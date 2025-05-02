package config

import (
	"errors"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

var _ RateLimiterConfig = (*rateLimiterConfig)(nil)

// rateLimiterConfig for default clients
type rateLimiterConfig struct {
	BucketCapacity       float64 `yaml:"capacity"`
	BucketRefillRate     float64 `yaml:"refill_rate"`
	BucketRefillInterval float64 `yaml:"refill_interval"`
}

// rateLimiterCustomConfig for VIP clients
type rateLimiterCustomConfig struct {
	BucketCapacity       float64 `yaml:"capacity_vip"`
	BucketRefillRate     float64 `yaml:"refill_rate_vip"`
	BucketRefillInterval float64 `yaml:"refill_interval"`
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

func (r *rateLimiterConfig) Capacity() float64 {
	return r.BucketCapacity
}

func (r *rateLimiterConfig) RefillRate() float64 {
	return r.BucketRefillRate
}

func (r *rateLimiterConfig) RefillInterval() float64 {
	return r.BucketRefillInterval
}

func NewRateLimiterCustomConfig(info *rateLimiter.Info) (RateLimiterConfig, error) {

	return &rateLimiterCustomConfig{
		BucketCapacity:       info.Capacity,
		BucketRefillRate:     info.RefillRate,
		BucketRefillInterval: info.Interval,
	}, nil
}

func (r *rateLimiterCustomConfig) Capacity() float64 {
	return r.BucketCapacity
}

func (r *rateLimiterCustomConfig) RefillRate() float64 {
	return r.BucketRefillRate
}

func (r *rateLimiterCustomConfig) RefillInterval() float64 {
	return r.BucketRefillInterval
}
