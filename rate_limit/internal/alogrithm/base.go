package alogrithm

import "time"

type LimiterConfig struct {
	Key         string
	Capacity    int
	RatePS      int // tokens/秒
	RatePPeriod int
	RefillRate  time.Duration // 補充時間間隔
}

func (l *LimiterConfig) SetCapacity(capacity int) {
	l.Capacity = capacity
}

func (l *LimiterConfig) SetRatePS(rate int) {
	l.RatePS = rate
}

func (l *LimiterConfig) SetRatePPeriod(rate int) {
	l.RatePPeriod = rate
}

func (l *LimiterConfig) SetRefillRate(refillRate time.Duration) {
	l.RefillRate = refillRate
}

func (l *LimiterConfig) SetKey(key string) {
	l.Key = key
}

func GetDefaultLimiterConfig() LimiterConfig {
	return LimiterConfig{
		Key:         "global",
		Capacity:    100,
		RatePS:      1,
		RatePPeriod: 1,
		RefillRate:  time.Second,
	}
}
