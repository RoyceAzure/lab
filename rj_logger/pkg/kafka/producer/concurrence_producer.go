package producer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/RoyceAzure/lab/rj_logger/pkg/kafka/config"
	ka_err "github.com/RoyceAzure/lab/rj_logger/pkg/kafka/errors"
)

// Producer interface defines the methods that a Kafka producer must implement
// 生產者會寫入到固定topic，由config.Config設置
type Producer interface {
	// Produce sends messages to Kafka
	Produce(ctx context.Context, msgs []kafka.Message) error
	// Close closes the producer
	Close(time.Duration) error
}

type ProducerError struct {
	Message kafka.Message
	Err     error
}

type option func(*ConcurrencekafkaProducer)

func SetHandlerSuccessfunc(f func(kafka.Message)) option {
	return func(n *ConcurrencekafkaProducer) {
		n.handlerSuccessfunc = f
	}
}
func SetHandlerFailedfunc(f func(ProducerError)) option {
	return func(n *ConcurrencekafkaProducer) {
		n.handlerErrorfunc = f
	}
}

// 同步模式，會block到所有消息都寫入
// 希望是併發安全，所有Logger調用同一個ConcurrencekafkaProducer
type ConcurrencekafkaProducer struct {
	isRunning          atomic.Bool //結束旗標
	buffer             []kafka.Message
	cfg                config.Config
	ctx                context.Context
	canceled           context.CancelFunc
	writer             Writer
	handlerSuccessfunc func(kafka.Message)
	handlerErrorfunc   func(ProducerError)
	receiverCh         chan kafka.Message
	isStopped          chan struct{} //內部程序是否已經停止旗標
}

// 目前默認是同步模式，會block到所有消息都寫入
func NewConcurrencekafkaProducer(w Writer, cfg config.Config, opts ...option) (*ConcurrencekafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, ka_err.ErrInvalidateParameter
	}
	if cfg.Topic == "" {
		return nil, ka_err.ErrInvalidateParameter
	}

	// writer := &kafka.Writer{
	// 	Addr:         kafka.TCP(cfg.Brokers...),
	// 	Balancer:     cfg.Balancer,
	// 	Topic:        cfg.Topic,
	// 	BatchSize:    cfg.BatchSize,
	// 	BatchTimeout: cfg.Timeout,
	// 	WriteTimeout: cfg.Timeout,
	// 	MaxAttempts:  cfg.RetryLimit,
	// 	Async:        cfg.Async,
	// 	RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),

	// 	// 錯誤處理
	// 	ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
	// 		log.Printf("kafka producer error: "+msg, args...)
	// 	}),
	// }

	ka := &ConcurrencekafkaProducer{
		writer: w,
		cfg:    cfg,
	}

	for _, opt := range opts {
		opt(ka)
	}

	return ka, nil
}

func (p *ConcurrencekafkaProducer) Start() {
	if !p.isRunning.CompareAndSwap(false, true) {
		return
	}
	p.ctx, p.canceled = context.WithCancel(context.Background())
	p.buffer = make([]kafka.Message, 0, p.cfg.BatchSize+500)
	p.receiverCh = make(chan kafka.Message, 50000)
	p.isStopped = make(chan struct{})
	go p.produce()
}

// Produce implements the Producer interface
// 非同步，若buffer已滿，放棄傳遞並回傳該訊息
func (p *ConcurrencekafkaProducer) Produce(ctx context.Context, msgs []kafka.Message) (errMsgs []kafka.Message, err error) {
	if !p.isRunning.Load() {
		return msgs, ka_err.ErrClientClosed
	}
	if len(msgs) == 0 {
		return nil, nil
	}

	errMsgs = make([]kafka.Message, 0, len(msgs))

	for i, msg := range msgs {
		select {
		case <-p.ctx.Done():
			for j := i; j < len(msgs); j++ {
				errMsgs = append(errMsgs, msgs[j])
			}
			return errMsgs, errors.New("producer is already closed")
		case p.receiverCh <- msg:
		default:
			errMsgs = append(errMsgs, msg)
			continue
		}
	}

	if len(errMsgs) > 0 {
		return errMsgs, fmt.Errorf("some msg send failed")
	}

	return nil, nil
}

// p.handlerSuccessfunc處理p.buffer所有Msg，不會清空buffer
func (p *ConcurrencekafkaProducer) batchHandleSuccess() {
	if p.handlerSuccessfunc != nil {
		for _, msg := range p.buffer {
			p.handlerSuccessfunc(msg)
		}
	}
}

