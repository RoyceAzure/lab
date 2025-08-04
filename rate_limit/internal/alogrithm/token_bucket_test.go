package alogrithm

import (
	"testing"
	"time"
)

func TestTokenBucket_Basic(t *testing.T) {
	// 建立一個容量為5，每秒補充2個token的bucket
	config := LimiterConfig{
		Capacity:   5,
		Rate:       2,
		RefillRate: 100 * time.Millisecond,
	}
	bucket := NewTokenBucket(&config)
	defer bucket.Stop()

	// 測試初始容量
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("應該允許第 %d 次請求", i+1)
		}
	}

	// 第6次應該被拒絕
	if bucket.Allow() {
		t.Error("超過容量限制應該被拒絕")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// 建立一個容量為2，每秒補充1個token的bucket
	config := LimiterConfig{
		Capacity:   2,
		Rate:       1,
		RefillRate: time.Second,
	}
	bucket := NewTokenBucket(&config)
	defer bucket.Stop()

	// 消耗所有token
	bucket.Allow()
	bucket.Allow()

	if bucket.Allow() {
		t.Error("應該沒有可用的token")
	}

	// 等待1.1秒，應該補充了1個token
	time.Sleep(1100 * time.Millisecond)

	if !bucket.Allow() {
		t.Error("應該有1個新的token可用")
	}

	if bucket.Allow() {
		t.Error("不應該有第2個token可用")
	}
}

func TestTokenBucket_Capacity(t *testing.T) {
	// 建立一個容量為2，每秒補充10個token的bucket（故意設定補充速率大於容量）
	config := LimiterConfig{
		Capacity:   2,
		Rate:       10,
		RefillRate: 100 * time.Millisecond,
	}
	bucket := NewTokenBucket(&config)
	defer bucket.Stop()

	// 消耗所有token
	bucket.Allow()
	bucket.Allow()

	// 等待足夠長的時間讓token補充
	time.Sleep(500 * time.Millisecond)

	// 應該只能使用2次（容量限制）
	if !bucket.Allow() {
		t.Error("應該有token可用")
	}
	if !bucket.Allow() {
		t.Error("應該有第2個token可用")
	}
	if bucket.Allow() {
		t.Error("不應該超過容量限制")
	}
}

func TestTokenBucket_Concurrent(t *testing.T) {
	config := LimiterConfig{
		Capacity:   100,
		Rate:       10,
		RefillRate: 100 * time.Millisecond,
	}
	bucket := NewTokenBucket(&config)
	defer bucket.Stop()

	// 並發測試
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				bucket.Allow()
				time.Sleep(50 * time.Millisecond)
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTokenBucket_DefaultConfig(t *testing.T) {
	// 測試使用預設配置
	bucket := NewTokenBucket(nil)
	defer bucket.Stop()

	// 檢查是否可以正常使用預設配置
	defaultConfig := GetDefaultLimiterConfig()

	// 測試初始容量
	for i := 0; i < defaultConfig.Capacity; i++ {
		if !bucket.Allow() {
			t.Errorf("使用預設配置時，應該允許第 %d 次請求", i+1)
		}
	}

	// 下一次應該被拒絕
	if bucket.Allow() {
		t.Error("使用預設配置時，超過容量限制應該被拒絕")
	}
}
