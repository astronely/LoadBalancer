package repository

import (
	"context"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
)

type RateLimiterRepository interface {
	Create(ctx context.Context, info *rateLimiter.Info) error
	Get(ctx context.Context, id string) (*rateLimiter.Info, error)
	Update(ctx context.Context, info *rateLimiter.Info) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*rateLimiter.Info, error)
}
