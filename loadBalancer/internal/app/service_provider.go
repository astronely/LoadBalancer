package app

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/backend"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/proxy"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/roundRobin"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/workerPool"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"log/slog"
	"net"
	"net/http"
)

// serviceProvider - DI container
type serviceProvider struct {
	loadBalancerConfig   config.LoadBalancerConfig
	backendConfig        config.BackendConfig
	healthCheckerConfig  config.HealthCheckerConfig
	rateLimiterConfig    config.RateLimiterConfig
	rateLimiterVipConfig config.RateLimiterConfig
	workerPoolConfig     config.WorkerPoolConfig

	backends []service.Backend

	proxy http.Handler

	roundRobin  service.LoadBalancer
	rateLimiter service.RateLimiter
	workerPool  service.WorkerPool
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

func (s *serviceProvider) RateLimiterConfig() config.RateLimiterConfig {
	if s.rateLimiterConfig == nil {
		cfg, err := config.NewRateLimiterConfig()
		if err != nil {
			panic("Error loading rateLimiter config: " + err.Error())
		}
		s.rateLimiterConfig = cfg
	}
	return s.rateLimiterConfig
}

func (s *serviceProvider) RateLimiterVipConfig() config.RateLimiterConfig {
	if s.rateLimiterVipConfig == nil {
		cfg, err := config.NewRateLimiterVipConfig()
		if err != nil {
			panic("Error loading rateLimiter config: " + err.Error())
		}
		s.rateLimiterVipConfig = cfg
	}
	return s.rateLimiterVipConfig
}

func (s *serviceProvider) WorkerPoolConfig() config.WorkerPoolConfig {
	if s.workerPoolConfig == nil {
		cfg, err := config.NewWorkerPoolConfig()
		if err != nil {
			panic("Error loading worker pool config: " + err.Error())
		}
		s.workerPoolConfig = cfg
	}
	return s.workerPoolConfig
}

func (s *serviceProvider) Proxy() http.Handler {
	if s.proxy == nil {
		newProxy := proxy.NewProxy(s.RoundRobin())

		keyFunc := func(r *http.Request) string {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			return host
		}

		limitedProxy := s.RateLimiter().Middleware(newProxy, keyFunc)

		// Start RateLimiter ticker to refill tokens
		go s.RateLimiter().Start()
		closer.Add(func() error {
			s.RateLimiter().Stop()
			slog.Info("RateLimiter Ticker stopped")
			return nil
		})

		s.proxy = limitedProxy
	}
	return s.proxy
}

func (s *serviceProvider) RoundRobin() service.LoadBalancer {
	if s.roundRobin == nil {
		s.roundRobin = roundRobin.NewRoundRobin(s.Backends())
	}
	return s.roundRobin
}

func (s *serviceProvider) Backends() []service.Backend {
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

func (s *serviceProvider) RateLimiter() service.RateLimiter {
	if s.rateLimiter == nil {
		s.rateLimiter = rateLimiter.NewRateLimiter(s.RateLimiterConfig())
	}
	return s.rateLimiter
}

func (s *serviceProvider) WorkerPool() service.WorkerPool {
	if s.workerPool == nil {
		s.workerPool = workerPool.NewWorkerPool(s.WorkerPoolConfig())
	}
	return s.workerPool
}
