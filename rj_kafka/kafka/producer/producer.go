package producer

import (
	"context"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/errors"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

// Producer interface defines the methods that a Kafka producer must implement
type Producer interface {
	// Produce sends messages to Kafka
	Produce(ctx context.Context, msgs []message.Message) error
	// Close closes the producer
	Close() error
}

// kafkaProducer implements the Producer interface
type kafkaProducer struct {
	writer *kafka.Writer
	cfg    *config.Config
	closed atomic.Bool
}

// New creates a new Kafka producer
func New(cfg *config.Config) (Producer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Async:        false,

		// 重試機制設置
		MaxAttempts: cfg.RetryAttempts, // 最大重試次數

		// 重連機制設置
		Transport: &kafka.Transport{
			Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
				dialer := &kafka.Dialer{
					Timeout:   10 * time.Second, // 連接超時
					DualStack: true,             // 支援 IPv4/IPv6
					KeepAlive: 30 * time.Second, // TCP keepalive
				}
				return dialer.DialContext(ctx, network, address)
			},
		},

		// 錯誤處理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("kafka producer error: "+msg, args...)
		}),

		// 壓縮設置
		Compression: kafka.Snappy,
	}

	return &kafkaProducer{
		writer: writer,
		cfg:    cfg,
	}, nil
}

// Produce implements the Producer interface
// 同步發送消息，會block到所有消息都寫入
func (p *kafkaProducer) Produce(ctx context.Context, msgs []message.Message) error {
	if p.closed.Load() {
		return errors.ErrProducerClosed
	}

	// 先檢查傳入的參數
	if len(msgs) == 0 {
		return nil // 或者返回一個參數錯誤
	}

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	var err error
	for attempt := 0; attempt <= p.cfg.RetryAttempts; attempt++ {
		// 檢查外部 context 是否已經取消
		if ctx.Err() != nil {
			return errors.NewKafkaError("Produce", p.cfg.Topic, ctx.Err())
		}
		//同步模式，會block到所有消息都寫入
		err = p.writer.WriteMessages(ctx, kafkaMsgs...)
		if err == nil {
			return nil
		}

		if !errors.IsTemporary(err) {
			break
		}
	}

	return errors.NewKafkaError("Produce", p.cfg.Topic, err)
}

// Close implements the Producer interface
func (p *kafkaProducer) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	return p.writer.Close()
}
