package alogrithm

import (
	"sync"
	"sync/atomic"
	"time"
)

/*
會有突刺問題
*/
type FixedWindow struct {
	LimiterConfig
	count     atomic.Int32
	startedAt time.Time
	mu        sync.RWMutex
}

func NewFixWindow(config *LimiterConfig) *FixedWindow {
	fw := &FixedWindow{
		count:     atomic.Int32{},
		startedAt: time.Now(),
		mu:        sync.RWMutex{},
	}
	if config != nil {
		fw.LimiterConfig = *config
	} else {
		fw.LimiterConfig = GetDefaultLimiterConfig()
	}

	return fw
}

func (w *FixedWindow) Allow() bool {
	current := time.Now()
	w.mu.RLock()
	needReset := current.Sub(w.startedAt) > w.RefillRate
	w.mu.RUnlock()

	if needReset {
		w.mu.Lock()
		if current.Sub(w.startedAt) > w.RefillRate {
			w.reset()
		}
		w.mu.Unlock()
	}

	for {
		current := w.count.Load()
		if current+1 > int32(w.Capacity) {
			return false
		}
		if w.count.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (w *FixedWindow) reset() {
	w.count.Store(0)
	w.startedAt = time.Now()
}
