package osbuildconfig

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	buildv1 "github.com/openshift/api/build/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/httpapi"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	repositoryosbuildconfig "github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	repositorysecret "github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/restapi"
)

const (
	Namespace         = "test_namespace"
	SecretName        = "test_secret"
	OSBuildConfigName = "test_osbuildconfig"
)

func TestOsbuildConfigAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "test")
}

var _ = Describe("OSBuildConfig rest API", func() {
	var (
		mockCtrl       *gomock.Controller
		osbuildConfig  v1alpha1.OSBuildConfig
		triggerWebhook *buildv1.WebHookTrigger
		secret         corev1.Secret
		secretVal      = "123"
		secretData     = map[string][]byte{
			"WebHookSecretKey": []byte(secretVal),
		}

		osBuildConfigRepository *repositoryosbuildconfig.MockRepository
		secretRepository        *repositorysecret.MockRepository
		osBuildCRCreator        *manifests.MockOSBuildCRCreator
		responseWriter          *httptest.ResponseRecorder
		osbuildConfigHandler    *OSBuildConfigHandler
		req                     *http.Request
		params                  restapi.TriggerBuildParams
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		osBuildConfigRepository = repositoryosbuildconfig.NewMockRepository(mockCtrl)
		osBuildCRCreator = manifests.NewMockOSBuildCRCreator(mockCtrl)
		secretRepository = repositorysecret.NewMockRepository(mockCtrl)
		osbuildConfigHandler = NewOSBuildConfigHandler(osBuildConfigRepository, secretRepository, osBuildCRCreator)

		secret = corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace: Namespace,
				Name:      SecretName,
			},
			Data: secretData,
		}
		params = restapi.TriggerBuildParams{
			Secret: secretVal,
		}
		secretReference := &buildv1.SecretLocalReference{
			Name: SecretName,
		}
		triggerWebhook = &buildv1.WebHookTrigger{
			SecretReference: secretReference,
		}

		triggerNewBuildUponAnyChange := true
		osbuildConfig = v1alpha1.OSBuildConfig{
			ObjectMeta: v1.ObjectMeta{
				Name: OSBuildConfigName,
			},
			Spec: v1alpha1.OSBuildConfigSpec{
				Details: v1alpha1.BuildDetails{
					Distribution:   "rhel-86",
					Customizations: nil,
					TargetImage: v1alpha1.TargetImage{
						Architecture:    "x86_64",
						TargetImageType: "edge-container",
						OSTree:          nil,
					},
				},
				Triggers: v1alpha1.BuildTriggers{
					ConfigChange: &triggerNewBuildUponAnyChange,
					WebHook:      triggerWebhook,
				},
			},
			Status: v1alpha1.OSBuildConfigStatus{},
		}

		responseWriter = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "test_request", nil)

		err := httpapi.Load()
		if err != nil {
			panic(err.Error())
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("trigger a build", func() {
		It("and succeed", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			secretRepository.EXPECT().Read(req.Context(), SecretName, Namespace).Return(&secret, nil)

			osBuildCRCreator.EXPECT().Create(req.Context(), &osbuildConfig).Return(nil)
			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusOK))

		})

		It("with not found response, because osbuildConfig doesn't exist", func() {
			// given
			returnErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(nil, returnErr)

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusNotFound))
		})

		It("with internalServerError response, because osbuildConfigRepository failed", func() {
			// given
			returnErr := errors.NewBadRequest("test")
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(nil, returnErr)

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusInternalServerError))
		})

		It("with bad request response, because osbuildConfig hasn't webhook", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			osbuildConfig.Spec.Triggers.WebHook = nil

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("with not found response, because the secret doesn't exist", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			returnErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
			secretRepository.EXPECT().Read(req.Context(), SecretName, Namespace).Return(nil, returnErr)

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusNotFound))
		})

		It("with internalServerError response, because getting the secret failed", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			returnErr := errors.NewBadRequest("test")
			secretRepository.EXPECT().Read(req.Context(), SecretName, Namespace).Return(nil, returnErr)

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusInternalServerError))
		})

		It("with forbidden response, because the webhook secret value different from the input param", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			secretRepository.EXPECT().Read(req.Context(), SecretName, Namespace).Return(&secret, nil)
			params.Secret = "456"

			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusForbidden))

		})

		It("with internalServerError response, because the creation of the osbuild CR failed", func() {
			// given
			osBuildConfigRepository.EXPECT().Read(req.Context(), OSBuildConfigName, Namespace).Return(&osbuildConfig, nil)
			secretRepository.EXPECT().Read(req.Context(), SecretName, Namespace).Return(&secret, nil)
			returnErr := errors.NewBadRequest("test")
			osBuildCRCreator.EXPECT().Create(gomock.Any(), &osbuildConfig).Return(returnErr)
			// when
			osbuildConfigHandler.TriggerBuild(responseWriter, req, Namespace, OSBuildConfigName, params)

			// then
			Expect(responseWriter.Result().StatusCode).To(Equal(http.StatusInternalServerError))

		})
	})

})
