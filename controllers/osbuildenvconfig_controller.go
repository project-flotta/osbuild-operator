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
	"path"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/go-logr/logr"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/repository/certificate"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/deployment"
	"github.com/project-flotta/osbuild-operator/internal/repository/job"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildenvconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/route"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/internal/repository/service"
	"github.com/project-flotta/osbuild-operator/internal/repository/virtualmachine"
	"github.com/project-flotta/osbuild-operator/internal/sshkey"
	"github.com/project-flotta/osbuild-operator/internal/templates"
)

const (
	osBuildOperatorFinalizer = "osbuilder.project-flotta.io/osBuildOperatorFinalizer"

	certificateDuration = 87600

	composerCertificateName = "composer-cert"

	ComposerComposerAPIServiceName = "osbuild-composer"
	composerComposerAPIPortName    = "composer-api"
	composerWorkerAPIServiceName   = "osbuild-worker"
	composerWorkerAPIPortName      = "worker-api"

	composerWorkerAPIRouteName = "osbuild-worker"

	certificateSecretCACrtKey = "ca.crt"
	certificateSecretCrtKey   = "tls.crt"
	certificateSecretKeyKey   = "tls.key"

	composerConfigMapName      = "osbuild-composer-config"
	composerConfigMapKey       = "osbuild-composer.toml"
	composerConfigTemplateFile = "osbuild-composer.toml"

	composerProxyConfigMapName      = "osbuild-composer-proxy-config"
	composerProxyConfigMapKey       = "envoy.yaml"
	composerProxyConfigTemplateFile = "composer-proxy-config.yaml"

	pgSSLModeDefault = "prefer"

	composerDeploymentName         = "composer"
	composerDeploymentTemplateFile = "composer-deployment.yaml"

	composerAPIInternalPort = 18080
	composerAPIExternalPort = 8080
	workerAPIInternalPort   = 18700
	workerAPIExternalPort   = 8700

	composerWorkerRequestJobTimeout = time.Second * 20

	envoyProxyCertsDir = "/etc/certs"

	workerSSHKeysSecretName    = "osbuild-worker-ssh" // #nosec G101
	workerSSHKeysPrivateKeyKey = "ssh-privatekey"
	workerSSHKeysPublicKeyKey  = "ssh-publickey"

	workerVMUsername     = "cloud-user"
	workerVMTemplateFile = "worker-vm.yaml"

	workerSSHServiceNameFormat = "worker-%s-ssh"
	workerSSHPortName          = "ssh"

	workerCertificateNameFormat = "worker-%s-cert"

	workerSetupPlaybookConfigMapNameFormat = "worker-%s-setup-playbook"
	workerSetupPlaybookConfigMapKey        = "playbook.yaml"
	workerSetupPlaybookTemplateFile        = "worker-config-ansible-playbook.yaml"

	workerRPMRepoDistribution      = "rhel-8-cdn"
	workerRHCredentialsDir         = "/var/secrets/redhat-portal-credentials" // #nosec G101
	workerRHCredentialsUsernameKey = "username"
	workerRHCredentialsPasswordKey = "password"
	workerOSBuildWorkerCertsDir    = "/var/secrets/osbuild-worker-certs"

	workerSetupAnsibleConfigConfigMapName = "osbuild-worker-setup-ansible-config"
	workerSetupAnsibleConfigConfigMapKey  = "ansible.cfg"
	workerSetupAnsibleConfigTemplateFile  = "worker-config-ansible-config.cfg"

	workerSetupAnsibleConfigSSHRetries = 5

	workerSetupInventoryConfigMapNameFormat = "worker-%s-inventory"
	workerSetupInventoryConfigMapKey        = "inventory.yaml"
	workerSetupInventoryTemplateFile        = "worker-config-ansible-inventory.yaml"

	workerOSBuildWorkerConfigConfigMapName                 = "osbuild-worker-config"
	workerOSBuildWorkerConfigConfigMapKey                  = "osbuild-worker.toml"
	workerOSBuildWorkerConfigTemplateFile                  = "worker-osbuild-worker-config.toml"
	workerOSBuildWorkerConfigDir                           = "/var/config"
	workerOSBuildWorkerConfigS3CredentialsFile             = "s3-creds"
	workerOSBuildWorkerConfigS3CABundleFile                = "s3-cabundle"
	workerOSBuildWorkerConfigContainerRegistryAuthFile     = "cir-creds"
	workerOSBuildWorkerConfigContainerRegistryCertsDir     = "registry-certs"
	workerOSBuildWorkerConfigContainerRegistryCABundleFile = "cir-cabundle.crt"

	workerOSBuildWorkerS3CredsDir                   = "/var/secrets/osbuild-s3-certs" // #nosec G101
	workerOSBuildWorkerS3CABundleDir                = "/var/secrets/osbuild-s3-ca-bundle"
	workerOSBuildWorkerContainerRegistryCredsDir    = "/var/secrets/osbuild-container-registry-certs" // #nosec G101
	workerOSBuildWorkerContainerRegistryCABundleDir = "/var/secrets/osbuild-container-registry-ca-bundle"

	workerOSBuildWorkerS3CredsAccessKeyIDKey     = "access-key-id"
	workerOSBuildWorkerS3CredsSecretAccessKeyKey = "secret-access-key"
	workerOSBuildWorkerCABundleKey               = "ca-bundle"

	workerSetupJobSSHKeyDir = "/var/secrets/ssh"

	workerSetupJobTemplateFile = "worker-setup-job.yaml"

	workerSetupJobNameFormat = "worker-%s-setup"
)

