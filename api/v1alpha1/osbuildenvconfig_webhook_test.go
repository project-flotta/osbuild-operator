package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	buildv1 "github.com/openshift/api/build/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	osbuildEnvConfigName = "test_osbuildenvconfig"
)

var _ = Describe("OSBuildEnvConfig Webhook", func() {
	var (
		osbuildEnvConfig     OSBuildEnvConfig
		osbuildEnvConfigList OSBuildEnvConfigList
		clientBuilder        *fake.ClientBuilder
	)

	BeforeEach(func() {
		scheme := pkgruntime.NewScheme()
		utilruntime.Must(AddToScheme(scheme))

		kClient = nil
		clientBuilder = fake.NewClientBuilder().WithScheme(scheme)

		osbuildEnvConfig = OSBuildEnvConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: osbuildEnvConfigName,
			},
			Spec: OSBuildEnvConfigSpec{
				RedHatCredsSecretReference: buildv1.SecretLocalReference{
					Name: "OriginalName",
				},
				Workers: []WorkerConfig{
					{
						Name:           "worker-1",
						VMWorkerConfig: &VMWorkerConfig{},
					},
					{
						Name:                 "worker-2",
						ExternalWorkerConfig: &ExternalWorkerConfig{},
					},
				},
			},
		}
		osbuildEnvConfigList = OSBuildEnvConfigList{
			Items: []OSBuildEnvConfig{
				osbuildEnvConfig,
			},
		}

	})

	Context("Test osBuildEnvConfigError", func() {
		It("Should return the same string", func() {
			// given
			message := "test message"
			// when
			err := osBuildEnvConfigError{
				error: message,
			}
			// then
			Expect(err.Error()).To(Equal(message))
		})
	})

	Context("Test create validation", func() {
		It("First instance creation should succeed", func() {
			// given
			kClient = clientBuilder.Build()
			// when
			err := osbuildEnvConfig.ValidateCreate()
			// then
			Expect(err).To(BeNil())
		})

		It("Second instance creation should fail", func() {
			// given
			kClient = clientBuilder.WithLists(&osbuildEnvConfigList).Build()
			// when
			err := osbuildEnvConfig.ValidateCreate()
			// then
			Expect(err).To(Equal(crAlreadyExists))
		})

		It("Non-unique worker names should fail", func() {
			// given
			kClient = clientBuilder.Build()
			osbuildEnvConfig.Spec.Workers[1].Name = osbuildEnvConfig.Spec.Workers[0].Name
			// when
			err := osbuildEnvConfig.ValidateCreate()
			// then
			Expect(err).To(Equal(workerNamesNotUnique))
		})

		It("Should fail if neither VMWorkerConfig not ExternalWorkerConfig are set", func() {
			// given
			kClient = clientBuilder.Build()
			worker := &osbuildEnvConfig.Spec.Workers[0]
			worker.VMWorkerConfig = nil
			// when
			err := osbuildEnvConfig.ValidateCreate()
			// then
			Expect(err.Error()).To(Equal(fmt.Sprintf(noWorkerConfigFormat, worker.Name)))
		})

		It("Should fail if both VMWorkerConfig and ExternalWorkerConfig are set", func() {
			// given
			kClient = clientBuilder.Build()
			worker := &osbuildEnvConfig.Spec.Workers[0]
			worker.ExternalWorkerConfig = &ExternalWorkerConfig{}
			// when
			err := osbuildEnvConfig.ValidateCreate()
			// then
			Expect(err.Error()).To(Equal(fmt.Sprintf(duplicateWorkerConfigFormat, worker.Name)))
		})
	})

	Context("Test default value", func() {
		It("Should not change anything", func() {
			// given
			original := osbuildEnvConfig.DeepCopy()
			// when
			osbuildEnvConfig.Default()
			// then
			Expect(osbuildEnvConfig).To(Equal(*original))
		})
	})

	Context("Test delete validation", func() {
		It("Should approve delete", func() {
			// when
			err := osbuildEnvConfig.ValidateDelete()
			// then
			Expect(err).To(BeNil())
		})
	})

	Context("Test update validation", func() {
		It("Should allow updating the finalizers", func() {
			// given
			updatedCR := osbuildEnvConfig.DeepCopy()
			updatedCR.ObjectMeta.Finalizers = []string{"osbuild"}
			// when
			err := updatedCR.ValidateUpdate(&osbuildEnvConfig)
			// then
			Expect(err).To(BeNil())
		})
		It("Should not allow updating the spec", func() {
			// given
			updatedCR := osbuildEnvConfig.DeepCopy()
			updatedCR.Spec.RedHatCredsSecretReference.Name = "UpdatedName"
			// when
			err := updatedCR.ValidateUpdate(&osbuildEnvConfig)
			// then
			Expect(err).To(Equal(updateNotSupported))
		})
	})

})
