//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package=composer -old-config-style -generate=types,client -o ../internal/composer/client.go  ../internal/composer/openapi.v2.yml
//go:generate mockgen -source=../internal/composer/client.go -package=composer -destination=../internal/composer/mock_osbuild_composer.go . ClientWithResponsesInterface

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/composer"
	repositoryosbuild "github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
)

var (
	uploadTypeForTargetImageType = map[osbuildv1alpha1.TargetImageType]composer.UploadTypes{
		osbuildv1alpha1.EdgeContainerImageType: composer.UploadTypesContainer,
		osbuildv1alpha1.EdgeInstallerImageType: composer.UploadTypesAwsS3,
	}
)

const (
	// Conditions Messages
	failedToSendPostRequestMsg      = "Failed to post a new composer build request"
	edgeContainerJobFinishedMsg     = "Edge-container job was finished successfully"
	EdgeContainerJobFailedMsg       = "Edge-container job was failed"
	EdgeContainerJobStillRunningMsg = "Edge-container job is still running"

	// OSBuildConditionTypes values
	containerBuildDone    = "containerBuildDone"
	failedContainerBuild  = "failedContainerBuild"
	startedContainerBuild = "startedContainerBuild"
	isoBuildDone          = "isoBuildDone"
	failedIsoBuild        = "failedIsoBuild"
	startedIsoBuild       = "startedIsoBuild"

	EmptyComposeID = ""
	emptyURL       = ""

	RequeueForLongDuration  = time.Minute * 2
	RequeueForShortDuration = time.Second * 10
)

