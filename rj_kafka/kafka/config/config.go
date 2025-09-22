package config

import (
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	ErrNoBrokers = errors.New("no brokers provided")
	ErrNoTopic   = errors.New("no topic provided")
)

// Config represents the configuration for Kafka client
type Config struct {
	Async bool
	// Broker 配置
	Brokers []string
	Topic   string

	// 消費者配置
	ConsumerGroup    string
	ConsumerMinBytes int
	ConsumerMaxBytes int
	ConsumerMaxWait  time.Duration
	CommitInterval   time.Duration
	Partition        int

	// 生產者配置
	BatchSize     int
	BatchTimeout  time.Duration
	RequiredAcks  int
	RetryAttempts int
	RetryDelay    time.Duration

	// 通用配置
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// 重連相關配置
	MaxRetryAttempts   int           `yaml:"max_retry_attempts"`   // 最大重試次數
	RetryBackoffMin    time.Duration `yaml:"retry_backoff_min"`    // 最小重試間隔
	RetryBackoffMax    time.Duration `yaml:"retry_backoff_max"`    // 最大重試間隔
	RetryBackoffFactor float64       `yaml:"retry_backoff_factor"` // 重試間隔增長因子
	AutoResetOffset    bool          `yaml:"auto_reset_offset"`    // 重連後是否重設 offset
	ReconnectWaitTime  time.Duration `yaml:"reconnect_wait_time"`  // 重連等待時間

	// 分區策略配置
	Balancer kafka.Balancer // 自定義負載平衡器
}

// GetBalancer 取得負載平衡器，如果沒有設定則使用預設的 LeastBytes
func (c *Config) GetBalancer() kafka.Balancer {
	if c.Balancer != nil {
		return c.Balancer
	}
	// 預設使用 LeastBytes
	return &kafka.LeastBytes{}
}

// DefaultConfig returns a Config with default settings
func DefaultConfig() *Config {
	return &Config{
		ConsumerMinBytes: 10e3, // 10KB
		ConsumerMaxBytes: 10e6, // 10MB
		ConsumerMaxWait:  time.Second,
		CommitInterval:   time.Second,
		BatchSize:        100,
		BatchTimeout:     time.Second,
		RequiredAcks:     -1, // 等待所有副本確認
		RetryAttempts:    3,
		RetryDelay:       time.Millisecond * 250,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}
	if c.Topic == "" {
		return ErrNoTopic
	}
	return nil
}

// GetBatchTimeout returns the BatchTimeout if set, otherwise returns a default value
func GetDefaultBatchTimeout() time.Duration {
	return 1 * time.Second
}