// p.handlerSuccessfunc處理p.buffer所有Msg，不會清空buffer
func (p *ConcurrencekafkaProducer) batchHandleFailed(err error) {
	if p.handlerErrorfunc != nil {
		for _, msg := range p.buffer {
			p.handlerErrorfunc(ProducerError{
				Message: msg,
				Err:     err,
			})
		}
	}
}

// sendMsg with retry
// return
// bool: isFatal, if it is true, need to inmiditily stopping producer
func (p *ConcurrencekafkaProducer) process() (bool, error) {
	for i := 0; i < p.cfg.RetryLimit; i++ {
		isfatal, err := p.sendMsgs()
		if err == nil {
			return false, nil
		}
		log.Printf("kafka producer error : %s", err.Error())
		if isfatal {
			return true, err
		}
		n := 1 << i
		time.Sleep(p.cfg.RetryDelay * time.Duration(n))
	}

	return false, fmt.Errorf("producer send msg retry failed after reach limit")
}

// 內部發送程序
// 結束條件就是完全消耗完receiverCh
func (p *ConcurrencekafkaProducer) produce() {
	defer func() {
		p.isStopped <- struct{}{}
		p.stop(time.Second*10, false)
	}()

	var (
		err     error
		isfatal bool
	)

	log.Printf("kafka producer start consumeing msg...")
	ticker := time.NewTicker(p.cfg.CommitInterval)
	defer ticker.Stop()

normalProcess:
	for msg := range p.receiverCh {
		select {
		case <-ticker.C:
			p.buffer = append(p.buffer, msg)
			if len(p.buffer) > 0 {
				isfatal, err = p.process()
				if err != nil {
					if isfatal {
						break normalProcess
					} else {
						p.batchHandleFailed(err)
					}
				} else {
					p.batchHandleSuccess()
				}
				p.buffer = p.buffer[:0]
			}
		default:
			p.buffer = append(p.buffer, msg)
			if len(p.buffer) >= p.cfg.BatchSize {
				isfatal, err = p.process()
				if err != nil {
					if isfatal {
						break normalProcess
					} else {
						p.batchHandleFailed(err)
					}
				} else {
					p.batchHandleSuccess()
				}
				p.buffer = p.buffer[:0]
			}
			//清空ticker
			ticker.Reset(p.cfg.CommitInterval)
			select {
			case <-ticker.C:
			default:
			}
		}
	}

	if isfatal {
		for msg := range p.receiverCh {
			p.buffer = append(p.buffer, msg)
		}
		p.batchHandleFailed(err)
	} else {
		//receiverCh已經處理完， 處理殘留於buffer資料
		if len(p.buffer) > 0 {
			_, err := p.process()
			if err != nil {
				p.batchHandleFailed(err)
			} else {
				p.batchHandleSuccess()
			}
		}
	}
	p.buffer = p.buffer[:0]

	log.Printf("kafka producer end consumeing msg...")
}

// 發送buffer裡面所有訊息給kafka
// 失敗會有retry，每次失敗固定等待p.cfg.CommitInterval時間
// return :
// true (fatal error), false (temp error)
func (p *ConcurrencekafkaProducer) sendMsgs() (bool, error) {
	var err error
	ctx, cancle := context.WithTimeout(context.Background(), time.Second*5)
	defer cancle()

	//若是同步模式，會block到所有消息都寫入
	log.Printf("kafka producer send msgs: %d msgs", len(p.buffer))
	err = p.writer.WriteMessages(ctx, p.buffer...)
	if err != nil {
		err = ka_err.NewKafkaError("Produce", p.cfg.Topic, err)
		return !ka_err.IsTemporaryError(err), err
	}

	return false, nil
}

// Block until close is finished
func (p *ConcurrencekafkaProducer) Close(waitTIme time.Duration) error {
	return p.stop(waitTIme, true)
}

// parm:
// needWait (bool) :　若為內部呼叫，使用false, 外部呼叫使用true
func (p *ConcurrencekafkaProducer) stop(waitTIme time.Duration, needWait bool) (err error) {
	if !p.isRunning.CompareAndSwap(true, false) {
		return nil
	}
	p.canceled()        //拒絕接收新資料
	close(p.receiverCh) // 等待內部處理完所有receiverCh 內部剩餘資料

	if needWait {
		select {
		case <-time.After(waitTIme):
			err = fmt.Errorf("procucer not closed within waitTIme some msgs will lose")
		case <-p.isStopped:
		}
	}
	kafka_err := p.writer.Close()
	err = errors.Join(err, kafka_err)
	return
}
