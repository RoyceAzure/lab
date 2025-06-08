package test

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func TestConsumerConnection(t *testing.T) {
	// 配置 reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		Topic:   "test-large-topic",
		GroupID: "test-consumer-group",
		// 從第一條消息開始讀取
		StartOffset: kafka.FirstOffset,
		// 設置較長的超時時間
		MaxWait: 3 * time.Second,
	})
	defer r.Close()

	// 設置讀取超時的 context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 讀取所有消息
	msgCount := 0
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			// 如果是超時錯誤，表示沒有更多消息了
			if err == context.DeadlineExceeded {
				t.Logf("No more messages available, total messages read: %d", msgCount)
				break
			}
			// 其他錯誤則表示連接問題
			assert.NoError(t, err, "Should be able to connect and read messages")
			return
		}

		// 輸出消息內容
		t.Logf("Message %d received: Partition=%d, Offset=%d, Key=%s, Value=%s",
			msgCount+1, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))
		msgCount++
	}

	// 驗證我們讀取到了所有消息
	assert.Equal(t, 3, msgCount, "Should read all 3 messages")
}
