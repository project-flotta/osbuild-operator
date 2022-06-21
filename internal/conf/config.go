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
