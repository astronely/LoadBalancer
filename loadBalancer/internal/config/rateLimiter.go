package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

var _ RateLimiterConfig = (*rateLimiterConfig)(nil)

type rateLimiterConfig struct {
	BucketCapacity   int `yaml:"capacity"`
	BucketRefillRate int `yaml:"refill_rate"`
}

func NewRateLimiterConfig() (RateLimiterConfig, error) {
	configFile, err := os.Open(configName)
	if err != nil {
		return nil, errors.New("no config file found with name " + configName)
	}
	defer configFile.Close()

	var cfg struct {
		RateLimiterConfig rateLimiterConfig `yaml:"rateLimiter"`
	}

	d := yaml.NewDecoder(configFile)

	if err = d.Decode(&cfg); err != nil {
		return nil, errors.New("error parsing config file " + configName)
	}

	slog.Debug("Config",
		"cfg", cfg)

	return &cfg.RateLimiterConfig, nil
}

func (r *rateLimiterConfig) Capacity() int {
	return r.BucketCapacity
}

func (r *rateLimiterConfig) RefillRate() int {
	return r.BucketRefillRate
}
