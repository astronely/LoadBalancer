package cache

import (
	"context"
	"time"
)

type RedisClient interface {
	HashSet(ctx context.Context, key string, values interface{}) error
	Set(ctx context.Context, key string, value interface{}) error
	Add(ctx context.Context, key string, value interface{}) error
	ZAdd(ctx context.Context, key string, value interface{}) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZScore(ctx context.Context, key string, value string) (float64, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	Get(ctx context.Context, key string) (interface{}, error)
	Exist(ctx context.Context, key string, value interface{}) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Ping(ctx context.Context) error
	Del(ctx context.Context, key string) error
	Close() error
	Scan(ctx context.Context) ([]string, error)
	SMembers(ctx context.Context, key string) ([]string, error)
}
