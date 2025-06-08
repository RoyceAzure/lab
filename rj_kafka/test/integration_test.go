package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/stretchr/testify/assert"
)

func TestProducerConsumerIntegration(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 1. 設置測試配置
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name, // 使用動態創建的 topic
		ConsumerGroup:  fmt.Sprintf("test-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   1,
		CommitInterval: time.Second,
	}

	// 2. 創建 Producer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	// 3. 創建 Consumer
	c, err := consumer.New(cfg)
	assert.NoError(t, err)
	defer func() {
		// 確保在測試結束時正確關閉 consumer
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.Close()
		assert.NoError(t, err)
		<-closeCtx.Done()
	}()

	// 4. 準備測試數據
	testMessages := []message.Message{
		{
			Key:   []byte("key1"),
			Value: []byte("value1"),
			Headers: []message.Header{
				{
					Key:   "header1",
					Value: []byte("value1"),
				},
			},
		},
		{
			Key:   []byte("key2"),
			Value: []byte("value2"),
			Headers: []message.Header{
				{
					Key:   "header2",
					Value: []byte("value2"),
				},
			},
		},
	}

	// 5. 設置 context 和超時
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 6. 啟動 consumer
	msgChan, errChan := c.Consume(ctx)

	// 7. 發送消息
	err = p.Produce(ctx, testMessages)
	assert.NoError(t, err)

	// 8. 接收和驗證消息
	receivedMessages := make([]message.Message, 0)
	messageCount := 0
	done := make(chan struct{})

	go func() {
		defer close(done)
		for messageCount < len(testMessages) {
			select {
			case msg := <-msgChan:
				receivedMessages = append(receivedMessages, msg)
				messageCount++
				// 立即提交消息
				err := c.CommitMessages(ctx, msg)
				if err != nil {
					t.Errorf("Failed to commit message: %v", err)
					return
				}
			case err := <-errChan:
				t.Errorf("Error consuming message: %v", err)
				return
			case <-ctx.Done():
				t.Error("Context cancelled while waiting for messages")
				return
			}
		}
	}()

	// 等待消息處理完成或超時
	select {
	case <-done:
		// 成功完成
	case <-time.After(10 * time.Second):
		t.Fatal("Test timeout waiting for messages")
	}

	// 9. 驗證接收到的消息
	assert.Equal(t, len(testMessages), len(receivedMessages))
	for i, sent := range testMessages {
		assert.Equal(t, string(sent.Key), string(receivedMessages[i].Key))
		assert.Equal(t, string(sent.Value), string(receivedMessages[i].Value))
		assert.Equal(t, len(sent.Headers), len(receivedMessages[i].Headers))
		for j, header := range sent.Headers {
			assert.Equal(t, header.Key, receivedMessages[i].Headers[j].Key)
			assert.Equal(t, string(header.Value), string(receivedMessages[i].Headers[j].Value))
		}
	}
}

func TestProducerConsumerWithLargeMessages(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 配置
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name, // 使用動態創建的 topic
		ConsumerGroup:  fmt.Sprintf("test-large-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      100,
		RequiredAcks:   1,
		CommitInterval: time.Second,
	}

	// 創建 Producer 和 Consumer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	c, err := consumer.New(cfg)
	assert.NoError(t, err)
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.Close()
		assert.NoError(t, err)
		<-closeCtx.Done()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 啟動 consumer
	msgChan, errChan := c.Consume(ctx)

	// 生成和發送大量測試數據
	producerMessageCount := 10000
	messages := make([]message.Message, 0, producerMessageCount)
	producerDone := make(chan struct{})

	go func() {
		defer close(producerDone)
		// 先準備所有消息
		for i := 0; i < producerMessageCount; i++ {
			msg := message.Message{
				Key:   []byte(fmt.Sprintf("key-%d", i)),
				Value: []byte(fmt.Sprintf("value-%d", i)),
				Headers: []message.Header{
					{
						Key:   "index",
						Value: []byte(fmt.Sprintf("%d", i)),
					},
				},
			}
			messages = append(messages, msg)
		}

		// 一次性發送所有消息
		if err := p.Produce(ctx, messages); err != nil {
			t.Errorf("Failed to produce messages: %v", err)
			return
		}
	}()

	// 接收和驗證消息
	receivedMessages := make([]message.Message, 0, producerMessageCount)
	messageCount := 0
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for messageCount < producerMessageCount {
			select {
			case msg := <-msgChan:
				receivedMessages = append(receivedMessages, msg)
				messageCount++
				// 立即提交消息
				if err := c.CommitMessages(ctx, msg); err != nil {
					t.Errorf("Failed to commit message: %v", err)
					return
				}
			case err := <-errChan:
				t.Errorf("Error consuming message: %v", err)
				return
			case <-ctx.Done():
				t.Error("Context cancelled while waiting for messages")
				return
			}
		}
	}()

	// 等待生產者和消費者完成
	select {
	case <-producerDone:
		t.Log("Producer done")
		select {
		case <-consumerDone:
			t.Log("Consumer done")
			// 成功完成
		case <-time.After(60 * time.Second):
			t.Fatal("Consumer timeout")
		}
	case <-time.After(60 * time.Second):
		t.Fatal("Producer timeout")
	}

	// 驗證消息數量
	assert.Equal(t, producerMessageCount, len(receivedMessages))

	// 驗證消息內容
	messageMap := make(map[string]bool)
	for _, msg := range receivedMessages {
		messageMap[string(msg.Key)] = true
	}

	for _, sent := range messages {
		assert.True(t, messageMap[string(sent.Key)], "Message not received: %s", string(sent.Key))
	}
}

