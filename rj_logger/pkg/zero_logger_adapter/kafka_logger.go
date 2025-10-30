package logger

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	lab_config "github.com/RoyceAzure/lab/rj_kafka/pkg/config"
	"github.com/RoyceAzure/lab/rj_kafka/pkg/model"
	"github.com/RoyceAzure/lab/rj_kafka/pkg/producer"
	"github.com/segmentio/kafka-go"
)

// for consumer  不需要接收來自zerolog的資訊
type KafkaLogger struct {
	cfg   *lab_config.Config
	w     producer.Producer
	logId atomic.Int64
}

func NewKafkaLogger(cfg *lab_config.Config) (*KafkaLogger, error) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     cfg.GetBalancer(),
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.CommitInterval + time.Millisecond*100,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		// 設置較短的超時時間以快速發現問題
		WriteTimeout: 5 * time.Second,
		// 設置重試
		MaxAttempts: 3,
	}

	p, err := producer.NewConcurrencekafkaProducer(w, *cfg)
	if err != nil {
		return nil, err
	}

	return &KafkaLogger{
		cfg: cfg,
		w:   p,
	}, nil
}

func (kw *KafkaLogger) Write(p []byte) (n int, err error) {
	if kw == nil {
		return 0, fmt.Errorf("file logger is not init")
	}

	kw.logId.Add(int64(1))
	kbuf := make([]byte, 0, 8)
	binary.BigEndian.PutUint64(kbuf, uint64(kw.logId.Load()))
	_, err = kw.w.Produce(context.Background(), []model.Message{
		{
			Key:   kbuf,
			Value: p,
		},
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (fw *KafkaLogger) Close() error {
	return fw.w.Close(time.Second * 15)
}
