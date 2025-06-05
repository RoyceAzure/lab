package config

import (
	"errors"
	"time"
)

var (
	ErrNoBrokers = errors.New("no brokers provided")
	ErrNoTopic   = errors.New("no topic provided")
)

// Config represents the configuration for Kafka client
type Config struct {
	// Broker 配置
	Brokers []string
	Topic   string

	// 消費者配置
	ConsumerGroup    string
	ConsumerMinBytes int
	ConsumerMaxBytes int
	ConsumerMaxWait  time.Duration
	CommitInterval   time.Duration

	// 生產者配置
	BatchSize     int
	BatchTimeout  time.Duration
	RequiredAcks  int
	RetryAttempts int
	RetryDelay    time.Duration

	// 通用配置
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
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
