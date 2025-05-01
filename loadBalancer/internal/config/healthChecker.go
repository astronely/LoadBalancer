package config

import (
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"time"
)

var _ HealthCheckerConfig = (*healthCheckerConfig)(nil)

type healthCheckerConfig struct {
	Time int `yaml:"interval"`
}

func NewHealthCheckerConfig() (HealthCheckerConfig, error) {
	data, err := os.ReadFile(configName)
	if err != nil {
		return nil, err
	}

	var cfg struct {
		HealthCheckerConfig healthCheckerConfig `yaml:"healthChecker"`
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	slog.Debug("Config",
		"cfg", &cfg.HealthCheckerConfig)

	return &cfg.HealthCheckerConfig, nil
}

func (b *healthCheckerConfig) Interval() time.Duration {
	return time.Duration(b.Time) * time.Second
}
