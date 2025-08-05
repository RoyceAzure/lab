package middleware

import (
	"net/http"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/RoyceAzure/lab/rate_limit/limiter"
)

type RateLimitType string

var (
	FixedWindow = RateLimitType("fixed_window")
	TokenBucket = RateLimitType("token_bucket")
	SlideWindow = RateLimitType("slide_window")
	RedisBucket = RateLimitType("redis_bucket")
)

// NewRateLimitMiddleware 創建非Scope限流中間件
func NewRateLimitMiddleware(rateLimitType RateLimitType, config *alogrithm.LimiterConfig, opts ...limiter.RateLimiterOption) func(http.Handler) http.Handler {
	var rateLimiter limiter.ILimitManager
	switch rateLimitType {
	case FixedWindow:
		rateLimiter = limiter.NewRateLimiter(config).UseFixedWindow()
	case TokenBucket:
		rateLimiter = limiter.NewRateLimiter(config).UseTokenBucket()
	case SlideWindow:
		rateLimiter = limiter.NewRateLimiter(config).UseSlideWindow()
	case RedisBucket:
		rateLimiter = limiter.NewRateLimiter(config, opts...).UseRedisTokenBucket()
	default:
		panic("invalid rate limit type")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed := rateLimiter.Allow(r.Context())
			if !allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