var (
	resultQuickRequeue = ctrl.Result{RequeueAfter: time.Second}
)

type composerConfigParametersKoji struct {
	LogLevel string
}

type composerConfigParametersWorker struct {
	LogLevel          string
	RequestJobTimeout string
	BasePath          string
}

type composerConfigParameters struct {
	Koji   composerConfigParametersKoji
	Worker composerConfigParametersWorker
}

// composerDeploymentParameters includes all the parameters needed to render the Composer Proxy Config and Deployment
type composerDeploymentParameters struct {
	ComposerDeploymentNamespace      string
	ComposerDeploymentName           string
	ComposerImageName                string
	ComposerImageTag                 string
	ProxyImageName                   string
	ProxyImageTag                    string
	ComposerAPIInternalPort          int
	ComposerAPIExternalPort          int
	WorkerAPIInternalPort            int
	WorkerAPIExternalPort            int
	PgSSLMode                        string
	ProxyCertsDir                    string
	PGSQLSecretName                  string
	ComposerConfigMapName            string
	ComposerCertsSecretName          string
	ComposerCertsSecretPublicCertKey string
	ComposerCertsSecretPrivateKeyKey string
	ComposerCertsSecretCACertKey     string
	ProxyConfigMapName               string
	ProxyWorkerAPIUpstreamTimeout    string
}

type workerSetupAnsibleConfigParameters struct {
	SSHRetries int
}

type workerSetupPlaybookParameters struct {
	RPMRepoDistribution                        string
	OSBuildComposerTag                         string
	OSBuildTag                                 string
	RHCredentialsDir                           string
	RHCredentialsUsernameKey                   string
	RHCredentialsPasswordKey                   string
	OSBuildWorkerCertsDir                      string
	WorkerAPIAddress                           string
	OSBuildWorkerConfigDir                     string
	OSBuildWorkerConfigFile                    string
	OSBuildWorkerS3CredsFile                   string
	OSBuildWorkerS3CredsDir                    string
	OSBuildWorkerS3CredsAccessKeyIDKey         string
	OSBuildWorkerS3CredsSecretAccessKeyKey     string
	OSBuildWorkerS3CABundleFile                string
	OSBuildWorkerS3CABundleDir                 string
	OSBuildWorkerS3CABundleKey                 string
	OSBuildWorkerContainerRegistryCredsDir     string
	OSBuildWorkerContainerRegistryAuthFile     string
	OSBuildWorkerContainerRegistryCertsDir     string
	OSBuildWorkerContainerRegistryCABundleFile string
	OSBuildWorkerContainerRegistryCABundleDir  string
	OSBuildWorkerContainerRegistryCABundleKey  string
}

type workerSetupInventoryParameters struct {
	Address string
	User    string
	SSHKey  string
}

type workerVMParameters struct {
	Namespace         string
	Name              string
	Hostname          string
	Username          string
	SSHKeysSecretName string
}

type workerOSBuildWorkerConfigGenericS3Parameters struct {
	Region              string
	Endpoint            string
	CABundleFile        *string
	SkipSSLVerification *bool
}

type workerOSBuildWorkerConfigS3Parameters struct {
	CredentialsFile string
	Bucket          string
	GenericS3       *workerOSBuildWorkerConfigGenericS3Parameters
}

type workerOSBuildWorkerConfigContainersParameters struct {
	AuthFile   string
	Domain     string
	PathPrefix string
	CertPath   string
	TLSVerify  bool
}

type workerOSBuildWorkerConfigParameters struct {
	S3Params         workerOSBuildWorkerConfigS3Parameters
	ContainersParams workerOSBuildWorkerConfigContainersParameters
}

type workerSetupJobParameters struct {
	WorkerConfigJobNamespace               string
	WorkerConfigJobName                    string
	WorkerConfigJobImageName               string
	WorkerConfigJobImageTag                string
	WorkerConfigAnsibleConfigConfigMapKey  string
	WorkerConfigInventoryConfigMapKey      string
	WorkerConfigPlaybookConfigMapKey       string
	WorkerConfigJobSSHKeyDir               string
	RHCredentialsDir                       string
	OSBuildWorkerCertsDir                  string
	WorkerSSHKeysSecretName                string
	WorkerConfigAnsibleConfigConfigMapName string
	WorkerConfigPlaybookConfigMapName      string
	WorkerConfigInventoryConfigMapName     string
	RedHatCredsSecretName                  string
	WorkerCertificateName                  string
	WorkerOSBuildWorkerConfigConfigMapName string
	OSBuildWorkerConfigDir                 string
	OSBuildWorkerS3CredsDir                string
	OSBuildWorkerContainerRegistryCredsDir string
	WorkerS3CredsSecretName                string
	WorkerContainerRegistryCredsSecretName string
}

