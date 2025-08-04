package alogrithm

import (
	"sync"
	"time"
)

/*
使用鎖實現，高QPS請採用其他窗口策略
*/
type SlideWindow struct {
	LimiterConfig
	window []time.Time
	mu     sync.Mutex
}

func NewSlideWindow(config *LimiterConfig) *SlideWindow {
	sw := &SlideWindow{
		window: make([]time.Time, 0),
		mu:     sync.Mutex{},
	}
	if config != nil {
		sw.LimiterConfig = *config
	} else {
		sw.LimiterConfig = GetDefaultLimiterConfig()
	}
	return sw
}

func (w *SlideWindow) Allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	validStart := len(w.window)
	for i, t := range w.window {
		diff := now.Sub(t)
		if diff < w.RefillRate {
			validStart = i
			break
		}
	}

	w.window = w.window[validStart:]
	if len(w.window) >= w.Capacity {
		return false
	}
	w.window = append(w.window, now)

	return true
}
