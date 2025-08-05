package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LimiterTestSuite struct {
	suite.Suite
	config *alogrithm.LimiterConfig
}

func (s *LimiterTestSuite) SetupTest() {
	s.config = &alogrithm.LimiterConfig{
		Capacity:    5,
		RatePS:      1,
		RatePPeriod: 1,
		RefillRate:  time.Second,
	}
}

func TestLimiterSuite(t *testing.T) {
	suite.Run(t, new(LimiterTestSuite))
}

func (s *LimiterTestSuite) TestFixedWindowLimiter() {
	limiter := NewRateLimiter(s.config).UseFixedWindow()
	ctx := context.Background()
	// 測試初始容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), limiter.Allow(ctx), "應該允許第 %d 次請求", i+1)
	}

	// 超過容量應該被拒絕
	require.False(s.T(), limiter.Allow(ctx), "超過容量限制應該被拒絕")

	// 等待一個時間窗口
	time.Sleep(s.config.RefillRate)

	// 應該可以重新接受請求
	require.True(s.T(), limiter.Allow(ctx), "新的時間窗口應該允許請求")
}

func (s *LimiterTestSuite) TestTokenBucketLimiter() {
	limiter := NewRateLimiter(s.config).UseTokenBucket()
	ctx := context.Background()
	defer limiter.(*RateLimiter).ILimiter.(*alogrithm.TokenBucket).Stop()

	// 測試初始容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), limiter.Allow(ctx), "應該允許第 %d 次請求", i+1)
	}

	// 超過容量應該被拒絕
	require.False(s.T(), limiter.Allow(ctx), "超過容量限制應該被拒絕")

	// 等待補充時間
	time.Sleep(s.config.RefillRate + 100*time.Millisecond)

	// 應該補充了一個 token
	require.True(s.T(), limiter.Allow(ctx), "應該有新的 token 可用")
}

func (s *LimiterTestSuite) TestSlideWindowLimiter() {
	limiter := NewRateLimiter(s.config).UseSlideWindow()
	ctx := context.Background()
	// 測試初始容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), limiter.Allow(ctx), "應該允許第 %d 次請求", i+1)
	}

	// 超過容量應該被拒絕
	require.False(s.T(), limiter.Allow(ctx), "超過容量限制應該被拒絕")

	// 等待一個完整的時間窗口
	time.Sleep(s.config.RefillRate)

	// 應該可以重新接受請求
	require.True(s.T(), limiter.Allow(ctx), "新的時間窗口應該允許請求")
}

func (s *LimiterTestSuite) TestDynamicSwitching() {
	// 初始使用 FixedWindow
	limiter := NewRateLimiter(s.config)
	fixedWindow := limiter.UseFixedWindow()
	ctx := context.Background()
	// 使用完初始容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), fixedWindow.Allow(ctx), "FixedWindow 應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), fixedWindow.Allow(ctx), "FixedWindow 超過容量限制應該被拒絕")

	// 切換到 TokenBucket
	tokenBucket := limiter.UseTokenBucket()
	defer tokenBucket.(*RateLimiter).ILimiter.(*alogrithm.TokenBucket).Stop()

	// TokenBucket 應該有新的容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), tokenBucket.Allow(ctx), "TokenBucket 應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), tokenBucket.Allow(ctx), "TokenBucket 超過容量限制應該被拒絕")

	// 切換到 SlideWindow
	slideWindow := limiter.UseSlideWindow()
	// SlideWindow 應該有新的容量
	for i := 0; i < s.config.Capacity; i++ {
		require.True(s.T(), slideWindow.Allow(ctx), "SlideWindow 應該允許第 %d 次請求", i+1)
	}
	require.False(s.T(), slideWindow.Allow(ctx), "SlideWindow 超過容量限制應該被拒絕")
}

func (s *LimiterTestSuite) TestConcurrentSwitching() {
	limiter := NewRateLimiter(s.config)
	done := make(chan bool)
	ctx := context.Background()
	// 並發切換和使用不同的限流器
	go func() {
		for i := 0; i < 10; i++ {
			l := limiter.UseFixedWindow()
			l.Allow(ctx)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			l := limiter.UseTokenBucket()
			l.Allow(ctx)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			l := limiter.UseSlideWindow()
			l.Allow(ctx)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// 等待所有 goroutine 完成
	for i := 0; i < 3; i++ {
		<-done
	}
}

func (s *LimiterTestSuite) TestConfigSharing() {
	limiter := NewRateLimiter(s.config)
	ctx := context.Background()
	// 修改配置
	newConfig := &alogrithm.LimiterConfig{
		Capacity:   3,
		RatePS:     2,
		RefillRate: 500 * time.Millisecond,
	}
	limiter.LimiterConfig = *newConfig

	// 測試所有限流器是否使用新配置
	limiters := []ILimitManager{
		limiter.UseFixedWindow(),
		limiter.UseTokenBucket(),
		limiter.UseSlideWindow(),
	}

	for _, l := range limiters {
		// 應該只允許 3 次請求
		for i := 0; i < 3; i++ {
			require.True(s.T(), l.Allow(ctx), "應該允許第 %d 次請求", i+1)
		}
		require.False(s.T(), l.Allow(ctx), "超過新容量限制應該被拒絕")
	}
}
