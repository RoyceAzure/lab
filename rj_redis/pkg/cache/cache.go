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

	// Pipeline
	Pipeline(ctx context.Context, command func(pipe redis.Pipeliner) error) ([]redis.Cmder, error)

	// Hash 相關操作
	HSet(ctx context.Context, key string, field string, value any) error
	HGet(ctx context.Context, key string, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key string, field string) (bool, error)
	HKeys(ctx context.Context, key string) ([]string, error)
	HVals(ctx context.Context, key string) ([]string, error)
	HLen(ctx context.Context, key string) (int64, error)
	HMSet(ctx context.Context, key string, value any) error
	HMGet(ctx context.Context, key string, fields ...string) ([]any, error)
	HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error)
	HIncrByFloat(ctx context.Context, key string, field string, increment float64) (float64, error)

	// 批量 Hash 操作
	HMSetMulti(ctx context.Context, items map[string]any) error
	HMGetAll(ctx context.Context, keys ...string) (map[string]map[string]string, error)
}
