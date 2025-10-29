package errors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/segmentio/kafka-go"
)

var (
	//
	ErrInvalidateParameter = errors.New("invalidate parameter")
	// ErrClientClosed 表示消費者或生產者已關閉
	ErrClientClosed = errors.New("consumer or producer is closed, please call Start() first")
	// ErrWriteTimeout 表示寫入超時
	ErrWriteTimeout = errors.New("write timeout")
	// ErrReadTimeout 表示讀取超時
	ErrReadTimeout = errors.New("read timeout")
	// ErrTopicNotFound 表示主題不存在
	ErrTopicNotFound = errors.New("topic not found")
	// ErrAuthenticationFailed 表示認證失敗
	ErrAuthenticationFailed = errors.New("authentication failed")
	// ErrPermissionDenied 表示權限不足
	ErrPermissionDenied = errors.New("permission denied")

	// ErrConsumerAlreadyRunning 表示消費者已經在運行
	ErrConsumerAlreadyRunning = errors.New("consumer is already running")
)

// KafkaError 代表 Kafka 操作錯誤
type KafkaError struct {
	Operation string
	Topic     string
	Err       error
}

func (e *KafkaError) Error() string {
	return fmt.Sprintf("kafka operation %s on topic %s failed: %v", e.Operation, e.Topic, e.Err)
}

// NewKafkaError 創建新的 KafkaError
func NewKafkaError(operation, topic string, err error) error {
	return &KafkaError{
		Operation: operation,
		Topic:     topic,
		Err:       err,
	}
}

// IsConnectionError 判斷是否為需要重置連接的錯誤
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// 檢查具體的網路錯誤
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// 檢查是否為系統錯誤
		var sysErr syscall.Errno
		if errors.As(netErr.Err, &sysErr) {
			switch sysErr {
			case syscall.ECONNREFUSED, // 連接被拒絕
				syscall.ECONNRESET,   // 連接被重置
				syscall.ECONNABORTED, // 連接中斷
				syscall.ENETUNREACH,  // 網路不可達
				syscall.ENETRESET,    // 網路重置
				syscall.ETIMEDOUT:    // 連接超時
				return true
			}
		}
		// 一般網路錯誤也需要重置連接
		return true
	}

	// 檢查 DNS 錯誤
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// 檢查特定的錯誤字符串
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "network is unreachable")
}

// IsTemporaryError 判斷是否為可重試的臨時錯誤
// return
// true 表示可重試, false 表示不可重試
func IsTemporaryError(err error) bool {
	if err == nil {
		return true
	}

	// 如果是需要重置連接的錯誤，不在這裡處理
	if IsConnectionError(err) {
		return false
	}

	// 如果是致命錯誤，不可重試
	if IsFatalError(err) {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// 檢查 Kafka 特定的可重試錯誤
	if errors.Is(err, kafka.LeaderNotAvailable) ||
		errors.Is(err, kafka.NotLeaderForPartition) ||
		errors.Is(err, kafka.RequestTimedOut) ||
		errors.Is(err, kafka.RebalanceInProgress) {
		return true
	}

	// 檢查上下文錯誤
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 檢查特定的錯誤字符串
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "no buffer space") ||
		strings.Contains(errStr, "too many open files") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "retriable") ||
		strings.Contains(errStr, "rebalance in progress") ||
		strings.Contains(errStr, "coordinator load in progress")
}

// IsFatalError 判斷是否為致命錯誤（不可重試）
func IsFatalError(err error) bool {
	if err == nil {
		return false
	}

	// 解包 KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		err = kafkaErr.Err
	}

	// 檢查 Kafka 特定的致命錯誤
	if errors.Is(err, kafka.TopicAuthorizationFailed) ||
		errors.Is(err, kafka.GroupAuthorizationFailed) ||
		errors.Is(err, kafka.ClusterAuthorizationFailed) {
		return true
	}

	// 檢查特定的錯誤字符串
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "authentication failed") ||
		strings.Contains(errStr, "authorization failed") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "topic not found") ||
		strings.Contains(errStr, "invalid topic") ||
		strings.Contains(errStr, "group id not found") ||
		strings.Contains(errStr, "illegal generation") ||
		strings.Contains(errStr, "unknown member id")
}
