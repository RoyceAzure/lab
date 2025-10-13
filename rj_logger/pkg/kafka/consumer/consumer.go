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

type ConsuemError struct {
	Message kafka.Message
	Err     error
}

type Consumer struct {
	isRunning      atomic.Bool
	retryTimes     int
	processerNum   int
	batchSize      int
	lastErrorTime  time.Time
	retryInterVal  time.Duration
	processWg      sync.WaitGroup //for process go routine
	handleResultWg sync.WaitGroup // for handler success, failed go routine
	ctx            context.Context
	cancel         context.CancelFunc
	processer      Processer
	reader         KafkaReader

	processChan        chan kafka.Message
	resultChan         chan kafka.Message
	dlq                chan ConsuemError
	handlerSuccessfunc func(kafka.Message)
	handlerErrorfunc   func(ConsuemError)
	isStopped          chan struct{}
}

// 不使用pipline，用內簽方式達到一體成形的處理消息
// readMsg -> processer處理消息 -> 根據結果決定commit
// 批次讀取是在conn這一層
func NewConsumer(reader KafkaReader, p Processer, handlerSuccessfunc func(kafka.Message), handlerErrorfunc func(ConsuemError)) *Consumer {
	if handlerSuccessfunc == nil {
		handlerSuccessfunc = DefaultHandleSucessFunc
	}

	if handlerErrorfunc == nil {
		handlerErrorfunc = DefaulthandlerErrorfunc
	}

	return &Consumer{
		reader:             reader,
		processerNum:       10,
		retryTimes:         0,
		retryInterVal:      100 * time.Millisecond,
		batchSize:          10000,
		processer:          p,
		resultChan:         make(chan kafka.Message, 10000),
		processChan:        make(chan kafka.Message, 1000),
		dlq:                make(chan ConsuemError, 1000),
		handlerSuccessfunc: handlerSuccessfunc,
		handlerErrorfunc:   handlerErrorfunc,
		isStopped:          make(chan struct{}),
	}
}

func (b *Consumer) Start() {
	if !b.isRunning.CompareAndSwap(false, true) {
		return
	}

	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.startConsumerLoop()
}

func (b *Consumer) startConsumerLoop() {
	b.handleResultWg.Add(1)
	go func() {
		defer b.handleResultWg.Done()
		b.handleResult(b.resultChan)
	}()
	b.handleResultWg.Add(1)
	go func() {
		defer b.handleResultWg.Done()
		b.handleError(b.dlq)
	}()
	for i := 0; i < b.processerNum; i++ {
		b.processWg.Add(1)
		go func() {
			defer b.processWg.Done()
			b.process(b.ctx, b.processChan, b.resultChan, b.dlq)
		}()
	}
	go b.readMsg(b.ctx, b.processChan) // kafka reader 只能有一個 goroutine
}

// 以關閉 in 當作結束訊號，會持續處理直到in沒有訊息
// 結束時會一併關閉out, dlq chan
// 批次處理，若其中一個錯誤，則整批批次失敗，錯誤訊息也會是同一個
func (b *Consumer) process(ctx context.Context,
	in <-chan kafka.Message,
	out chan<- kafka.Message,
	dlq chan<- ConsuemError) {

	ticker := time.NewTicker(100 * time.Millisecond)
	batch := make([]kafka.Message, 0, b.batchSize)

	processMsg := func() {
		if len(batch) > 0 {
			err := b.processer.Process(ctx, batch)
			if err != nil {
				for _, msg := range batch {
					dlq <- ConsuemError{
						Message: msg,
						Err:     err,
					}
				}
			} else {
				for _, msg := range batch {
					out <- msg
				}
			}
			batch = batch[:0]
		}
	}

	for {
		select {
		case msg, ok := <-in:
			if !ok {
				processMsg()
				return
			}
			batch = append(batch, msg)
		case <-ticker.C:
			processMsg()
		}
	}
}

// 由額外goroutine執行
// 注意,kafka reader 並非併發安全，一個goroutine要使用一個reader
// 讀取訊息並放入chan， chan 須由參數傳入
// 依據錯誤類型有重試機制，或者直接關閉consumer
func (b *Consumer) readMsg(ctx context.Context, in chan<- kafka.Message) {
	defer b.stop()
	defer close(b.processChan)

	//重要，1. chan 要由發送者負責關閉 2 .且不論任何原因發送者return, chan都必須關閉

	var (
		msg kafka.Message
		err error
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err = b.reader.FetchMessage(ctx)

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
			in <- msg
		}
	}
}

func (b *Consumer) retryBackoff() error {
	if b.retryTimes >= 3 {
		return fmt.Errorf("kafka consumer 重試次數過多，將關閉consumer")
	}

	if b.lastErrorTime.Add(b.retryInterVal).After(time.Now()) {
		b.retryTimes++
	} else {
		b.retryTimes = 1
		b.retryInterVal = 100 * time.Millisecond
	}

	time.Sleep(b.retryInterVal)
	b.retryInterVal *= 2
	b.lastErrorTime = time.Now()

	return nil
}

// 藉由關閉in來退出
func (b *Consumer) handleResult(in <-chan kafka.Message) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	toCommit := make([]kafka.Message, 0, b.batchSize)

	commitMsgs := func() {
		if len(toCommit) > 0 {
			ctx, cancle := context.WithTimeout(context.Background(), 1*time.Second)
			b.reader.CommitMessages(ctx, toCommit...)
			cancle()
			toCommit = toCommit[:0]
		}
	}

	for {
		select {
		case msg, ok := <-in:
			if !ok {
				commitMsgs()
				return
			}
			b.handlerSuccessfunc(msg)
			toCommit = append(toCommit, msg)
		case <-ticker.C:
			commitMsgs()
		}
	}
}

func (b *Consumer) handleError(dlq <-chan ConsuemError) {
	for err := range dlq {
		b.handlerErrorfunc(err)
	}
}

// 訊號關閉時
// reader停止讀取
// 一定時間內消耗剩餘chan 內部資料，並記錄到已完成slice
// 若時間到未結束  則提交已完成slice, 這樣kafka可以記錄正確的已完成資料
// 剩餘資料可以直接拋棄
func (b *Consumer) Stop(timeout time.Duration) error {
	b.stop()
	return nil
}

// 1.kafka rader停止讀取
// 2.關閉 processChan
// 3.等待所有process goroutine消耗完processChan剩餘Msg
// 4.關閉resultChan
// 5.等待處理result goroutine 提交所有變更
// 6. result goroutine處理完後，於 isHandleResultCompelete chan傳遞消息
// 7. 時間內接收到isHandleResultCompelete 訊號  表示優雅關閉成功
func (b *Consumer) stop() {
	if !b.isRunning.CompareAndSwap(true, false) {
		return
	}
	b.cancel()
	b.processWg.Wait()
	close(b.resultChan)
	close(b.dlq)
	b.handleResultWg.Wait()
	close(b.isStopped)
}

func (b *Consumer) C() chan struct{} {
	return b.isStopped
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

func DefaultHandleSucessFunc(msg kafka.Message) {
	log.Printf("consume message success :%v", msg)
}

func DefaulthandlerErrorfunc(msg ConsuemError) {
	log.Printf("consume message failed :%v, err : %s", msg.Message, msg.Err)
}
