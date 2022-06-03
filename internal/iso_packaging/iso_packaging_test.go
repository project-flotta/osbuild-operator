package iso_packaging_test

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	buildv1 "github.com/openshift/api/build/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/iso_packaging"
)

var _ = Describe("IsoPackageJob", Ordered, func() {
	var (
		k8sClient client.Client
		testEnv   *envtest.Environment
		buildInfo *v1alpha1.OSBuild
		buildEnv  *v1alpha1.OSBuildEnvConfig
	)

	BeforeAll(func() {
		By("bootstrapping test environment")
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: false,
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {

		buildInfo = &v1alpha1.OSBuild{
			TypeMeta: metav1.TypeMeta{
				Kind:       "OSBuild",
				APIVersion: "v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       "test",
			},
			Spec: v1alpha1.OSBuildSpec{
				Kickstart: &v1alpha1.NameRef{Name: "testKS"}},
			Status: v1alpha1.OSBuildStatus{
				ComposerIso: "http://127.0.0.1/fedora.iso"},
		}

		buildEnv = &v1alpha1.OSBuildEnvConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: v1alpha1.OSBuildEnvConfigSpec{
				S3Service: v1alpha1.S3ServiceConfig{
					AWS: &v1alpha1.AWSS3ServiceConfig{
						CredsSecretReference: buildv1.SecretLocalReference{
							Name: "test-secret",
						},
						Region: "test",
						Bucket: "test",
					},
				},
			},
		}

	})

	// @TODO Needs the following test:
	// Generics3 basic
	// Generics3 with custom CA
	// No S3 or Generics3 should fail.
	// Delete function is working as expected.
	// IfFinished all testcases

	It("Run the job correctly", func() {
		// given
		builder, err := iso_packaging.NewBuilderJob(k8sClient, buildInfo, buildEnv, "busybox")
		Expect(err).NotTo(HaveOccurred())

		// when
		err = builder.Start(context.TODO())
		Expect(err).NotTo(HaveOccurred())

		// then
		resultJob := batchv1.Job{}
		err = k8sClient.Get(context.TODO(), client.ObjectKey{
			Namespace: buildInfo.Namespace,
			Name:      buildInfo.Name,
		}, &resultJob)
		Expect(err).NotTo(HaveOccurred())

		containers := resultJob.Spec.Template.Spec.Containers
		Expect(containers).To(HaveLen(1))
		Expect(containers[0].VolumeMounts).To(HaveLen(1))
		Expect(containers[0].VolumeMounts[0].Name).To(Equal("config"))
		Expect(containers[0].VolumeMounts[0].MountPath).To(Equal("/opt/iso_package/"))

		// Env section
		Expect(containers[0].EnvFrom).To(HaveLen(1))
		Expect(containers[0].EnvFrom[0].SecretRef.Name).To(Equal("test-secret"))
		Expect(containers[0].Env).To(HaveLen(1))
		Expect(containers[0].Env[0].Name).To(Equal("AWS_DEFAULT_REGION"))
		Expect(containers[0].Env[0].Value).To(Equal("test"))

		// Command section
		cmd := containers[0].Command
		Expect(containers[0].Command).To(HaveLen(6))
		Expect(cmd[len(cmd)-1]).To(Equal("s3://test/default_test_test.iso"))
		Expect(cmd[len(cmd)-2]).To(Equal("--upload-target"))

		//job details
		job := resultJob.Spec.Template.Spec
		Expect(job.Volumes).To(HaveLen(1))
		Expect(job.Volumes[0].Name).To(Equal("config"))
		Expect(job.Volumes[0].VolumeSource.Projected.Sources).To(HaveLen(1))
		Expect(job.Volumes[0].VolumeSource.Projected.Sources[0].ConfigMap.Name).To(Equal("testKS"))

		finished, err := builder.IsFinished()
		Expect(finished).To(BeFalse())
		Expect(err).NotTo(HaveOccurred())
	})
})
