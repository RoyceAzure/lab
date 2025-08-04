package alogrithm

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RsBucketToken struct {
	LimiterConfig
	client RedisClient
}

// RedisClient 介面定義
type RedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

func NewRsBucketToken(client RedisClient, config *LimiterConfig) *RsBucketToken {
	rb := &RsBucketToken{
		client: client,
	}

	if config != nil {
		rb.LimiterConfig = *config
	} else {
		rb.LimiterConfig = GetDefaultLimiterConfig()
	}

	return rb
}

func (r *RsBucketToken) Allow(ctx context.Context, key string) bool {
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local initTokens = tonumber(ARGV[4])

		-- 取得或初始化 bucket 狀態
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local currentTokens = tonumber(bucket[1])
		local lastRefill = tonumber(bucket[2])

		-- 如果 key 不存在，進行初始化
		if currentTokens == nil then
			currentTokens = initTokens
			lastRefill = now
			redis.call('HMSET', key, 'tokens', currentTokens, 'last_refill', lastRefill)
			redis.call('EXPIRE', key, 60) -- 設置 60 秒過期，避免佔用太多記憶體
		end

		-- 計算需要補充的 tokens
		local elapsedSeconds = (now - lastRefill) / 1000000000 -- 轉換為秒
		local tokensToAdd = elapsedSeconds * rate
		
		-- 更新 tokens
		currentTokens = math.min(capacity, currentTokens + tokensToAdd)
		
		-- 如果沒有足夠的 tokens，返回 false
		if currentTokens < 1 then
			-- 更新最後補充時間，避免頻繁計算
			redis.call('HMSET', key, 'tokens', currentTokens, 'last_refill', now)
			return 0
		end
		
		-- 扣減一個 token 並更新狀態
		currentTokens = currentTokens - 1
		redis.call('HMSET', key, 'tokens', currentTokens, 'last_refill', now)
		
		return 1
	`

	// 執行 Lua 腳本
	result, err := r.client.Eval(
		ctx,
		luaScript,
		[]string{key},
		r.Capacity,
		r.Rate,
		time.Now().UnixNano(),
		r.Capacity, // 初始 tokens 設為最大容量
	).Int64()

	if err != nil {
		return false
	}

	return result == 1
}