// OSBuildEnvConfigReconciler reconciles a OSBuildEnvConfig object
type OSBuildEnvConfigReconciler struct {
	Scheme                     *runtime.Scheme
	OSBuildEnvConfigRepository osbuildenvconfig.Repository
	CertificateRepository      certificate.Repository
	ConfigMapRepository        configmap.Repository
	DeploymentRepository       deployment.Repository
	JobRepository              job.Repository
	ServiceRepository          service.Repository
	SecretRepository           secret.Repository
	RouteRepository            route.Repository
	VirtualMachineRepository   virtualmachine.Repository
	SSHKeyGenerator            sshkey.SSHKeyGenerator
}

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OSBuildEnvConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OSBuildEnvConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithValues("osbuildenvconfig", req.Name)
	reqLogger.Info("Reconciling OSBuildEnvConfig")

	instance, err := r.OSBuildEnvConfigRepository.Read(ctx, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found. Not a fatal error.
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "get failed for KubeFiler", "name", req.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	// now that we have the resource. determine if its alive or pending deletion
	if instance.GetDeletionTimestamp() != nil {
		// its being deleted
		if controllerutil.ContainsFinalizer(instance, osBuildOperatorFinalizer) {
			// and our finalizer is present
			return r.Finalize(ctx, reqLogger, instance)
		}
		return ctrl.Result{}, nil
	}
	// resource is alive
	return r.Update(ctx, reqLogger, instance)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OSBuildEnvConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&osbuildv1alpha1.OSBuildEnvConfig{}).
		Complete(r)
}

func (r *OSBuildEnvConfigReconciler) Update(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig) (ctrl.Result, error) {
	reqLogger.Info("Updating state for OSBuildEnvConfig",
		"name", instance.Name,
		"UID", instance.UID,
	)

	created, err := r.addFinalizer(ctx, instance)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	} else if created {
		reqLogger.Info("Added finalizer")
		return resultQuickRequeue, nil
	}

	created, err = r.ensureComposerWorkerAPIRouteExists(ctx, instance)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	} else if created {
		reqLogger.Info("Generated Route for the Composer's Worker API")
		return resultQuickRequeue, nil
	}

	composerWorkerAPIRouteHost, err := r.getComposerWorkerAPIRouteHost(ctx)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	} else if composerWorkerAPIRouteHost == nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}

	created, err = r.ensureComposerExists(ctx, reqLogger, instance, *composerWorkerAPIRouteHost)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	} else if created {
		return resultQuickRequeue, nil
	}

	created, err = r.ensureWorkersExists(ctx, reqLogger, instance, *composerWorkerAPIRouteHost)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	} else if created {
		return resultQuickRequeue, nil
	}

	return ctrl.Result{}, nil
}

func (r *OSBuildEnvConfigReconciler) ensureComposerExists(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, composerWorkerAPIRouteHost string) (bool, error) {
	created, err := r.ensureCertificateExists(
		ctx,
		reqLogger,
		instance,
		composerCertificateName,
		[]string{
			ComposerComposerAPIServiceName,
			composerWorkerAPIServiceName,
			composerWorkerAPIRouteHost,
		},
	)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Created composer certificate")
		return true, nil
	}

	updated, err := r.ensureSecretOwnedByInstance(ctx, reqLogger, instance, composerCertificateName)
	if err != nil {
		return false, err
	} else if updated {
		reqLogger.Info("Updated owner reference for the composer certificate secret", "secretName", composerCertificateName)
		return true, nil
	}

	composerDeploymentParams := composerDeploymentParameters{
		ComposerDeploymentNamespace:      conf.GlobalConf.WorkingNamespace,
		ComposerDeploymentName:           composerDeploymentName,
		ComposerImageName:                conf.GlobalConf.ComposerImageName,
		ComposerImageTag:                 conf.GlobalConf.ComposerImageTag,
		ProxyImageName:                   conf.GlobalConf.EnvoyProxyImageName,
		ProxyImageTag:                    conf.GlobalConf.EnvoyProxyImageTag,
		ComposerAPIInternalPort:          composerAPIInternalPort,
		ComposerAPIExternalPort:          composerAPIExternalPort,
		WorkerAPIInternalPort:            workerAPIInternalPort,
		WorkerAPIExternalPort:            workerAPIExternalPort,
		PgSSLMode:                        pgSSLModeDefault,
		ProxyCertsDir:                    envoyProxyCertsDir,
		PGSQLSecretName:                  "",
		ComposerConfigMapName:            composerConfigMapName,
		ComposerCertsSecretName:          composerCertificateName,
		ComposerCertsSecretPublicCertKey: certificateSecretCrtKey,
		ComposerCertsSecretPrivateKeyKey: certificateSecretKeyKey,
		ComposerCertsSecretCACertKey:     certificateSecretCACrtKey,
		ProxyConfigMapName:               composerProxyConfigMapName,
		ProxyWorkerAPIUpstreamTimeout:    (composerWorkerRequestJobTimeout * 2).String(),
	}

	if instance.Spec.Composer != nil && instance.Spec.Composer.PSQL != nil {
		composerDeploymentParams.PGSQLSecretName = instance.Spec.Composer.PSQL.ConnectionSecretReference.Name
		if instance.Spec.Composer.PSQL.SSLMode != nil {
			composerDeploymentParams.PgSSLMode = string(*instance.Spec.Composer.PSQL.SSLMode)
		}
	} else {
		return false, fmt.Errorf("creating a PSQL service is not yet implemented")
	}

	created, err = r.ensureComposerConfigMapExists(ctx, instance)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Composer configuration configMap")
		return true, nil
	}

	created, err = r.ensureComposerProxyConfigMapExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Composer Proxy configuration configMap")
		return true, nil
	}

	created, err = r.ensureComposerDeploymentExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Composer Deployment")
		return true, nil
	}

	created, err = r.ensureComposerComposerAPIServiceExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Service for the Composer's Composer API")
		return true, nil
	}

	created, err = r.ensureComposerWorkerAPIServiceExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Service for the Composer's Worker API")
		return true, nil
	}

	return false, nil
}

