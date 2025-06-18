package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	Client *redis.Client
	prefix string
}

func NewRedisCache(redisClient *redis.Client, prefix string) cache.Cache {
	return &RedisCache{
		Client: redisClient,
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
	return r.Client.Ping(ctx).Result()
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, error) {
	return r.Client.Get(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.Client.Set(ctx, r.setPrefixKey(key), value, ttl).Err()
}

// SCAN
func (r *RedisCache) ScanAllData(ctx context.Context, count int64) ([]any, error) {
	var cursor uint64
	var allKeys []string
	for {
		keys, nextCursor, err := r.Client.Scan(ctx, cursor, r.setPrefixKey("*"), count).Result()
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}
	return r.Client.MGet(ctx, allKeys...).Result()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, r.setPrefixKey(key)).Err()
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.Client.Exists(ctx, r.setPrefixKey(key)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]any, error) {
	return r.Client.MGet(ctx, r.setPrefixKeys(keys...)...).Result()
}

func (r *RedisCache) MSet(ctx context.Context, items map[string]any) error {
	prefixMap := make(map[string]any)
	for k, v := range items {
		prefixMap[r.setPrefixKey(k)] = v
	}
	return r.Client.MSet(ctx, prefixMap).Err()
}

func (r *RedisCache) MDelete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, r.setPrefixKeys(keys...)...).Err()
}

func (r *RedisCache) Clear(ctx context.Context) error {
	return r.Client.FlushDB(ctx).Err()
}

func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.Client.Keys(ctx, r.setPrefixKey(pattern)).Result()
}

func (r *RedisCache) Pipeline(ctx context.Context, command func(pipe redis.Pipeliner) error) ([]redis.Cmder, error) {
	return r.Client.Pipelined(ctx, command)
}

// Hash 相關操作
func (r *RedisCache) HSet(ctx context.Context, key string, field string, value any) error {
	return r.Client.HSet(ctx, r.setPrefixKey(key), field, value).Err()
}

func (r *RedisCache) HGet(ctx context.Context, key string, field string) (string, error) {
	return r.Client.HGet(ctx, r.setPrefixKey(key), field).Result()
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]any, error) {
	// 先獲取原始的 map[string]string
	strMap, err := r.Client.HGetAll(ctx, r.setPrefixKey(key)).Result()
	if err != nil {
		return nil, err
	}

	// 轉換為 map[string]any
	result := make(map[string]any, len(strMap))
	for k, v := range strMap {
		// 嘗試解析為數字
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			result[k] = num
			continue
		}
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			result[k] = num
			continue
		}
		// 嘗試解析為布爾值
		if b, err := strconv.ParseBool(v); err == nil {
			result[k] = b
			continue
		}
		// 如果是 JSON 字符串，嘗試解析
		if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
			var jsonValue any
			if err := json.Unmarshal([]byte(v), &jsonValue); err == nil {
				result[k] = jsonValue
				continue
			}
		}
		// 默認為字符串
		result[k] = v
	}

	return result, nil
}

func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return r.Client.HDel(ctx, r.setPrefixKey(key), fields...).Err()
}

func (r *RedisCache) HExists(ctx context.Context, key string, field string) (bool, error) {
	return r.Client.HExists(ctx, r.setPrefixKey(key), field).Result()
}

