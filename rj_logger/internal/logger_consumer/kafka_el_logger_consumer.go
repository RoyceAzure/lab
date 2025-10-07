package logger_consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/RoyceAzure/lab/rj_logger/pkg/elsearch"
	"github.com/segmentio/kafka-go"
)

// for kafka consumer
type KafkaElProcesser struct {
	bufferSize     int
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
func NewKafkaElProcesser(elDao elsearch.IElSearchDao, bufferSize int, commitInterval time.Duration) (*KafkaElProcesser, error) {
	return &KafkaElProcesser{
		dao:            elDao,
		bufferSize:     bufferSize,
		commitInterval: commitInterval,
	}, nil
}

// for log 採用最多處理一次的訊息模式
// 將in chan關閉視為結束訊號
// 處理完本身持有msg，送入out，就return
// 併發操作
// todo 失敗紀錄Log, 或者寫入DB
// 失敗的訊息會直接commit, 後續由DLQ處理
func (p *KafkaElProcesser) Process(ctx context.Context, in <-chan kafka.Message, out chan<- kafka.Message, dlq chan<- kafka.Message) {
	documents := make([]map[string]interface{}, 0, p.bufferSize)
	toCommits := make([]kafka.Message, 0, p.bufferSize)

	ticker := time.NewTicker(p.commitInterval)
	defer ticker.Stop()

	handlerMsg := func() {
		err := p.dao.BatchInsert(string(toCommits[0].Key), documents)
		if err != nil {
			for _, m := range toCommits {
				dlq <- m
			}
			log.Printf("Elastic logger consumer write log get err : %s", err.Error())
		}
		for _, m := range toCommits {
			out <- m
		}
	}

	for {
		select {
		case <-ticker.C:
			if len(documents) > 0 {
				handlerMsg()
				documents = documents[:0]
				toCommits = toCommits[:0]
			}
		case msg, ok := <-in:
			if !ok {
				if len(documents) > 0 {
					handlerMsg()
				}
				return
			}
			apppendMsg(msg, &documents, &toCommits, dlq)
		}
	}
}

func apppendMsg(
	msg kafka.Message,
	documents *[]map[string]interface{},
	toCommits *[]kafka.Message,
	dlq chan<- kafka.Message) {

	var doc map[string]interface{}
	if err := json.Unmarshal(msg.Value, &doc); err != nil {
		// log.Printf("Elastic logger consumer transform log content get err : %s", err.Error())
		dlq <- msg
		return
	}
	*documents = append(*documents, doc)
	*toCommits = append(*toCommits, msg)
}
