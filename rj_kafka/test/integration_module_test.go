package test

import (
	"context"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/stretchr/testify/assert"
)

func TestProducerConsumerModule(t *testing.T) {
	// 基本配置
	cfg := &config.Config{
		Brokers:        []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		Topic:          "test-large-topic",
		ConsumerGroup:  "test-module-group",
		BatchSize:      1,
		BatchTimeout:   time.Second,
		RetryAttempts:  3,
		RequiredAcks:   1, // 改為只需要 leader 確認
		CommitInterval: time.Second,
		// 添加消費者配置
		ConsumerMinBytes: 1,
		ConsumerMaxBytes: 10e6,
		ConsumerMaxWait:  time.Second,
	}

	// 創建 producer
	p, err := producer.New(cfg)
	assert.NoError(t, err, "Should create producer")
	defer p.Close()

	// 創建測試消息
	testMsgs := []message.Message{
		{Key: []byte("key1"), Value: []byte("test message 1")},
		{Key: []byte("key2"), Value: []byte("test message 2")},
		{Key: []byte("key3"), Value: []byte("test message 3")},
	}

	// 發送消息
	ctx := context.Background()
	err = p.Produce(ctx, testMsgs)
	assert.NoError(t, err, "Should produce messages")

	// 創建 consumer
	c, err := consumer.New(cfg)
	assert.NoError(t, err, "Should create consumer")
	defer c.Close()

	// 設置讀取超時
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 開始消費消息
	msgChan, errChan, err := c.Consume(ctxTimeout)
	assert.NoError(t, err, "Should consume messages")

	// 讀取並驗證消息
	receivedMsgs := make([]message.Message, 0)
	expectedCount := len(testMsgs)

	for len(receivedMsgs) < expectedCount {
		select {
		case msg := <-msgChan:
			t.Logf("Received message: Key=%s, Value=%s", string(msg.Key), string(msg.Value))
			receivedMsgs = append(receivedMsgs, msg)

			// 提交消息
			err := c.CommitMessages(ctx, msg)
			assert.NoError(t, err, "Should commit message")

		case err := <-errChan:
			if consumerErr, ok := err.(*consumer.ConsumerError); ok && !consumerErr.Fatal {
				t.Logf("Received non-fatal error: %v", err)
				continue
			}
			t.Fatalf("Received fatal error: %v", err)

		case <-ctxTimeout.Done():
			t.Fatalf("Timeout waiting for messages. Received %d of %d messages",
				len(receivedMsgs), expectedCount)
		}
	}

	// 驗證收到的消息數量
	assert.Equal(t, expectedCount, len(receivedMsgs),
		"Should receive all messages")

	// 驗證消息內容
	for i, msg := range receivedMsgs {
		assert.Equal(t, string(testMsgs[i].Key), string(msg.Key),
			"Message key should match")
		assert.Equal(t, string(testMsgs[i].Value), string(msg.Value),
			"Message value should match")
	}
}
