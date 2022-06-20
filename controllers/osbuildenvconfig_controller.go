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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"

	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/repository/certificate"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/deployment"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildenvconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/internal/repository/service"
	"github.com/project-flotta/osbuild-operator/internal/templates"
)

const (
	osBuildOperatorFinalizer = "osbuilder.project-flotta.io/osBuildOperatorFinalizer"

	certificateDuration = 87600

	composerCertificateName = "composer-cert"

	composerComposerAPIServiceName = "osbuild-composer"
	composerComposerAPIPortName    = "composer-api"
	composerWorkerAPIServiceName   = "osbuild-worker"
	composerWorkerAPIPortName      = "worker-api"

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

	composerImageName = "quay.io/app-sre/composer"
	composerImageTag  = "fc87b17"

	envoyProxyImageName = "docker.io/envoyproxy/envoy"
	envoyProxyImageTag  = "v1.21-latest"

	composerAPIInternalPort = 18080
	composerAPIExternalPort = 8080
	workerAPIInternalPort   = 18700
	workerAPIExternalPort   = 8700

	envoyProxyCertsDir = "/etc/certs"
)

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
}

// OSBuildEnvConfigReconciler reconciles a OSBuildEnvConfig object
type OSBuildEnvConfigReconciler struct {
	Scheme                     *runtime.Scheme
	OSBuildEnvConfigRepository osbuildenvconfig.Repository
	CertificateRepository      certificate.Repository
	ConfigMapRepository        configmap.Repository
	DeploymentRepository       deployment.Repository
	ServiceRepository          service.Repository
	SecretRepository           secret.Repository
}

//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=osbuilder.project-flotta.io,resources=osbuildenvconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
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
		return ctrl.Result{Requeue: true}, err
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
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Added finalizer")
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = r.ensureCertificateExists(
		ctx,
		reqLogger,
		instance,
		composerCertificateName,
		[]string{
			composerComposerAPIServiceName,
			composerWorkerAPIServiceName,
		},
	)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Created composer certificate")
		return ctrl.Result{Requeue: true}, nil
	}

	composerDeploymentParams := composerDeploymentParameters{
		ComposerDeploymentNamespace:      conf.GlobalConf.WorkingNamespace,
		ComposerDeploymentName:           composerDeploymentName,
		ComposerImageName:                composerImageName,
		ComposerImageTag:                 composerImageTag,
		ProxyImageName:                   envoyProxyImageName,
		ProxyImageTag:                    envoyProxyImageTag,
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
	}

	if instance.Spec.Composer != nil && instance.Spec.Composer.PSQL != nil {
		composerDeploymentParams.PGSQLSecretName = instance.Spec.Composer.PSQL.ConnectionSecretReference.Name
		if instance.Spec.Composer.PSQL.SSLMode != nil {
			composerDeploymentParams.PgSSLMode = string(*instance.Spec.Composer.PSQL.SSLMode)
		}
	} else {
		return ctrl.Result{Requeue: true}, fmt.Errorf("creating a PSQL service is not yet implemented")
	}

	created, err = r.ensureComposerConfigMapExists(ctx, instance)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Generated Composer configuration configMap")
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = r.ensureComposerProxyConfigMapExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Generated Composer Proxy configuration configMap")
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = r.ensureComposerDeploymentExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Generated Composer Deployment")
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = r.ensureComposerComposerAPIServiceExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Generated Service for the Composer's Composer API")
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = r.ensureComposerWorkerAPIServiceExists(ctx, instance, &composerDeploymentParams)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if created {
		reqLogger.Info("Generated Service for the Composer's Worker API")
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
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

func (r *OSBuildEnvConfigReconciler) ensureComposerConfigMapExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	composerConfigParams := struct {
		Koji struct {
			LogLevel string
		}
		Worker struct {
			LogLevel          string
			RequestJobTimeout string
			BasePath          string
		}
	}{
		Koji: struct{ LogLevel string }{
			LogLevel: "info",
		},
		Worker: struct {
			LogLevel          string
			RequestJobTimeout string
			BasePath          string
		}{
			LogLevel:          "info",
			RequestJobTimeout: "20s",
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
	return r.ensureComposerServiceExists(ctx, composerComposerAPIServiceName, composerComposerAPIPortName, composerDeploymentParams.ComposerAPIExternalPort, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerWorkerAPIServiceExists(ctx context.Context, instance *osbuildv1alpha1.OSBuildEnvConfig, composerDeploymentParams *composerDeploymentParameters) (bool, error) {
	return r.ensureComposerServiceExists(ctx, composerWorkerAPIServiceName, composerWorkerAPIPortName, composerDeploymentParams.WorkerAPIExternalPort, instance)
}

func (r *OSBuildEnvConfigReconciler) ensureComposerServiceExists(ctx context.Context, serviceName, portName string, targetPort int, instance *osbuildv1alpha1.OSBuildEnvConfig) (bool, error) {
	_, err := r.ServiceRepository.Read(ctx, serviceName, conf.GlobalConf.WorkingNamespace)
	if err == nil {
		return false, nil
	}

	if errors.IsNotFound(err) {
		service, err := r.generateComposerService(serviceName, portName, targetPort, instance)
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

func (r *OSBuildEnvConfigReconciler) generateComposerService(serviceName, portName string, targetPort int, instance *osbuildv1alpha1.OSBuildEnvConfig) (*corev1.Service, error) {
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
					Port:       443,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(targetPort),
				},
			},
			Selector: map[string]string{
				"app": "osbuild-composer",
			},
		},
	}

	return service, controllerutil.SetControllerReference(instance, service, r.Scheme)
}

func (r *OSBuildEnvConfigReconciler) Finalize(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig) (ctrl.Result, error) {
	// By default, cert-manager does not delete the secret is creates when the certificate is deleted so need to delete it manually
	deleted, err := r.ensureComposerCertSecretDeleted(ctx)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	if deleted {
		reqLogger.Info("composer certificate secret was deleted")
		return ctrl.Result{Requeue: true}, nil
	}

	err = r.removeFinalizer(ctx, reqLogger, instance)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
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

func (r *OSBuildEnvConfigReconciler) ensureComposerCertSecretDeleted(ctx context.Context) (bool, error) {
	return r.ensureSecretDeleted(ctx, composerCertificateName)
}

func (r *OSBuildEnvConfigReconciler) ensureSecretDeleted(ctx context.Context, secretName string) (bool, error) {
	secret, err := r.SecretRepository.Read(ctx, secretName, conf.GlobalConf.WorkingNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	err = r.SecretRepository.Delete(ctx, secret)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *OSBuildEnvConfigReconciler) removeFinalizer(ctx context.Context, reqLogger logr.Logger, instance *osbuildv1alpha1.OSBuildEnvConfig) error {
	reqLogger.Info("Removing finalizer")

	oldInstance := instance.DeepCopy()
	controllerutil.RemoveFinalizer(instance, osBuildOperatorFinalizer)
	return r.OSBuildEnvConfigRepository.Patch(ctx, oldInstance, instance)
}
