package redis

import (
	"context"
	"strings"
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
	var builder strings.Builder
	builder.Grow(len(r.prefix) + 1 + len(key))
	builder.WriteString(r.prefix)
	builder.WriteString(":")
	builder.WriteString(key)
	return builder.String()
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

// SCAN
func (r *RedisCache) ScanAllData(ctx context.Context, count int64) ([]any, error) {
	var cursor uint64
	var allKeys []string
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, r.setPrefixKey("*"), count).Result()
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}
	return r.client.MGet(ctx, allKeys...).Result()
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

// Hash 相關操作
func (r *RedisCache) HSet(ctx context.Context, key string, field string, value any) error {
	return r.client.HSet(ctx, r.setPrefixKey(key), field, value).Err()
}

func (r *RedisCache) HGet(ctx context.Context, key string, field string) (any, error) {
	return r.client.HGet(ctx, r.setPrefixKey(key), field).Result()
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, r.setPrefixKey(key), fields...).Err()
}

func (r *RedisCache) HExists(ctx context.Context, key string, field string) (bool, error) {
	return r.client.HExists(ctx, r.setPrefixKey(key), field).Result()
}

func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	return r.client.HKeys(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HVals(ctx context.Context, key string) ([]string, error) {
	return r.client.HVals(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HLen(ctx context.Context, key string) (int64, error) {
	return r.client.HLen(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HMSet(ctx context.Context, key string, fields map[string]any) error {
	return r.client.HMSet(ctx, r.setPrefixKey(key), fields).Err()
}

func (r *RedisCache) HMGet(ctx context.Context, key string, fields ...string) ([]any, error) {
	return r.client.HMGet(ctx, r.setPrefixKey(key), fields...).Result()
}

func (r *RedisCache) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	return r.client.HIncrBy(ctx, r.setPrefixKey(key), field, increment).Result()
}

func (r *RedisCache) HIncrByFloat(ctx context.Context, key string, field string, increment float64) (float64, error) {
	return r.client.HIncrByFloat(ctx, r.setPrefixKey(key), field, increment).Result()
}

// 批量 Hash 操作
func (r *RedisCache) HMGetAll(ctx context.Context, keys ...string) (map[string]map[string]string, error) {
	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.MapStringStringCmd, len(keys))

	// 將所有命令加入 pipeline
	for _, key := range keys {
		cmds[key] = pipe.HGetAll(ctx, r.setPrefixKey(key))
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 收集結果
	result := make(map[string]map[string]string, len(keys))
	for key, cmd := range cmds {
		fields, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		result[key] = fields
	}

	return result, nil
}

func (r *RedisCache) HMSetMulti(ctx context.Context, items map[string]map[string]any) error {
	pipe := r.client.Pipeline()

	// 將所有命令加入 pipeline
	for key, fields := range items {
		pipe.HMSet(ctx, r.setPrefixKey(key), fields)
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) HDelMulti(ctx context.Context, items map[string][]string) error {
	pipe := r.client.Pipeline()

	// 將所有命令加入 pipeline
	for key, fields := range items {
		pipe.HDel(ctx, r.setPrefixKey(key), fields...)
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) HExistsMulti(ctx context.Context, items map[string][]string) (map[string]map[string]bool, error) {
	pipe := r.client.Pipeline()
	cmds := make(map[string]map[string]*redis.BoolCmd)

	// 將所有命令加入 pipeline
	for key, fields := range items {
		cmds[key] = make(map[string]*redis.BoolCmd)
		for _, field := range fields {
			cmds[key][field] = pipe.HExists(ctx, r.setPrefixKey(key), field)
		}
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 收集結果
	result := make(map[string]map[string]bool)
	for key, fieldCmds := range cmds {
		result[key] = make(map[string]bool)
		for field, cmd := range fieldCmds {
			exists, err := cmd.Result()
			if err != nil {
				return nil, err
			}
			result[key][field] = exists
		}
	}

	return result, nil
}

func (r *RedisCache) HLenMulti(ctx context.Context, keys ...string) (map[string]int64, error) {
	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.IntCmd, len(keys))

	// 將所有命令加入 pipeline
	for _, key := range keys {
		cmds[key] = pipe.HLen(ctx, r.setPrefixKey(key))
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 收集結果
	result := make(map[string]int64, len(keys))
	for key, cmd := range cmds {
		length, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		result[key] = length
	}

	return result, nil
}
