package errors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/segmentio/kafka-go"
)

var (
	ErrProducerClosed = errors.New("producer is closed")
	ErrConsumerClosed = errors.New("consumer is closed")
	ErrWriteTimeout   = errors.New("write timeout")
	ErrReadTimeout    = errors.New("read timeout")
)

// KafkaError represents a Kafka-specific error
type KafkaError struct {
	Op      string // 操作名稱
	Topic   string // 主題
	Message string // 錯誤訊息
	Err     error  // 原始錯誤
}

func (e *KafkaError) Error() string {
	if e.Topic != "" {
		return fmt.Sprintf("%s failed for topic %s: %v", e.Op, e.Topic, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", e.Op, e.Err)
}

func (e *KafkaError) Unwrap() error {
	return e.Err
}

// Is implements error comparison for KafkaError
func (e *KafkaError) Is(target error) bool {
	// 如果目標錯誤是 KafkaError
	if t, ok := target.(*KafkaError); ok {
		// 比較操作名稱和主題（如果有指定）
		if t.Op != "" && t.Op != e.Op {
			return false
		}
		if t.Topic != "" && t.Topic != e.Topic {
			return false
		}
		// 如果內部錯誤存在，遞迴比較
		if t.Err != nil {
			return errors.Is(e.Err, t.Err)
		}
		return true
	}
	// 如果目標錯誤不是 KafkaError，比較內部錯誤
	return errors.Is(e.Err, target)
}

// NewKafkaError creates a new KafkaError
func NewKafkaError(op, topic string, err error) *KafkaError {
	return &KafkaError{
		Op:    op,
		Topic: topic,
		Err:   err,
	}
}

// IsTemporary returns true if the error is temporary and can be retried
func IsTemporary(err error) bool {
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		return isTemporaryError(kafkaErr.Err)
	}
	return isTemporaryError(err)
}

// isTemporaryError checks if the error is temporary
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// 檢查我們自定義的臨時錯誤
	if errors.Is(err, ErrWriteTimeout) || errors.Is(err, ErrReadTimeout) {
		return true
	}

	// 檢查 kafka-go 特定的錯誤
	if errors.Is(err, kafka.LeaderNotAvailable) ||
		errors.Is(err, kafka.NotLeaderForPartition) ||
		errors.Is(err, kafka.RequestTimedOut) {
		return true
	}

	// 檢查網絡相關錯誤
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// 檢查常見的臨時錯誤字符串
	errStr := err.Error()
	temporaryErrors := []string{
		"connection refused",
		"broken pipe",
		"connection reset by peer",
		"no route to host",
		"network is unreachable",
		"operation timed out",
		"too many open files",
		"no buffer space",
		"connection timed out",
	}

	for _, tempErr := range temporaryErrors {
		if strings.Contains(strings.ToLower(errStr), tempErr) {
			return true
		}
	}

	return false
}

// 新增一些輔助函數來檢查特定類型的錯誤
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"broken pipe",
		"connection reset",
		"no route to host",
		"network is unreachable",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}

	return false
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	if errors.Is(err, ErrWriteTimeout) || errors.Is(err, ErrReadTimeout) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

// IsTemporaryProducerError 判斷是否為生產者的臨時錯誤
func IsTemporaryProducerError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// kafka-go 特定的臨時錯誤
	if errors.Is(err, kafka.LeaderNotAvailable) ||
		errors.Is(err, kafka.NotLeaderForPartition) ||
		errors.Is(err, kafka.RequestTimedOut) {
		return true
	}

	// 檢查錯誤字符串
	errStr := strings.ToLower(err.Error())
	temporaryErrors := []string{
		"connection refused",
		"broken pipe",
		"connection reset",
		"no route to host",
		"network is unreachable",
		"timeout",
		"leader not available",
		"not leader for partition",
	}

	for _, tempErr := range temporaryErrors {
		if strings.Contains(errStr, tempErr) {
			return true
		}
	}

	return false
}

// IsTemporaryConsumerError 判斷是否為消費者的臨時錯誤
func IsTemporaryConsumerError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// 檢查錯誤字符串
	errStr := strings.ToLower(err.Error())
	temporaryErrors := []string{
		"connection refused",
		"broken pipe",
		"connection reset",
		"no route to host",
		"network is unreachable",
		"timeout",
		"offset out of range",
		"group coordinator not available",
		"not coordinator for group",
	}

	for _, tempErr := range temporaryErrors {
		if strings.Contains(errStr, tempErr) {
			return true
		}
	}

	return false
}

// IsFatalConsumerError 判斷是否為消費者的致命錯誤
func IsFatalConsumerError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// Context 錯誤
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 消費者已關閉
	if errors.Is(err, ErrConsumerClosed) {
		return true
	}

	// 檢查錯誤字符串
	errStr := strings.ToLower(err.Error())
	fatalErrors := []string{
		"authentication failed",
		"sasl authentication failed",
		"authorization failed",
		"not authorized",
		"invalid configuration",
		"configuration error",
		"illegal generation",
		"unknown member id",
		"group id not found",
		"coordinator load in progress",
	}

	for _, fatalErr := range fatalErrors {
		if strings.Contains(errStr, fatalErr) {
			return true
		}
	}

	return false
}

// IsFatalProducerError 判斷是否為生產者的致命錯誤
func IsFatalProducerError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// Context 錯誤
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 生產者已關閉
	if errors.Is(err, ErrProducerClosed) {
		return true
	}

	// 檢查錯誤字符串
	errStr := strings.ToLower(err.Error())
	fatalErrors := []string{
		"authentication failed",
		"sasl authentication failed",
		"authorization failed",
		"not authorized",
		"invalid configuration",
		"configuration error",
		"topic authorization failed",
		"invalid topic",
		"topic not found",
	}

	for _, fatalErr := range fatalErrors {
		if strings.Contains(errStr, fatalErr) {
			return true
		}
	}

	return false
}
