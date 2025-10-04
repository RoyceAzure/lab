package producer

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	ka_err "github.com/RoyceAzure/lab/rj_kafka/kafka/errors"
)

// Producer interface defines the methods that a Kafka producer must implement
// 生產者會寫入到固定topic，由config.Config設置
type Producer interface {
	// Produce sends messages to Kafka
	Produce(ctx context.Context, msgs []kafka.Message) error
	// Close closes the producer
	Close(time.Duration) error
}

// 同步模式，會block到所有消息都寫入
// 希望是併發安全，所有Logger調用同一個kafkaProducer
type kafkaProducer struct {
	writer     *kafka.Writer
	buffer     []kafka.Message
	receiverCh chan kafka.Message
	cfg        *config.Config
	closed     atomic.Bool // 避免重複呼叫Close() 旗標
	innerClose chan struct{}
}

// 目前默認是同步模式，會block到所有消息都寫入
func New(cfg *config.Config) (Producer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     cfg.Balancer,
		Topic:        cfg.Topic,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxAttempts:  cfg.RetryAttempts,
		Async:        cfg.Async,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),

		// 錯誤處理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("kafka producer error: "+msg, args...)
		}),
	}

	ka := &kafkaProducer{
		writer:     writer,
		cfg:        cfg,
		buffer:     make([]kafka.Message, 0, cfg.BatchSize+100),
		receiverCh: make(chan kafka.Message, cfg.BatchSize),
		innerClose: make(chan struct{}),
	}
	go ka.produce()

	return ka, nil
}

// Produce implements the Producer interface
// 非同步，若buffer已滿，將丟棄訊息
func (p *kafkaProducer) Produce(ctx context.Context, msgs []kafka.Message) error {
	if p.closed.Load() {
		return ka_err.ErrClientClosed
	}

	select {
	case <-p.innerClose:
		//內部消費程序意外結束
		p.Close(time.Second * 3)
		return ka_err.ErrClientClosed
	default:
		// 先檢查傳入的參數
		if len(msgs) == 0 {
			return nil // 或者返回一個參數錯誤
		}

		for _, msg := range msgs {
			select {
			case p.receiverCh <- msg:
			default:
				continue
			}
		}

		return nil
	}
}

// 結束條件就是完全消耗完receiverCh
func (p *kafkaProducer) produce() {
	log.Printf("kafka producer start consumeing msg...")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer func() {
		p.innerClose <- struct{}{}
	}()

	for msg := range p.receiverCh {
		select {
		case <-ticker.C:
			if len(p.buffer) > 0 {
				err := p.sendMsgs()
				if err != nil {
					log.Printf("kafka producer error : %s", err.Error())
				}
			}
		default:
			p.buffer = append(p.buffer, msg)
			if len(p.buffer) > p.cfg.BatchSize {
				err := p.sendMsgs()
				if err != nil {
					log.Printf("kafka producer error : %s", err.Error())
				}
			}
		}
	}

	if len(p.buffer) > 0 {
		err := p.sendMsgs()
		if err != nil {
			log.Printf("kafka producer error : %s", err.Error())
		}
	}
	log.Printf("kafka producer end consumeing msg...")
}

// 發送buffer裡面所有訊息給kafka
// 失敗會有retry
// 最後不論成功或者失敗，都會清空buffer
func (p *kafkaProducer) sendMsgs() error {
	var err error
	ctx, cancle := context.WithTimeout(context.Background(), time.Second)
	defer cancle()

	for attempt := 0; attempt <= p.cfg.RetryAttempts; attempt++ {
		// 檢查外部 context 是否已經取消
		if ctx.Err() != nil {
			return ka_err.NewKafkaError("Produce", p.cfg.Topic, ctx.Err())
		}
		//若是同步模式，會block到所有消息都寫入
		err = p.writer.WriteMessages(ctx, p.buffer...)
		if err == nil {
			p.buffer = p.buffer[:0]
			return nil
		}

		if !ka_err.IsTemporaryError(err) {
			break
		}
	}

	//buffer訊息發送失敗，可以考慮紀錄起來
	p.buffer = p.buffer[:0]
	return ka_err.NewKafkaError("Produce", p.cfg.Topic, err)
}

// Close implements the Producer interface
func (p *kafkaProducer) Close(waitTIme time.Duration) (err error) {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	defer func() {
		kafka_err := p.writer.Close()
		err = errors.Join(err, kafka_err)
	}()

	select {
	case <-time.After(waitTIme):
		err = errors.New("kafka producer close timeout!!")
	case <-p.innerClose:
		return
	}

	return
}
