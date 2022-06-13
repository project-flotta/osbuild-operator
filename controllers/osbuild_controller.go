//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package=composer -old-config-style -generate=types,client -o ../internal/composer/client.go  ../internal/composer/openapi.v2.yml

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
	"fmt"
	"net/http"
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

const (
	// Conditions Messages
	failedToSendPostRequestMsg      = "Failed to post a new composer build request"
	edgeContainerJobFinishedMsg     = "Edge-container job was finished successfully"
	edgeContainerJobFailedMsg       = "Edge-container job was failed"
	edgeContainerJobStillRunningMsg = "Edge-container job is still running"

	// OSBuildConditionTypes values
	containerBuildDone    = "containerBuildDone"
	failedContainerBuild  = "failedContainerBuild"
	startedContainerBuild = "startedContainerBuild"
	isoBuildDone          = "isoBuildDone"
	failedIsoBuild        = "failedIsoBuild"
	startedIsoBuild       = "startedIsoBuild"

	// Image types
	edgeContainerImgType = "edge-container"
	//edgeInstallerImgType = "edge-installer"

	emptyComposeID = ""

	requeueForLongDuration  = time.Minute * 2
	requeueForShortDuration = time.Second * 10
)

// OSBuildReconciler reconciles a OSBuild object
type OSBuildReconciler struct {
	Scheme            *runtime.Scheme
	OSBuildRepository repositoryosbuild.Repository
	ComposerClient    *composer.Client
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
		return ctrl.Result{Requeue: true, RequeueAfter: requeueForShortDuration}, nil
	}

	if osBuild.DeletionTimestamp != nil {
		// The OSBuild CRs that were created by that OSBuildConfig would be deleted
		// thanks to setting controller reference for each OSBuild CR
		return ctrl.Result{}, nil
	}

	lastBuildStatus := osbuildv1alpha1.OSBuildConditionType("")
	if osBuild.Status.Conditions != nil {
		conditionLen := len(osBuild.Status.Conditions)
		lastBuildStatus = osBuild.Status.Conditions[conditionLen-1].Type
	}

	if osBuild.Status.ContainerComposeId == emptyComposeID {
		// if the edge container wasn't created yet - schedule a new build
		logger.Info("create an edge-container")
		err = r.postComposeEdgeContainer(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to create an edge-container")
			return ctrl.Result{Requeue: true, RequeueAfter: requeueForLongDuration}, nil
		}

		logger.Info("new job created for edge-container, requeue to sample its status")
		return ctrl.Result{Requeue: true, RequeueAfter: requeueForLongDuration}, nil
	}

	if lastBuildStatus == startedContainerBuild {
		// if the edge container already created but wasn't finish yet - check the build status
		logger.Info("update the edge-container's compose ID job status")
		composeStatus, err := r.updateContainerComposeStatus(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to get compose ID status")
			return ctrl.Result{Requeue: true, RequeueAfter: requeueForShortDuration}, nil
		}

		// the build is still in progress - requeue
		if composeStatus == composer.ComposeStatusValuePending {
			logger.Info(fmt.Sprintf("the job ID %s, is still in progress", osBuild.Status.ContainerComposeId))
			return ctrl.Result{Requeue: true, RequeueAfter: requeueForLongDuration}, nil
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if lastBuildStatus == failedContainerBuild {
		logger.Error(fmt.Errorf("failed to build edge container"), "")
		return ctrl.Result{}, nil
	}

	// if the build was finished successfully and the target image is edge-container then return
	if lastBuildStatus == containerBuildDone && osBuild.Spec.Details.TargetImage.TargetImageType == edgeContainerImgType {
		logger.Info(fmt.Sprintf("the job ID %s, Finished", osBuild.Status.ContainerComposeId))
		return ctrl.Result{}, nil
	}

	if osBuild.Status.IsoComposeId == emptyComposeID {
		// if the edge installer build wasn't created yet - schedule a new build
		// TODO postComposeEdgeInstaller - schedule a new build
		return ctrl.Result{}, nil
	}

	if lastBuildStatus == startedIsoBuild {
		// if the edge installer already created but wasn't finish yet - check the build status
		logger.Info("update the edge-installer's compose ID job status")
		composeStatus, err := r.updateIsoComposeStatus(ctx, logger, osBuild)
		if err != nil {
			logger.Error(err, "failed to get compose ID status")
			return ctrl.Result{Requeue: true, RequeueAfter: requeueForShortDuration}, nil
		}

		// the build is still in progress - requeue
		if composeStatus == composer.ComposeStatusValuePending {
			logger.Info(fmt.Sprintf("the job ID %s, is still in progress", osBuild.Status.IsoComposeId))
			return ctrl.Result{Requeue: true, RequeueAfter: requeueForLongDuration}, nil
		}
	}

	// the build was failed - return with error
	if lastBuildStatus == failedIsoBuild {
		logger.Error(fmt.Errorf("failed building the edge installer"), "")
		return ctrl.Result{}, nil
	}

	// the build was finished successfully - continue with repackaging the iso image
	if lastBuildStatus == isoBuildDone {
		// TODO repackaging the iso image with a kickstart file
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildReconciler) updateContainerComposeStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (composer.ComposeStatusValue, error) {
	composeStatus, err := r.checkComposeIDStatus(ctx, logger, osBuild.Status.ContainerComposeId)
	if err != nil {
		logger.Error(err, "failed to get compose ID status")
		return "", err
	}

	err = r.updateOSBuildConditionStatus(ctx, logger, osBuild, composeStatus, containerBuildDone, failedContainerBuild, startedContainerBuild)
	if err != nil {
		logger.Error(err, "failed to update OSBuild condition status")
		return "", err
	}
	return composeStatus, nil
}

func (r *OSBuildReconciler) updateIsoComposeStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) (composer.ComposeStatusValue, error) {
	composeStatus, err := r.checkComposeIDStatus(ctx, logger, osBuild.Status.IsoComposeId)
	if err != nil {
		logger.Error(err, "failed to get compose ID status")
		return "", err
	}

	err = r.updateOSBuildConditionStatus(ctx, logger, osBuild, composeStatus, isoBuildDone, failedIsoBuild, startedIsoBuild)
	if err != nil {
		logger.Error(err, "failed to update OSBuild condition status")
		return "", err
	}
	return composeStatus, nil
}

func (r *OSBuildReconciler) updateOSBuildConditionStatus(ctx context.Context, logger logr.Logger,
	osBuild *osbuildv1alpha1.OSBuild, composeStatus composer.ComposeStatusValue,
	buildDoneValue osbuildv1alpha1.OSBuildConditionType, buildFailedValue osbuildv1alpha1.OSBuildConditionType,
	buildStartedValue osbuildv1alpha1.OSBuildConditionType) error {

	if composeStatus == composer.ComposeStatusValueSuccess {
		return r.updateOSBuildStatus(ctx, logger, osBuild, edgeContainerJobFinishedMsg, buildDoneValue, emptyComposeID, emptyComposeID)
	}

	if composeStatus == composer.ComposeStatusValueFailure {
		return r.updateOSBuildStatus(ctx, logger, osBuild, edgeContainerJobFailedMsg, buildFailedValue, emptyComposeID, emptyComposeID)
	}

	if composeStatus == composer.ComposeStatusValuePending {
		return r.updateOSBuildStatus(ctx, logger, osBuild, edgeContainerJobStillRunningMsg, buildStartedValue, emptyComposeID, emptyComposeID)
	}

	return nil
}

func (r *OSBuildReconciler) postComposeEdgeContainer(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild) error {
	customizations := r.createCustomizations(osBuild.Spec.Details.Customizations)
	imageRequest := r.createImageRequest(&osBuild.Spec.Details.TargetImage, edgeContainerImgType)

	body := composer.PostComposeJSONRequestBody{
		Customizations: customizations,
		Distribution:   osBuild.Spec.Details.Distribution,
		ImageRequest:   imageRequest,
	}

	// post compos:
	response, err := r.ComposerClient.PostCompose(ctx, body)
	if err != nil {
		logger.Error(err, "failed to post a new request")
		errUpdating := r.updateOSBuildStatus(ctx, logger, osBuild, failedToSendPostRequestMsg, failedContainerBuild, emptyComposeID, emptyComposeID)
		if errUpdating != nil {
			logger.Error(errUpdating, "failed to update OSBuild condition status")
		}
		return err
	}

	composerResponse, err := composer.ParsePostComposeResponse(response)
	if err != nil {
		logger.Error(err, "failed parsing the response of postCompose")
		return err
	}
	if composerResponse.StatusCode() != http.StatusCreated {
		errorMsg := fmt.Sprintf("postCompose request failed for OSBuild %s, with status code %v, and body %s", osBuild.Name, composerResponse.StatusCode(), string(composerResponse.Body))
		err = fmt.Errorf(errorMsg)
		logger.Error(err, "postCompose request failed")
		errUpdating := r.updateOSBuildStatus(ctx, logger, osBuild, errorMsg, failedContainerBuild, emptyComposeID, emptyComposeID)
		if errUpdating != nil {
			logger.Error(errUpdating, "failed to update OSBuild condition status")
		}
		return err
	}

	containerComposeId := composerResponse.JSON201.Id.String()
	logger.Info("postComposer request was sent and trigger a new compose ID %s", containerComposeId)

	return r.updateOSBuildStatus(ctx, logger, osBuild, edgeContainerJobStillRunningMsg, startedContainerBuild, containerComposeId, emptyComposeID)
}

func (r *OSBuildReconciler) updateOSBuildStatus(ctx context.Context, logger logr.Logger, osBuild *osbuildv1alpha1.OSBuild,
	msg string, conditionType osbuildv1alpha1.OSBuildConditionType, containerComposeId string, isoComposeId string) error {
	patch := client.MergeFrom(osBuild.DeepCopy())
	if containerComposeId != emptyComposeID {
		osBuild.Status.ContainerComposeId = containerComposeId
	}

	if isoComposeId != emptyComposeID {
		osBuild.Status.IsoComposeId = isoComposeId
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

func (r *OSBuildReconciler) checkComposeIDStatus(ctx context.Context, logger logr.Logger, composeID string) (composer.ComposeStatusValue, error) {
	response, err := r.ComposerClient.GetComposeStatus(ctx, composeID)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to get compose ID %s status", composeID))
		return "", err
	}
	composerResponse, err := composer.ParseGetComposeStatusResponse(response)
	if err != nil {
		logger.Error(err, "failed to parse getCompose response")
		return "", err
	}
	if composerResponse.JSON200 != nil {
		return composerResponse.JSON200.Status, nil
	}
	return "", fmt.Errorf("something went wrong with requesting the composeID %v", composerResponse.StatusCode())
}
func (r *OSBuildReconciler) buildRepositories() []composer.Repository {
	baseosUrl := "https://cdn.redhat.com/content/dist/rhel8/8.5/x86_64/baseos/os"
	baseosgpgkey := "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBErgSTsBEACh2A4b0O9t+vzC9VrVtL1AKvUWi9OPCjkvR7Xd8DtJxeeMZ5eF\n0HtzIG58qDRybwUe89FZprB1ffuUKzdE+HcL3FbNWSSOXVjZIersdXyH3NvnLLLF\n0DNRB2ix3bXG9Rh/RXpFsNxDp2CEMdUvbYCzE79K1EnUTVh1L0Of023FtPSZXX0c\nu7Pb5DI5lX5YeoXO6RoodrIGYJsVBQWnrWw4xNTconUfNPk0EGZtEnzvH2zyPoJh\nXGF+Ncu9XwbalnYde10OCvSWAZ5zTCpoLMTvQjWpbCdWXJzCm6G+/hx9upke546H\n5IjtYm4dTIVTnc3wvDiODgBKRzOl9rEOCIgOuGtDxRxcQkjrC+xvg5Vkqn7vBUyW\n9pHedOU+PoF3DGOM+dqv+eNKBvh9YF9ugFAQBkcG7viZgvGEMGGUpzNgN7XnS1gj\n/DPo9mZESOYnKceve2tIC87p2hqjrxOHuI7fkZYeNIcAoa83rBltFXaBDYhWAKS1\nPcXS1/7JzP0ky7d0L6Xbu/If5kqWQpKwUInXtySRkuraVfuK3Bpa+X1XecWi24JY\nHVtlNX025xx1ewVzGNCTlWn1skQN2OOoQTV4C8/qFpTW6DTWYurd4+fE0OJFJZQF\nbuhfXYwmRlVOgN5i77NTIJZJQfYFj38c/Iv5vZBPokO6mffrOTv3MHWVgQARAQAB\ntDNSZWQgSGF0LCBJbmMuIChyZWxlYXNlIGtleSAyKSA8c2VjdXJpdHlAcmVkaGF0\nLmNvbT6JAjYEEwECACAFAkrgSTsCGwMGCwkIBwMCBBUCCAMEFgIDAQIeAQIXgAAK\nCRAZni+R/UMdUWzpD/9s5SFR/ZF3yjY5VLUFLMXIKUztNN3oc45fyLdTI3+UClKC\n2tEruzYjqNHhqAEXa2sN1fMrsuKec61Ll2NfvJjkLKDvgVIh7kM7aslNYVOP6BTf\nC/JJ7/ufz3UZmyViH/WDl+AYdgk3JqCIO5w5ryrC9IyBzYv2m0HqYbWfphY3uHw5\nun3ndLJcu8+BGP5F+ONQEGl+DRH58Il9Jp3HwbRa7dvkPgEhfFR+1hI+Btta2C7E\n0/2NKzCxZw7Lx3PBRcU92YKyaEihfy/aQKZCAuyfKiMvsmzs+4poIX7I9NQCJpyE\nIGfINoZ7VxqHwRn/d5mw2MZTJjbzSf+Um9YJyA0iEEyD6qjriWQRbuxpQXmlAJbh\n8okZ4gbVFv1F8MzK+4R8VvWJ0XxgtikSo72fHjwha7MAjqFnOq6eo6fEC/75g3NL\nGht5VdpGuHk0vbdENHMC8wS99e5qXGNDued3hlTavDMlEAHl34q2H9nakTGRF5Ki\nJUfNh3DVRGhg8cMIti21njiRh7gyFI2OccATY7bBSr79JhuNwelHuxLrCFpY7V25\nOFktl15jZJaMxuQBqYdBgSay2G0U6D1+7VsWufpzd/Abx1/c3oi9ZaJvW22kAggq\ndzdA27UUYjWvx42w9menJwh/0jeQcTecIUd0d0rFcw/c1pvgMMl/Q73yzKgKYw==\n=zbHE\n-----END PGP PUBLIC KEY BLOCK-----\n-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBFsy23UBEACUKSphFEIEvNpy68VeW4Dt6qv+mU6am9a2AAl10JANLj1oqWX+\noYk3en1S6cVe2qehSL5DGVa3HMUZkP3dtbD4SgzXzxPodebPcr4+0QNWigkUisri\nXGL5SCEcOP30zDhZvg+4mpO2jMi7Kc1DLPzBBkgppcX91wa0L1pQzBcvYMPyV/Dh\nKbQHR75WdkP6OA2JXdfC94nxYq+2e0iPqC1hCP3Elh+YnSkOkrawDPmoB1g4+ft/\nxsiVGVy/W0ekXmgvYEHt6si6Y8NwXgnTMqxeSXQ9YUgVIbTpsxHQKGy76T5lMlWX\n4LCOmEVomBJg1SqF6yi9Vu8TeNThaDqT4/DddYInd0OO69s0kGIXalVgGYiW2HOD\nx2q5R1VGCoJxXomz+EbOXY+HpKPOHAjU0DB9MxbU3S248LQ69nIB5uxysy0PSco1\nsdZ8sxRNQ9Dw6on0Nowx5m6Thefzs5iK3dnPGBqHTT43DHbnWc2scjQFG+eZhe98\nEll/kb6vpBoY4bG9/wCG9qu7jj9Z+BceCNKeHllbezVLCU/Hswivr7h2dnaEFvPD\nO4GqiWiwOF06XaBMVgxA8p2HRw0KtXqOpZk+o+sUvdPjsBw42BB96A1yFX4jgFNA\nPyZYnEUdP6OOv9HSjnl7k/iEkvHq/jGYMMojixlvXpGXhnt5jNyc4GSUJQARAQAB\ntDNSZWQgSGF0LCBJbmMuIChhdXhpbGlhcnkga2V5KSA8c2VjdXJpdHlAcmVkaGF0\nLmNvbT6JAjkEEwECACMFAlsy23UCGwMHCwkIBwMCAQYVCAIJCgsEFgIDAQIeAQIX\ngAAKCRD3b2bD1AgnknqOD/9fB2ASuG2aJIiap4kK58R+RmOVM4qgclAnaG57+vjI\nnKvyfV3NH/keplGNRxwqHekfPCqvkpABwhdGEXIE8ILqnPewIMr6PZNZWNJynZ9i\neSMzVuCG7jDoGyQ5/6B0f6xeBtTeBDiRl7+Alehet1twuGL1BJUYG0QuLgcEzkaE\n/gkuumeVcazLzz7L12D22nMk66GxmgXfqS5zcbqOAuZwaA6VgSEgFdV2X2JU79zS\nBQJXv7NKc+nDXFG7M7EHjY3Rma3HXkDbkT8bzh9tJV7Z7TlpT829pStWQyoxKCVq\nsEX8WsSapTKA3P9YkYCwLShgZu4HKRFvHMaIasSIZWzLu+RZH/4yyHOhj0QB7XMY\neHQ6fGSbtJ+K6SrpHOOsKQNAJ0hVbSrnA1cr5+2SDfel1RfYt0W9FA6DoH/S5gAR\ndzT1u44QVwwp3U+eFpHphFy//uzxNMtCjjdkpzhYYhOCLNkDrlRPb+bcoL/6ePSr\n016PA7eEnuC305YU1Ml2WcCn7wQV8x90o33klJmEkWtXh3X39vYtI4nCPIvZn1eP\nVy+F+wWt4vN2b8oOdlzc2paOembbCo2B+Wapv5Y9peBvlbsDSgqtJABfK8KQq/jK\nYl3h5elIa1I3uNfczeHOnf1enLOUOlq630yeM/yHizz99G1g+z/guMh5+x/OHraW\niA==\n=+Gxh\n-----END PGP PUBLIC KEY BLOCK-----\n"

	appstreamUrl := "https://cdn.redhat.com/content/dist/rhel8/8.5/x86_64/appstream/os"
	appstreamgpgkey := "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBErgSTsBEACh2A4b0O9t+vzC9VrVtL1AKvUWi9OPCjkvR7Xd8DtJxeeMZ5eF\n0HtzIG58qDRybwUe89FZprB1ffuUKzdE+HcL3FbNWSSOXVjZIersdXyH3NvnLLLF\n0DNRB2ix3bXG9Rh/RXpFsNxDp2CEMdUvbYCzE79K1EnUTVh1L0Of023FtPSZXX0c\nu7Pb5DI5lX5YeoXO6RoodrIGYJsVBQWnrWw4xNTconUfNPk0EGZtEnzvH2zyPoJh\nXGF+Ncu9XwbalnYde10OCvSWAZ5zTCpoLMTvQjWpbCdWXJzCm6G+/hx9upke546H\n5IjtYm4dTIVTnc3wvDiODgBKRzOl9rEOCIgOuGtDxRxcQkjrC+xvg5Vkqn7vBUyW\n9pHedOU+PoF3DGOM+dqv+eNKBvh9YF9ugFAQBkcG7viZgvGEMGGUpzNgN7XnS1gj\n/DPo9mZESOYnKceve2tIC87p2hqjrxOHuI7fkZYeNIcAoa83rBltFXaBDYhWAKS1\nPcXS1/7JzP0ky7d0L6Xbu/If5kqWQpKwUInXtySRkuraVfuK3Bpa+X1XecWi24JY\nHVtlNX025xx1ewVzGNCTlWn1skQN2OOoQTV4C8/qFpTW6DTWYurd4+fE0OJFJZQF\nbuhfXYwmRlVOgN5i77NTIJZJQfYFj38c/Iv5vZBPokO6mffrOTv3MHWVgQARAQAB\ntDNSZWQgSGF0LCBJbmMuIChyZWxlYXNlIGtleSAyKSA8c2VjdXJpdHlAcmVkaGF0\nLmNvbT6JAjYEEwECACAFAkrgSTsCGwMGCwkIBwMCBBUCCAMEFgIDAQIeAQIXgAAK\nCRAZni+R/UMdUWzpD/9s5SFR/ZF3yjY5VLUFLMXIKUztNN3oc45fyLdTI3+UClKC\n2tEruzYjqNHhqAEXa2sN1fMrsuKec61Ll2NfvJjkLKDvgVIh7kM7aslNYVOP6BTf\nC/JJ7/ufz3UZmyViH/WDl+AYdgk3JqCIO5w5ryrC9IyBzYv2m0HqYbWfphY3uHw5\nun3ndLJcu8+BGP5F+ONQEGl+DRH58Il9Jp3HwbRa7dvkPgEhfFR+1hI+Btta2C7E\n0/2NKzCxZw7Lx3PBRcU92YKyaEihfy/aQKZCAuyfKiMvsmzs+4poIX7I9NQCJpyE\nIGfINoZ7VxqHwRn/d5mw2MZTJjbzSf+Um9YJyA0iEEyD6qjriWQRbuxpQXmlAJbh\n8okZ4gbVFv1F8MzK+4R8VvWJ0XxgtikSo72fHjwha7MAjqFnOq6eo6fEC/75g3NL\nGht5VdpGuHk0vbdENHMC8wS99e5qXGNDued3hlTavDMlEAHl34q2H9nakTGRF5Ki\nJUfNh3DVRGhg8cMIti21njiRh7gyFI2OccATY7bBSr79JhuNwelHuxLrCFpY7V25\nOFktl15jZJaMxuQBqYdBgSay2G0U6D1+7VsWufpzd/Abx1/c3oi9ZaJvW22kAggq\ndzdA27UUYjWvx42w9menJwh/0jeQcTecIUd0d0rFcw/c1pvgMMl/Q73yzKgKYw==\n=zbHE\n-----END PGP PUBLIC KEY BLOCK-----\n-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBFsy23UBEACUKSphFEIEvNpy68VeW4Dt6qv+mU6am9a2AAl10JANLj1oqWX+\noYk3en1S6cVe2qehSL5DGVa3HMUZkP3dtbD4SgzXzxPodebPcr4+0QNWigkUisri\nXGL5SCEcOP30zDhZvg+4mpO2jMi7Kc1DLPzBBkgppcX91wa0L1pQzBcvYMPyV/Dh\nKbQHR75WdkP6OA2JXdfC94nxYq+2e0iPqC1hCP3Elh+YnSkOkrawDPmoB1g4+ft/\nxsiVGVy/W0ekXmgvYEHt6si6Y8NwXgnTMqxeSXQ9YUgVIbTpsxHQKGy76T5lMlWX\n4LCOmEVomBJg1SqF6yi9Vu8TeNThaDqT4/DddYInd0OO69s0kGIXalVgGYiW2HOD\nx2q5R1VGCoJxXomz+EbOXY+HpKPOHAjU0DB9MxbU3S248LQ69nIB5uxysy0PSco1\nsdZ8sxRNQ9Dw6on0Nowx5m6Thefzs5iK3dnPGBqHTT43DHbnWc2scjQFG+eZhe98\nEll/kb6vpBoY4bG9/wCG9qu7jj9Z+BceCNKeHllbezVLCU/Hswivr7h2dnaEFvPD\nO4GqiWiwOF06XaBMVgxA8p2HRw0KtXqOpZk+o+sUvdPjsBw42BB96A1yFX4jgFNA\nPyZYnEUdP6OOv9HSjnl7k/iEkvHq/jGYMMojixlvXpGXhnt5jNyc4GSUJQARAQAB\ntDNSZWQgSGF0LCBJbmMuIChhdXhpbGlhcnkga2V5KSA8c2VjdXJpdHlAcmVkaGF0\nLmNvbT6JAjkEEwECACMFAlsy23UCGwMHCwkIBwMCAQYVCAIJCgsEFgIDAQIeAQIX\ngAAKCRD3b2bD1AgnknqOD/9fB2ASuG2aJIiap4kK58R+RmOVM4qgclAnaG57+vjI\nnKvyfV3NH/keplGNRxwqHekfPCqvkpABwhdGEXIE8ILqnPewIMr6PZNZWNJynZ9i\neSMzVuCG7jDoGyQ5/6B0f6xeBtTeBDiRl7+Alehet1twuGL1BJUYG0QuLgcEzkaE\n/gkuumeVcazLzz7L12D22nMk66GxmgXfqS5zcbqOAuZwaA6VgSEgFdV2X2JU79zS\nBQJXv7NKc+nDXFG7M7EHjY3Rma3HXkDbkT8bzh9tJV7Z7TlpT829pStWQyoxKCVq\nsEX8WsSapTKA3P9YkYCwLShgZu4HKRFvHMaIasSIZWzLu+RZH/4yyHOhj0QB7XMY\neHQ6fGSbtJ+K6SrpHOOsKQNAJ0hVbSrnA1cr5+2SDfel1RfYt0W9FA6DoH/S5gAR\ndzT1u44QVwwp3U+eFpHphFy//uzxNMtCjjdkpzhYYhOCLNkDrlRPb+bcoL/6ePSr\n016PA7eEnuC305YU1Ml2WcCn7wQV8x90o33klJmEkWtXh3X39vYtI4nCPIvZn1eP\nVy+F+wWt4vN2b8oOdlzc2paOembbCo2B+Wapv5Y9peBvlbsDSgqtJABfK8KQq/jK\nYl3h5elIa1I3uNfczeHOnf1enLOUOlq630yeM/yHizz99G1g+z/guMh5+x/OHraW\niA==\n=+Gxh\n-----END PGP PUBLIC KEY BLOCK-----\n"
	checkGpg := true
	rhsm := true
	return []composer.Repository{
		{
			Baseurl:  &baseosUrl,
			CheckGpg: &checkGpg,
			Gpgkey:   &baseosgpgkey,
			Rhsm:     &rhsm,
		},
		{
			Baseurl:  &appstreamUrl,
			CheckGpg: &checkGpg,
			Gpgkey:   &appstreamgpgkey,
			Rhsm:     &rhsm,
		},
	}
}

func (r *OSBuildReconciler) createImageRequest(targetImage *osbuildv1alpha1.TargetImage, targetImageType osbuildv1alpha1.TargetImageType) *composer.ImageRequest {
	uploadOptions := composer.UploadOptions(composer.AWSS3UploadOptions{Region: ""})

	// TODO[ECOPROJECT-902]- add repositories to OSBuildConfig and OSBuildConfigTemplate types
	imageRequest := composer.ImageRequest{
		Architecture:  string(targetImage.Architecture),
		ImageType:     composer.ImageTypes(targetImageType),
		UploadOptions: &uploadOptions,
		Repositories:  r.buildRepositories(),
	}
	if targetImage.OSTree != nil {
		imageRequest.Ostree = (*composer.OSTree)(targetImage.OSTree.DeepCopy())
	}
	return &imageRequest
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