func (r *OSBuildEnvConfigReconciler) ensureWorkersExists(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, composerWorkerAPIRouteHost string) (bool, error) {
	created, err := r.ensureWorkerConfigAnsibleConfigExists(ctx, instance)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated ConfigMap for Ansible Config")
		return true, nil
	}

	created, err = r.ensureOSBuildWorkerConfigExists(ctx, instance)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated ConfigMap for Worker configuration")
		return true, nil
	}

	for i := range instance.Spec.Workers {
		created, err = r.ensureWorkerExists(ctx, reqLogger, instance, &instance.Spec.Workers[i], composerWorkerAPIRouteHost)
		if err != nil {
			return false, err
		} else if created {
			return true, nil
		}
	}

	return false, nil
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerExists(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, worker *osbuildv1alpha1.WorkerConfig, composerWorkerAPIRouteHost string) (bool, error) {
	var workerAddress string
	var workerUser string
	var workerSSHKeySecretName string
	var workerAPIAddress string

	if worker.VMWorkerConfig != nil {
		created, err := r.ensureWorkerSSHKeysSecretExists(ctx, instance)
		if err != nil {
			return false, err
		} else if created {
			reqLogger.Info("Generated Secret for SSH Keys")
			return true, nil
		}

		created, err = r.ensureWorkerVMExists(ctx, worker, instance)
		if err != nil {
			return false, err
		} else if created {
			reqLogger.Info("Generated VM for Worker", "name", worker.Name)
			return true, nil
		}

		created, err = r.ensureWorkerVMSSHServiceExists(ctx, worker.Name, instance)
		if err != nil {
			return false, err
		} else if created {
			reqLogger.Info("Generated SSH Service for Worker", "name", worker.Name)
			return true, nil
		}

		ready, err := r.ensureWorkerVMIsReady(ctx, worker.Name)
		if err != nil {
			return false, err
		} else if !ready {
			reqLogger.Info("Worker VM is not ready yet", "name", worker.Name)
			return true, nil
		}

		workerAddress = fmt.Sprintf(workerSSHServiceNameFormat, worker.Name)
		workerUser = workerVMUsername
		workerSSHKeySecretName = workerSSHKeysSecretName
		workerAPIAddress = composerWorkerAPIServiceName
	} else {
		workerAddress = worker.ExternalWorkerConfig.Address
		workerUser = worker.ExternalWorkerConfig.User
		workerSSHKeySecretName = worker.ExternalWorkerConfig.SSHKeySecretReference.Name
		workerAPIAddress = composerWorkerAPIRouteHost
	}

	created, err := r.ensureWorkerCertificateExists(ctx, reqLogger, instance, worker.Name)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Certificate for Worker", "name", worker.Name)
		return true, nil
	}

	updated, err := r.ensureWorkerCertificateSecretOwnedByInstance(ctx, reqLogger, instance, worker.Name)
	if err != nil {
		return false, err
	} else if updated {
		reqLogger.Info("Updated owner reference for worker certificate secret", "name", worker.Name)
		return true, nil
	}

	created, err = r.ensureWorkerConfigPlaybookExists(ctx, instance, worker.Name, workerAPIAddress)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated ConfigMap for Ansible Playbook for Worker", "name", worker.Name)
		return true, nil
	}

	created, err = r.ensureWorkerConfigInventoryExists(ctx, instance, worker.Name, workerAddress, workerUser)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated ConfigMap for Inventory file for Worker", "name", worker.Name)
		return true, nil
	}

	created, err = r.ensureWorkerSetupJobExists(ctx, instance, worker.Name, workerSSHKeySecretName)
	if err != nil {
		return false, err
	} else if created {
		reqLogger.Info("Generated Setup Job for Worker", "name", worker.Name)
		return true, nil
	}

	return false, nil
}

