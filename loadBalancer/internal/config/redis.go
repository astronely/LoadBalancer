package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log/slog"
	"net"
	"os"
	"time"
)

type redisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`

	ConnectionTimeoutVar int `yaml:"connection_timeout_sec"`

	MaxIdleVar     int `yaml:"max_idle"`
	IdleTimeoutVar int `yaml:"idle_timeout_sec"`
}

func NewRedisConfig() (RedisConfig, error) {
	configFile, err := os.Open(configName)
	if err != nil {
		return nil, errors.New("no config file found with name " + configName)
	}
	defer configFile.Close()

	var cfg struct {
		RedisConfig redisConfig `yaml:"redis"`
	}

	d := yaml.NewDecoder(configFile)

	if err = d.Decode(&cfg); err != nil {
		return nil, errors.New("error parsing config file " + configName)
	}

	slog.Info("Config",
		"cfg", cfg)

	return &cfg.RedisConfig, nil
}

func (cfg *redisConfig) Address() string {
	return net.JoinHostPort(cfg.Host, cfg.Port)
}

func (cfg *redisConfig) ConnectionTimeout() time.Duration {
	return time.Duration(cfg.ConnectionTimeoutVar) * time.Second
}

func (cfg *redisConfig) MaxIdle() int {
	return cfg.MaxIdleVar
}

func (cfg *redisConfig) IdleTimeout() time.Duration {
	return time.Duration(cfg.IdleTimeoutVar) * time.Second
}
