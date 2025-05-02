package rateLimiter

import (
	"context"
	"encoding/json"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"github.com/astronely/loadBalancer/loadBalancer/internal/repository"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/client/cache"
	"log/slog"
)

type repo struct {
	client cache.RedisClient
}

func NewRepository(client cache.RedisClient) repository.RateLimiterRepository {
	return &repo{client: client}
}

func (r *repo) Create(ctx context.Context, info *rateLimiter.Info) error {
	bytesData, err := json.Marshal(info)
	if err != nil {
		slog.Error("Error marshalling rate limiter info",
			"error", err.Error(),
		)
		return err
	}

	err = r.client.Set(ctx, info.ID, bytesData)
	if err != nil {
		slog.Error("Error creating rate limiter info",
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *repo) Get(ctx context.Context, id string) (*rateLimiter.Info, error) {
	bytesInfo, err := r.client.Get(ctx, id)
	if err != nil {
		slog.Error("Error getting rate limiter info",
			"error", err.Error(),
		)
		return nil, err
	}

	var info *rateLimiter.Info
	err = json.Unmarshal([]byte(bytesInfo.(string)), &info)
	if err != nil {
		slog.Error("Error unmarshalling rate limiter info",
			"error", err.Error(),
		)
		return nil, err
	}
	return info, nil
}

func (r *repo) Update(ctx context.Context, info *rateLimiter.Info) error {
	bytesInfo, err := json.Marshal(info)
	if err != nil {
		slog.Error("Error marshalling rate limiter info",
			"error", err.Error(),
		)
		return err
	}

	err = r.client.Set(ctx, info.ID, bytesInfo)
	if err != nil {
		slog.Error("Error updating rate limiter info",
			"error", err.Error(),
		)
		return err
	}
	return nil
}

func (r *repo) Delete(ctx context.Context, id string) error {
	err := r.client.Del(ctx, id)
	if err != nil {
		slog.Error("Error deleting rate limiter info",
			"error", err.Error(),
		)
		return err
	}
	return nil
}

func (r *repo) List(ctx context.Context) ([]*rateLimiter.Info, error) {
	var list []*rateLimiter.Info
	var info *rateLimiter.Info

	keys, err := r.client.Scan(ctx)
	if err != nil {
		slog.Error("Error listing rate limiter info",
			"error", err.Error(),
		)
		return nil, err
	}

	for _, key := range keys {
		slog.Debug("Key",
			"key", key,
		)
		bytesInfo, err := r.client.Get(ctx, key)
		if err != nil {
			slog.Error("Error listing rate limiter info",
				"error", err.Error(),
			)
			return nil, err
		}
		err = json.Unmarshal([]byte(bytesInfo.(string)), &info)
		if err != nil {
			slog.Error("Error unmarshalling rate limiter info",
				"error", err.Error(),
			)
			return nil, err
		}
		list = append(list, info)
	}

	return list, nil
}
