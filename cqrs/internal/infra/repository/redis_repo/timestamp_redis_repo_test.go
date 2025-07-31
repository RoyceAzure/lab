package redis_repo

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type TimestampRedisRepoTestSuite struct {
	suite.Suite
	redisClient *redis.Client
	repo        *TimestampRedisRepo
	ctx         context.Context
}

func TestTimestampRedisRepo(t *testing.T) {
	suite.Run(t, new(TimestampRedisRepoTestSuite))
}

func (suite *TimestampRedisRepoTestSuite) SetupSuite() {
	suite.redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	suite.repo = NewTimestampRepo(suite.redisClient)
	suite.ctx = context.Background()
}

func (suite *TimestampRedisRepoTestSuite) TearDownSuite() {
	suite.redisClient.Close()
}

func (suite *TimestampRedisRepoTestSuite) SetupTest() {
	suite.redisClient.FlushDB(suite.ctx)
}

func (suite *TimestampRedisRepoTestSuite) TestGetProductTimestampKey() {
	testCases := []struct {
		productID string
		event     string
		expected  string
	}{
		{
			productID: "test123",
			event:     "stock_updated",
			expected:  "product:test123:stock_updated:timestamp",
		},
		{
			productID: "prod456",
			event:     "reserved_changed",
			expected:  "product:prod456:reserved_changed:timestamp",
		},
	}

	for _, tc := range testCases {
		actual := suite.repo.GetProductTimestampKey(tc.productID, tc.event)
		suite.Equal(tc.expected, actual)
	}
}

func (suite *TimestampRedisRepoTestSuite) TestSetProductTimestamp() {
	productID := "test123"
	event := "stock_updated"
	key := suite.repo.GetProductTimestampKey(productID, event)

	// 測試場景 1: key 不存在時的首次設置
	timestamp1 := time.Now().UnixMilli()
	updated, err := suite.repo.SetProductTimestamp(suite.ctx, key, timestamp1)
	suite.NoError(err)
	suite.True(updated, "首次設置應該返回 true")

	// 驗證值已正確設置
	val, err := suite.redisClient.Get(suite.ctx, key).Int64()
	suite.NoError(err)
	suite.Equal(timestamp1, val)

	// 測試場景 2: 設置較小的時間戳
	smallerTimestamp := timestamp1 - 1000
	updated, err = suite.repo.SetProductTimestamp(suite.ctx, key, smallerTimestamp)
	suite.NoError(err)
	suite.False(updated, "較小的時間戳不應該更新成功")

	// 驗證值未被更新
	val, err = suite.redisClient.Get(suite.ctx, key).Int64()
	suite.NoError(err)
	suite.Equal(timestamp1, val, "值應該保持不變")

	// 測試場景 3: 設置較大的時間戳
	largerTimestamp := timestamp1 + 1000
	updated, err = suite.repo.SetProductTimestamp(suite.ctx, key, largerTimestamp)
	suite.NoError(err)
	suite.True(updated, "較大的時間戳應該更新成功")

	// 驗證值已被更新
	val, err = suite.redisClient.Get(suite.ctx, key).Int64()
	suite.NoError(err)
	suite.Equal(largerTimestamp, val)
}

func (suite *TimestampRedisRepoTestSuite) TestSetProductTimestamp_Concurrent() {
	productID := "test123"
	event := "stock_updated"
	key := suite.repo.GetProductTimestampKey(productID, event)
	baseTimestamp := time.Now().UnixMilli()

	const numGoroutines = 10
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	// 從大到小的時間戳測試
	for i := numGoroutines - 1; i >= 0; i-- {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			timestamp := baseTimestamp + int64(i)
			updated, err := suite.repo.SetProductTimestamp(suite.ctx, key, timestamp)
			suite.NoError(err)
			results <- updated
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(results)

	// 收集結果
	successCount := 0
	for updated := range results {
		if updated {
			successCount++
		}
	}

	// 驗證結果
	suite.Equal(1, successCount, "只應該有一個最大的時間戳更新成功")

	// 驗證最終儲存的值是最大的時間戳
	val, err := suite.redisClient.Get(suite.ctx, key).Int64()
	suite.NoError(err)
	suite.Equal(baseTimestamp+numGoroutines-1, val, "應該儲存最大的時間戳")
}

func (suite *TimestampRedisRepoTestSuite) TestSetProductTimestamp_InvalidData() {
	productID := "test123"
	event := "stock_updated"
	key := suite.repo.GetProductTimestampKey(productID, event)

	// 先設置一個無效的值到 Redis
	err := suite.redisClient.Set(suite.ctx, key, "invalid", 0).Err()
	suite.NoError(err)

	// 嘗試設置新的時間戳
	timestamp := time.Now().UnixMilli()
	_, err = suite.repo.SetProductTimestamp(suite.ctx, key, timestamp)
	suite.Error(err, "應該處理無效的數據格式")
}
