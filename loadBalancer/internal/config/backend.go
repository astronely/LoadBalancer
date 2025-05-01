package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

var _ BackendConfig = (*backendConfig)(nil)

type backendConfig struct {
	BackendAddresses []string `yaml:"addresses"`
}

func NewBackendConfig() (BackendConfig, error) {
	configFile, err := os.Open(configName)
	if err != nil {
		return nil, errors.New("no config file found with name " + configName)
	}
	defer configFile.Close()

	var cfg struct {
		BackendConfig backendConfig `yaml:"backends"`
	}

	d := yaml.NewDecoder(configFile)

	if err = d.Decode(&cfg); err != nil {
		return nil, errors.New("error parsing config file " + configName)
	}

	slog.Debug("Config",
		"cfg", cfg)

	return &cfg.BackendConfig, nil
}

func (b *backendConfig) Addresses() []string {
	return b.BackendAddresses
}
