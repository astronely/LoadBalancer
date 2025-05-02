package app

import (
	"context"
	"github.com/astronely/loadBalancer/loadBalancer/internal/api/loadBalancer"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/repository"
	rateLimiterRepo "github.com/astronely/loadBalancer/loadBalancer/internal/repository/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/backend"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/proxy"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/roundRobin"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/workerPool"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/client/cache"
	cacheClient "github.com/astronely/loadBalancer/loadBalancer/pkg/client/cache/redis"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"github.com/redis/go-redis/v9"
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
	redisConfig          config.RedisConfig

	rdb         *redis.Client
	redisClient cache.RedisClient

	backends []service.Backend

	proxy http.Handler

	roundRobin  service.LoadBalancer
	rateLimiter service.RateLimiter
	workerPool  service.WorkerPool

	rateLimiterRepository repository.RateLimiterRepository

	loadBalancerImpl *loadBalancer.Implementation
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

func (s *serviceProvider) RedisConfig() config.RedisConfig {
	if s.redisConfig == nil {
		cfg, err := config.NewRedisConfig()
		if err != nil {
			panic("Error loading redis config: " + err.Error())
		}
		s.redisConfig = cfg
	}
	return s.redisConfig
}

func (s *serviceProvider) Rdb(_ context.Context) *redis.Client {
	if s.rdb == nil {
		rdb := redis.NewClient(&redis.Options{
			Addr:            s.RedisConfig().Address(),
			MaxIdleConns:    s.RedisConfig().MaxIdle(),
			ConnMaxIdleTime: s.RedisConfig().IdleTimeout(),
		})
		slog.Debug("Redis initialized",
			"addr", s.RedisConfig().Address())

		closer.Add(func() error {
			err := rdb.Close()
			if err != nil {
				slog.Error("Error closing redis connection: " + err.Error())
				return err
			}

			slog.Info("Redis closed gracefully")
			return nil
		})

		s.rdb = rdb
	}

	return s.rdb
}

func (s *serviceProvider) RedisClient(ctx context.Context) cache.RedisClient {
	if s.redisClient == nil {
		redisClient := cacheClient.NewClient(s.Rdb(ctx), s.RedisConfig())
		s.redisClient = redisClient
	}

	return s.redisClient
}

func (s *serviceProvider) Proxy(ctx context.Context) http.Handler {
	if s.proxy == nil {
		newProxy := proxy.NewProxy(s.RoundRobin(ctx))

		keyFunc := func(r *http.Request) string {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			return host
		}

		limitedProxy := s.RateLimiter(ctx).Middleware(newProxy, keyFunc)

		// Start RateLimiter ticker to refill tokens
		go s.RateLimiter(ctx).Start()
		closer.Add(func() error {
			s.RateLimiter(ctx).Stop()
			slog.Info("RateLimiter Ticker stopped")
			return nil
		})

		s.proxy = limitedProxy
	}
	return s.proxy
}

func (s *serviceProvider) RoundRobin(ctx context.Context) service.LoadBalancer {
	if s.roundRobin == nil {
		s.roundRobin = roundRobin.NewRoundRobin(s.Backends(ctx))
	}
	return s.roundRobin
}

func (s *serviceProvider) Backends(_ context.Context) []service.Backend {
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

func (s *serviceProvider) RateLimiter(ctx context.Context) service.RateLimiter {
	if s.rateLimiter == nil {
		s.rateLimiter = rateLimiter.NewRateLimiter(s.RateLimiterConfig(), s.RateLimiterRepository(ctx))
	}
	return s.rateLimiter
}

func (s *serviceProvider) WorkerPool(_ context.Context) service.WorkerPool {
	if s.workerPool == nil {
		s.workerPool = workerPool.NewWorkerPool(s.WorkerPoolConfig())
	}
	return s.workerPool
}

func (s *serviceProvider) RateLimiterRepository(ctx context.Context) repository.RateLimiterRepository {
	if s.rateLimiterRepository == nil {
		s.rateLimiterRepository = rateLimiterRepo.NewRepository(s.RedisClient(ctx))
	}
	return s.rateLimiterRepository
}

func (s *serviceProvider) LoadBalancerImpl(ctx context.Context) *loadBalancer.Implementation {
	if s.loadBalancerImpl == nil {
		s.loadBalancerImpl = loadBalancer.NewImplementation(s.Proxy(ctx), s.RateLimiter(ctx), s.WorkerPool(ctx))
	}
	return s.loadBalancerImpl
}