// OSBuildReconciler reconciles a OSBuild object
type OSBuildReconciler struct {
	Scheme            *runtime.Scheme
	OSBuildRepository repositoryosbuild.Repository
	ComposerClient    composer.ClientWithResponsesInterface
}

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuilds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OSBuild object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OSBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("osbuild", req.Name)

	osBuild, err := r.OSBuildRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, fmt.Sprintf("OSBuild %s wasn't found", req.Name))
			return ctrl.Result{}, nil
		}
		logger.Error(err, fmt.Sprintf("OSBuild %s cannot be retrieve", req.Name))
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
	}

	if osBuild.DeletionTimestamp != nil {
		// The OSBuild CRs that were created by that OSBuildConfig would be deleted
		// thanks to setting controller reference for each OSBuild CR
		return ctrl.Result{}, nil
	}

	if osBuild.Spec.Details != nil {
		// build edge-container image
		return r.buildEdgeContainerImage(ctx, logger, osBuild)
	} else if osBuild.Spec.EdgeInstallerDetails != nil {
		return r.buildEdgeInstallerImage(ctx, logger, osBuild)
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildReconciler) buildEdgeContainerImage(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (ctrl.Result, error) {
	if osBuild.Status.ContainerComposeId == EmptyComposeID {
		// if the edge container wasn't created yet - schedule a new build
		logger.Info("create an edge-container")
		err := r.postComposeEdgeContainer(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to create an edge-container")
			return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
		}

		logger.Info("new job created for edge-container, requeue to sample its status")
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
	}

	lastBuildStatus := osbuildv1alpha1.OSBuildConditionType("")
	if osBuild.Status.Conditions != nil {
		conditionLen := len(osBuild.Status.Conditions)
		lastBuildStatus = osBuild.Status.Conditions[conditionLen-1].Type
	}

	switch lastBuildStatus {
	case startedContainerBuild:
		// if the edge container already created but wasn't finish yet - check the build status
		logger.Info("update the edge-container's compose ID job status")
		composeStatus, err := r.updateContainerComposeStatus(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to get compose ID status")
			return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
		}

		// the build is still in progress - requeue
		if composeStatus == composer.ComposeStatusValuePending {
			logger.Info(fmt.Sprintf("the job ID %s, is still in progress", osBuild.Status.ContainerComposeId))
			return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
		}

		return ctrl.Result{Requeue: true}, nil

	case failedContainerBuild:
		logger.Error(fmt.Errorf("failed to build edge container"), "")
		return ctrl.Result{}, nil

	case containerBuildDone:
		logger.Info(fmt.Sprintf("the job ID %s, Finished", osBuild.Status.ContainerComposeId))
		return ctrl.Result{}, nil

	default:
		logger.Error(fmt.Errorf("failed to parse condition status"), "")
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
	}
}

func (r *OSBuildReconciler) buildEdgeInstallerImage(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (ctrl.Result, error) {
	if osBuild.Status.IsoComposeId == EmptyComposeID {
		// if the edge installer build wasn't created yet - schedule a new build
		// [ECOPROJECT-917] TODO postComposeEdgeInstaller - schedule a new build
		return ctrl.Result{}, nil
	}

	lastBuildStatus := osbuildv1alpha1.OSBuildConditionType("")
	if osBuild.Status.Conditions != nil {
		conditionLen := len(osBuild.Status.Conditions)
		lastBuildStatus = osBuild.Status.Conditions[conditionLen-1].Type
	}

	switch lastBuildStatus {
	case startedIsoBuild:
		// if the edge installer already created but wasn't finish yet - check the build status
		logger.Info("update the edge-installer's compose ID job status")
		composeStatus, err := r.updateIsoComposeStatus(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to get compose ID status")
			return ctrl.Result{Requeue: true, RequeueAfter: RequeueForShortDuration}, nil
		}

		// the build is still in progress - requeue
		if composeStatus == composer.ComposeStatusValuePending {
			logger.Info(fmt.Sprintf("the job ID %s, is still in progress", osBuild.Status.IsoComposeId))
			return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
		}

		return ctrl.Result{Requeue: true}, nil

	case failedIsoBuild:
		// the build was failed - return with error
		logger.Error(fmt.Errorf("failed building the edge installer"), "")
		return ctrl.Result{}, nil

	case isoBuildDone:
		// the build was finished successfully - continue with repackaging the iso image
		// TODO repackaging the iso image with a kickstart file
		return ctrl.Result{}, nil

	default:
		logger.Error(fmt.Errorf("failed to parse condition status"), "")
		return ctrl.Result{Requeue: true, RequeueAfter: RequeueForLongDuration}, nil
	}
}

func (r *OSBuildReconciler) updateContainerComposeStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (composer.ComposeStatusValue, error) {
	composeStatus, err := r.checkComposeIDStatus(ctx, logger, osBuild.Status.ContainerComposeId)
	if err != nil {
		logger.Error(err, "failed to get compose ID status")
		return "", err
	}

	status := composeStatus.Status
	buildUrl, err := r.getBuildUrl(logger, composeStatus)
	if err != nil {
		return "", err
	}

	err = r.updateOSBuildConditionStatus(ctx, logger, osBuild, status, containerBuildDone, failedContainerBuild, startedContainerBuild, buildUrl, emptyURL)
	if err != nil {
		logger.Error(err, "failed to update OSBuild condition status")
		return "", err
	}
	return status, nil
}

func (r *OSBuildReconciler) getBuildUrl(logger logr.Logger, composeStatus *composer.ComposeStatus) (string, error) {
	if composeStatus.ImageStatus.UploadStatus == nil {
		logger.Info("field uploadStatus is nil")
		return emptyURL, nil
	}

	jsonUploadStatus, err := json.Marshal(composeStatus.ImageStatus.UploadStatus.Options)
	if err != nil {
		logger.Error(err, "cannot marshal the field `Options`")
		return emptyURL, err
	}

	var buildUrl string
	switch composeStatus.ImageStatus.UploadStatus.Type {
	case composer.UploadTypesAwsS3:
		var awsS3UploadStatus composer.AWSS3UploadStatus
		err = json.Unmarshal(jsonUploadStatus, &awsS3UploadStatus)
		if err != nil {
			logger.Error(err, "cannot convert the field `Options` to type AWSS3UploadStatus")
			return emptyURL, err
		}
		buildUrl = awsS3UploadStatus.Url
	case composer.UploadTypesContainer:
		var containerUploadStatus composer.ContainerUploadStatus
		err = json.Unmarshal(jsonUploadStatus, &containerUploadStatus)
		if err != nil {
			logger.Error(err, "cannot convert the field `Options` to type ContainerUploadStatus")
			return emptyURL, err
		}
		buildUrl = containerUploadStatus.Url
	default:
		return emptyURL, fmt.Errorf("unsupported upload status type %s", composeStatus.ImageStatus.UploadStatus.Type)
	}

	return buildUrl, nil
}

func (r *OSBuildReconciler) updateIsoComposeStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (composer.ComposeStatusValue, error) {
	composeStatus, err := r.checkComposeIDStatus(ctx, logger, osBuild.Status.IsoComposeId)
	if err != nil {
		logger.Error(err, "failed to get compose ID status")
		return "", err
	}

	status := composeStatus.Status
	buildUrl, err := r.getBuildUrl(logger, composeStatus)
	if err != nil {
		return "", err
	}

	err = r.updateOSBuildConditionStatus(ctx, logger, osBuild, status, isoBuildDone, failedIsoBuild, startedIsoBuild, emptyURL, buildUrl)
	if err != nil {
		logger.Error(err, "failed to update OSBuild condition status")
		return "", err
	}
	return status, nil
}

func (r *OSBuildReconciler) updateOSBuildConditionStatus(ctx context.Context, logger logr.Logger,
	osBuild *osbuildv1alpha1.OSBuild, composeStatus composer.ComposeStatusValue,
	buildDoneValue osbuildv1alpha1.OSBuildConditionType, buildFailedValue osbuildv1alpha1.OSBuildConditionType,
	buildStartedValue osbuildv1alpha1.OSBuildConditionType, edgeContainerUrl string, edgeInstallerUrl string) error {

	if composeStatus == composer.ComposeStatusValueSuccess {
		return r.updateOSBuildStatus(ctx, logger, osBuild, edgeContainerJobFinishedMsg, buildDoneValue, EmptyComposeID, EmptyComposeID, edgeContainerUrl, edgeInstallerUrl)
	}

	if composeStatus == composer.ComposeStatusValueFailure {
		return r.updateOSBuildStatus(ctx, logger, osBuild, EdgeContainerJobFailedMsg, buildFailedValue, EmptyComposeID, EmptyComposeID, edgeContainerUrl, edgeInstallerUrl)
	}

	if composeStatus == composer.ComposeStatusValuePending {
		return r.updateOSBuildStatus(ctx, logger, osBuild, EdgeContainerJobStillRunningMsg, buildStartedValue, EmptyComposeID, EmptyComposeID, edgeContainerUrl, edgeInstallerUrl)
	}

	return nil
}

func (r *OSBuildReconciler) postComposeEdgeContainer(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) error {
	customizations := r.createCustomizations(osBuild.Spec.Details.Customizations)

	imageRequest, err := r.createImageRequest(osBuild, osbuildv1alpha1.EdgeContainerImageType)
	if err != nil {
		return err
	}

	body := composer.PostComposeJSONRequestBody{
		Customizations: customizations,
		Distribution:   osBuild.Spec.Details.Distribution,
		ImageRequest:   imageRequest,
	}

	// post compose:
	composerResponse, err := r.ComposerClient.PostComposeWithResponse(ctx, body)
	if err != nil {
		logger.Error(err, "failed to post a new request")

		errUpdating := r.updateOSBuildStatus(ctx, logger, osBuild, failedToSendPostRequestMsg, failedContainerBuild, EmptyComposeID, EmptyComposeID, emptyURL, emptyURL)
		if errUpdating != nil {
			logger.Error(errUpdating, "failed to update OSBuild condition status")
		}
		return err
	}

	if composerResponse.StatusCode() != http.StatusCreated {
		errorMsg := fmt.Sprintf("postCompose request failed for OSBuild %s, with status code %v, and body %s", osBuild.Name, composerResponse.StatusCode(), string(composerResponse.Body))
		err = fmt.Errorf(errorMsg)
		logger.Error(err, "postCompose request failed")

		errUpdating := r.updateOSBuildStatus(ctx, logger, osBuild, errorMsg, failedContainerBuild, EmptyComposeID, EmptyComposeID, emptyURL, emptyURL)
		if errUpdating != nil {
			logger.Error(errUpdating, "failed to update OSBuild condition status")
		}
		return err
	}

	containerComposeId := composerResponse.JSON201.Id.String()
	logger.Info("postComposer request was sent and trigger a new compose ID ", "container compose ID: ", containerComposeId)

	return r.updateOSBuildStatus(ctx, logger, osBuild, EdgeContainerJobStillRunningMsg, startedContainerBuild, containerComposeId, EmptyComposeID, emptyURL, emptyURL)
}

func (r *OSBuildReconciler) updateOSBuildStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild,
	msg string, conditionType osbuildv1alpha1.OSBuildConditionType, containerComposeId string, isoComposeId string,
	edgeContainerUrl string, edgeInstallerUrl string) error {
	patch := client.MergeFrom(osBuild.DeepCopy())
	if containerComposeId != EmptyComposeID {
		osBuild.Status.ContainerComposeId = containerComposeId
	}

	if isoComposeId != EmptyComposeID {
		osBuild.Status.IsoComposeId = isoComposeId
	}

	if edgeContainerUrl != emptyURL {
		osBuild.Status.ContainerUrl = edgeContainerUrl
	}

	if edgeInstallerUrl != emptyURL {
		osBuild.Status.IsoUrl = edgeInstallerUrl
	}

	if osBuild.Status.Conditions == nil {
		osBuild.Status.Conditions = []osbuildv1alpha1.OSBuildCondition{}
	}

	conditionArrLen := len(osBuild.Status.Conditions)
	if conditionArrLen > 0 {
		lastConditionType := osBuild.Status.Conditions[conditionArrLen-1].Type
		if lastConditionType == conditionType {
			logger.Info("conditionType did not change ", " lastConditionType ", lastConditionType)
			return nil
		}
	}

	osBuild.Status.Conditions = append(osBuild.Status.Conditions, osbuildv1alpha1.OSBuildCondition{
		Type:    conditionType,
		Message: &msg,
	})

	errPatch := r.OSBuildRepository.PatchStatus(ctx, osBuild, &patch)
	if errPatch != nil {
		logger.Error(errPatch, "Failed to patch OSBuild status")
		return errPatch
	}

	return nil
}

func (r *OSBuildReconciler) checkComposeIDStatus(ctx context.Context, logger logr.Logger, composeID string) (*composer.ComposeStatus, error) {
	composerResponse, err := r.ComposerClient.GetComposeStatusWithResponse(ctx, composeID)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to get compose ID %s status", composeID))
		return nil, err
	}

	if composerResponse.JSON200 != nil {
		logger.Info(fmt.Sprintf("Image building status %v", composerResponse.JSON200.ImageStatus))
		return composerResponse.JSON200, nil

	}
	return nil, fmt.Errorf("something went wrong with requesting the composeID %v", composerResponse.StatusCode())
}

