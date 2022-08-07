package iso_packaging

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws"
)

const (
	templateCommand string = "isopackage %s -ks %s --upload-target"
	kickstart       string = "init.ks"
)

var (
	jobTTLAfterFinish int32 = 100   // Job will be deleted after finished with this
	deadlineSeconds   int64 = 36000 // Job will be terminated after a hour.
)

// Builder is a struct that manages a build to package an iso
type Builder struct {
	client      client.Client
	jobSpec     *batchv1.Job
	build       *v1alpha1.OSBuild
	buildConfig *v1alpha1.OSBuildEnvConfig
	baseImage   string
}

func NewBuilderJob(client client.Client, build *v1alpha1.OSBuild, buildConfig *v1alpha1.OSBuildEnvConfig, image string) (*Builder, error) {

	if build == nil || buildConfig == nil {
		return nil, fmt.Errorf("Invalid config added")
	}

	return &Builder{
		client:      client,
		jobSpec:     nil,
		build:       build,
		buildConfig: buildConfig,
		baseImage:   image,
	}, nil
}

func (b *Builder) Start(ctx context.Context) error {
	sourceISO := b.build.Status.ComposerIso
	if sourceISO == "" {
		return fmt.Errorf("Cannot parse invalid iso image")
	}
	_, err := url.Parse(sourceISO)
	if err != nil {
		return err
	}

	if b.build.Spec.EdgeInstallerDetails.Kickstart == nil {
		return fmt.Errorf("kickstart is not defined")
	}

	command := fmt.Sprintf(templateCommand, b.build.Status.ComposerIso, kickstart)

	jobSpec := b.getIsoPackageJob(command)

	err = b.addCredentialsToJob(jobSpec)
	if err != nil {
		return fmt.Errorf("cannot add credentials: %v", err)
	}

	err = b.client.Create(ctx, jobSpec)
	if err != nil {
		return fmt.Errorf("Cannot applied job: %v", err)
	}
	b.jobSpec = jobSpec
	return nil
}

func (b *Builder) getIsoPackageJob(command string) *batchv1.Job {
	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.build.Name,
			Namespace: b.build.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: b.build.APIVersion,
					Kind:       b.build.Kind,
					Name:       b.build.Name,
					UID:        b.build.UID,
				},
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &jobTTLAfterFinish,
			ActiveDeadlineSeconds:   &deadlineSeconds,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "iso-command",
							Image:   b.baseImage,
							Command: strings.Split(command, " "),
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/opt/iso_package/",
								}},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					Volumes: []v1.Volume{
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								Projected: &v1.ProjectedVolumeSource{
									Sources: []v1.VolumeProjection{{
										ConfigMap: &v1.ConfigMapProjection{
											LocalObjectReference: v1.LocalObjectReference{
												Name: b.build.Spec.EdgeInstallerDetails.Kickstart.Name,
											},
										},
									}},
								},
							},
						},
					},
				},
			},
		},
	}
	return jobSpec
}

func (b *Builder) addCredentialsToJob(job *batchv1.Job) error {

	if b.buildConfig.Spec.S3Service.AWS != nil {
		b.addS3CredentialsToJob(job)
		return nil
	}

	if b.buildConfig.Spec.S3Service.GenericS3 != nil {
		b.addGenericS3CredentialsToJob(job)
		return nil
	}

	return fmt.Errorf("Credentials to store the iso is not present")
}

func (b *Builder) addS3CredentialsToJob(jobSpec *batchv1.Job) {
	config := b.buildConfig.Spec.S3Service.AWS

	jobSpec.Spec.Template.Spec.Containers[0].EnvFrom = []v1.EnvFromSource{
		{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{Name: config.CredsSecretReference.Name},
				Optional:             aws.Bool(false)},
		},
	}
	jobSpec.Spec.Template.Spec.Containers[0].Env = []v1.EnvVar{
		{
			Name:  "AWS_DEFAULT_REGION",
			Value: config.Region,
		},
	}

	jobSpec.Spec.Template.Spec.Containers[0].Command = append(
		jobSpec.Spec.Template.Spec.Containers[0].Command,
		gets3Target(config.Bucket, b.build))
}

func (b *Builder) addGenericS3CredentialsToJob(jobSpec *batchv1.Job) {
	config := b.buildConfig.Spec.S3Service.GenericS3
	jobSpec.Spec.Template.Spec.Containers[0].EnvFrom = []v1.EnvFromSource{
		{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{Name: config.CredsSecretReference.Name},
				Optional:             aws.Bool(false)},
		},
	}
	jobSpec.Spec.Template.Spec.Containers[0].Env = []v1.EnvVar{
		{
			Name:  "AWS_DEFAULT_REGION",
			Value: config.Region,
		},
		{
			Name:  "AWS_ENDPOINT_URL",
			Value: config.Endpoint,
		},
		{
			Name:  "AWS_NO_VERIFY_SSL",
			Value: fmt.Sprintf("%v", config.SkipSSLVerification),
		},
	}

	if config.CABundleSecretReference != nil {
		// From here: https://go.dev/src/crypto/x509/root_unix.go
		jobSpec.Spec.Template.Spec.Containers[0].Env = append(
			jobSpec.Spec.Template.Spec.Containers[0].Env,
			v1.EnvVar{Name: "SSL_CERT_DIR", Value: "/cabundle"})

		jobSpec.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			jobSpec.Spec.Template.Spec.Containers[0].VolumeMounts,
			v1.VolumeMount{Name: "cabundle", ReadOnly: true, MountPath: "/cabundle/"})

		jobSpec.Spec.Template.Spec.Volumes = append(
			jobSpec.Spec.Template.Spec.Volumes,
			v1.Volume{
				Name: "cabundle",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: config.CABundleSecretReference.Name,
					},
				},
			})
	}

	jobSpec.Spec.Template.Spec.Containers[0].Command = append(
		jobSpec.Spec.Template.Spec.Containers[0].Command,
		gets3Target(config.Bucket, b.build))
}

func (b *Builder) IsFinished() (bool, error) {
	job := batchv1.Job{}
	err := b.client.Get(context.TODO(), client.ObjectKey{
		Namespace: b.jobSpec.Namespace,
		Name:      b.jobSpec.Name,
	}, &job)

	if err != nil {
		return false, fmt.Errorf("Cannot get job: %s", err)
	}

	b.jobSpec = &job
	for _, c := range job.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
			if c.Type == batchv1.JobFailed {
				return true, errors.New("Cannot repackage the ISO image correctly")
			}
			return true, nil
		}
	}
	return false, nil
}

func (b *Builder) Delete() error {
	if b.jobSpec == nil {
		return errors.New("Cannot delete a non started job")
	}

	err := b.client.Delete(context.TODO(), b.jobSpec)
	if err != nil {
		return fmt.Errorf("Cannot delete job: %v", err)
	}
	return nil
}

func gets3Target(bucket string, build *v1alpha1.OSBuild) string {
	return fmt.Sprintf("s3://%s/%s_%s_%s.iso", bucket, build.Namespace, build.Name, build.UID)
}
