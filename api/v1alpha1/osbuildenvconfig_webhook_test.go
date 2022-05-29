package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		It("Should not allow updating", func() {
			// when
			err := osbuildEnvConfig.ValidateUpdate(&osbuildEnvConfig)
			// then
			Expect(err).To(Equal(updateNotSupported))
		})
	})

})
