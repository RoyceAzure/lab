package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	// 基本操作
	Ping(ctx context.Context) (string, error)
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 批量操作
	MGet(ctx context.Context, keys ...string) ([]any, error)
	MSet(ctx context.Context, items map[string]any) error
	MDelete(ctx context.Context, keys ...string) error

	// 辅助功能
	Clear(ctx context.Context) error
	Keys(ctx context.Context, pattern string) ([]string, error)

	//Pipeline
	Pipeline(ctx context.Context, command func(pipe redis.Pipeliner) error) ([]redis.Cmder, error)
}
