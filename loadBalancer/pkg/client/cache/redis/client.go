package redis

import (
	"context"
	"errors"
	"github.com/astronely/financial-helper_microservices/apiGateway/pkg/logger"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/client/cache"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisClient struct {
	rdb    *redis.Client
	config config.RedisConfig
}

func NewClient(rdb *redis.Client, config config.RedisConfig) cache.RedisClient {
	return &redisClient{
		rdb:    rdb,
		config: config,
	}
}

func (r *redisClient) HashSet(ctx context.Context, key string, values interface{}) error {
	if err := r.rdb.HSet(ctx, key, values).Err(); err != nil {
		logger.Error("redis HSet failed",
			"error", err)
		return err
	}
	return nil
}

func (r *redisClient) Add(ctx context.Context, key string, value interface{}) error {
	if err := r.rdb.SAdd(ctx, key, value).Err(); err != nil {
		logger.Error("redis add failed",
			"error", err)
		return err
	}
	return nil
}

func (r *redisClient) ZAdd(ctx context.Context, key string, value interface{}) error {
	if err := r.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: value,
	}).Err(); err != nil {
		logger.Error("redis ZAdd failed")
		return err
	}
	return nil
}

func (r *redisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	result, err := r.rdb.ZRange(ctx, key, start, stop).Result()
	//logger.Debug("redis ZRange result",
	//	"start", start,
	//	"stop", stop,
	//	"result", result,
	//	"key", key)
	if err != nil {
		logger.Error("redis ZRange failed",
			"error", err)
		return nil, err
	}
	return result, nil
}

func (r *redisClient) Set(ctx context.Context, key string, value interface{}) error {
	if err := r.rdb.Set(ctx, key, value, 0).Err(); err != nil {
		logger.Error("redis set failed",
			"error", err)
		return err
	}
	return nil
}

func (r *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		logger.Error("redis HGetAll failed",
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

func (r *redisClient) Get(ctx context.Context, key string) (interface{}, error) {
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		logger.Error("redis get failed",
			"error", err,
		)
		return nil, err
	}
	return result, nil
}

func (r *redisClient) Exist(ctx context.Context, key string, value interface{}) (bool, error) {
	result, err := r.rdb.SIsMember(ctx, key, value).Result()
	if err != nil {
		logger.Error("redis SIsMember failed",
			"error", err,
		)
		return false, err
	}
	return result, nil
}

func (r *redisClient) ZScore(ctx context.Context, key string, value string) (float64, error) {
	score, err := r.rdb.ZScore(ctx, key, value).Result()
	logger.Debug("redis ZScore",
		"key", key,
		"value", value)
	if errors.Is(err, redis.Nil) {
		return -1, nil
	}
	if err != nil {
		return -1, err
	}

	return score, nil
}

func (r *redisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := r.rdb.Expire(ctx, key, expiration).Err(); err != nil {
		logger.Error("redis Expire failed",
			"error", err,
		)
		return err
	}
	return nil
}

func (r *redisClient) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}

func (r *redisClient) Close() error {
	return r.rdb.Close()
}

func (r *redisClient) Del(ctx context.Context, key string) error {
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis Del failed",
			"error", err,
		)
		return err
	}
	return nil
}

func (r *redisClient) Scan(ctx context.Context) ([]string, error) {
	var (
		cursor uint64
		keys   []string
		err    error
	)
	for {
		var batch []string
		batch, cursor, err = r.rdb.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func (r *redisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	members, err := r.rdb.SMembers(ctx, key).Result()
	if err != nil {
		logger.Error("redis SMembers failed",
			"error", err,
		)
		return nil, err
	}

	return members, nil
}
