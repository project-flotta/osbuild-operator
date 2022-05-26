package httpapi

import "github.com/kelseyhightower/envconfig"

type HttpAPIConfig struct {
	// The port of the HTTPs server
	HttpPort uint16 `envconfig:"HTTP_PORT" default:"8080"`

	// The port of the HTTPs server
	ProbesPort uint16 `envconfig:"PROBES_PORT" default:"8081"`

	// Kubeconfig specifies path to a kubeconfig file if the server is run outside of a cluster
	Kubeconfig string `envconfig:"KUBECONFIG" default:""`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

var GlobalHttpAPIConf *HttpAPIConfig

func Load() error {
	var c HttpAPIConfig
	err := envconfig.Process("", &c)
	if err != nil {
		return err
	}
	GlobalHttpAPIConf = &c
	return nil
}