func (r *OSBuildReconciler) createImageRequest(osBuild *osbuildv1alpha1.OSBuild, targetImageType osbuildv1alpha1.TargetImageType) (*composer.ImageRequest, error) {
	uploadOptions, err := r.getUploadOptions(osBuild, targetImageType)
	if err != nil {
		return nil, err
	}

	// TODO[ECOPROJECT-902]- add repositories to OSBuildConfig and OSBuildConfigTemplate types
	imageRequest := composer.ImageRequest{
		Architecture:  string(osBuild.Spec.Details.TargetImage.Architecture),
		ImageType:     composer.ImageTypes(targetImageType),
		UploadOptions: uploadOptions,
	}

	if osBuild.Spec.Details.TargetImage.Repositories != nil {
		var repos []composer.Repository
		for _, osbuildRepo := range *osBuild.Spec.Details.TargetImage.Repositories {
			composerRepo := osbuildRepo.DeepCopy()
			repos = append(repos, (composer.Repository)(*composerRepo))
		}
		imageRequest.Repositories = repos
	}

	if osBuild.Spec.Details.TargetImage.OSTree != nil {
		imageRequest.Ostree = (*composer.OSTree)(osBuild.Spec.Details.TargetImage.OSTree.DeepCopy())
	}

	return &imageRequest, nil
}

