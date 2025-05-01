package config

import (
	"github.com/joho/godotenv"
	"time"
)

const (
	configName = "local.yaml"
)

func Load(path string) error {
	err := godotenv.Load(path)
	return err
}

type LoadBalancerConfig interface {
	Address() string
}

type BackendConfig interface {
	Addresses() []string
}

type HealthCheckerConfig interface {
	Interval() time.Duration
}

type RateLimiterConfig interface {
	Capacity() int
	RefillRate() int
}
