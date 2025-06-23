package consumer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/errors"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

// ConsumerError 代表消費者錯誤
type ConsumerError struct {
	// Fatal 表示是否為致命錯誤，需要終止消費
	Fatal bool
	// Err 原始錯誤
	Err error
	// RetryAttempt 當前重試次數
	RetryAttempt int
}

func (e *ConsumerError) Error() string {
	if e.Fatal {
		return fmt.Sprintf("fatal error (attempt %d): %v", e.RetryAttempt, e.Err)
	}
	return fmt.Sprintf("temporary error (attempt %d): %v", e.RetryAttempt, e.Err)
}

// Consumer interface defines the methods that a Kafka consumer must implement
type Consumer interface {
	Consume() (<-chan message.Message, <-chan error, error)
	CommitMessages(msgs ...message.Message) error
	Close() error
}

// 實現錯誤恢復機制
// 使用者只要於呼叫Consumer後，不斷重msgCh, errCh 接收訊息即可
// 只有當使用者呼叫Close()，才會關閉msgCh, errCh
type kafkaConsumer struct {
	reader    *kafka.Reader
	cfg       *config.Config
	closed    atomic.Bool          // 控制是否已關閉
	consuming atomic.Bool          // 控制是否正在消費
	msgCh     chan message.Message //回傳訊息channel
	errCh     chan error           //回傳錯誤channel

	// 錯誤聚合
	lastError     atomic.Value // 最後一次錯誤
	errorCount    atomic.Int32 // 相同錯誤的次數
	lastErrorTime atomic.Value // 最後一次錯誤的時間
}

// New creates a new Kafka consumer
func New(cfg *config.Config) (Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       cfg.ConsumerMinBytes,
		MaxBytes:       cfg.ConsumerMaxBytes,
		MaxWait:        cfg.ConsumerMaxWait,
		CommitInterval: cfg.CommitInterval,
		Partition:      cfg.Partition,

		// 重連機制設定
		Dialer: &kafka.Dialer{
			Timeout:   cfg.RetryBackoffMax,
			DualStack: true,
			KeepAlive: 30 * time.Second,
		},

		// 錯誤處理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// 不處理，讓我們自己的錯誤處理機制來處理
		}),

		// 讀取重試設定
		ReadBatchTimeout: cfg.RetryBackoffMax,
		ReadLagInterval:  1 * time.Minute,
		ReadBackoffMin:   cfg.RetryBackoffMin,
		ReadBackoffMax:   cfg.RetryBackoffMax,
	})

	consumer := &kafkaConsumer{
		reader: reader,
		cfg:    cfg,
		msgCh:  make(chan message.Message),
		errCh:  make(chan error),
	}

	consumer.lastErrorTime.Store(time.Now())
	return consumer, nil
}

// shouldReportError 判斷是否需要報告錯誤
func (c *kafkaConsumer) shouldReportError(err error) bool {
	lastErr := c.lastError.Load()
	lastErrTime := c.lastErrorTime.Load().(time.Time)

	// 如果是新的錯誤類型，或者距離上次錯誤已經超過30秒
	if lastErr == nil || lastErr.(error).Error() != err.Error() || time.Since(lastErrTime) > 30*time.Second {
		c.lastError.Store(err)
		c.lastErrorTime.Store(time.Now())
		c.errorCount.Store(1)
		return true
	}

	// 累加錯誤次數
	count := c.errorCount.Add(1)

	// 每累積10次錯誤才報告一次
	return count%10 == 0
}

// Consume implements the Consumer interface
func (c *kafkaConsumer) Consume() (<-chan message.Message, <-chan error, error) {
	// 檢查是否已關閉
	if c.closed.Load() {
		return nil, nil, errors.ErrClientClosed
	}

	// 確保只能同時執行一個消費循環
	if !c.consuming.CompareAndSwap(false, true) {
		return nil, nil, errors.ErrConsumerAlreadyRunning
	}

	go c.consumeLoop()

	return c.msgCh, c.errCh, nil
}

// consumeLoop 處理消費循環
func (c *kafkaConsumer) consumeLoop() {
	defer func() {
		c.consuming.Store(false)
		if r := recover(); r != nil {
			if c.closed.Load() {
				return
			}
			c.errCh <- &ConsumerError{
				Fatal: true,
				Err:   fmt.Errorf("consumer panic: %v", r),
			}
		}
	}()

	log.Println("consumeLoop start")
	for {
		if c.closed.Load() {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.cfg.ReadTimeout)
		defer cancel()

		msg, err := c.reader.ReadMessage(ctx)
		if c.closed.Load() {
			return
		}

		if err != nil {
			// 處理錯誤
			kafkaErr := errors.NewKafkaError("Consume", c.cfg.Topic, err)
			isFatal := errors.IsFatalError(err)

			// 只有在需要報告錯誤時才發送到錯誤通道
			if c.shouldReportError(kafkaErr) {
				c.errCh <- &ConsumerError{
					Fatal: isFatal,
					Err:   kafkaErr,
				}
			}

			// 如果是致命錯誤，關閉消費者
			if isFatal {
				c.close()
				return
			}

			// 對於非致命錯誤，讓 kafka-go 的重試機制處理
			if errors.IsConnectionError(err) {
				time.Sleep(c.cfg.RetryBackoffMin)
				continue
			}

			continue
		}

		// 成功讀取消息，發送到消息通道
		c.msgCh <- message.FromKafkaMessage(msg)
	}
}

// CommitMessages implements the Consumer interface
func (c *kafkaConsumer) CommitMessages(msgs ...message.Message) error {
	if c.closed.Load() {
		return errors.ErrClientClosed
	}

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.WriteTimeout)
	defer cancel()

	err := c.reader.CommitMessages(ctx, kafkaMsgs...)
	if err != nil {
		return errors.NewKafkaError("CommitMessages", c.cfg.Topic, err)
	}

	return nil
}

// close 關閉msgCh, errCh, 設置旗號
// 以下情況會自動呼叫close:
// 1. 使用者呼叫Close()
// 2. 消費者發生致命錯誤
// 3. 消費者於重新連接時發生錯誤
func (c *kafkaConsumer) close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.consuming.Store(false)
	err := c.reader.Close()
	close(c.msgCh)
	close(c.errCh)
	return err
}

func (c *kafkaConsumer) Close() error {
	return c.close()
}
