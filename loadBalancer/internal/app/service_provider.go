package app

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/backend"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/domain"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/proxy"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/roundRobin"
	"log/slog"
	"net/http"
)

type serviceProvider struct {
	loadBalancerConfig  config.LoadBalancerConfig
	backendConfig       config.BackendConfig
	healthCheckerConfig config.HealthCheckerConfig

	proxy http.Handler

	roundRobin domain.LoadBalancer

	backends []*backend.Backend
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) LoadBalancerConfig() config.LoadBalancerConfig {
	if s.loadBalancerConfig == nil {
		cfg, err := config.NewLoadBalancerConfig()
		if err != nil {
			panic("Error loading load balancer config: " + err.Error())
		}
		s.loadBalancerConfig = cfg
	}
	return s.loadBalancerConfig
}
func (s *serviceProvider) BackendConfig() config.BackendConfig {
	if s.backendConfig == nil {
		cfg, err := config.NewBackendConfig()
		if err != nil {
			panic("Error loading backend config: " + err.Error())
		}
		s.backendConfig = cfg
	}
	return s.backendConfig
}

func (s *serviceProvider) HealthCheckerConfig() config.HealthCheckerConfig {
	if s.healthCheckerConfig == nil {
		cfg, err := config.NewHealthCheckerConfig()
		if err != nil {
			panic("Error loading health checker config: " + err.Error())
		}
		s.healthCheckerConfig = cfg
	}
	return s.healthCheckerConfig
}

func (s *serviceProvider) Proxy() http.Handler {
	if s.proxy == nil {
		s.proxy = proxy.NewProxy(s.RoundRobin())
	}
	return s.proxy
}

func (s *serviceProvider) RoundRobin() domain.LoadBalancer {
	if s.roundRobin == nil {
		s.roundRobin = roundRobin.NewRoundRobin(s.Backends())
	}

	return s.roundRobin
}

func (s *serviceProvider) Backends() []*backend.Backend {
	if s.backends == nil {
		cfg := s.BackendConfig()
		for _, address := range cfg.Addresses() {
			b, err := backend.NewBackend(address)
			if err != nil {
				slog.Error("Error creating backend",
					"error", err.Error(),
				)
			}
			s.backends = append(s.backends, b)
		}
	}
	return s.backends
}
