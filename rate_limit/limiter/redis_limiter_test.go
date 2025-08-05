package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RedisLimiterTestSuite struct {
	suite.Suite
	config      *alogrithm.LimiterConfig
	redisConfig RedisConfig
	ctx         context.Context
	limiter     *RateLimiter
}

func (s *RedisLimiterTestSuite) SetupSuite() {
	s.config = &alogrithm.LimiterConfig{
		Capacity:    5,
		RatePS:      1,
		RatePPeriod: 1,
		RefillRate:  time.Second,
	}
	s.redisConfig = RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "password",
	}
	s.ctx = context.Background()
}

func (s *RedisLimiterTestSuite) SetupTest() {
	s.limiter = NewRateLimiter(s.config, WithRedisConfig(&s.redisConfig))
	// 清空資料庫
	s.limiter.client.FlushDB(s.ctx)
}

func (s *RedisLimiterTestSuite) TearDownTest() {
	s.limiter.client.Close()
}

func TestRedisLimiterSuite(t *testing.T) {
	suite.Run(t, new(RedisLimiterTestSuite))
}

func (s *RedisLimiterTestSuite) TestBasicRateLimit() {
	limiter := s.limiter.UseRedisTokenBucket()
	key := "test-basic"

	// 測試初始容量
	limiter.SetKey(key)
	for i := 0; i < s.config.Capacity; i++ {
		allowed := limiter.Allow(s.ctx)
		require.True(s.T(), allowed, "應該允許第 %d 次請求", i+1)
	}

	// 超過容量應該被拒絕
	allowed := limiter.Allow(s.ctx)
	require.False(s.T(), allowed, "超過容量限制應該被拒絕")

	// 等待補充時間
	time.Sleep(s.config.RefillRate + 100*time.Millisecond)

	// 應該補充了一個 token
	allowed = limiter.Allow(s.ctx)
	require.True(s.T(), allowed, "等待後應該有新的 token 可用")
}

func (s *RedisLimiterTestSuite) TestMultipleKeys() {
	limiter := s.limiter.UseRedisTokenBucket()
	key1 := "test-key1"
	key2 := "test-key2"

	// 測試 key1
	limiter.SetKey(key1)
	for i := 0; i < s.config.Capacity; i++ {
		allowed := limiter.Allow(s.ctx)
		require.True(s.T(), allowed, "key1 應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), limiter.Allow(s.ctx), "key1 超過容量限制應該被拒絕")

	// key2 應該不受 key1 影響
	limiter.SetKey(key2)
	for i := 0; i < s.config.Capacity; i++ {
		allowed := limiter.Allow(s.ctx)
		require.True(s.T(), allowed, "key2 應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), limiter.Allow(s.ctx), "key2 超過容量限制應該被拒絕")
}

func (s *RedisLimiterTestSuite) TestConcurrentAccess() {
	limiter := s.limiter.UseRedisTokenBucket()
	limiter.SetKey("test-concurrent")
	done := make(chan bool)
	results := make(chan bool, 100)

	// 並發訪問同一個 key
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				results <- limiter.Allow(s.ctx)
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 5; i++ {
		<-done
	}
	close(results)

	// 統計允許的請求數量
	allowedCount := 0
	for allowed := range results {
		if allowed {
			allowedCount++
		}
	}

	// 檢查是否符合限流規則
	require.LessOrEqual(s.T(), allowedCount, s.config.Capacity,
		"並發請求中允許的總數不應超過容量")
}

func (s *RedisLimiterTestSuite) TestConfigChange() {
	limiter := s.limiter.UseRedisTokenBucket()
	key := "test-config"

	// 使用初始配置
	limiter.SetKey(key)
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), limiter.Allow(s.ctx), "應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), limiter.Allow(s.ctx), "超過容量限制應該被拒絕")

	// 修改配置
	newConfig := &alogrithm.LimiterConfig{
		Capacity:   3,
		RatePS:     2,
		RefillRate: 500 * time.Millisecond,
	}
	s.limiter = NewRateLimiter(newConfig, WithRedisConfig(&s.redisConfig))
	limiter = s.limiter.UseRedisTokenBucket()
	limiter.SetKey("test-config-new")

	// 使用新配置
	for i := 0; i < newConfig.Capacity; i++ {
		require.True(s.T(), limiter.Allow(s.ctx), "新配置應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), limiter.Allow(s.ctx), "超過新容量限制應該被拒絕")
}

func (s *RedisLimiterTestSuite) TestRedisConnectionError() {
	// 使用錯誤的 Redis 配置
	badConfig := RedisConfig{
		Host:     "nonexistent",
		Port:     6379,
		Password: "wrong",
	}

	// 這應該會建立失敗的連接
	limiter := NewRateLimiter(s.config, WithRedisConfig(&badConfig))
	redisLimiter := limiter.UseRedisTokenBucket()
	redisLimiter.SetKey("test-error")
	// 應該返回 false 而不是 panic
	require.False(s.T(), redisLimiter.Allow(s.ctx),
		"Redis 連接錯誤時應該拒絕請求")
}
