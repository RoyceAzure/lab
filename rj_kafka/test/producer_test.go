package test

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func TestProducerConnection(t *testing.T) {
	// 配置 writer
	w := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9092", "localhost:9093", "localhost:9094"),
		Topic:        "test-topic",
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: time.Second,
		// 設置較短的超時時間以快速發現問題
		WriteTimeout: 5 * time.Second,
		// 設置重試
		MaxAttempts: 3,
	}
	defer w.Close()

	// 創建一條測試消息
	msg := kafka.Message{
		Key:   []byte("test-key"),
		Value: []byte("test-message"),
	}

	// 嘗試發送消息
	ctx := context.Background()
	err := w.WriteMessages(ctx, msg)

	// 驗證結果
	assert.NoError(t, err, "Should be able to send message")
}
