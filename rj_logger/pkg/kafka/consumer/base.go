package consumer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

type Processer interface {
	Process(chan<- []kafka.Message) bool
}

type BaseConsumer struct {
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   atomic.Bool
	reader      *kafka.Reader
	isReadBatch atomic.Bool
	minBatch    int
	maxBatch    int
}

// 不使用pipline，用內簽方式達到一體成形的處理消息
// readMsg -> processer處理消息 -> 根據結果決定commit
func NewBaseConsumer(reader *kafka.Reader) *BaseConsumer {
	return &BaseConsumer{
		reader: reader,
	}
}

func (b *BaseConsumer) Start() {
	if !b.isRunning.CompareAndSwap(false, true) {
		return
	}

	b.wg.Add(1)
	b.ctx, b.cancel = context.WithCancel(context.Background())
	go b.consumerLoop()
}

func (b *BaseConsumer) consumerLoop() {
	defer b.wg.Done()

	select {
	case <-b.ctx.Done():
		return
	default:
		//是否批次讀取  worker轉換  處理消息  根據結果commit 或者直接commit
	}

}

func (b *BaseConsumer) setReadBatch() {
	b.isReadBatch.Store(true)
}

func (b *BaseConsumer) setReadOne() {
	b.isReadBatch.Store(false)
}

func (b *BaseConsumer) Stop(timeout time.Duration) error {
	if !b.isRunning.CompareAndSwap(true, false) {
		return nil
	}

	b.cancel()

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil // 正常結束
	case <-time.After(timeout):
		return errors.New("kafka consumer close timeout")
	}
}
