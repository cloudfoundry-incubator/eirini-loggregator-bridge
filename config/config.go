package config

import (
	"errors"
)

type LoggregatorOptions struct {
	CAPath, CertPath, KeyPath, Endpoint string
}

type ConfigType struct {
	Namespace           string `mapstructure:"namespace"`
	LoggregatorEndpoint string `mapstructure:"loggregator-endpoint"`
	LoggregatorCAPath   string `mapstructure:"loggregator-ca-path"`
	LoggregatorCertPath string `mapstructure:"loggregator-cert-path"`
	LoggregatorKeyPath  string `mapstructure:"loggregator-key-path"`
}

func (conf ConfigType) GetLoggregatorOptions() LoggregatorOptions {
	return LoggregatorOptions{
		CAPath:   conf.LoggregatorCAPath,
		CertPath: conf.LoggregatorCertPath,
		KeyPath:  conf.LoggregatorKeyPath,
		Endpoint: conf.LoggregatorEndpoint,
	}
}

func (conf ConfigType) Validate() error {
	if conf.Namespace == "" {
		return errors.New("namespace is missing from configuration")
	}
	if conf.LoggregatorEndpoint == "" {
		return errors.New("loggregator-endpoint is missing from configuration")
	}
	if conf.LoggregatorCAPath == "" {
		return errors.New("loggregator-ca-path is missing from configuration")
	}
	if conf.LoggregatorCertPath == "" {
		return errors.New("loggregator-cert-path is missing from configuration")
	}
	if conf.LoggregatorKeyPath == "" {
		return errors.New("loggregator-key-path is missing from configuration")
	}
	return nil
}
