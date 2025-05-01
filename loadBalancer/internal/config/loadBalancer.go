package config

import (
	"gopkg.in/yaml.v3"
	"log/slog"
	"net"
	"os"
)

var _ LoadBalancerConfig = (*loadBalancerConfig)(nil)

type loadBalancerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

func NewLoadBalancerConfig() (LoadBalancerConfig, error) {
	data, err := os.ReadFile(configName)
	if err != nil {
		return nil, err
	}

	var cfg struct {
		LoadBalancerConfig loadBalancerConfig `yaml:"loadBalancer"`
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	slog.Debug("Config",
		"cfg", &cfg.LoadBalancerConfig)

	return &cfg.LoadBalancerConfig, nil
}

func (b *loadBalancerConfig) Address() string {
	return net.JoinHostPort(b.Host, b.Port)
}
