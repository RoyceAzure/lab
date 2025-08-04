package alogrithm

import "time"

type LimiterConfig struct {
	Capacity   int
	Rate       float64       // tokens/秒
	RefillRate time.Duration // 補充時間間隔
}

func (l *LimiterConfig) SetCapacity(capacity int) *LimiterConfig {
	l.Capacity = capacity
	return l
}

func (l *LimiterConfig) SetRate(rate float64) *LimiterConfig {
	l.Rate = rate
	return l
}

func (l *LimiterConfig) SetRefillRate(refillRate time.Duration) *LimiterConfig {
	l.RefillRate = refillRate
	return l
}

func GetDefaultLimiterConfig() LimiterConfig {
	return LimiterConfig{
		Capacity:   100,
		Rate:       1,
		RefillRate: time.Second,
	}
}
