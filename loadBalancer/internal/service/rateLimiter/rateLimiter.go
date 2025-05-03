package rateLimiter

import (
	"context"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	TokenBucketModel "github.com/astronely/loadBalancer/loadBalancer/internal/model/TokenBucket"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/repository"
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

	rateLimiterRepo repository.RateLimiterRepository
}

func NewRateLimiter(cfg config.RateLimiterConfig, repo repository.RateLimiterRepository) service.RateLimiter {
	return &RateLimiter{
		config:       cfg,
		customConfig: make(map[string]config.RateLimiterConfig),
		buckets:      make(map[string]*TokenBucket),
		ticker:       time.NewTicker(time.Duration(cfg.RefillInterval()) * time.Second),
		done:         make(chan struct{}),

		rateLimiterRepo: repo,
	}
}

// Start ticker to refill tokens
func (r *RateLimiter) Start() {
	for {
		select {
		case <-r.ticker.C:
			r.mu.Lock()
			for id, bucket := range r.buckets {
				bucket.refill()
				slog.Info("RateLimiter ticker",
					"bucket ID", id,
					"bucket capacity", bucket.tokens)
			}
			r.mu.Unlock()
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

func (r *RateLimiter) Add(ctx context.Context, info *rateLimiter.Info) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.rateLimiterRepo.Create(ctxWithTimeout, info)
	if err != nil {
		return err
	}

	cfg, err := config.NewRateLimiterCustomConfig(info)
	if err != nil {
		slog.Error("failed to create RateLimiterCustomConfig",
			"error", err.Error(),
		)
		return err
	}

	r.SetClientConfig(info.ID, cfg)

	return nil
}

func (r *RateLimiter) Get(ctx context.Context, id string) (*rateLimiter.FullInfo, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	info, err := r.rateLimiterRepo.Get(ctxWithTimeout, id)
	if err != nil {
		return nil, err
	}
	fullInfo := r.buckets[info.ID]

	return &rateLimiter.FullInfo{
		ID:         info.ID,
		Capacity:   info.Capacity,
		TokensLeft: fullInfo.tokens,
		RefillRate: info.RefillRate,
		LastRefill: fullInfo.lastRefill,
		Interval:   info.Interval,
	}, nil
}

func (r *RateLimiter) Delete(ctx context.Context, id string) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.rateLimiterRepo.Delete(ctxWithTimeout, id)
	if err != nil {
		return err
	}

	r.mu.Lock()
	delete(r.customConfig, id)
	r.buckets[id] = NewTokenBucket(r.config)
	r.mu.Unlock()

	return nil
}

func (r *RateLimiter) Update(ctx context.Context, info *rateLimiter.Info) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.rateLimiterRepo.Update(ctxWithTimeout, info)
	if err != nil {
		return err
	}

	cfg, err := config.NewRateLimiterCustomConfig(info)
	if err != nil {
		slog.Error("failed to create RateLimiterCustomConfig",
			"error", err.Error(),
		)
		return err
	}
	r.SetClientConfig(info.ID, cfg)

	return nil
}

func (r *RateLimiter) CheckAll(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	clients, err := r.rateLimiterRepo.List(ctxWithTimeout)
	if err != nil {
		return err
	}

	for _, client := range clients {
		cfg, err := config.NewRateLimiterCustomConfig(client)
		if err != nil {
			slog.Error("failed to create RateLimiterCustomConfig",
				"error", err.Error(),
			)
			return err
		}
		r.SetClientConfig(client.ID, cfg)
	}

	return nil
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

func (r *RateLimiter) Clients() map[string]*TokenBucketModel.TokenBucket {
	clients := make(map[string]*TokenBucketModel.TokenBucket)
	r.mu.RLock()
	for id, bucket := range r.buckets {
		clients[id] = &TokenBucketModel.TokenBucket{
			Config:     bucket.config,
			Tokens:     bucket.tokens,
			LastRefill: bucket.lastRefill,
		}
	}
	r.mu.RUnlock()
	return clients
}
