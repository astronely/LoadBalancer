package rateLimiter

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// TokenBucket part of RateLimiting algorithm
type TokenBucket struct {
	config     config.RateLimiterConfig
	tokens     float64
	lastRefill time.Time
	mu         sync.RWMutex
}

func NewTokenBucket(cfg config.RateLimiterConfig) *TokenBucket {
	slog.Debug("TokenBucket initialized",
		"tokens", cfg.Capacity(),
		"refillRate", cfg.RefillRate())

	return &TokenBucket{
		config:     cfg,
		tokens:     cfg.Capacity(),
		lastRefill: time.Now(),
	}
}

func (t *TokenBucket) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tokens >= 1 {
		t.tokens--
		return true
	}
	return false
}

func (t *TokenBucket) refill() {
	t.mu.Lock()
	defer t.mu.Unlock()
	toAdd := t.config.RefillRate()
	t.tokens += toAdd
	if t.tokens > t.config.Capacity() {
		t.tokens = t.config.Capacity()
	}
}

// RateLimiter Implementation of RateLimiting algorithm
type RateLimiter struct {
	config       config.RateLimiterConfig
	customConfig map[string]config.RateLimiterConfig
	buckets      map[string]*TokenBucket
	ticker       *time.Ticker
	done         chan struct{}
	mu           sync.RWMutex
}

func NewRateLimiter(cfg config.RateLimiterConfig) service.RateLimiter {
	return &RateLimiter{
		config:       cfg,
		customConfig: make(map[string]config.RateLimiterConfig),
		buckets:      make(map[string]*TokenBucket),
		ticker:       time.NewTicker(time.Duration(cfg.RefillInterval()) * time.Second),
		done:         make(chan struct{}),
	}
}

// Start ticker to refill tokens
func (r *RateLimiter) Start() {
	for {
		select {
		case <-r.ticker.C:
			r.mu.RLock()
			for id, bucket := range r.buckets {
				bucket.refill()
				slog.Info("RateLimiter ticker",
					"bucket ID", id,
					"bucket capacity", bucket.tokens)
			}
			r.mu.RUnlock()
		case <-r.done:
			r.ticker.Stop()
			return
		}
	}
}

// Stop ticker
func (r *RateLimiter) Stop() {
	close(r.done)
}

// Allow checking for enough tokens for a request
func (r *RateLimiter) Allow(id string) bool {
	r.mu.RLock()
	bucket, ok := r.buckets[id]
	r.mu.RUnlock()

	if !ok {
		r.mu.Lock()
		if bucket, ok = r.buckets[id]; !ok {
			cfg, ok2 := r.customConfig[id]
			if !ok2 {
				cfg = r.config
			}
			bucket = NewTokenBucket(cfg)
			r.buckets[id] = bucket
		}
		r.mu.Unlock()
	}
	return bucket.Allow()
}

func (r *RateLimiter) SetClientConfig(id string, cfg config.RateLimiterConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.customConfig[id] = cfg
	r.buckets[id] = NewTokenBucket(cfg)
}

// Middleware is http.HandlerFunc used for each response
func (r *RateLimiter) Middleware(next http.Handler, keyFunc func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		id := keyFunc(req)
		if !r.Allow(id) {
			slog.Info("Too many requests",
				"id", id,
				"url", req.URL.String(),
				"method", req.Method,
			)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) Clients() map[string]*rateLimiter.TokenBucket {
	clients := make(map[string]*rateLimiter.TokenBucket)
	r.mu.RLock()
	for id, bucket := range r.buckets {
		clients[id] = &rateLimiter.TokenBucket{
			Config:     bucket.config,
			Tokens:     bucket.tokens,
			LastRefill: bucket.lastRefill,
		}
	}
	r.mu.RUnlock()
	return clients
}
