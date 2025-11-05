package producer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/pkg/config"
	ka_err "github.com/RoyceAzure/lab/rj_kafka/pkg/errors"
	"github.com/RoyceAzure/lab/rj_kafka/pkg/model"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

// Producer interface defines the methods that a Kafka producer must implement
// 生產者會寫入到固定topic，由config.Config設置
type Producer interface {
	// Produce sends messages to Kafka
	Produce(ctx context.Context, msgs []model.Message) ([]model.Message, error)
	// Close closes the producer
	Close(time.Duration) error
}

type Option func(*ConcurrencekafkaProducer)

func SetHandlerSuccessfunc(f func(model.Message)) Option {
	return func(n *ConcurrencekafkaProducer) {
		n.handlerSuccessfunc = f
	}
}
func SetHandlerFailedfunc(f func(model.ProducerError)) Option {
	return func(n *ConcurrencekafkaProducer) {
		n.handlerErrorfunc = f
	}
}

// 同步模式，會block到所有消息都寫入
// 希望是併發安全，所有Logger調用同一個ConcurrencekafkaProducer
type ConcurrencekafkaProducer struct {
	isRunning          atomic.Bool //結束旗標
	buffer             []model.Message
	cfg                config.Config
	writer             Writer
	handlerSuccessfunc func(model.Message) //使用自訂義結構，可方便日後擴展
	handlerErrorfunc   func(model.ProducerError)
	receiverCh         chan []model.Message
	isStopped          chan struct{} //內部程序是否已經停止旗標
	chanMutex          sync.RWMutex
	logger             *zerolog.Logger
}

// 目前默認是同步模式，會block到所有消息都寫入
func NewConcurrencekafkaProducer(w Writer, cfg config.Config, opts ...Option) (*ConcurrencekafkaProducer, error) {
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

	logger := zerolog.New(os.Stdout).Level(zerolog.Level(cfg.LogLevel)).With().Timestamp().Logger()
	ka := &ConcurrencekafkaProducer{
		writer: w,
		cfg:    cfg,
		logger: &logger,
	}

	for _, opt := range opts {
		opt(ka)
	}

	if cfg.ProducerHandlerSuccessfunc != nil {
		ka.handlerSuccessfunc = cfg.ProducerHandlerSuccessfunc
	}
	if cfg.ProducerHandlerErrorfunc != nil {
		ka.handlerErrorfunc = cfg.ProducerHandlerErrorfunc
	}

	return ka, nil
}

func (p *ConcurrencekafkaProducer) Start() {
	if !p.isRunning.CompareAndSwap(false, true) {
		return
	}
	p.buffer = make([]model.Message, 0, p.cfg.BatchSize+512)
	p.receiverCh = make(chan []model.Message, p.cfg.BatchSize/2)
	p.isStopped = make(chan struct{})
	go p.produce()
}

// Produce implements the Producer interface
// 非同步，若buffer已滿，放棄傳遞並回傳該訊息
func (p *ConcurrencekafkaProducer) Produce(ctx context.Context, msgs []model.Message) (errMsgs []model.Message, err error) {
	p.chanMutex.RLock()
	defer p.chanMutex.RUnlock()
	if !p.isRunning.Load() {
		return msgs, ka_err.ErrClientClosed
	}
	if len(msgs) == 0 {
		return nil, nil
	}

	errMsgs = make([]model.Message, 0, len(msgs))

	select {
	case p.receiverCh <- msgs:
	default:
		p.logger.Error().Msg("kafka producer receiverCh is full, some msg will be lost")
		errMsgs = append(errMsgs, msgs...)
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
			p.handlerErrorfunc(model.ProducerError{
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
		p.logger.Error().Msgf("kafka producer error : %s", err.Error())
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
		p.logger.Info().Msg("kafka producer end consumeing msg...")
		close(p.isStopped)
		p.stop(time.Second*15, false)
	}()

	var (
		err     error
		isfatal bool
	)

	p.logger.Info().Msg("kafka producer start consumeing msg...")
	ticker := time.NewTicker(p.cfg.CommitInterval)
	defer ticker.Stop()

	processMsg := func(limit int) (bool, error) {
		if len(p.buffer) >= limit {
			p.logger.Debug().Msgf("kafka producer process msg by limit: %d msgs, buffer count: %d", limit, len(p.buffer))
			isfatal, err = p.process()
			if err != nil {
				if isfatal {
					return true, err
				} else {
					p.batchHandleFailed(err)
				}
			} else {
				p.batchHandleSuccess()
			}
			//清空buffer
			p.buffer = p.buffer[:0]
			//發送完訊息後，重置ticker
			ticker.Reset(p.cfg.CommitInterval)
			select {
			case <-ticker.C:
			default:
			}
		}
		return false, nil
	}

	for msg := range p.receiverCh {
		p.buffer = append(p.buffer, msg...)
		select {
		case <-ticker.C:
			p.logger.Debug().Msg("kafka producer triggered by ticker")
			isfatal, err = processMsg(1)
		default:
			p.logger.Debug().Msg("kafka producer triggered by default")
			isfatal, err = processMsg(int(float64(p.cfg.BatchSize) * 0.8))
		}

		if isfatal {
			p.logger.Error().Msgf("kafka producer process fatal error: %s", err.Error())
			p.stop(time.Nanosecond, false)
			//目的是要觸發關閉chans
			//才能針對channel剩餘的資料進行處理
			for msg := range p.receiverCh {
				p.buffer = append(p.buffer, msg...)
			}
			if len(p.buffer) > 0 {
				if err != nil {
					p.batchHandleFailed(err)
				} else {
					p.batchHandleSuccess()
				}
				p.buffer = p.buffer[:0]
			}
			break
		}
	}
	//處理剩餘buffer內訊息
	p.logger.Debug().Msgf("kafka producer process remaining msg: %d msgs", len(p.buffer))
	processMsg(1)
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
	p.logger.Debug().Msgf("kafka producer send msgs: %d msgs", len(p.buffer))

	msgs := make([]kafka.Message, 0, len(p.buffer))
	for _, msg := range p.buffer {
		msgs = append(msgs, msg.ToKafkaMessage())
	}

	err = p.writer.WriteMessages(ctx, msgs...)
	if err != nil {
		err = ka_err.NewKafkaError("Produce", p.cfg.Topic, err)
		return !ka_err.IsTemporaryError(err), err
	}

	return false, nil
}

// waitTIme (time.Duration) : 給予producer處理剩餘訊息的時間，若超過時間，則會印出錯誤
func (p *ConcurrencekafkaProducer) Close(waitTIme time.Duration) error {
	return p.stop(waitTIme, true)
}

func (p *ConcurrencekafkaProducer) C() chan struct{} {
	return p.isStopped
}

// parm:
// needWait (bool) :　是否等待內部循環結束，若為內部呼叫，使用false, 外部呼叫使用true
func (p *ConcurrencekafkaProducer) stop(waitTIme time.Duration, needWait bool) (err error) {
	if !p.isRunning.CompareAndSwap(true, false) {
		return nil
	}

	p.chanMutex.Lock()
	close(p.receiverCh)
	p.chanMutex.Unlock()

	if needWait {
		select {
		case <-time.After(waitTIme):
			err = fmt.Errorf("procucer not closed within waitTIme some msgs will lose")
		case <-p.isStopped:
		}
	}

	kafka_err := p.writer.Close()
	err = errors.Join(err, kafka_err)
	if err != nil {
		p.logger.Error().Msgf("kafka producer stop error: %s", err.Error())
	}
	return
}
