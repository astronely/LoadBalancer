package rateLimiter

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"sync"
	"time"
)

// TokenBucket part of RateLimiting algorithm
type TokenBucket struct {
	Config     config.RateLimiterConfig
	Tokens     float64
	LastRefill time.Time
	mu         sync.RWMutex
}
