package test

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/balancer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
	}

	// 2. 創建 Producer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	// 3. 創建 Consumer
	c, err := consumer.New(cfg)
	assert.NoError(t, err)

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
	msgChan, errChan, err := c.Consume()
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
			case err := <-errChan:
				t.Logf("Error consuming message: %v", err)
			case <-ctx.Done():
				t.Logf("Context cancelled while waiting for messages")
				return
			}
		}
	}()

	// 等待消息處理完成或超時
	select {
	case <-done:
		c.Close()
		p.Close()
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
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
	}

	// 創建 Producer 和 Consumer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	c, err := consumer.New(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 啟動 consumer
	msgChan, errChan, err := c.Consume()
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
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	done := make(chan struct{})

	// 創建第一個消費者
	consumer1, err := consumer.New(cfg)
	assert.NoError(t, err)

	// 啟動第一個消費者
	msgChan1, errChan1, err := consumer1.Consume()
	assert.NoError(t, err, "Should consume messages")

	// 發送消息的 goroutine
	wg.Add(1)
	totalExpectedMessages := 0

	p, err := producer.New(cfg)
	assert.NoError(t, err)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
				// 發送新消息
				msg := message.Message{
					Key:   []byte(fmt.Sprintf("key-%d", totalExpectedMessages)),
					Value: []byte(fmt.Sprintf("value-%d", totalExpectedMessages)),
				}
				err = p.Produce(ctx, []message.Message{msg})
				assert.NoError(t, err)
				totalExpectedMessages++
				time.Sleep(2 * time.Second) // 每隔2秒發送一次
			}
		}
	}()

	// 等待一段時間後創建第二個消費者
	time.Sleep(10 * time.Second)

	consumer2, err := consumer.New(cfg)
	assert.NoError(t, err)

	// 啟動第二個消費者
	msgChan2, errChan2, err := consumer2.Consume()
	assert.NoError(t, err, "Should consume messages")

	// 追蹤兩個消費者接收到的消息
	consumer1Messages := make(map[string]bool)
	consumer2Messages := make(map[string]bool)

	// 使用 WaitGroup 來等待消息處理完成
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case msg := <-msgChan1:
				consumer1Messages[string(msg.Key)] = true
			case msg := <-msgChan2:
				consumer2Messages[string(msg.Key)] = true
			case err := <-errChan1:
				t.Logf("Consumer 1 error: %v", err)
			case err := <-errChan2:
				t.Logf("Consumer 2 error: %v", err)

			}
		}
	}()

	// 模擬外部信號來結束測試
	go func() {
		time.Sleep(30 * time.Second) // 模擬一些處理時間
		t.Logf("結束發送，關閉channle，準備驗證測試結果")
		close(done) // 發送結束信號
		consumer1.Close()
		consumer2.Close()
		p.Close()
	}()

	wg.Wait()

	totalReceivedMessages := len(consumer1Messages) + len(consumer2Messages)

	// 驗證消息分配
	t.Logf("Consumer 1 received %d messages", len(consumer1Messages))
	t.Logf("Consumer 2 received %d messages", len(consumer2Messages))
	t.Logf("totalReceivedMessages: %d", totalReceivedMessages)
	t.Logf("totalExpectedMessages: %d", totalExpectedMessages)
	// 確保所有消息都被處理
	require.Equal(t, totalExpectedMessages, totalReceivedMessages)
}