func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	return r.Client.HKeys(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HVals(ctx context.Context, key string) ([]string, error) {
	return r.Client.HVals(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HLen(ctx context.Context, key string) (int64, error) {
	return r.Client.HLen(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) HMSet(ctx context.Context, key string, value any) error {
	// 將 value 轉換為 map
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(jsonData, &fields); err != nil {
		return err
	}
	return r.Client.HMSet(ctx, r.setPrefixKey(key), fields).Err()
}

func (r *RedisCache) HMGet(ctx context.Context, key string, fields ...string) ([]any, error) {
	return r.Client.HMGet(ctx, r.setPrefixKey(key), fields...).Result()
}

func (r *RedisCache) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	return r.Client.HIncrBy(ctx, r.setPrefixKey(key), field, increment).Result()
}

func (r *RedisCache) HIncrByFloat(ctx context.Context, key string, field string, increment float64) (float64, error) {
	return r.Client.HIncrByFloat(ctx, r.setPrefixKey(key), field, increment).Result()
}

// 批量 Hash 操作
func (r *RedisCache) HMGetAll(ctx context.Context, keys ...string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	// 使用 Pipeline 批量獲取
	pipe := r.Client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.HGetAll(ctx, r.setPrefixKey(key))
	}

	// 執行 Pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 處理結果
	for i, cmd := range cmds {
		strMap, err := cmd.Result()
		if err != nil {
			return nil, err
		}

		// 轉換為 map[string]any
		anyMap := make(map[string]any, len(strMap))
		for k, v := range strMap {
			// 嘗試解析為數字
			if num, err := strconv.ParseInt(v, 10, 64); err == nil {
				anyMap[k] = num
				continue
			}
			if num, err := strconv.ParseFloat(v, 64); err == nil {
				anyMap[k] = num
				continue
			}
			// 嘗試解析為布爾值
			if b, err := strconv.ParseBool(v); err == nil {
				anyMap[k] = b
				continue
			}
			// 如果是 JSON 字符串，嘗試解析
			if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
				var jsonValue any
				if err := json.Unmarshal([]byte(v), &jsonValue); err == nil {
					anyMap[k] = jsonValue
					continue
				}
			}
			// 默認為字符串
			anyMap[k] = v
		}

		result[keys[i]] = anyMap
	}

	return result, nil
}

func (r *RedisCache) HMSetMulti(ctx context.Context, items map[string]any) error {
	pipe := r.Client.Pipeline()

	for key, value := range items {
		// 將 value 轉換為 map
		jsonData, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var fields map[string]interface{}
		if err := json.Unmarshal(jsonData, &fields); err != nil {
			return err
		}

		// 處理巢狀結構
		flattenedFields := make(map[string]interface{})
		flattenMap("", fields, flattenedFields)

		// 將 interface{} 轉換為 string
		stringFields := make(map[string]string)
		for k, v := range flattenedFields {
			switch val := v.(type) {
			case string:
				stringFields[k] = val
			case []interface{}:
				// 處理切片，將其轉換為逗號分隔的字符串
				strSlice := make([]string, len(val))
				for i, item := range val {
					strSlice[i] = fmt.Sprint(item)
				}
				stringFields[k] = strings.Join(strSlice, ",")
			default:
				stringFields[k] = fmt.Sprint(v)
			}
		}

		pipe.HMSet(ctx, r.setPrefixKey(key), stringFields)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// flattenMap 將巢狀 map 扁平化
func flattenMap(prefix string, m map[string]interface{}, result map[string]interface{}) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			flattenMap(key, val, result)
		default:
			result[key] = v
		}
	}
}

func (r *RedisCache) HDelMulti(ctx context.Context, items map[string][]string) error {
	pipe := r.Client.Pipeline()

	// 將所有命令加入 pipeline
	for key, fields := range items {
		pipe.HDel(ctx, r.setPrefixKey(key), fields...)
	}

	// 執行 pipeline
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) HExistsMulti(ctx context.Context, items map[string][]string) (map[string]map[string]bool, error) {
	pipe := r.Client.Pipeline()
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
	pipe := r.Client.Pipeline()
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

// Set 操作
func (r *RedisCache) SAdd(ctx context.Context, key string, members ...any) error {
	return r.Client.SAdd(ctx, r.setPrefixKey(key), members...).Err()
}

func (r *RedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.Client.SMembers(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) SRem(ctx context.Context, key string, members ...any) error {
	return r.Client.SRem(ctx, r.setPrefixKey(key), members...).Err()
}

func (r *RedisCache) SCard(ctx context.Context, key string) (int64, error) {
	return r.Client.SCard(ctx, r.setPrefixKey(key)).Result()
}

func (r *RedisCache) SIsMember(ctx context.Context, key string, member any) (bool, error) {
	return r.Client.SIsMember(ctx, r.setPrefixKey(key), member).Result()
}
