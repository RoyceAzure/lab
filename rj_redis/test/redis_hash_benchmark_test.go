package test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/stretchr/testify/require"
)

func setupBenchmark(b *testing.B) (cache.Cache, context.Context) {
	redisClient, err := redis_client.GetRedisClient(testRedisAddr, redis_client.WithPassword(testRedisPassword))
	require.NoError(b, err)
	cache := redis_cache.NewRedisCache(redisClient, "benchmark_hash")
	ctx := context.Background()

	// 清理測試數據
	err = cache.Clear(ctx)
	require.NoError(b, err)

	return cache, ctx
}

func BenchmarkHashSetAndGet(b *testing.B) {
	cache, ctx := setupBenchmark(b)

	b.Run("HSet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("product:%d", i)
			err := cache.HSet(ctx, key, "name", fmt.Sprintf("Product %d", i))
			require.NoError(b, err)
		}
	})

	b.Run("HGet", func(b *testing.B) {
		key := "product:0"
		err := cache.HSet(ctx, key, "name", "Test Product")
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cache.HGet(ctx, key, "name")
			require.NoError(b, err)
		}
	})
}

func BenchmarkHashBatchOperations(b *testing.B) {
	cache, ctx := setupBenchmark(b)

	b.Run("HMSet", func(b *testing.B) {
		fields := map[string]any{
			"name":     "Test Product",
			"price":    "99.99",
			"stock":    "100",
			"category": "Electronics",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("product:%d", i)
			err := cache.HMSet(ctx, key, fields)
			require.NoError(b, err)
		}
	})

	b.Run("HMGet", func(b *testing.B) {
		key := "product:0"
		fields := map[string]any{
			"name":     "Test Product",
			"price":    "99.99",
			"stock":    "100",
			"category": "Electronics",
		}
		err := cache.HMSet(ctx, key, fields)
		require.NoError(b, err)

		fieldNames := []string{"name", "price", "stock", "category"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cache.HMGet(ctx, key, fieldNames...)
			require.NoError(b, err)
		}
	})
}

func BenchmarkHashGetAll(b *testing.B) {
	cache, ctx := setupBenchmark(b)

	// 準備測試數據
	key := "product:0"
	fields := map[string]any{
		"name":     "Test Product",
		"price":    "99.99",
		"stock":    "100",
		"category": "Electronics",
		"code":     "P001",
		"reserved": "10",
	}
	err := cache.HMSet(ctx, key, fields)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.HGetAll(ctx, key)
		require.NoError(b, err)
	}
}

func BenchmarkHashIncrBy(b *testing.B) {
	cache, ctx := setupBenchmark(b)

	key := "product:0"
	err := cache.HSet(ctx, key, "stock", "100")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.HIncrBy(ctx, key, "stock", 1)
		require.NoError(b, err)
	}
}

func BenchmarkHashComplexOperations(b *testing.B) {
	cache, ctx := setupBenchmark(b)

	// 模擬複雜的商品操作場景
	b.Run("ProductUpdate", func(b *testing.B) {
		key := "product:0"
		fields := map[string]any{
			"name":     "Test Product",
			"price":    "99.99",
			"stock":    "100",
			"category": "Electronics",
		}
		err := cache.HMSet(ctx, key, fields)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 模擬更新庫存和預訂數量
			_, err := cache.HIncrBy(ctx, key, "stock", -1)
			require.NoError(b, err)
			_, err = cache.HIncrBy(ctx, key, "reserved", 1)
			require.NoError(b, err)

			// 讀取更新後的狀態
			_, err = cache.HGetAll(ctx, key)
			require.NoError(b, err)
		}
	})
}

// BenchmarkHashPerformance 提供詳細的性能分析
func BenchmarkHashPerformance(b *testing.B) {
	cache, ctx := setupBenchmark(b)
	defer cache.Clear(ctx)

	// 記錄初始記憶體狀態
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.TotalAlloc
	initialSys := m.Sys

	// 準備測試數據
	key := "product:0"
	fields := map[string]any{
		"name":     "Test Product",
		"price":    "99.99",
		"stock":    "100",
		"category": "Electronics",
		"code":     "P001",
		"reserved": "10",
	}

	// 測試寫入性能
	b.Run("WritePerformance", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			err := cache.HMSet(ctx, key, fields)
			require.NoError(b, err)
		}
		duration := time.Since(start)

		// 計算每秒操作數
		opsPerSec := float64(b.N) / duration.Seconds()
		b.ReportMetric(opsPerSec, "ops/sec")
	})

	// 測試讀取性能
	b.Run("ReadPerformance", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			_, err := cache.HGetAll(ctx, key)
			require.NoError(b, err)
		}
		duration := time.Since(start)

		// 計算每秒操作數
		opsPerSec := float64(b.N) / duration.Seconds()
		b.ReportMetric(opsPerSec, "ops/sec")
	})

	// 測試並發性能
	b.Run("ConcurrentPerformance", func(b *testing.B) {
		b.SetParallelism(10) // 設置並發數
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// 模擬實際使用場景
				err := cache.HMSet(ctx, key, fields)
				require.NoError(b, err)
				_, err = cache.HGetAll(ctx, key)
				require.NoError(b, err)
				_, err = cache.HIncrBy(ctx, key, "stock", 1)
				require.NoError(b, err)
			}
		})
	})

	// 記錄最終記憶體狀態
	runtime.ReadMemStats(&m)
	finalAlloc := m.TotalAlloc
	finalSys := m.Sys

	// 報告記憶體使用情況
	b.ReportMetric(float64(finalAlloc-initialAlloc)/float64(b.N), "B/op")
	b.ReportMetric(float64(finalSys-initialSys)/float64(b.N), "sys-B/op")
}

// BenchmarkHashLatency 測試延遲情況
func BenchmarkHashLatency(b *testing.B) {
	cache, ctx := setupBenchmark(b)
	defer cache.Clear(ctx)

	key := "product:0"
	fields := map[string]any{
		"name":     "Test Product",
		"price":    "99.99",
		"stock":    "100",
		"category": "Electronics",
	}

	// 準備測試數據
	err := cache.HMSet(ctx, key, fields)
	require.NoError(b, err)

	// 測試延遲
	b.Run("Latency", func(b *testing.B) {
		var totalLatency time.Duration
		var maxLatency time.Duration
		var minLatency time.Duration = time.Hour // 設置一個初始最大值

		for i := 0; i < b.N; i++ {
			start := time.Now()
			_, err := cache.HGetAll(ctx, key)
			require.NoError(b, err)
			latency := time.Since(start)

			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
			if latency < minLatency {
				minLatency = latency
			}
		}

		avgLatency := totalLatency / time.Duration(b.N)
		b.ReportMetric(float64(avgLatency.Microseconds()), "avg-μs/op")
		b.ReportMetric(float64(maxLatency.Microseconds()), "max-μs/op")
		b.ReportMetric(float64(minLatency.Microseconds()), "min-μs/op")
	})
}