func TestProducerConsumerWithProductIDBalancer(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	const testDuration = 60 * time.Second

	// 定義平衡函數  與 balancer.go 中的 Balance 函數相同
	balance := func(msg message.Message, numPartitions int) int {
		productID := string(msg.Key)
		if productID == "" {
			return 0 // 或者用隨機分配
		}

		hash := fnv.New32a()
		hash.Write([]byte(productID))

		return int(hash.Sum32() % uint32(numPartitions))
	}

	// 配置
	cfg := &config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Topic:          testClusterConfig.Topics[0].Name, // 使用動態創建的 topic
		ConsumerGroup:  fmt.Sprintf("test-product-id-group-%s-%d", t.Name(), time.Now().UnixNano()),
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   1,
		CommitInterval: time.Second,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
	}

	// 使用 ProductIDBalancer
	balancer := balancer.NewProductIDBalancer(testClusterConfig.Topics[0].Partitions)

	// 設置 Producer 使用 balancer
	cfg.Balancer = balancer

	// 創建 Producer 和 Consumer
	p, err := producer.New(cfg)
	assert.NoError(t, err)
	defer p.Close()

	// 建立n個消費者
	consumers := make([]consumer.Consumer, testClusterConfig.Topics[0].Partitions)
	for i := 0; i < testClusterConfig.Topics[0].Partitions; i++ {
		c, err := consumer.New(cfg)
		assert.NoError(t, err)
		consumers[i] = c
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 準備測試數據
	testMsgNum := 12
	testMessages := make([]message.Message, testMsgNum)
	for i := 0; i < testMsgNum; i++ {
		testMessages[i] = message.Message{
			Key:   []byte(fmt.Sprintf("product-%d", (i+1)%3)),
			Value: []byte(fmt.Sprintf("value%d", i+1)),
		}
	}

	//先分配消息到各個分區，驗證用
	sendMessages := make([][]message.Message, 6) // 有6個分區
	for i := 0; i < testMsgNum; i++ {
		msg := testMessages[i]
		partition := balance(msg, 6)
		sendMessages[partition] = append(sendMessages[partition], msg)
	}

	receivedMessages := make([][]message.Message, 6)
	messageCount := 0
	done := make(chan struct{})

	// 消費消息
	msgChans := make([]<-chan message.Message, 6)
	errChans := make([]<-chan error, 6)
	for i := 0; i < 6; i++ {
		msgChans[i], errChans[i], err = consumers[i].Consume()
		assert.NoError(t, err, "Should consume messages")
	}
	go func() {
		for {
			select {
			case msg := <-msgChans[0]:
				t.Logf("收到分區0的消息: %v", msg)
				receivedMessages[0] = append(receivedMessages[0], msg)
				messageCount++
			case msg := <-msgChans[1]:
				t.Logf("收到分區1的消息: %v", msg)
				receivedMessages[1] = append(receivedMessages[1], msg)
				messageCount++
			case msg := <-msgChans[2]:
				t.Logf("收到分區2的消息: %v", msg)
				receivedMessages[2] = append(receivedMessages[2], msg)
				messageCount++
			case msg := <-msgChans[3]:
				t.Logf("收到分區3的消息: %v", msg)
				receivedMessages[3] = append(receivedMessages[3], msg)
				messageCount++
			case msg := <-msgChans[4]:
				t.Logf("收到分區4的消息: %v", msg)
				receivedMessages[4] = append(receivedMessages[4], msg)
				messageCount++
			case msg := <-msgChans[5]:
				t.Logf("收到分區5的消息: %v", msg)
				receivedMessages[5] = append(receivedMessages[5], msg)
				messageCount++
				// 加入錯誤處理
			case err := <-errChans[0]:
				t.Logf("分區0錯誤: %v", err)
			case err := <-errChans[1]:
				t.Logf("分區1錯誤: %v", err)
			case err := <-errChans[2]:
				t.Logf("分區2錯誤: %v", err)
			case err := <-errChans[3]:
				t.Logf("分區3錯誤: %v", err)
			case err := <-errChans[4]:
				t.Logf("分區4錯誤: %v", err)
			case err := <-errChans[5]:
				t.Logf("分區5錯誤: %v", err)
			case <-done:
				t.Logf("接收到 producer 結束發送的信號，結束消費")
				return
			}
		}
	}()

	// // 發送消息
	// go func() {
	// 	defer close(done)
	// 	for _, msg := range testMessages {
	// 		time.Sleep(1 * time.Second)
	// 		err = p.Produce(ctx, []message.Message{msg})
	// 		assert.NoError(t, err)
	// 	}
	// 	// 等待10秒，讓消費者有時間消費
	// 	time.Sleep(10 * time.Second)
	// }()

	// 發送消息
	go func() {
		defer close(done)
		err = p.Produce(ctx, testMessages)
		assert.NoError(t, err)
		// 等待10秒，讓消費者有時間消費
		time.Sleep(10 * time.Second)
	}()

	// 等待消息處理完成或超時
	select {
	case <-done:
		time.Sleep(2 * time.Second)

		// 先取消上下文
		cancel()

		// 再關閉 consumers
		for _, c := range consumers {
			c.Close()
		}
		p.Close()
	case <-time.After(testDuration):
		t.Fatal("Test timeout waiting for messages")
	}

	// 驗證接收到的消息順序
	printMessages := func(t *testing.T, received [][]message.Message, sent [][]message.Message) {
		t.Log("接收到的消息分佈：")
		for partitionID, messages := range received {
			if len(messages) > 0 {
				t.Logf("分區 %d 收到的消息:", partitionID)
				for _, msg := range messages {
					t.Logf("    Key: %s, Value: %s", string(msg.Key), string(msg.Value))
				}
			}
		}

		t.Log("\n預期的消息分佈：")
		for partitionID, messages := range sent {
			if len(messages) > 0 {
				t.Logf("分區 %d 應該收到的消息:", partitionID)
				for _, msg := range messages {
					t.Logf("    Key: %s, Value: %s", string(msg.Key), string(msg.Value))
				}
			}
		}
	}
	printMessages(t, receivedMessages, sendMessages)
}
