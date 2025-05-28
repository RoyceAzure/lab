package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(redisClient *redis.Client, prefix string) cache.Cache {
	return &RedisCache{
		client: redisClient,
		prefix: prefix,
	}
}

var _ cache.Cache = (*RedisCache)(nil)

func (r *RedisCache) setPrefixKey(key string) string {
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

func (r *RedisCache) setPrefixKeys(keys ...string) []string {
	for i, key := range keys {
		keys[i] = r.setPrefixKey(key)
	}
	return keys
}

func (r *RedisCache) Ping(ctx context.Context) (string, error) {
	return r.client.Ping(ctx).Result()
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, error) {
	return r.client.Get(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.client.Set(ctx, r.setPrefixKey(key), value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.setPrefixKey(key)).Err()
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, r.setPrefixKey(key)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]any, error) {
	return r.client.MGet(ctx, r.setPrefixKeys(keys...)...).Result()
}

func (r *RedisCache) MSet(ctx context.Context, items map[string]any) error {
	prefixMap := make(map[string]any)
	for k, v := range items {
		prefixMap[r.setPrefixKey(k)] = v
	}
	return r.client.MSet(ctx, prefixMap).Err()
}

func (r *RedisCache) MDelete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, r.setPrefixKeys(keys...)...).Err()
}

func (r *RedisCache) Clear(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, r.setPrefixKey(pattern)).Result()
}

func (r *RedisCache) Pipeline(ctx context.Context, command func(pipe redis.Pipeliner) error) ([]redis.Cmder, error) {
	return r.client.Pipelined(ctx, command)
}
