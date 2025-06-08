package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	Consume(ctx context.Context) (<-chan message.Message, <-chan error)
	CommitMessages(ctx context.Context, msgs ...message.Message) error
	Close() error
}

// kafkaConsumer implements the Consumer interface
type kafkaConsumer struct {
	reader        *kafka.Reader
	cfg           *config.Config
	closed        atomic.Bool
	consuming     atomic.Bool // 控制是否正在消費
	retryAttempts atomic.Int32
	lastError     atomic.Value
	msgCh         chan message.Message
	errCh         chan error
	stopCh        chan struct{} // 用於停止消費循環
	mu            sync.Mutex    // 保護重要操作
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

		// 重連機制設定
		Dialer: &kafka.Dialer{
			Timeout:   cfg.RetryBackoffMax,
			DualStack: true,
			KeepAlive: 30 * time.Second,
		},

		// 錯誤處理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("Kafka Reader Error: "+msg, args...)
		}),

		// 讀取重試設定
		ReadBatchTimeout: cfg.RetryBackoffMax,
		ReadLagInterval:  1 * time.Minute,
		ReadBackoffMin:   cfg.RetryBackoffMin,
		ReadBackoffMax:   cfg.RetryBackoffMax,
	})

	return &kafkaConsumer{
		reader: reader,
		cfg:    cfg,
		msgCh:  make(chan message.Message),
		errCh:  make(chan error),
		stopCh: make(chan struct{}),
	}, nil
}

// calculateBackoff 計算下一次重試的等待時間
func (c *kafkaConsumer) calculateBackoff(attempt int32) time.Duration {
	backoff := float64(c.cfg.RetryBackoffMin)
	for i := int32(0); i < attempt && backoff < float64(c.cfg.RetryBackoffMax); i++ {
		backoff *= c.cfg.RetryBackoffFactor
	}
	if backoff > float64(c.cfg.RetryBackoffMax) {
		backoff = float64(c.cfg.RetryBackoffMax)
	}
	return time.Duration(backoff)
}

// resetConnection 重置連接
func (c *kafkaConsumer) resetConnection(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 關閉當前的 reader
	if err := c.reader.Close(); err != nil {
		log.Printf("Error closing reader during reset: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.cfg.Brokers,
		Topic:          c.cfg.Topic,
		GroupID:        c.cfg.ConsumerGroup,
		MinBytes:       c.cfg.ConsumerMinBytes,
		MaxBytes:       c.cfg.ConsumerMaxBytes,
		MaxWait:        c.cfg.ConsumerMaxWait,
		CommitInterval: c.cfg.CommitInterval,
		Dialer: &kafka.Dialer{
			Timeout:   c.cfg.RetryBackoffMax,
			DualStack: true,
			KeepAlive: 30 * time.Second,
		},
	})

	c.reader = reader
	return nil
}

// Consume implements the Consumer interface
func (c *kafkaConsumer) Consume(ctx context.Context) (<-chan message.Message, <-chan error) {
	// 檢查是否已關閉
	if c.closed.Load() {
		c.errCh <- &ConsumerError{
			Fatal: true,
			Err:   errors.ErrClientClosed,
		}
		close(c.msgCh)
		close(c.errCh)
		return c.msgCh, c.errCh
	}

	// 確保只能同時執行一個消費循環
	if !c.consuming.CompareAndSwap(false, true) {
		c.errCh <- &ConsumerError{
			Fatal: true,
			Err:   errors.ErrConsumerAlreadyRunning,
		}
		return c.msgCh, c.errCh
	}

	// 重置 channels
	c.mu.Lock()
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	c.mu.Unlock()

	// 啟動消費循環
	go c.consumeLoop(ctx)

	return c.msgCh, c.errCh
}

// consumeLoop 處理消費循環
func (c *kafkaConsumer) consumeLoop(ctx context.Context) {
	defer func() {
		c.consuming.Store(false)
		if r := recover(); r != nil {
			c.errCh <- &ConsumerError{
				Fatal: true,
				Err:   fmt.Errorf("consumer panic: %v", r),
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		default:
			if c.closed.Load() {
				c.errCh <- &ConsumerError{
					Fatal: true,
					Err:   errors.ErrClientClosed,
				}
				return
			}

			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				kafkaErr := errors.NewKafkaError("Consume", c.cfg.Topic, err)

				// 1. 先檢查是否為致命錯誤
				if errors.IsFatalError(err) {
					c.errCh <- &ConsumerError{
						Fatal: true,
						Err:   kafkaErr,
					}
					return
				}

				// 2. 檢查是否為連接錯誤
				if errors.IsConnectionError(err) {
					// 嘗試重置連接
					if resetErr := c.resetConnection(ctx); resetErr != nil {
						c.errCh <- &ConsumerError{
							Fatal: true,
							Err:   fmt.Errorf("failed to reset connection: %w", resetErr),
						}
						return
					}

					// 發送非致命錯誤通知
					c.errCh <- &ConsumerError{
						Fatal: false,
						Err:   kafkaErr,
					}

					// 等待一段時間後繼續
					select {
					case <-c.stopCh:
						return
					case <-time.After(c.cfg.ReconnectWaitTime):
						continue
					}
				}

				// 3. 其他錯誤，通知後繼續
				c.errCh <- &ConsumerError{
					Fatal: false,
					Err:   kafkaErr,
				}
				continue
			}

			c.msgCh <- message.FromKafkaMessage(msg)
		}
	}
}

// CommitMessages implements the Consumer interface
func (c *kafkaConsumer) CommitMessages(ctx context.Context, msgs ...message.Message) error {
	if c.closed.Load() {
		return errors.ErrClientClosed
	}

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	err := c.reader.CommitMessages(ctx, kafkaMsgs...)
	if err != nil {
		return errors.NewKafkaError("CommitMessages", c.cfg.Topic, err)
	}

	return nil
}

// Close implements the Consumer interface
func (c *kafkaConsumer) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	c.mu.Lock()
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}
	c.mu.Unlock()

	// 等待消費循環結束
	for c.consuming.Load() {
		time.Sleep(100 * time.Millisecond)
	}

	close(c.msgCh)
	close(c.errCh)
	return c.reader.Close()
}