func (r *OSBuildReconciler) getUploadOptions(osBuild *osbuildv1alpha1.OSBuild, targetImageType osbuildv1alpha1.TargetImageType) (*composer.UploadOptions, error) {
	var uploadOptions composer.UploadOptions
	switch uploadTypeForTargetImageType[targetImageType] {
	case composer.UploadTypesAwsS3:
		uploadOptions = composer.UploadOptions(composer.AWSS3UploadOptions{Region: ""})
	case composer.UploadTypesContainer:
		splitName := strings.Split(osBuild.Name, "-")
		imageName := fmt.Sprintf("%s/%s", osBuild.Namespace, strings.Join(splitName[:len(splitName)-1], ""))
		imageTag := splitName[len(splitName)-1]
		uploadOptions = composer.UploadOptions(composer.ContainerUploadOptions{Name: &imageName, Tag: &imageTag})
	default:
		return nil, fmt.Errorf("unsupported TargetImageType: %s", targetImageType)
	}
	return &uploadOptions, nil
}

func (r *OSBuildReconciler) createCustomizations(osbuildCustomizations *osbuildv1alpha1.Customizations) *composer.Customizations {
	if osbuildCustomizations == nil {
		return nil
	}

	customizationIsEmpty := true
	composerCustomizations := composer.Customizations{}
	if osbuildCustomizations.Users != nil && len(osbuildCustomizations.Users) > 0 {
		var users []composer.User
		for _, cstmzUser := range osbuildCustomizations.Users {
			user := cstmzUser.DeepCopy()
			users = append(users, (composer.User)(*user))
		}
		customizationIsEmpty = false
		composerCustomizations.Users = &users
	}

	if osbuildCustomizations.Services != nil {
		var services struct {
			Disabled *[]string `json:"disabled,omitempty"`
			Enabled  *[]string `json:"enabled,omitempty"`
		}

		if osbuildCustomizations.Services.Enabled != nil && len(osbuildCustomizations.Services.Enabled) > 0 {
			customizationIsEmpty = false
			services.Enabled = &osbuildCustomizations.Services.Enabled
			composerCustomizations.Services = &services
		}
		if osbuildCustomizations.Services.Disabled != nil && len(osbuildCustomizations.Services.Disabled) > 0 {
			customizationIsEmpty = false
			services.Disabled = &osbuildCustomizations.Services.Disabled
			composerCustomizations.Services = &services
		}
	}
	if osbuildCustomizations.Packages != nil {
		customizationIsEmpty = false
		composerCustomizations.Packages = &osbuildCustomizations.Packages
	}

	if customizationIsEmpty {
		return nil
	}

	return &composerCustomizations
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&osbuildv1alpha1.OSBuild{}).
		Complete(r)
}
