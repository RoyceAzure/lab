package logger

import (
	"context"
	"fmt"
	"time"

	lab_config "github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/segmentio/kafka-go"
)

type kafkaLoggerConfig struct {
	AutoResetOffset bool `yaml:"auto_reset_offset"` // 重連後是否重設 offset
	modulerBytes    []byte
	moduler         string

	// Broker 配置
	Topic   string
	Brokers []string

	// 消費者配置
	ConsumerGroup    string
	ConsumerMinBytes int
	ConsumerMaxBytes int
	Partition        int
	ConsumerMaxWait  time.Duration
	CommitInterval   time.Duration

	// 生產者配置
	RequiredAcks  int
	RetryAttempts int
	BatchSize     int
	BatchTimeout  time.Duration
	RetryDelay    time.Duration

	// 通用配置
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// 重連相關配置
	MaxRetryAttempts   int           `yaml:"max_retry_attempts"`   // 最大重試次數
	RetryBackoffMin    time.Duration `yaml:"retry_backoff_min"`    // 最小重試間隔
	RetryBackoffMax    time.Duration `yaml:"retry_backoff_max"`    // 最大重試間隔
	RetryBackoffFactor float64       `yaml:"retry_backoff_factor"` // 重試間隔增長因子
	ReconnectWaitTime  time.Duration `yaml:"reconnect_wait_time"`  // 重連等待時間

	// 分區策略配置
	Balancer kafka.Balancer // 自定義負載平衡器
}

// for consumer  不需要接收來自zerolog的資訊
type KafkaLogger struct {
	cf *kafkaLoggerConfig
	w  producer.Producer
}

func transCf(config kafkaLoggerConfig) *lab_config.Config {
	return &lab_config.Config{
		Brokers: config.Brokers,
		Topic:   config.Topic,

		// 生產者配置
		BatchSize:     config.BatchSize,
		BatchTimeout:  config.BatchTimeout,
		RequiredAcks:  config.RequiredAcks,
		RetryAttempts: config.RetryAttempts,
		RetryDelay:    config.RetryDelay,
		Async:         true,
		// 通用配置
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,

		// 重連相關配置
		MaxRetryAttempts:   config.MaxRetryAttempts,
		RetryBackoffMin:    config.RetryBackoffMin,
		RetryBackoffMax:    config.RetryBackoffMax,
		RetryBackoffFactor: config.RetryBackoffFactor,
		AutoResetOffset:    config.AutoResetOffset,
		ReconnectWaitTime:  config.ReconnectWaitTime,
	}
}

func NewKafkaLogger(config *kafkaLoggerConfig) (*KafkaLogger, error) {
	p, err := producer.New(transCf(*config))
	if err != nil {
		return nil, err
	}

	config.modulerBytes = []byte(config.moduler)

	return &KafkaLogger{
		cf: config,
		w:  p,
	}, nil
}

func (kw *KafkaLogger) Write(p []byte) (n int, err error) {
	if kw == nil {
		return 0, fmt.Errorf("file logger is not init")
	}

	err = kw.w.Produce(context.Background(), []message.Message{
		{
			Key:   kw.cf.modulerBytes,
			Value: p,
		},
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (fw *KafkaLogger) Close() error {
	return fw.w.Close()
}
