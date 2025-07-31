package consumer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"sync"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/producer"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

/*
負責接收redis更新商品資資訊，更新db product 資料
*/
type ProductConsumer struct {
	consumer  consumer.Consumer
	closeChan chan struct{}
	closeOnce sync.Once
	handler   ProductConsumHandler
}

type ProductConsumHandler struct {
	productRepo   db.IProductRepository
	timestampRepo redis_repo.ITimestampRedisRepository
}

func (h *ProductConsumHandler) Handle(ctx context.Context, cmd producer.ProductCommand, data model.Product) error {
	switch cmd {
	case producer.ProductCommandAdd:
		return h.productRepo.CreateProduct(ctx, &data)
	case producer.ProductCommandUpdateReserved:
		return h.productRepo.UpdateReserved(ctx, data.ProductID, data.Reserved)
	case producer.ProductCommandDelete:
		return h.productRepo.HardDeleteProduct(ctx, data.ProductID)
	case producer.ProductCommandUpdate:
		return h.productRepo.UpdateProduct(ctx, &data)
	}
	return ErrCommandTypeNotFound
}

func (h *ProductConsumHandler) transformData(msg message.Message) (producer.ProductCommand, *model.Product) {
	var product model.Product
	err := json.Unmarshal(msg.Value, &product)
	if err != nil {
		return "", nil
	}
	return producer.ProductCommand(msg.Headers[0].Value), &product
}

/*
不是所有message都有timestamp 只檢查有timestamp的message
*/
func (h *ProductConsumHandler) setAndCheckProductTimestamp(ctx context.Context, key string, msg message.Message) bool {
	for _, header := range msg.Headers {
		if header.Key == "timestamp" {
			timestamp := binary.BigEndian.Uint64(header.Value)
			if b, _ := h.timestampRepo.SetProductTimestamp(ctx, key, int64(timestamp)); !b {
				return false
			}
			return true
		}
	}
	//不是所有message都有timestamp 只檢查有timestamp的message
	return true
}

func NewProductConsumer(productRepo db.IProductRepository, timestampRepo redis_repo.ITimestampRedisRepository, consumer consumer.Consumer) *ProductConsumer {
	return &ProductConsumer{handler: ProductConsumHandler{productRepo: productRepo, timestampRepo: timestampRepo}, consumer: consumer, closeChan: make(chan struct{})}
}

func (c *ProductConsumer) checkIsClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *ProductConsumer) Start(ctx context.Context) error {
	if c.checkIsClosed() {
		return ErrConsumerClosed
	}

	msgChan, errChan, err := c.consumer.Consume()

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-c.closeChan:
				return
			case msg := <-msgChan:
				evt, data := c.handler.transformData(msg)
				if err != nil {
					log.Println("error", err)
					continue
				}

				key := c.handler.timestampRepo.GetProductTimestampKey(string(msg.Key), string(evt))

				if !c.handler.setAndCheckProductTimestamp(ctx, key, msg) {
					log.Printf("warning: product %s timestamp not match", string(msg.Key))
					continue
				}

				err = c.handler.Handle(ctx, evt, *data)
				if err != nil {
					log.Println("error", err)
					continue
				}
			case err := <-errChan:
				log.Println("error", err)
			}
		}
	}()

	return nil
}

func (c *ProductConsumer) Stop() {
	if c.checkIsClosed() {
		return
	}

	c.closeOnce.Do(func() {
		close(c.closeChan)
	})

	c.consumer.Close()
}
