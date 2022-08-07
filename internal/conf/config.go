package conf

import (
	"github.com/kelseyhightower/envconfig"
)

type OperatorConfig struct {
	// The address the metric endpoint binds to.
	MetricsAddr string `envconfig:"METRICS_ADDR" default:"127.0.0.1:8080"`

	// The address the probe endpoint binds to.
	ProbeAddr string `envconfig:"PROBE_ADDR" default:":8081"`

	// If Webhooks are enabled, an admission webhook is created and checked when
	// any user submits any change to any project-flotta.io CRD.
	EnableWebhooks bool `envconfig:"ENABLE_WEBHOOKS" default:"true"`

	// WebhookPort is the port that the webhook server serves at.
	WebhookPort int `envconfig:"WEBHOOK_PORT" default:"9443"`

	// Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool `envconfig:"LEADER_ELECT" default:"true"`

	LeaderElectionResourceName string `envconfig:"LEADER_ELECTION_RESOURCE_NAME" default:"bfdcaedc.osbuilder.project-flotta.io"`

	// WorkingNamespace must be set to the operator's namespace
	WorkingNamespace string `envconfig:"WORKING_NAMESPACE" required:"true"`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// CAIssuerName is the name of the cert-manager issuer in the operator's namespace used for the environment setup
	CAIssuerName string `envconfig:"CA_ISSUER_NAME" required:"true"`

	// TemplatesDir is the path to the directory where the templates are stored
	TemplatesDir string `envconfig:"TEMPLATES_DIR" default:"/templates"`

	// WorkerSetupImageName is the name of the image to use for the Worker Setup job
	WorkerSetupImageName string `envconfig:"WORKER_SETUP_IMAGE_NAME" default:"quay.io/project-flotta/osbuild-operator-worker-setup"`

	// WorkerSetupImageTag is the tag of the image to use for the Worker Setup job
	WorkerSetupImageTag string `envconfig:"WORKER_SETUP_IMAGE_TAG" default:"v0.1"`

	// ComposerImageName is the name of the image to use for the Composer API
	ComposerImageName string `envconfig:"COMPOSER_IMAGE_NAME" default:"quay.io/app-sre/composer"`

	// ComposerImageTag is the tag of the image to use for the Composer API
	ComposerImageTag string `envconfig:"COMPOSER_IMAGE_TAG" default:"d3dde77"`

	// EnvoyProxyImageName is the name of the image to use for the Composer's Proxy server
	EnvoyProxyImageName string `envconfig:"ENVOY_PROXY_IMAGE_NAME" default:"docker.io/envoyproxy/envoy"`

	// EnvoyProxyImageTag is the tag of the image to use for the Composer API
	EnvoyProxyImageTag string `envconfig:"ENVOY_PROXY_IMAGE_TAG" default:"v1.21-latest"`

	// WorkerOSBuildComposerVersion is the release tag of OSBuild-Composer for the osbuild-worker RPM
	WorkerOSBuildComposerVersion string `envconfig:"WORKER_OSBUILD_COMPOSER_VERSION" default:"v59"`

	// WorkerOSBuildVersion is the release tag of OSBuild for the osbuild RPM
	WorkerOSBuildVersion string `envconfig:"WORKER_OSBUILD_VERSION" default:"v63"`

	// RepositoriesDir is the path to the directory where the repositories information files are stored
	RepositoriesDir string `envconfig:"REPOSITORIES_DIR" default:"/etc/osbuild/repositories"`

	// BaseISOContainerImage is the container image to run the iso-package job
	BaseISOContainerImage string `envconfig:"BASE_ISO_CONTAINER_IMAGE" required:"true" default:"controller:latest"`
}

var GlobalConf *OperatorConfig

func Load() error {
	var c OperatorConfig
	err := envconfig.Process("", &c)
	if err != nil {
		return err
	}
	GlobalConf = &c
	return nil
}
