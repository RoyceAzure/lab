package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/redis/go-redis/v9"
)

type ILimiter interface {
	Allow(ctx context.Context) bool
	SetCapacity(capacity int)
	SetRefillRate(refillRate time.Duration)
	SetRatePS(rate int)
	SetRatePPeriod(rate int)
	SetKey(key string)
}

type ILimitManager interface {
	ILimiter
	UseFixedWindow() ILimitManager
	UseTokenBucket() ILimitManager
	UseSlideWindow() ILimitManager
	UseRedisTokenBucket() ILimitManager
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

type RateLimiter struct {
	ILimiter
	LimiterConfig alogrithm.LimiterConfig
	config        *RedisConfig
	client        *redis.Client
}

type RateLimiterOption func(*RateLimiter)

func WithRedisConfig(config *RedisConfig) RateLimiterOption {
	return func(r *RateLimiter) {
		r.config = config
		r.client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
			Password: config.Password,
		})
	}
}

func NewRateLimiter(config *alogrithm.LimiterConfig, opts ...RateLimiterOption) *RateLimiter {
	r := &RateLimiter{
		LimiterConfig: *config,
	}

	for _, opt := range opts {
		opt(r)
	}
	return r
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

func (r *RateLimiter) UseRedisTokenBucket() ILimitManager {
	return r.newRateLimiter(alogrithm.NewRsBucketToken(r.client, &r.LimiterConfig))
}
