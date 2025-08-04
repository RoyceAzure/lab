package limiter

import (
	"context"
	"fmt"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/redis/go-redis/v9"
)

type ILimiter interface {
	Allow() bool
}

type ILimitManager interface {
	ILimiter
	UseFixedWindow() ILimitManager
	UseTokenBucket() ILimitManager
	UseSlideWindow() ILimitManager
}

type RateLimiter struct {
	ILimiter
	LimiterConfig alogrithm.LimiterConfig
}

func NewRateLimiter(config *alogrithm.LimiterConfig) *RateLimiter {
	return &RateLimiter{
		LimiterConfig: *config,
	}
}

func (r *RateLimiter) newRateLimiter(limiter ILimiter) *RateLimiter {
	return &RateLimiter{
		ILimiter:      limiter,
		LimiterConfig: r.LimiterConfig,
	}
}

func (r *RateLimiter) UseFixedWindow() ILimitManager {
	return r.newRateLimiter(alogrithm.NewFixWindow(&r.LimiterConfig))
}

func (r *RateLimiter) UseTokenBucket() ILimitManager {
	return r.newRateLimiter(alogrithm.NewTokenBucket(&r.LimiterConfig))
}

func (r *RateLimiter) UseSlideWindow() ILimitManager {
	return r.newRateLimiter(alogrithm.NewSlideWindow(&r.LimiterConfig))
}

type RedisConfig struct {
	host     string
	port     int
	password string
}

type IRedisRateLimiter interface {
	Allow(ctx context.Context, key string) bool
}

type IRedisLimiterManager interface {
	IRedisRateLimiter
	UseBucketToken() IRedisLimiterManager
}

type RedisRateLimiter struct {
	IRedisRateLimiter
	LimiterConfig alogrithm.LimiterConfig
	config        RedisConfig
	client        *redis.Client
}

func NewRedisRateLimiter(config *alogrithm.LimiterConfig, redisConfig RedisConfig) *RedisRateLimiter {
	return &RedisRateLimiter{
		LimiterConfig: *config,
		config:        redisConfig,
		client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", redisConfig.host, redisConfig.port),
			Password: redisConfig.password,
		}),
	}
}

func (r *RedisRateLimiter) newRedisRateLimiter(limiter IRedisRateLimiter) *RedisRateLimiter {
	return &RedisRateLimiter{
		IRedisRateLimiter: limiter,
		LimiterConfig:     r.LimiterConfig,
	}
}

func (r *RedisRateLimiter) UseBucketToken() IRedisLimiterManager {
	return r.newRedisRateLimiter(alogrithm.NewRsBucketToken(r.client, &r.LimiterConfig))
}
