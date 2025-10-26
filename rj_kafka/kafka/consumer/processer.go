package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RoyceAzure/rj/infra/elsearch"
	"github.com/segmentio/kafka-go"
)

type Processer interface {
	Process(ctx context.Context, msgs []kafka.Message) error //必須是無狀態
}

// for kafka consumer
type KafkaElProcesser struct {
	dao            elsearch.IElSearchDao
	commitInterval time.Duration
}

// // 預設的Kafka logger消費者配置
// func GetDefaultConfigForLogger() *config.Config {
// 	return &config.Config{
// 		ConsumerMinBytes: 1024 * 1024,      //1M
// 		ConsumerMaxBytes: 1024 * 1024 * 10, //10M
// 		ConsumerMaxWait:  time.Millisecond * 100,
// 		CommitInterval:   time.Second * 5,
// 		RetryBackoffMax:  1 * time.Second,
// 		RetryBackoffMin:  100 * time.Millisecond,
// 	}
// }

// bufferSize int 最大batch size
// Process操作將會是併發，所以batch size應設為 需求數/併發數
func NewKafkaElProcesser(elDao elsearch.IElSearchDao, commitInterval time.Duration) (*KafkaElProcesser, error) {
	return &KafkaElProcesser{
		dao:            elDao,
		commitInterval: commitInterval,
	}, nil
}

// 批次處理，單一失敗則全部失敗
func (p *KafkaElProcesser) Process(ctx context.Context, msgs []kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	documents := make([]map[string]interface{}, 0)

	for _, msg := range msgs {
		doc := make(map[string]any)
		if err := json.Unmarshal(msg.Value, &doc); err != nil {
			return err
		}
		documents = append(documents, doc)
	}
	return p.dao.BatchInsert(string(msgs[0].Key), documents)
}