func (r *OSBuildEnvConfigReconciler) ensureCertificateExists(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, certificateName string, dnsNames []string) (bool, error) {
	_, err := r.CertificateRepository.Read(ctx, certificateName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		certificate, err := r.generateCertificate(ctx, instance, certificateName, dnsNames)
		if err != nil {
			return false, err
		}

		err = r.CertificateRepository.Create(ctx, certificate)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateCertificate(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, certificateName string, dnsNames []string) (*certmanagerv1.Certificate, error) {
	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certificateName,
			Namespace: conf.GlobalConf.WorkingNamespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: certificateName,
			PrivateKey: &certmanagerv1.CertificatePrivateKey{
				Algorithm: "ECDSA",
				Size:      256,
			},
			DNSNames: dnsNames,
			Duration: &metav1.Duration{
				Duration: time.Hour * certificateDuration,
			},
			IssuerRef: certmanagermetav1.ObjectReference{
				Group: "cert-manager.io",
				Kind:  "Issuer",
				Name:  conf.GlobalConf.CAIssuerName,
			},
		},
	}

	return certificate, controllerutil.SetControllerReference(instance, certificate, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerCertificateSecretOwnedByInstance(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, workerName string) (bool, error) {
	return r.ensureSecretOwnedByInstance(ctx, reqLogger, instance, fmt.Sprintf(workerCertificateNameFormat, workerName))
}

func (r *OSBuildEnvConfigReconciler) ensureSecretOwnedByInstance(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, secretName string) (bool, error) {
	secret, err := r.SecretRepository.Read(ctx, secretName, conf.GlobalConf.WorkingNamespace)
	if err != nil {
		return false, err
	}

	if len(secret.ObjectMeta.OwnerReferences) > 0 {
		// Do not check who is the owner since the cert-manager may be configured to own its secrets
		// In that case, the secret will be deleted with the certificate and the osbuildenvconfig doesn't need to own it
		return false, nil
	}

	oldSecret := secret.DeepCopy()
	err = controllerutil.SetControllerReference(instance, secret, r.Scheme)
	if err != nil {
		return false, err
	}

	err = r.SecretRepository.Patch(ctx, oldSecret, secret)
	if err != nil {
		return false, err
	}

	return true, err
}

func (r *OSBuildEnvConfigReconciler) ensureComposerConfigMapExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	composerConfigParams := composerConfigParameters{
		Koji: composerConfigParametersKoji{
			LogLevel: "info",
		},
		Worker: composerConfigParametersWorker{
			LogLevel:          "info",
			RequestJobTimeout: composerWorkerRequestJobTimeout.String(),
			BasePath:          "/api/worker/v1",
		},
	}
	return r.ensureConfigMapForTemplateFileExists(ctx, composerConfigMapName, composerConfigMapKey, composerConfigTemplateFile, composerConfigParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerProxyConfigMapExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, composerDeploymentParams *composerDeploymentParameters) (bool, error) {
	return r.ensureConfigMapForTemplateFileExists(ctx, composerProxyConfigMapName, composerProxyConfigMapKey, composerProxyConfigTemplateFile, composerDeploymentParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureConfigMapForTemplateFileExists(ctx context.Context, configMapName, configMapKey string, templateFile string, templateParams interface{}, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.ConfigMapRepository.Read(ctx, configMapName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		configMap, err := r.generateConfigMapForTemplateFile(configMapName, configMapKey, templateFile, templateParams, instance)
		if err != nil {
			return false, err
		}

		err = r.ConfigMapRepository.Create(ctx, configMap)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateConfigMapForTemplateFile(configMapName, configMapKey string, templateFile string, templateParams interface{}, instance *osbuildv1alpha1.OSBuildEnvConfig) (*corev1.ConfigMap, error) {
	buf, err := templates.LoadFromTemplateFile(templateFile, templateParams)
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: conf.GlobalConf.WorkingNamespace,
		},
		Data: map[string]string{
			configMapKey: buf.String(),
		},
	}

	return configMap, controllerutil.SetControllerReference(instance, configMap, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerDeploymentExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, composerDeploymentParams *composerDeploymentParameters) (bool, error) {
	_, err := r.DeploymentRepository.Read(ctx, composerDeploymentName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		composerDeployment, err := r.generateComposerDeployment(composerDeploymentParams, instance)
		if err != nil {
			return false, err
		}

		err = r.DeploymentRepository.Create(ctx, composerDeployment)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateComposerDeployment(composerDeploymentParams *composerDeploymentParameters, instance *osbuildv1alpha1.OSBuildEnvConfig) (*appsv1.Deployment, error) {
	buf, err := templates.LoadFromTemplateFile(composerDeploymentTemplateFile, composerDeploymentParams)
	if err != nil {
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(buf.Bytes(), nil, nil)
	if err != nil {
		return nil, err

	}

	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize the deployment object")
	}

	return deployment, controllerutil.SetControllerReference(instance, deployment, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerComposerAPIServiceExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, composerDeploymentParams *composerDeploymentParameters) (bool, error) {
	return r.ensureComposerServiceExists(ctx, ComposerComposerAPIServiceName, composerComposerAPIPortName, composerDeploymentParams.ComposerAPIExternalPort, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerWorkerAPIServiceExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, composerDeploymentParams *composerDeploymentParameters) (bool, error) {
	return r.ensureComposerServiceExists(ctx, composerWorkerAPIServiceName, composerWorkerAPIPortName, composerDeploymentParams.WorkerAPIExternalPort, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerServiceExists(ctx context.Context, serviceName, portName string, targetPort int, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	return r.ensureServiceExists(ctx, serviceName, portName, 443, targetPort, map[string]string{"app": "osbuild-composer"}, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureServiceExists(ctx context.Context, serviceName, portName string, port, targetPort int, selector map[string]string, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.ServiceRepository.Read(ctx, serviceName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		service, err := r.generateService(serviceName, portName, port, targetPort, selector, instance)
		if err != nil {
			return false, err
		}

		err = r.ServiceRepository.Create(ctx, service)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateService(serviceName, portName string, port, targetPort int, selector map[string]string, instance *osbuildv1alpha1.OSBuildEnvConfig) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: conf.GlobalConf.WorkingNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       portName,
					Port:       int32(port),
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(targetPort),
				},
			},
			Selector: selector,
		},
	}

	return service, controllerutil.SetControllerReference(instance, service, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerWorkerAPIRouteExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.RouteRepository.Read(ctx, composerWorkerAPIRouteName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		route, err := r.generateComposerWorkerAPIRoute(instance)
		if err != nil {
			return false, err
		}

		err = r.RouteRepository.Create(ctx, route)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateComposerWorkerAPIRoute(instance *osbuildv1alpha1.OSBuildEnvConfig) (*routev1.Route, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      composerWorkerAPIRouteName,
			Namespace: conf.GlobalConf.WorkingNamespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: composerWorkerAPIServiceName,
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
		},
	}
	return route, controllerutil.SetControllerReference(instance, route, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) getComposerWorkerAPIRouteHost(ctx context.Context) (*string, error) {
	route, err := r.RouteRepository.Read(ctx, composerWorkerAPIRouteName, conf.GlobalConf.WorkingNamespace)
	if err != nil {
		return nil, err
	}

	if len(route.Status.Ingress) == 0 {
		return nil, nil
	}

	ingress := &route.Status.Ingress[0]
	for _, condition := range ingress.Conditions {
		if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
			return &ingress.Host, nil
		}
	}

	return nil, nil
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerSSHKeysSecretExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.SecretRepository.Read(ctx, workerSSHKeysSecretName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		privateKeyBytes, publicKeyBytes, err := r.SSHKeyGenerator.GenerateSSHKeyPair()
		if err != nil {
			return false, err
		}
		workerSSLSecret, err := r.generateWorkerSSHKeysSecret(instance, privateKeyBytes, publicKeyBytes)
		if err != nil {
			return false, err
		}

		err = r.SecretRepository.Create(ctx, workerSSLSecret)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateWorkerSSHKeysSecret(instance *osbuildv1alpha1.OSBuildEnvConfig, privateKeyBytes, publicKeyBytes []byte) (*corev1.Secret, error) {
	immutable := true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workerSSHKeysSecretName,
			Namespace: conf.GlobalConf.WorkingNamespace,
		},
		Immutable: &immutable,
		StringData: map[string]string{
			workerSSHKeysPrivateKeyKey: string(privateKeyBytes),
			workerSSHKeysPublicKeyKey:  string(publicKeyBytes),
		},
	}

	return secret, controllerutil.SetControllerReference(instance, secret, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerVMExists(ctx context.Context, worker *osbuildv1alpha1.WorkerConfig, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.VirtualMachineRepository.Read(ctx, worker.Name, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		workerVm, err := r.generateWorkerVM(worker, instance)
		if err != nil {
			return false, err
		}

		err = r.VirtualMachineRepository.Create(ctx, workerVm)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateWorkerVM(worker *osbuildv1alpha1.WorkerConfig, instance *osbuildv1alpha1.OSBuildEnvConfig) (*kubevirtv1.VirtualMachine, error) {
	vmParameters := &workerVMParameters{
		Namespace:         conf.GlobalConf.WorkingNamespace,
		Name:              worker.Name,
		Hostname:          worker.Name,
		Username:          workerVMUsername,
		SSHKeysSecretName: workerSSHKeysSecretName,
	}

	buf, err := templates.LoadFromTemplateFile(workerVMTemplateFile, vmParameters)
	if err != nil {
		return nil, err
	}

	obj, groupVersionKind, err := serializer.NewCodecFactory(r.Scheme).UniversalDeserializer().Decode(buf.Bytes(), nil, nil)
	if err != nil {
		return nil, err
	}
	if *groupVersionKind != kubevirtv1.VirtualMachineGroupVersionKind {
		return nil, fmt.Errorf("failed to deserializer into a VirtualMachine CR")
	}

	vm, ok := obj.(*kubevirtv1.VirtualMachine)
	if !ok {
		return nil, fmt.Errorf("failed to cast into a VirtualMachine object")
	}

	vm.Spec.DataVolumeTemplates[0].Spec.Source = &worker.VMWorkerConfig.DataVolumeSource

	return vm, controllerutil.SetControllerReference(instance, vm, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerVMSSHServiceExists(ctx context.Context, workerName string, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	return r.ensureServiceExists(ctx, fmt.Sprintf(workerSSHServiceNameFormat, workerName), workerSSHPortName, 22, 22, map[string]string{"osbuild-worker": workerName}, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerVMIsReady(ctx context.Context, workerName string) (bool, error) {
	vm, err := r.VirtualMachineRepository.Read(ctx, workerName, conf.GlobalConf.WorkingNamespace)
	if err != nil {
		return false, err
	}

	for _, condition := range vm.Status.Conditions {
		if condition.Type == kubevirtv1.VirtualMachineReady && condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerCertificateExists(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig, workerName string) (bool, error) {
	return r.ensureCertificateExists(ctx, reqLogger, instance, fmt.Sprintf(workerCertificateNameFormat, workerName), []string{workerName})
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerConfigPlaybookExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, workerName, workerAPIAddress string) (bool, error) {
	workerSetupPlaybookParams := workerSetupPlaybookParameters{
		RPMRepoDistribution:                        workerRPMRepoDistribution,
		OSBuildComposerTag:                         conf.GlobalConf.WorkerOSBuildComposerVersion,
		OSBuildTag:                                 conf.GlobalConf.WorkerOSBuildVersion,
		RHCredentialsDir:                           workerRHCredentialsDir,
		RHCredentialsUsernameKey:                   workerRHCredentialsUsernameKey,
		RHCredentialsPasswordKey:                   workerRHCredentialsPasswordKey,
		OSBuildWorkerCertsDir:                      workerOSBuildWorkerCertsDir,
		WorkerAPIAddress:                           workerAPIAddress,
		OSBuildWorkerConfigDir:                     workerOSBuildWorkerConfigDir,
		OSBuildWorkerConfigFile:                    workerOSBuildWorkerConfigConfigMapKey,
		OSBuildWorkerS3CredsFile:                   workerOSBuildWorkerConfigS3CredentialsFile,
		OSBuildWorkerS3CredsDir:                    workerOSBuildWorkerS3CredsDir,
		OSBuildWorkerS3CredsAccessKeyIDKey:         workerOSBuildWorkerS3CredsAccessKeyIDKey,
		OSBuildWorkerS3CredsSecretAccessKeyKey:     workerOSBuildWorkerS3CredsSecretAccessKeyKey,
		OSBuildWorkerS3CABundleFile:                workerOSBuildWorkerConfigS3CABundleFile,
		OSBuildWorkerS3CABundleDir:                 workerOSBuildWorkerS3CABundleDir,
		OSBuildWorkerS3CABundleKey:                 workerOSBuildWorkerCABundleKey,
		OSBuildWorkerContainerRegistryCredsDir:     workerOSBuildWorkerContainerRegistryCredsDir,
		OSBuildWorkerContainerRegistryAuthFile:     workerOSBuildWorkerConfigContainerRegistryAuthFile,
		OSBuildWorkerContainerRegistryCertsDir:     workerOSBuildWorkerConfigContainerRegistryCertsDir,
		OSBuildWorkerContainerRegistryCABundleFile: workerOSBuildWorkerConfigContainerRegistryCABundleFile,
		OSBuildWorkerContainerRegistryCABundleDir:  workerOSBuildWorkerContainerRegistryCABundleDir,
		OSBuildWorkerContainerRegistryCABundleKey:  workerOSBuildWorkerCABundleKey,
	}
	return r.ensureConfigMapForTemplateFileExists(ctx, fmt.Sprintf(workerSetupPlaybookConfigMapNameFormat, workerName), workerSetupPlaybookConfigMapKey, workerSetupPlaybookTemplateFile, workerSetupPlaybookParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerConfigAnsibleConfigExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	workerSetupAnsibleConfigParams := workerSetupAnsibleConfigParameters{
		SSHRetries: workerSetupAnsibleConfigSSHRetries,
	}
	return r.ensureConfigMapForTemplateFileExists(ctx, workerSetupAnsibleConfigConfigMapName, workerSetupAnsibleConfigConfigMapKey, workerSetupAnsibleConfigTemplateFile, workerSetupAnsibleConfigParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerConfigInventoryExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, workerName, workerAddress, workerUser string) (bool, error) {
	workerSetupInventoryParams := workerSetupInventoryParameters{
		Address: workerAddress,
		User:    workerUser,
		SSHKey:  path.Join(workerSetupJobSSHKeyDir, workerSSHKeysPrivateKeyKey),
	}
	return r.ensureConfigMapForTemplateFileExists(ctx, fmt.Sprintf(workerSetupInventoryConfigMapNameFormat, workerName), workerSetupInventoryConfigMapKey, workerSetupInventoryTemplateFile, workerSetupInventoryParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureOSBuildWorkerConfigExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	workerOSBuildWorkerConfigParams := workerOSBuildWorkerConfigParameters{
		S3Params: workerOSBuildWorkerConfigS3Parameters{
			CredentialsFile: workerOSBuildWorkerConfigS3CredentialsFile,
		},
		ContainersParams: workerOSBuildWorkerConfigContainersParameters{
			AuthFile:   workerOSBuildWorkerConfigContainerRegistryAuthFile,
			Domain:     instance.Spec.ContainerRegistryService.Domain,
			PathPrefix: instance.Spec.ContainerRegistryService.PathPrefix,
			TLSVerify:  true,
		},
	}

	if instance.Spec.S3Service.AWS != nil {
		workerOSBuildWorkerConfigParams.S3Params.Bucket = instance.Spec.S3Service.AWS.Bucket
	} else {
		workerOSBuildWorkerConfigParams.S3Params.Bucket = instance.Spec.S3Service.GenericS3.Bucket
		workerOSBuildWorkerConfigParams.S3Params.GenericS3 = &workerOSBuildWorkerConfigGenericS3Parameters{
			Region:   instance.Spec.S3Service.GenericS3.Region,
			Endpoint: instance.Spec.S3Service.GenericS3.Endpoint,
		}
		if instance.Spec.S3Service.GenericS3.CABundleSecretReference != nil {
			caBundleFile := workerOSBuildWorkerConfigS3CABundleFile
			workerOSBuildWorkerConfigParams.S3Params.GenericS3.CABundleFile = &caBundleFile
		}
		if instance.Spec.S3Service.GenericS3.SkipSSLVerification != nil {
			workerOSBuildWorkerConfigParams.S3Params.GenericS3.SkipSSLVerification = instance.Spec.S3Service.GenericS3.SkipSSLVerification
		}
	}

	if instance.Spec.ContainerRegistryService.CABundleSecretReference != nil {
		workerOSBuildWorkerConfigParams.ContainersParams.CertPath = workerOSBuildWorkerConfigContainerRegistryCertsDir
	}
	if instance.Spec.ContainerRegistryService.SkipSSLVerification != nil {
		workerOSBuildWorkerConfigParams.ContainersParams.TLSVerify = !*instance.Spec.ContainerRegistryService.SkipSSLVerification
	}

	return r.ensureConfigMapForTemplateFileExists(ctx, workerOSBuildWorkerConfigConfigMapName, workerOSBuildWorkerConfigConfigMapKey, workerOSBuildWorkerConfigTemplateFile, workerOSBuildWorkerConfigParams, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureWorkerSetupJobExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, workerName, workerSSHKeySecretName string) (bool, error) {
	_, err := r.JobRepository.Read(ctx, fmt.Sprintf(workerSetupJobNameFormat, workerName), conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		workerSetupJob, err := r.generateWorkerSetupJob(workerName, workerSSHKeySecretName, instance)
		if err != nil {
			return false, err
		}

		err = r.JobRepository.Create(ctx, workerSetupJob)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, err
}

func (r *OSBuildEnvConfigReconciler) generateWorkerSetupJob(workerName, workerSSHKeySecretName string, instance *osbuildv1alpha1.OSBuildEnvConfig) (*batchv1.Job, error) {
	workerSetupJobParams := workerSetupJobParameters{
		WorkerConfigJobNamespace:               conf.GlobalConf.WorkingNamespace,
		WorkerConfigJobName:                    fmt.Sprintf(workerSetupJobNameFormat, workerName),
		WorkerConfigJobImageName:               conf.GlobalConf.WorkerSetupImageName,
		WorkerConfigJobImageTag:                conf.GlobalConf.WorkerSetupImageTag,
		WorkerConfigAnsibleConfigConfigMapKey:  workerSetupAnsibleConfigConfigMapKey,
		WorkerConfigInventoryConfigMapKey:      workerSetupInventoryConfigMapKey,
		WorkerConfigPlaybookConfigMapKey:       workerSetupPlaybookConfigMapKey,
		WorkerConfigJobSSHKeyDir:               workerSetupJobSSHKeyDir,
		RHCredentialsDir:                       workerRHCredentialsDir,
		OSBuildWorkerCertsDir:                  workerOSBuildWorkerCertsDir,
		WorkerSSHKeysSecretName:                workerSSHKeySecretName,
		WorkerConfigAnsibleConfigConfigMapName: workerSetupAnsibleConfigConfigMapName,
		WorkerConfigPlaybookConfigMapName:      fmt.Sprintf(workerSetupPlaybookConfigMapNameFormat, workerName),
		WorkerConfigInventoryConfigMapName:     fmt.Sprintf(workerSetupInventoryConfigMapNameFormat, workerName),
		RedHatCredsSecretName:                  instance.Spec.RedHatCredsSecretReference.Name,
		WorkerCertificateName:                  fmt.Sprintf(workerCertificateNameFormat, workerName),
		WorkerOSBuildWorkerConfigConfigMapName: workerOSBuildWorkerConfigConfigMapName,
		OSBuildWorkerConfigDir:                 workerOSBuildWorkerConfigDir,
		OSBuildWorkerS3CredsDir:                workerOSBuildWorkerS3CredsDir,
		OSBuildWorkerContainerRegistryCredsDir: workerOSBuildWorkerContainerRegistryCredsDir,
		WorkerContainerRegistryCredsSecretName: instance.Spec.ContainerRegistryService.CredsSecretReference.Name,
	}

	if instance.Spec.S3Service.AWS != nil {
		workerSetupJobParams.WorkerS3CredsSecretName = instance.Spec.S3Service.AWS.CredsSecretReference.Name
	} else {
		workerSetupJobParams.WorkerS3CredsSecretName = instance.Spec.S3Service.GenericS3.CredsSecretReference.Name
	}

	buf, err := templates.LoadFromTemplateFile(workerSetupJobTemplateFile, workerSetupJobParams)
	if err != nil {
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(buf.Bytes(), nil, nil)
	if err != nil {
		return nil, err

	}

	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize the job object")
	}

	if instance.Spec.S3Service.GenericS3 != nil && instance.Spec.S3Service.GenericS3.CABundleSecretReference != nil {
		caBundleSecretVolumeName := "s3-ca-bundle" // #nosec G101
		caBundleSecretVolume := corev1.Volume{
			Name: caBundleSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: instance.Spec.S3Service.GenericS3.CABundleSecretReference.Name,
				},
			},
		}
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, caBundleSecretVolume)

		caBundleVolumeMount := corev1.VolumeMount{
			Name:      caBundleSecretVolumeName,
			MountPath: workerOSBuildWorkerS3CABundleDir,
		}
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, caBundleVolumeMount)
	}

	if instance.Spec.ContainerRegistryService.CABundleSecretReference != nil {
		caBundleSecretVolumeName := "cir-ca-bundle" // #nosec G101
		caBundleSecretVolume := corev1.Volume{
			Name: caBundleSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: instance.Spec.ContainerRegistryService.CABundleSecretReference.Name,
				},
			},
		}
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, caBundleSecretVolume)

		caBundleVolumeMount := corev1.VolumeMount{
			Name:      caBundleSecretVolumeName,
			MountPath: workerOSBuildWorkerContainerRegistryCABundleDir,
		}
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, caBundleVolumeMount)
	}

	return job, controllerutil.SetControllerReference(instance, job, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) Finalize(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig) (ctrl.Result, error) {
	err := r.removeFinalizer(ctx, reqLogger, instance)
	if err != nil {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *OSBuildEnvConfigReconciler) addFinalizer(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	if controllerutil.ContainsFinalizer(instance, osBuildOperatorFinalizer) {
		return false, nil
	}
	oldInstance := instance.DeepCopy()
	controllerutil.AddFinalizer(instance, osBuildOperatorFinalizer)
	return true, r.OSBuildEnvConfigRepository.Patch(ctx, oldInstance, instance)
}

func (r *OSBuildEnvConfigReconciler) removeFinalizer(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig) error {
	reqLogger.Info("Removing finalizer")

	oldInstance := instance.DeepCopy()
	controllerutil.RemoveFinalizer(instance, osBuildOperatorFinalizer)
	return r.OSBuildEnvConfigRepository.Patch(ctx, oldInstance, instance)
}
