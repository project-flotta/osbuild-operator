package controllers_test

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/controllers"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
)

var _ = Describe("OSBuildConfig Controller", func() {
	const (
		instanceNamespace = "osbuild"
		instanceName      = "osbuild_test"

		distribution = "rhel-86"
		triggeredBy  = "UpdateCR"
		architecture = "x86_64"
	)
	var (
		mockCtrl                *gomock.Controller
		osBuildRepository       *osbuild.MockRepository
		osBuildConfigRepository *osbuildconfig.MockRepository
		osBuildCRCreator        *manifests.MockOSBuildCRCreator
		reconciler              *controllers.OSBuildConfigReconciler
		requestContext          context.Context
		osbuildConfigInstance   *osbuildv1alpha1.OSBuildConfig
		customizations          *osbuildv1alpha1.Customizations
		osbuildInstance         *osbuildv1alpha1.OSBuild

		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      instanceName,
				Namespace: instanceNamespace,
			},
		}

		resultShortRequeue = ctrl.Result{Requeue: true, RequeueAfter: controllers.RequeueForShortDuration}
		resultLongRequeue  = ctrl.Result{Requeue: true, RequeueAfter: controllers.RequeueForLongDuration}
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
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		osBuildRepository = osbuild.NewMockRepository(mockCtrl)
		osBuildConfigRepository = osbuildconfig.NewMockRepository(mockCtrl)
		osBuildCRCreator = manifests.NewMockOSBuildCRCreator(mockCtrl)

		reconciler = &controllers.OSBuildConfigReconciler{
			OSBuildConfigRepository: osBuildConfigRepository,
			OSBuildRepository:       osBuildRepository,
			OSBuildCRCreator:        osBuildCRCreator,
		}

		requestContext = context.TODO()

		errNotFound = errors.NewNotFound(schema.GroupResource{}, "Requested resource was not found")
		errFailed = errors.NewInternalError(fmt.Errorf("Server encounter and error"))

		customizations = &osbuildv1alpha1.Customizations{
			Packages: packages,
			Users:    []osbuildv1alpha1.User{usr1, usr2},
			Services: &osbuildv1alpha1.Services{
				Disabled: disabledServices,
				Enabled:  enabledServices,
			},
		}
		osbuildConfigInstance = &osbuildv1alpha1.OSBuildConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: instanceNamespace,
			},
			Spec: osbuildv1alpha1.OSBuildConfigSpec{
				Details: osbuildv1alpha1.BuildDetails{
					Distribution:   distribution,
					Customizations: customizations,
					TargetImage: osbuildv1alpha1.TargetImage{
						Architecture:    architecture,
						TargetImageType: osbuildv1alpha1.EdgeContainerImageType,
						OSTree:          nil,
					},
				},

				Triggers: osbuildv1alpha1.BuildTriggers{},
				Template: nil,
			},
			Status: osbuildv1alpha1.OSBuildConfigStatus{},
		}

		osbuildInstance = &osbuildv1alpha1.OSBuild{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: instanceNamespace,
			},
			Spec: osbuildv1alpha1.OSBuildSpec{
				Details: &osbuildv1alpha1.BuildDetails{
					Distribution:   distribution,
					Customizations: customizations,
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
	})

	AfterEach(func() {
		osbuildConfigInstance.DeletionTimestamp = nil
		osbuildConfigInstance.Status = osbuildv1alpha1.OSBuildConfigStatus{}
		osbuildInstance.Status = osbuildv1alpha1.OSBuildStatus{}
	})

	Context("Failure to get OSBuildConfig", func() {
		It("Should return Done when the instance is not found", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(nil, errNotFound)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("Should return requeue when failed to get the instance", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(nil, errFailed)

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
			osbuildConfigInstance.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildConfigInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))
		})
	})

	Context("New OSBuild instance is needed", func() {
		DescribeTable("should requeue when failing to patch update", func(lastKnownUserConfiguration *osbuildv1alpha1.UserConfiguration, annotation map[string]string) {
			// given
			osbuildConfigInstance.Status.LastKnownUserConfiguration = lastKnownUserConfiguration
			osbuildConfigInstance.Spec.Details.Customizations = &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}
			osbuildConfigInstance.Annotations = annotation
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildConfigInstance, nil)
			osBuildConfigRepository.EXPECT().PatchStatus(requestContext, osbuildConfigInstance, gomock.Any()).Return(errFailed)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		},
			Entry("because of LastKnownUserConfiguration is nil", nil, map[string]string{}),
			Entry("because of LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{}),
			Entry("because of the webHookAnnotationKey was changed", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
			Entry("because of the webHookAnnotationKey was changed and also the LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
		)

		DescribeTable("should requeue when failing to create edge-container", func(lastKnownUserConfiguration *osbuildv1alpha1.UserConfiguration, annotation map[string]string) {
			// given
			osbuildConfigInstance.Status.LastKnownUserConfiguration = lastKnownUserConfiguration
			osbuildConfigInstance.Spec.Details.Customizations = &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}
			osbuildConfigInstance.Annotations = annotation
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildConfigInstance, nil)
			osBuildConfigRepository.EXPECT().PatchStatus(requestContext, osbuildConfigInstance, gomock.Any()).Return(nil)
			osBuildCRCreator.EXPECT().Create(requestContext, osbuildConfigInstance, osbuildv1alpha1.EdgeContainerImageType).Return(errFailed)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		},
			Entry("because of LastKnownUserConfiguration is nil", nil, map[string]string{}),
			Entry("because of LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{}),
			Entry("because of the webHookAnnotationKey was changed", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
			Entry("because of the webHookAnnotationKey was changed and also the LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
		)

		DescribeTable("should requeue when creating a new build", func(lastKnownUserConfiguration *osbuildv1alpha1.UserConfiguration, annotation map[string]string) {
			// given
			osbuildConfigInstance.Status.LastKnownUserConfiguration = lastKnownUserConfiguration
			osbuildConfigInstance.Spec.Details.Customizations = &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}
			osbuildConfigInstance.Annotations = annotation
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildConfigInstance, nil)
			osBuildConfigRepository.EXPECT().PatchStatus(requestContext, osbuildConfigInstance, gomock.Any()).Return(nil).Times(2)
			osBuildCRCreator.EXPECT().Create(requestContext, osbuildConfigInstance, osbuildv1alpha1.EdgeContainerImageType).Return(nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))
		},
			Entry("because of LastKnownUserConfiguration is nil", nil, map[string]string{}),
			Entry("because of LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{}),
			Entry("because of the webHookAnnotationKey was changed", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1", "pkg2"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
			Entry("because of the webHookAnnotationKey was changed and also the LastKnownUserConfiguration is different from the current userConfiguration", &osbuildv1alpha1.UserConfiguration{Customizations: &osbuildv1alpha1.Customizations{Packages: []string{"pkg1"}}}, map[string]string{"last_webhook_trigger_ts": "1111"}),
		)
	})

	Context("OSBuildConfig status need to be updated", func() {
		var osBuildName string
		BeforeEach(func() {
			osbuildConfigInstance.Status.LastKnownUserConfiguration = &osbuildv1alpha1.UserConfiguration{
				Customizations: osbuildConfigInstance.Spec.Details.Customizations.DeepCopy(),
				Template:       nil,
			}
			edgeContainerTargetType := osbuildv1alpha1.EdgeContainerImageType
			osbuildConfigInstance.Status.LastBuildType = &edgeContainerTargetType
			osBuildConfigRepository.EXPECT().Read(requestContext, instanceName, instanceNamespace).Return(osbuildConfigInstance, nil)

			lastVersion := 5
			osbuildConfigInstance.Status.LastVersion = &lastVersion
			osBuildName = fmt.Sprintf("%s-%d", osbuildConfigInstance.Name, lastVersion)
		})
		It("because after sorting the user configuration it's the same object and should requeue on error getting last OSBuild instance ", func() {
			// given
			osbuildConfigInstance.Spec.Details.Customizations.Packages = []string{"pkg2", "pkg1"}
			osbuildConfigInstance.Status.LastKnownUserConfiguration.Customizations.Packages = []string{"pkg1", "pkg2"}
			userKey := "key"
			osbuildConfigInstance.Spec.Details.Customizations.Users = []osbuildv1alpha1.User{{
				Key:  &userKey,
				Name: "user2",
			}, {
				Key:  &userKey,
				Name: "user1",
			},
			}

			osbuildConfigInstance.Status.LastKnownUserConfiguration.Customizations.Users = []osbuildv1alpha1.User{{
				Key:  &userKey,
				Name: "user1",
			}, {
				Key:  &userKey,
				Name: "user2",
			},
			}
			osbuildConfigInstance.Spec.Details.Customizations.Services = &osbuildv1alpha1.Services{
				Disabled: []string{"s3", "s2", "s1"},
				Enabled:  []string{"s4", "s1"},
			}
			osbuildConfigInstance.Status.LastKnownUserConfiguration.Customizations.Services = &osbuildv1alpha1.Services{
				Disabled: []string{"s1", "s2", "s3"},
				Enabled:  []string{"s1", "s4"},
			}
			osbuildConfigInstance.Spec.Template = &osbuildv1alpha1.Template{
				OSBuildConfigTemplateRef: "abc",
				Parameters: []osbuildv1alpha1.ParameterValue{
					{
						Name:  "b",
						Value: "2",
					},
					{
						Name:  "a",
						Value: "1",
					},
				},
			}
			osbuildConfigInstance.Status.LastKnownUserConfiguration.Template = &osbuildv1alpha1.Template{
				OSBuildConfigTemplateRef: "abc",
				Parameters: []osbuildv1alpha1.ParameterValue{
					{
						Name:  "a",
						Value: "1",
					},
					{
						Name:  "b",
						Value: "2",
					},
				},
			}

			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(nil, errNotFound)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))
		})

		It("should requeue if fail getting last OSBuild instance", func() {
			// given
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(nil, errFailed)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))

		})

		It("should done when last OSBuild instance has failed", func() {
			// given
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:   osbuildv1alpha1.ConditionInProgress,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionFailed,
					Status: metav1.ConditionTrue,
				},
			}
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("should requeue if last OSBuild instance still InProgress", func() {
			// given
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:   osbuildv1alpha1.ConditionInProgress,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionFailed,
					Status: metav1.ConditionFalse,
				},
			}
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultLongRequeue))

		})

		It("should done if last OSBuild instance is ready and target image type is edge-container", func() {
			// given
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:   osbuildv1alpha1.ConditionInProgress,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   osbuildv1alpha1.ConditionFailed,
					Status: metav1.ConditionFalse,
				},
			}
			osbuildConfigInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeContainerImageType
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("should done if last OSBuild instance is ready and last target image type is edge-installer", func() {
			// given
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:   osbuildv1alpha1.ConditionInProgress,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   osbuildv1alpha1.ConditionFailed,
					Status: metav1.ConditionFalse,
				},
			}
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("should requeue if the last OSBuild instance status is not clear", func() {
			// given
			osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
				{
					Type:   osbuildv1alpha1.ConditionInProgress,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionReady,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   osbuildv1alpha1.ConditionFailed,
					Status: metav1.ConditionFalse,
				},
			}
			osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)

			// when
			result, err := reconciler.Reconcile(requestContext, request)

			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultShortRequeue))

		})

		Context("should create new OSBuild instance for edge-installer if the last OSBuild instance is ready", func() {
			BeforeEach(func() {
				// given
				osbuildConfigInstance.Spec.Details.TargetImage.TargetImageType = osbuildv1alpha1.EdgeInstallerImageType
				edgeContainerTargetType := osbuildv1alpha1.EdgeContainerImageType
				osbuildConfigInstance.Status.LastBuildType = &edgeContainerTargetType
				osbuildInstance.Status.Conditions = []osbuildv1alpha1.Condition{
					{
						Type:   osbuildv1alpha1.ConditionInProgress,
						Status: metav1.ConditionFalse,
					},
					{
						Type:   osbuildv1alpha1.ConditionReady,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   osbuildv1alpha1.ConditionFailed,
						Status: metav1.ConditionFalse,
					},
				}
				osBuildRepository.EXPECT().Read(requestContext, osBuildName, instanceNamespace).Return(osbuildInstance, nil)
			})
			It("should requeue for short duration if fail on creation", func() {
				// given
				osBuildCRCreator.EXPECT().Create(requestContext, osbuildConfigInstance, osbuildv1alpha1.EdgeInstallerImageType).Return(errFailed)

				// when
				result, err := reconciler.Reconcile(requestContext, request)

				// then
				Expect(err).To(BeNil())
				Expect(result).To(Equal(resultShortRequeue))
			})

			It("should requeue for long duration if the new OSBuild instance was created", func() {
				// given
				osBuildConfigRepository.EXPECT().PatchStatus(requestContext, osbuildConfigInstance, gomock.Any()).Return(nil)
				osBuildCRCreator.EXPECT().Create(requestContext, osbuildConfigInstance, osbuildv1alpha1.EdgeInstallerImageType).Return(nil)

				// when
				result, err := reconciler.Reconcile(requestContext, request)

				// then
				Expect(err).To(BeNil())
				Expect(result).To(Equal(resultLongRequeue))
			})
		})
	})
})
