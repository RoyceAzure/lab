package alogrithm

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RsTokenBucketTestSuite struct {
	suite.Suite
	client *redis.Client
	ctx    context.Context
}

func (s *RsTokenBucketTestSuite) SetupSuite() {
	s.client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password",
		DB:       0,
	})
	s.ctx = context.Background()

	// 測試連線
	err := s.client.Ping(s.ctx).Err()
	require.NoError(s.T(), err, "Redis連線失敗")
}

func (s *RsTokenBucketTestSuite) TearDownSuite() {
	s.client.Close()
}

func (s *RsTokenBucketTestSuite) SetupTest() {
	// 每個測試前清空資料庫
	s.client.FlushDB(s.ctx)
}

func TestRsTokenBucketSuite(t *testing.T) {
	suite.Run(t, new(RsTokenBucketTestSuite))
}

func (s *RsTokenBucketTestSuite) TestBasicRateLimit() {
	config := LimiterConfig{
		Capacity:   5,
		Rate:       2,
		RefillRate: 100 * time.Millisecond,
	}
	limiter := NewRsBucketToken(s.client, &config)

	// 測試初始容量
	for i := 0; i < 5; i++ {
		allowed := limiter.Allow(s.ctx, "test-basic")
		require.True(s.T(), allowed, "應該允許第 %d 次請求", i+1)
	}

	// 第6次應該被拒絕
	allowed := limiter.Allow(s.ctx, "test-basic")
	require.False(s.T(), allowed, "超過容量限制應該被拒絕")
}

func (s *RsTokenBucketTestSuite) TestTokenRefill() {
	config := LimiterConfig{
		Capacity:   2,
		Rate:       1,
		RefillRate: time.Second,
	}
	limiter := NewRsBucketToken(s.client, &config)
	key := "test-refill"

	// 消耗所有token
	require.True(s.T(), limiter.Allow(s.ctx, key), "第一次請求應該被允許")
	require.True(s.T(), limiter.Allow(s.ctx, key), "第二次請求應該被允許")
	require.False(s.T(), limiter.Allow(s.ctx, key), "第三次請求應該被拒絕")

	// 等待token補充
	time.Sleep(1100 * time.Millisecond)

	// 應該補充了一個token
	require.True(s.T(), limiter.Allow(s.ctx, key), "等待後應該有一個新的token")
	require.False(s.T(), limiter.Allow(s.ctx, key), "不應該有第二個token")
}

func (s *RsTokenBucketTestSuite) TestMultipleKeys() {
	config := LimiterConfig{
		Capacity:   2,
		Rate:       1,
		RefillRate: time.Second,
	}
	limiter := NewRsBucketToken(s.client, &config)

	// 測試不同的key是否互相獨立
	key1 := "test-key1"
	key2 := "test-key2"

	// key1 消耗token
	require.True(s.T(), limiter.Allow(s.ctx, key1), "key1第一次請求應該被允許")
	require.True(s.T(), limiter.Allow(s.ctx, key1), "key1第二次請求應該被允許")
	require.False(s.T(), limiter.Allow(s.ctx, key1), "key1第三次請求應該被拒絕")

	// key2 應該有獨立的token
	require.True(s.T(), limiter.Allow(s.ctx, key2), "key2第一次請求應該被允許")
	require.True(s.T(), limiter.Allow(s.ctx, key2), "key2第二次請求應該被允許")
	require.False(s.T(), limiter.Allow(s.ctx, key2), "key2第三次請求應該被拒絕")
}

func (s *RsTokenBucketTestSuite) TestDefaultConfig() {
	l := NewRsBucketToken(s.client, nil)
	key := "test-default"

	defaultConfig := GetDefaultLimiterConfig()

	// 測試預設容量
	for i := 0; i < defaultConfig.Capacity; i++ {
		allowed := l.Allow(s.ctx, key)
		require.True(s.T(), allowed, "使用預設配置時，應該允許第 %d 次請求", i+1)
	}

	// 超過預設容量應該被拒絕
	allowed := l.Allow(s.ctx, key)
	require.False(s.T(), allowed, "使用預設配置時，超過容量限制應該被拒絕")
}

func (s *RsTokenBucketTestSuite) TestConcurrent() {
	config := LimiterConfig{
		Capacity:   10,
		Rate:       2,
		RefillRate: 100 * time.Millisecond,
	}
	limiter := NewRsBucketToken(s.client, &config)
	key := "test-concurrent"

	// 並發測試
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 3; j++ {
				limiter.Allow(s.ctx, key)
				time.Sleep(50 * time.Millisecond)
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 5; i++ {
		<-done
	}
}
