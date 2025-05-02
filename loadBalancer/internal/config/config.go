package config

import (
	"log/slog"
	"time"
)

var configName string

func Load(cfgName string) error {
	configName = cfgName
	slog.Info("Loading config file",
		"filename", cfgName)
	return nil
}

// Interfaces for every config

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
	Capacity() float64
	RefillRate() float64
	RefillInterval() float64
}

type WorkerPoolConfig interface {
	PoolSize() int
	QueueSize() int
}
