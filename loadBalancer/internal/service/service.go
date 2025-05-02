package service

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/workerPool"
	"net/http"
	"net/url"
)

// RateLimiter interface for rateLimiter service
type RateLimiter interface {
	Start()
	Stop()
	SetClientConfig(id string, cfg config.RateLimiterConfig)
	Allow(id string) bool
	Middleware(next http.Handler, keyFunc func(*http.Request) string) http.Handler
	Clients() map[string]*rateLimiter.TokenBucket
}

// WorkerPool interface for workerPool service
type WorkerPool interface {
	Start() chan workerPool.Job
	Worker(jobs <-chan workerPool.Job)
}

// LoadBalancer interface for loadBalancer service
type LoadBalancer interface {
	Next() (Backend, error)
}

type Backend interface {
	SetAlive(value bool)
	IsAlive() bool
	GetURL() *url.URL
}
