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
	msgChan, errChan, err := c.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

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
	msgChan, errChan, err := c.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

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
	msgChan, errChan, err := c.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

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

// TestBrokerFailureRecovery 測試 broker 離線和重連的情況
func TestBrokerFailureRecovery(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 配置較短的重試時間
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name,
		ConsumerGroup:  fmt.Sprintf("test-failure-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  5,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   -1, // 等待所有副本確認
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
	msgChan, errChan, err := c.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

	// 準備測試消息
	messages := []message.Message{
		{
			Key:   []byte("pre-failure"),
			Value: []byte("message before broker failure"),
		},
	}

	// 發送第一條消息
	err = p.Produce(ctx, messages)
	assert.NoError(t, err)

	// 等待第一條消息被消費
	receivedPreFailure := false
	timeout := time.After(10 * time.Second)

	for !receivedPreFailure {
		select {
		case msg := <-msgChan:
			if string(msg.Key) == "pre-failure" {
				receivedPreFailure = true
				err := c.CommitMessages(ctx, msg)
				assert.NoError(t, err)
			}
		case err := <-errChan:
			t.Errorf("Error consuming message: %v", err)
			return
		case <-timeout:
			t.Fatal("Timeout waiting for pre-failure message")
			return
		}
	}

	// 模擬 broker 故障：停止其中一個 broker
	// 注意：這裡需要實際執行 docker stop 命令
	t.Log("Simulating broker failure...")
	// TODO: 在這裡添加實際停止 broker 的程式碼

	// 嘗試在 broker 故障期間發送消息
	failureMessages := []message.Message{
		{
			Key:   []byte("during-failure"),
			Value: []byte("message during broker failure"),
		},
	}

	// 這裡預期會有錯誤或重試
	err = p.Produce(ctx, failureMessages)
	t.Logf("Produce during failure result: %v", err)

	// 等待一段時間
	time.Sleep(5 * time.Second)

	// 重啟 broker
	t.Log("Restarting broker...")
	// TODO: 在這裡添加實際重啟 broker 的程式碼

	// 等待 broker 重新上線
	time.Sleep(10 * time.Second)

	// 發送恢復後的消息
	recoveryMessages := []message.Message{
		{
			Key:   []byte("post-recovery"),
			Value: []byte("message after broker recovery"),
		},
	}

	err = p.Produce(ctx, recoveryMessages)
	assert.NoError(t, err)

	// 驗證恢復後的消息是否被正確接收
	receivedPostRecovery := false
	timeout = time.After(10 * time.Second)

	for !receivedPostRecovery {
		select {
		case msg := <-msgChan:
			if string(msg.Key) == "post-recovery" {
				receivedPostRecovery = true
				err := c.CommitMessages(ctx, msg)
				assert.NoError(t, err)
			}
		case err := <-errChan:
			t.Errorf("Error consuming message: %v", err)
			return
		case <-timeout:
			t.Fatal("Timeout waiting for post-recovery message")
			return
		}
	}
}

// TestProducerRetryMechanism 測試生產者重試機制
func TestProducerRetryMechanism(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 配置較短的重試間隔
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name,
		ConsumerGroup:  fmt.Sprintf("test-retry-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   -1, // 等待所有副本確認
		CommitInterval: time.Second,
	}

	// 創建 Producer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	// 創建一個上下文，設定較短的超時時間
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 準備測試消息
	messages := []message.Message{
		{
			Key:   []byte("retry-test"),
			Value: []byte("message for retry testing"),
			Headers: []message.Header{
				{
					Key:   "attempt",
					Value: []byte("1"),
				},
			},
		},
	}

	// 在發送消息之前模擬網路問題或其他故障
	// TODO: 在這裡添加模擬網路問題的程式碼

	// 發送消息並觀察重試行為
	err = p.Produce(ctx, messages)
	if err != nil {
		t.Logf("Expected error occurred: %v", err)
	}

	// 恢復正常連接
	// TODO: 在這裡添加恢復連接的程式碼

	// 再次發送消息，確認系統已恢復正常
	err = p.Produce(ctx, messages)
	assert.NoError(t, err)
}

// TestConsumerRebalancing 測試消費者重平衡
func TestConsumerRebalancing(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name,
		ConsumerGroup:  fmt.Sprintf("test-rebalance-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   1,
		CommitInterval: time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 創建第一個消費者
	consumer1, err := consumer.New(cfg)
	assert.NoError(t, err)
	defer consumer1.Close()

	// 啟動第一個消費者
	msgChan1, errChan1, err := consumer1.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

	// 發送一些初始消息
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	initialMessages := make([]message.Message, 10)
	for i := 0; i < 10; i++ {
		initialMessages[i] = message.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: []byte(fmt.Sprintf("value-%d", i)),
		}
	}

	err = p.Produce(ctx, initialMessages)
	assert.NoError(t, err)

	// 等待一些消息被第一個消費者處理
	time.Sleep(5 * time.Second)

	// 創建第二個消費者（使用相同的消費者組）
	consumer2, err := consumer.New(cfg)
	assert.NoError(t, err)
	defer consumer2.Close()

	// 啟動第二個消費者
	msgChan2, errChan2, err := consumer2.Consume(ctx)
	assert.NoError(t, err, "Should consume messages")

	// 發送更多消息
	additionalMessages := make([]message.Message, 10)
	for i := 0; i < 10; i++ {
		additionalMessages[i] = message.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i+10)),
			Value: []byte(fmt.Sprintf("value-%d", i+10)),
		}
	}

	err = p.Produce(ctx, additionalMessages)
	assert.NoError(t, err)

	// 追蹤兩個消費者接收到的消息
	consumer1Messages := make(map[string]bool)
	consumer2Messages := make(map[string]bool)
	messageCount := 0
	totalExpectedMessages := 20

	// 設定超時
	timeout := time.After(30 * time.Second)

	// 收集兩個消費者的消息
	for messageCount < totalExpectedMessages {
		select {
		case msg := <-msgChan1:
			consumer1Messages[string(msg.Key)] = true
			messageCount++
			err := consumer1.CommitMessages(ctx, msg)
			assert.NoError(t, err)
		case msg := <-msgChan2:
			consumer2Messages[string(msg.Key)] = true
			messageCount++
			err := consumer2.CommitMessages(ctx, msg)
			assert.NoError(t, err)
		case err := <-errChan1:
			t.Errorf("Consumer 1 error: %v", err)
			return
		case err := <-errChan2:
			t.Errorf("Consumer 2 error: %v", err)
			return
		case <-timeout:
			t.Fatal("Test timeout")
			return
		}
	}

	// 驗證消息分配
	t.Logf("Consumer 1 received %d messages", len(consumer1Messages))
	t.Logf("Consumer 2 received %d messages", len(consumer2Messages))

	// 確保所有消息都被處理
	assert.Equal(t, totalExpectedMessages, len(consumer1Messages)+len(consumer2Messages))

	// 關閉第二個消費者，觀察重平衡
	err = consumer2.Close()
	assert.NoError(t, err)

	// 發送更多消息，確認第一個消費者接收所有消息
	finalMessages := make([]message.Message, 5)
	for i := 0; i < 5; i++ {
		finalMessages[i] = message.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i+20)),
			Value: []byte(fmt.Sprintf("value-%d", i+20)),
		}
	}

	err = p.Produce(ctx, finalMessages)
	assert.NoError(t, err)

	// 驗證第一個消費者接收到所有新消息
	finalMessageCount := 0
	timeout = time.After(10 * time.Second)

	for finalMessageCount < 5 {
		select {
		case msg := <-msgChan1:
			finalMessageCount++
			err := consumer1.CommitMessages(ctx, msg)
			assert.NoError(t, err)
		case err := <-errChan1:
			t.Errorf("Consumer 1 error after rebalance: %v", err)
			return
		case <-timeout:
			t.Fatal("Timeout waiting for final messages")
			return
		}
	}
}
