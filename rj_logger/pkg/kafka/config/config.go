package config

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Config represents the configuration for Kafka client
type Config struct {
	Async bool
	// Broker 配置
	Brokers []string
	Topic   string

	// 消費者配置
	ConsumerGroup string
	Partition     int

	// 生產者配置
	RequiredAcks int

	// 通用配置
	Timeout        time.Duration
	RetryDelay     time.Duration
	RetryLimit     int
	RetryFactor    int
	WorkerNum      int
	BatchSize      int
	MinBytes       int
	MaxBytes       int
	MaxWait        time.Duration
	CommitInterval time.Duration

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
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        time.Second,
		CommitInterval: 100 * time.Millisecond,
		BatchSize:      1000,
		Timeout:        time.Second,
		RequiredAcks:   -1, // 等待所有副本確認
		RetryLimit:     3,
		RetryDelay:     time.Millisecond * 200,
		RetryFactor:    2,
	}
}

// GetBatchTimeout returns the BatchTimeout if set, otherwise returns a default value
func GetDefaultBatchTimeout() time.Duration {
	return 1 * time.Second
}