// 測試一筆一筆發送 有夠慢	完全無法使用  使用上使用batch發送
func TestProducerConsumerWithLargeMessagesOneByOne(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 配置
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name, // 使用動態創建的 topic
		ConsumerGroup:  fmt.Sprintf("test-large-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      100,
		RequiredAcks:   1,
		CommitInterval: time.Second,
	}

	// 創建 Producer 和 Consumer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	c, err := consumer.New(cfg)
	assert.NoError(t, err)
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.Close()
		assert.NoError(t, err)
		<-closeCtx.Done()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 啟動 consumer
	msgChan, errChan := c.Consume(ctx)

	// 生成和發送大量測試數據
	producerMessageCount := 100
	messages := make([]message.Message, 0, producerMessageCount)
	producerDone := make(chan struct{})

	go func() {
		defer close(producerDone)
		// 先準備所有消息
		for i := 0; i < producerMessageCount; i++ {
			msg := message.Message{
				Key:   []byte(fmt.Sprintf("key-%d", i)),
				Value: []byte(fmt.Sprintf("value-%d", i)),
				Headers: []message.Header{
					{
						Key:   "index",
						Value: []byte(fmt.Sprintf("%d", i)),
					},
				},
			}
			messages = append(messages, msg)
			// 一次性發送一筆
			writeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			if err := p.Produce(writeCtx, []message.Message{msg}); err != nil {
				t.Errorf("Failed to produce messages: %v", err)
				cancel()
				return
			}
			cancel()
		}
	}()

	// 接收和驗證消息
	receivedMessages := make([]message.Message, 0, producerMessageCount)
	messageCount := 0
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for messageCount < producerMessageCount {
			select {
			case msg := <-msgChan:
				receivedMessages = append(receivedMessages, msg)
				messageCount++
				// 立即提交消息
				if err := c.CommitMessages(ctx, msg); err != nil {
					t.Errorf("Failed to commit message: %v", err)
					return
				}
			case err := <-errChan:
				t.Errorf("Error consuming message: %v", err)
				return
			case <-ctx.Done():
				t.Error("Context cancelled while waiting for messages")
				return
			}
		}
	}()

	// 等待生產者和消費者完成
	select {
	case <-producerDone:
		t.Log("Producer done")
		select {
		case <-consumerDone:
			t.Log("Consumer done")
			// 成功完成
		case <-time.After(60 * time.Second):
			t.Fatal("Consumer timeout")
		}
	case <-time.After(60 * time.Second):
		t.Fatal("Producer timeout")
	}

	// 驗證消息數量
	assert.Equal(t, producerMessageCount, len(receivedMessages))

	// 驗證消息內容
	messageMap := make(map[string]bool)
	for _, msg := range receivedMessages {
		messageMap[string(msg.Key)] = true
	}

	for _, sent := range messages {
		assert.True(t, messageMap[string(sent.Key)], "Message not received: %s", string(sent.Key))
	}
}
