package redis_repo

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type ITimestampRedisRepository interface {
	// GetProductTimestampKey 取得商品時間戳key
	GetProductTimestampKey(productID string, evt string) string
	/*
		使用redis lua script 來實現原子性
		1. 如果key不存在，直接設置並返回 true
		2. 如果key存在，比較新時間戳和當前時間戳，如果新時間戳大於當前時間戳，更新並返回 true
		3. 否則返回 false，不更新
	*/
	SetProductTimestamp(ctx context.Context, key string, timestamp int64) (bool, error)
}

type TimestampRedisRepo struct {
	timestampCache *redis.Client
}

func NewTimestampRepo(timestampCache *redis.Client) *TimestampRedisRepo {
	return &TimestampRedisRepo{timestampCache: timestampCache}
}

func (s *TimestampRedisRepo) GetProductTimestampKey(productID string, evt string) string {
	return fmt.Sprintf("product:%s:%s:timestamp", productID, evt)
}

/*
使用redis lua script 來實現原子性
1. 如果key不存在，直接設置並返回 true
2. 如果key存在，比較新時間戳和當前時間戳，如果新時間戳大於當前時間戳，更新並返回 true
3. 否則返回 false，不更新
*/
func (s *TimestampRedisRepo) SetProductTimestamp(ctx context.Context, key string, timestamp int64) (bool, error) {
	const compareAndSetScript = `
    local key = KEYS[1]
    local new_timestamp = tonumber(ARGV[1])
    
    -- 獲取當前儲存的時間戳
    local current = redis.call('GET', key)
    
    -- 如果 key 不存在，直接設置並返回 true
    if not current then
        redis.call('SET', key, new_timestamp)
        return 1
    end
    
    -- 轉換為數字進行比較
    current = tonumber(current)
    
    -- 如果新時間戳大於當前時間戳，更新並返回 true
    if new_timestamp > current then
        redis.call('SET', key, new_timestamp)
        return 1
    end
    
    -- 否則返回 false，不更新
    return 0
    `

	result, err := s.timestampCache.Eval(ctx, compareAndSetScript, []string{key}, timestamp).Result()
	if err != nil {
		return false, fmt.Errorf("failed to compare and set timestamp: %w", err)
	}

	// 將結果轉換為布林值
	success, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type: %T", result)
	}

	return success == 1, nil
}
