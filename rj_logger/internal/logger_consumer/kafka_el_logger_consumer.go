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

// 同一個模組使用同一個主題 使用timestamp讓elastic 做log排序
// 同一個獏組使用多個分區消耗以應付大量訊息
// consumer數量 分區數量要先想好  在創建對應數量的KafkaElLoggerConsumer
// 若consumer logger 自己本身執行有錯誤  就直接寫入到elastic
// 如果要使用pipline模式  就會有消息丟失風險  因為consumer讀取後就會自動commit，並不知道消息處理結果
func NewKafkaElLoggerConsumer(elDao elsearch.IElSearchDao, kafkaConsumer consumer.Consumer) (*KafkaElLoggerConsumer, error) {
	return &KafkaElLoggerConsumer{
		dao:           elDao,
		kafkaConsumer: kafkaConsumer,
	}, nil
}

// 每次batch size 數量message傳入 使用batch insert
func (fc *KafkaElLoggerConsumer) handler(message []byte) error {
	if fc.closed.Load() {
		return fmt.Errorf("consumer is closed")
	}

	_, err := fc.elLogger.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message to log file: %w", err)
	}
	return nil
}

// 使用 batch insert
func (fc *KafkaElLoggerConsumer) Start(queueName string) error {
	if fc.closed.Load() {
		return fmt.Errorf("consumer is closed")
	}

	return fc.IConsumer.Consume(queueName, fc.handler)
}

func (fc *KafkaElLoggerConsumer) Close() error {
	return errors.Join(
		fc.elLogger.Close(),
		fc.IConsumer.Close(),
	)
}
