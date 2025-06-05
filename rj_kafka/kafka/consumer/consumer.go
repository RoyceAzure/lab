package consumer

import (
	"context"
	"sync/atomic"

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
}

func (e *ConsumerError) Error() string {
	if e.Fatal {
		return "fatal error: " + e.Err.Error()
	}
	return "temporary error: " + e.Err.Error()
}

// Consumer interface defines the methods that a Kafka consumer must implement
type Consumer interface {
	Consume(ctx context.Context) (<-chan message.Message, <-chan error)
	CommitMessages(ctx context.Context, msgs ...message.Message) error
	Close() error
}

// kafkaConsumer implements the Consumer interface
type kafkaConsumer struct {
	reader *kafka.Reader
	cfg    *config.Config
	closed atomic.Bool
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
	})

	return &kafkaConsumer{
		reader: reader,
		cfg:    cfg,
	}, nil
}

// Consume implements the Consumer interface
func (c *kafkaConsumer) Consume(ctx context.Context) (<-chan message.Message, <-chan error) {
	msgCh := make(chan message.Message)
	errCh := make(chan error)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				errCh <- &ConsumerError{
					Fatal: true,
					Err:   ctx.Err(),
				}
				return
			default:
				if c.closed.Load() {
					errCh <- &ConsumerError{
						Fatal: true,
						Err:   errors.ErrConsumerClosed,
					}
					return
				}

				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					kafkaErr := errors.NewKafkaError("Consume", c.cfg.Topic, err)

					if errors.IsFatalConsumerError(err) {
						errCh <- &ConsumerError{
							Fatal: true,
							Err:   kafkaErr,
						}
						return
					}

					// 可重試的錯誤
					errCh <- &ConsumerError{
						Fatal: false,
						Err:   kafkaErr,
					}
					continue
				}

				msgCh <- message.FromKafkaMessage(msg)
			}
		}
	}()

	return msgCh, errCh
}

// CommitMessages implements the Consumer interface
func (c *kafkaConsumer) CommitMessages(ctx context.Context, msgs ...message.Message) error {
	if c.closed.Load() {
		return errors.ErrConsumerClosed
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
	return c.reader.Close()
}
