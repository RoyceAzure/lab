package consumer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

type Processer interface {
	Process(context.Context, <-chan kafka.Message, chan<- kafka.Message) //必須是無狀態
}

type BaseConsumer struct {
	isAutoCommit            bool
	isRunning               atomic.Bool
	retryTimes              int
	processerNum            int
	batchSize               int
	lastErrorTime           time.Time
	retryInterVal           time.Duration
	wg                      sync.WaitGroup
	ctx                     context.Context
	cancel                  context.CancelFunc
	processer               Processer
	processChan             chan kafka.Message
	resultChan              chan kafka.Message
	isHandleResultCompelete chan struct{}
	reader                  *kafka.Reader
}

// 不使用pipline，用內簽方式達到一體成形的處理消息
// readMsg -> processer處理消息 -> 根據結果決定commit
// 批次讀取是在conn這一層
func NewBaseConsumer(reader *kafka.Reader, p Processer) *BaseConsumer {
	return &BaseConsumer{
		reader:                  reader,
		processerNum:            10,
		retryTimes:              0,
		retryInterVal:           100 * time.Millisecond,
		batchSize:               10000,
		processer:               p,
		resultChan:              make(chan kafka.Message, 10000),
		processChan:             make(chan kafka.Message, 1000),
		isHandleResultCompelete: make(chan struct{}),
	}
}

func (b *BaseConsumer) Start() {
	if !b.isRunning.CompareAndSwap(false, true) {
		return
	}

	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.startConsumerLoop()
}

func (b *BaseConsumer) startConsumerLoop() {
	go b.readMsg(b.ctx, b.processChan) // kafka reader 只能有一個 goroutine

	for i := 0; i < b.processerNum; i++ {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.processer.Process(b.ctx, b.processChan, b.resultChan)
		}()
	}
	go b.handleResult(b.resultChan, b.isHandleResultCompelete)
}

// 由額外goroutine執行
// 注意,kafka reader 並非併發安全，一個goroutine要使用一個reader
// 讀取訊息並放入chan， chan 須由參數傳入
// 依據錯誤類型有重試機制，或者直接關閉consumer
func (b *BaseConsumer) readMsg(ctx context.Context, ch chan<- kafka.Message) {
	defer close(ch)

	//重要，1. chan 要由發送者負責關閉 2 .且不論任何原因發送者return, chan都必須關閉

	var (
		msg kafka.Message
		err error
	)

	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			if b.isAutoCommit {
				msg, err = b.reader.ReadMessage(b.ctx)
			} else {
				msg, err = b.reader.FetchMessage(b.ctx)
			}

			if err != nil {
				// Fatal 錯誤 - 停止消費者
				if errors.Is(err, io.EOF) ||
					errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) {
					log.Printf("kafka reader 已經關閉,將關閉consumer : %v", err)
					return
				}

				if IsKafkaAuthError(err) {
					log.Printf("kafka reader 遭遇權限錯誤, 將關閉consumer: %v", err)
					return
				}

				log.Printf("讀取錯誤: %v", err)
				err := b.retryBackoff()
				if err != nil {
					log.Printf("%s", err)
					return
				}

				continue
			}
			ch <- msg
		}
	}
}

func (b *BaseConsumer) retryBackoff() error {
	if b.retryTimes >= 3 {
		return fmt.Errorf("kafka consumer 重試次數過多，將關閉consumer")
	}

	if b.lastErrorTime.Add(b.retryInterVal).After(time.Now()) {
		b.retryTimes++
	} else {
		b.retryTimes = 1
		b.retryInterVal = 100 * time.Millisecond
	}

	return b.waitWithBackoff()
}

func (b *BaseConsumer) waitWithBackoff() error {
	select {
	case <-b.ctx.Done():
		return fmt.Errorf("kafka consumer 已經關閉")
	case <-time.After(b.retryInterVal):
		b.retryInterVal *= 2
		b.lastErrorTime = time.Now()
		return nil
	}
}

// 藉由關閉in來退出
func (b *BaseConsumer) handleResult(in <-chan kafka.Message, isHandleResultCompelete chan<- struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	toCommit := make([]kafka.Message, 0, b.batchSize)

	for msg := range in {
		select {
		case <-ticker.C:
			toCommit = append(toCommit, msg)
			if len(toCommit) > 0 {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
				b.reader.CommitMessages(ctx, toCommit...)
				toCommit = toCommit[:0]
			}
		default:
			toCommit = append(toCommit, msg)
		}
	}

	if len(toCommit) > 0 {
		ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
		b.reader.CommitMessages(ctx, toCommit...)
	}

	isHandleResultCompelete <- struct{}{}
}

// 訊號關閉時
// reader停止讀取
// 一定時間內消耗剩餘chan 內部資料，並記錄到已完成slice
// 若時間到未結束  則提交已完成slice, 這樣kafka可以記錄正確的已完成資料
// 剩餘資料可以直接拋棄
func (b *BaseConsumer) Stop(timeout time.Duration) error {
	if !b.isRunning.CompareAndSwap(true, false) {
		return nil
	}
	t := time.NewTicker(timeout)
	b.stop()
	select {
	case <-t.C:
		return fmt.Errorf("無法在時間內關閉kafka consumer, 將來可能會有Msg被重複消費")
	case <-b.isHandleResultCompelete:
		return nil
	}
}

// 1.kafka rader停止讀取
// 2.關閉 processChan
// 3.等待所有process goroutine消耗完processChan剩餘Msg
// 4.關閉resultChan
// 5.等待處理result goroutine 提交所有變更
// 6. result goroutine處理完後，於 isHandleResultCompelete chan傳遞消息
// 7. 時間內接收到isHandleResultCompelete 訊號  表示優雅關閉成功
func (b *BaseConsumer) stop() {
	b.cancel()
	b.wg.Wait()
	close(b.resultChan)
}

func IsKafkaAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// SASL Authentication errors
	if strings.Contains(errStr, "SASL Authentication Failed") ||
		strings.Contains(errStr, "Authentication failed") ||
		strings.Contains(errStr, "invalid credentials") ||
		strings.Contains(errStr, "SaslAuthenticationException") {
		return true
	}

	// SSL/TLS Authentication errors
	if strings.Contains(errStr, "SSL handshake failed") ||
		strings.Contains(errStr, "failed authentication due to") ||
		strings.Contains(errStr, "certificate") {
		return true
	}

	// Authorization errors
	if strings.Contains(errStr, "not authorized") ||
		strings.Contains(errStr, "authorization failed") {
		return true
	}

	return false
}
