package controllers_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/controllers"
	"github.com/project-flotta/osbuild-operator/internal/composer"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
)

var _ = Describe("OSBuild Controller", func() {
	const (
		instanceNamespace = "osbuild"
		instanceName      = "osbuild_test"

		distribution = "rhel-86"
		triggeredBy  = "UpdateCR"
		architecture = "x86_64"

		// Conditions Messages
		failedToSendPostRequestMsg = "Failed to post a new composer build request"
		buildJobFinishedMsg        = "Build job was finished successfully"
		buildJobFailedMsg          = "Build job was failed"
		buildJobStillRunningMsg    = "Build job is still running"
	)
	var (
		mockCtrl          *gomock.Controller
		scheme            *runtime.Scheme
		osBuildRepository *osbuild.MockRepository
		composerClient    *composer.MockClientWithResponsesInterface
		reconciler        *controllers.OSBuildReconciler
		requestContext    context.Context
		osbuildInstance   *osbuildv1alpha1.OSBuild

		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      instanceName,
				Namespace: instanceNamespace,
			},
		}

		composerPostResponseCreated         composer.PostComposeResponse
		composerPostResponseFailed          composer.PostComposeResponse
		composerGetStatusFailed             composer.GetComposeStatusResponse
		composerGetStatusDone               composer.GetComposeStatusResponse
		composerGetStatusPending            composer.GetComposeStatusResponse
		composerGetStatusResponseBadRequest composer.GetComposeStatusResponse

		resultShortRequeue = ctrl.Result{Requeue: true, RequeueAfter: controllers.RequeueForShortDuration}
		resultLongRequeue  = ctrl.Result{Requeue: true, RequeueAfter: controllers.RequeueForLongDuration}
		resultRequeue      = ctrl.Result{Requeue: true}
		resultDone         = ctrl.Result{}

		errNotFound  error
		errFailed    error
		packages     = []string{"pkg1", "pkg2"}
		sshPublicKey = "publicKey"

		usr1 = osbuildv1alpha1.User{
			Groups: &[]string{"group1", "group2"},
			Key:    &sshPublicKey,
			Name:   "usr1",
		}
		usr2 = osbuildv1alpha1.User{
			Groups: &[]string{"group3", "group4"},
			Key:    &sshPublicKey,
			Name:   "usr2",
		}
		disabledServices = []string{"s1", "s2"}
		enabledServices  = []string{"s3", "s4"}
		zeroUuid         = "00000000-0000-0000-0000-000000000000"
		buildUrl         = "http://test/test"
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		osBuildRepository = osbuild.NewMockRepository(mockCtrl)
		composerClient = composer.NewMockClientWithResponsesInterface(mockCtrl)

		scheme = runtime.NewScheme()
		err := clientgoscheme.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = osbuildv1alpha1.AddToScheme(scheme)
		Expect(err).To(BeNil())

		reconciler = &controllers.OSBuildReconciler{
			Scheme:            scheme,
			OSBuildRepository: osBuildRepository,
			ComposerClient:    composerClient,
		}

		requestContext = context.TODO()

		errNotFound = errors.NewNotFound(schema.GroupResource{}, "Requested resource was not found")
		errFailed = errors.NewInternalError(fmt.Errorf("Server encounter and error"))

		osbuildInstance = &osbuildv1alpha1.OSBuild{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: instanceNamespace,
			},
			Spec: osbuildv1alpha1.OSBuildSpec{
				Details: &osbuildv1alpha1.BuildDetails{
					Distribution: distribution,
					Customizations: &osbuildv1alpha1.Customizations{
						Packages: packages,
						Users:    []osbuildv1alpha1.User{usr1, usr2},
						Services: &osbuildv1alpha1.Services{
							Disabled: disabledServices,
							Enabled:  enabledServices,
						},
					},
					TargetImage: osbuildv1alpha1.TargetImage{
						Architecture:    architecture,
						TargetImageType: osbuildv1alpha1.EdgeContainerImageType,
						OSTree:          nil,
					},
				},
				TriggeredBy: triggeredBy,
			},
			Status: osbuildv1alpha1.OSBuildStatus{},
		}

		composerPostResponseCreated = composer.PostComposeResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusCreated,
			},
			JSON201: &composer.ComposeId{
				Id: uuid.MustParse(zeroUuid),
			},
		}

		composerPostResponseFailed = composer.PostComposeResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
			},
		}

		composerGetStatusResponseBadRequest = composer.GetComposeStatusResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
			},
		}

		composerGetStatusFailed = composer.GetComposeStatusResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &composer.ComposeStatus{
				Id: zeroUuid,
				ImageStatus: composer.ImageStatus{
					Status: composer.ImageStatusValueFailure,
				},
				Status: composer.ComposeStatusValueFailure,
			},
		}

		composerGetStatusDone = composer.GetComposeStatusResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &composer.ComposeStatus{
				Id: zeroUuid,
				ImageStatus: composer.ImageStatus{
					Status: composer.ImageStatusValueSuccess,
					UploadStatus: &composer.UploadStatus{
						Options: composer.AWSS3UploadStatus{
							Url: buildUrl,
						},
						Status: "",
						Type:   "aws.s3",
					},
				},
				Status: composer.ComposeStatusValueSuccess,
			},
		}

		composerGetStatusPending = composer.GetComposeStatusResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &composer.ComposeStatus{
				Id: zeroUuid,
				ImageStatus: composer.ImageStatus{
					Status: composer.ImageStatusValueBuilding,
				},
				Status: composer.ComposeStatusValuePending,
			},
		}
	})

	AfterEach(func() {
		osbuildInstance.DeletionTimestamp = nil
		osbuildInstance.Status.Conditions = nil
		osbuildInstance.Status.ComposeId = controllers.EmptyComposeID
	})

	Context("Failure to get OSBuild instance", func() {
		It("Should return Done when the instance is not found", func() {
			// given
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(nil, errNotFound)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("Should return requeue when failed to get the instance", func() {
			// given
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(nil, errFailed)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		})
	})

	Context("Handle deletion", func() {
		It("Should return Done if the OSBuild CR was deleted", func() {
			// given
			osbuildInstance.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))
		})
	})

	Context("ComposeId is empty so create postCompose request ", func() {
		BeforeEach(func() {
			// given
			osBuildRepository.EXPECT().PatchStatus(requestContext, osbuildInstance, gomock.Any()).Return(nil)
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)
		})

		DescribeTable("should requeue for long duration if failed on postCompose with an error", func(targetImageType osbuildv1alpha1.TargetImageType) {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = targetImageType
			composerClient.EXPECT().PostComposeWithResponse(requestContext, gomock.Any()).DoAndReturn(
				func(ctx context.Context, body composer.PostComposeJSONRequestBody, reqEditors ...interface{}) (*composer.PostComposeResponse, error) {
					Expect(body.ImageRequest.ImageType).To(Equal(composer.ImageTypes(targetImageType)))
					return nil, errFailed
				},
			)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
			checkConditionArr(osbuildv1alpha1.ConditionFailed, failedToSendPostRequestMsg, osbuildInstance.Status.Conditions)
		},
			Entry("target image type is edge-container", osbuildv1alpha1.EdgeContainerImageType),
			Entry("target image type is guest-image (qcow2)", osbuildv1alpha1.GuestImageImageType),
		)

		DescribeTable("should requeue for long duration if failed on postCompose with status code `bad request`", func(targetImageType osbuildv1alpha1.TargetImageType) {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = targetImageType
			composerClient.EXPECT().PostComposeWithResponse(requestContext, gomock.Any()).DoAndReturn(
				func(ctx context.Context, body composer.PostComposeJSONRequestBody, reqEditors ...interface{}) (*composer.PostComposeResponse, error) {
					Expect(body.ImageRequest.ImageType).To(Equal(composer.ImageTypes(targetImageType)))
					return &composerPostResponseFailed, nil
				},
			)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
			checkConditionArr(osbuildv1alpha1.ConditionFailed, "", osbuildInstance.Status.Conditions)
		},
			Entry("target image type is edge-container", osbuildv1alpha1.EdgeContainerImageType),
			Entry("target image type is guest-image (qcow2)", osbuildv1alpha1.GuestImageImageType),
		)

		DescribeTable("should requeue for long duration if succeeded to create a new job", func(targetImageType osbuildv1alpha1.TargetImageType) {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = targetImageType
			composerClient.EXPECT().PostComposeWithResponse(requestContext, gomock.Any()).DoAndReturn(
				func(ctx context.Context, body composer.PostComposeJSONRequestBody, reqEditors ...interface{}) (*composer.PostComposeResponse, error) {
					Expect(body.ImageRequest.ImageType).To(Equal(composer.ImageTypes(targetImageType)))
					return &composerPostResponseCreated, nil
				},
			)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
			osbuildStatus := osbuildInstance.Status
			Expect(osbuildStatus.ComposeId).To(Equal(composerPostResponseCreated.JSON201.Id.String()))
			checkConditionArr(osbuildv1alpha1.ConditionInProgress, buildJobStillRunningMsg, osbuildInstance.Status.Conditions)
		},
			Entry("target image type is edge-container", osbuildv1alpha1.EdgeContainerImageType),
			Entry("target image type is guest-image (qcow2)", osbuildv1alpha1.GuestImageImageType),
		)

	})

	Context("Last Build Status is InProgress", func() {
		BeforeEach(func() {
			msg := buildJobStillRunningMsg
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeContainerImageType
			osbuildInstance.Status.ComposeId = zeroUuid
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:    osbuildv1alpha1.ConditionInProgress,
					Status:  metav1.ConditionTrue,
					Message: &msg,
				},
				{
					Type:    osbuildv1alpha1.ConditionReady,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
				{
					Type:    osbuildv1alpha1.ConditionFailed,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
			}

			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)
		})

		It("should requeue for short duration if failed to getComposerStatus with error", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(nil, errFailed)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		})

		It("should requeue for short duration if failed to getComposerStatus with failure status code", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(&composerGetStatusResponseBadRequest, nil)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		})

		It("should requeue for long duration if job status is still pending", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(&composerGetStatusPending, nil)
			osBuildRepository.EXPECT().PatchStatus(requestContext, osbuildInstance, gomock.Any()).Return(nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
			checkConditionArr(osbuildv1alpha1.ConditionInProgress, buildJobStillRunningMsg, osbuildInstance.Status.Conditions)

		})

		It("should requeue if job status was changed from InProgress to success", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(&composerGetStatusDone, nil)
			osBuildRepository.EXPECT().PatchStatus(requestContext, osbuildInstance, gomock.Any()).Return(nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultRequeue))
			Expect(osbuildInstance.Status.AccessUrl).To(Equal(buildUrl))
			Expect(osbuildInstance.Status.ComposeId).To(Equal(zeroUuid))

			checkConditionArr(osbuildv1alpha1.ConditionReady, buildJobFinishedMsg, osbuildInstance.Status.Conditions)
		})

		It("should requeue if job status was changed from InProgress to failed", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(&composerGetStatusFailed, nil)
			osBuildRepository.EXPECT().PatchStatus(requestContext, osbuildInstance, gomock.Any()).Return(nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultRequeue))
			checkConditionArr(osbuildv1alpha1.ConditionFailed, buildJobFailedMsg, osbuildInstance.Status.Conditions)

		})

		It("should requeue if job status was changed from InProgress to success but fail on patch status", func() {
			// given
			composerClient.EXPECT().GetComposeStatusWithResponse(requestContext, zeroUuid).Return(&composerGetStatusDone, nil)
			osBuildRepository.EXPECT().PatchStatus(requestContext, osbuildInstance, gomock.Any()).Return(errFailed)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		})
	})

	Context("Failed to build an image", func() {
		It("should done", func() {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeContainerImageType
			msg := buildJobFailedMsg
			osbuildInstance.Status.ComposeId = zeroUuid
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:    osbuildv1alpha1.ConditionInProgress,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
				{
					Type:    osbuildv1alpha1.ConditionReady,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
				{
					Type:    osbuildv1alpha1.ConditionFailed,
					Status:  metav1.ConditionTrue,
					Message: &msg,
				},
			}

			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))
		})
	})

	Context("Container Build is done", func() {
		It("should return done", func() {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeContainerImageType
			osbuildInstance.Status.ComposeId = zeroUuid
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:    osbuildv1alpha1.ConditionInProgress,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:    osbuildv1alpha1.ConditionFailed,
					Status:  metav1.ConditionFalse,
					Message: nil,
				},
			}
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))
		})

	})

	Context("Container Build is empty", func() {
		It("should return done", func() {
			// given
			osbuildInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeContainerImageType
			osbuildInstance.Status.ComposeId = zeroUuid
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{}
			osBuildRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
		})
	})

})

func checkConditionArr(requiredStatus osbuildv1alpha1.ConditionType, requiredMsg string, conditions []osbuildv1alpha1.Condition) {
	for _, c := range conditions {
		if c.Type == requiredStatus {
			Expect(c.Status).To(Equal(metav1.ConditionTrue))
			if requiredMsg != "" {
				Expect(*c.Message).To(Equal(requiredMsg))
			}
		} else {
			Expect(c.Status).To(Equal(metav1.ConditionFalse))
		}
	}
}
