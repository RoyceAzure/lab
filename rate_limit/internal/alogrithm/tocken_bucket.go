package alogrithm

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

/*
請使用 defer 呼叫 Stop()
*/
type TokenBucket struct {
	LimiterConfig
	current      atomic.Int64
	lastRefilled atomic.Int64
	cancel       chan struct{}
	once         sync.Once //for close background
}

/*
請使用 defer 呼叫 Stop()
*/
func NewTokenBucket(config *LimiterConfig) *TokenBucket {
	t := &TokenBucket{
		cancel: make(chan struct{}),
	}

	if config != nil {
		t.LimiterConfig = *config
	} else {
		t.LimiterConfig = GetDefaultLimiterConfig()
	}

	t.current.Store(int64(t.Capacity))
	t.lastRefilled.Store(time.Now().UnixNano())
	go t.background()
	return t
}

func (t *TokenBucket) Allow(ctx context.Context) bool {
	for {
		current := t.current.Load()
		if current <= 0 {
			return false
		}
		if t.current.CompareAndSwap(current, current-1) {
			return true
		}
	}
}

func (t *TokenBucket) background() {
	ticker := time.NewTicker(t.RefillRate)
	defer ticker.Stop()
	for {
		select {
		case <-t.cancel:
			return
		case <-ticker.C:
			for {
				newToken := t.current.Load() + int64(t.RatePPeriod)
				if newToken > int64(t.Capacity) {
					newToken = int64(t.Capacity)
				}
				if t.current.CompareAndSwap(t.current.Load(), newToken) {
					break
				}
			}
		}
	}
}

func (t *TokenBucket) Stop() {
	t.once.Do(func() {
		close(t.cancel)
	})
}
