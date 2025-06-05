package producer

import (
	"context"
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
	}

	return &kafkaProducer{
		writer: writer,
		cfg:    cfg,
	}, nil
}

// Produce implements the Producer interface
func (p *kafkaProducer) Produce(ctx context.Context, msgs []message.Message) error {
	if p.closed.Load() {
		return errors.ErrProducerClosed
	}

	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	var err error
	for attempt := 0; attempt <= p.cfg.RetryAttempts; attempt++ {
		err = p.writer.WriteMessages(ctx, kafkaMsgs...)
		if err == nil {
			return nil
		}

		if !errors.IsTemporary(err) {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.cfg.RetryDelay):
			continue
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
