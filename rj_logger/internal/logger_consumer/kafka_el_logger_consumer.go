package logger_consumer

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/RoyceAzure/lab/rj_logger/pkg/kafka/config"
	"github.com/RoyceAzure/lab/rj_logger/pkg/kafka/consumer"
	"github.com/RoyceAzure/rj/infra/elsearch"
)

type KafkaElLoggerConsumer struct {
	dao           elsearch.IElSearchDao
	kafkaConsumer consumer.Consumer
	closed        atomic.Bool // 添加狀態追踪
}

// 預設的Kafka logger消費者配置
func GetDefaultConfigForLogger() *config.Config {
	return &config.Config{
		ConsumerMinBytes: 1024 * 1024,      //1M
		ConsumerMaxBytes: 1024 * 1024 * 10, //10M
		ConsumerMaxWait:  time.Millisecond * 100,
		CommitInterval:   time.Second * 5,
		RetryBackoffMax:  1 * time.Second,
		RetryBackoffMin:  100 * time.Millisecond,
	}
}

func NewKafkaElLoggerConsumer(elDao elsearch.IElSearchDao, kafkaConsumer consumer.Consumer) (*KafkaElLoggerConsumer, error) {
	return &KafkaElLoggerConsumer{
		dao:           elDao,
		kafkaConsumer: kafkaConsumer,
	}, nil
}

func (fc *ElLoggerConsumer) handler(message []byte) error {
	if fc.closed.Load() {
		return fmt.Errorf("consumer is closed")
	}

	_, err := fc.elLogger.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message to log file: %w", err)
	}
	return nil
}

func (fc *ElLoggerConsumer) Start(queueName string) error {
	if fc.closed.Load() {
		return fmt.Errorf("consumer is closed")
	}

	return fc.IConsumer.Consume(queueName, fc.handler)
}

func (fc *ElLoggerConsumer) Close() error {
	return errors.Join(
		fc.elLogger.Close(),
		fc.IConsumer.Close(),
	)
}
