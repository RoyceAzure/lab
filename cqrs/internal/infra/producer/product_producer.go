package producer

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
)

type ProductProducer struct {
	producer producer.Producer
}

type ProductCommand string

var (
	ProductCommandAdd            ProductCommand = "add"
	ProductCommandUpdate         ProductCommand = "update"
	ProductCommandUpdateReserved ProductCommand = "update_reserved"
	ProductCommandDelete         ProductCommand = "delete"
)

func NewProductProducer(producer producer.Producer) *ProductProducer {
	return &ProductProducer{producer: producer}
}

func (p *ProductProducer) AddNewProduct(ctx context.Context, product *model.Product) error {
	msg, err := p.convertToMessage(ProductCommandAdd, product, 0)
	if err != nil {
		return err
	}

	return p.producer.Produce(ctx, []message.Message{msg})
}

func (p *ProductProducer) UpdateProductStock(ctx context.Context, product *model.Product) error {
	msg, err := p.convertToMessage(ProductCommandUpdate, product, 0)
	if err != nil {
		return err
	}

	return p.producer.Produce(ctx, []message.Message{msg})
}

func (p *ProductProducer) UpdateProductReserved(ctx context.Context, product *model.Product, timestamp int64) error {
	msg, err := p.convertToMessage(ProductCommandUpdateReserved, product, timestamp)
	if err != nil {
		return err
	}

	return p.producer.Produce(ctx, []message.Message{msg})
}

func (p *ProductProducer) UpdateProduct(ctx context.Context, product *model.Product) error {
	msg, err := p.convertToMessage(ProductCommandUpdate, product, 0)
	if err != nil {
		return err
	}

	return p.producer.Produce(ctx, []message.Message{msg})
}

func (p *ProductProducer) DeleteProduct(ctx context.Context, productID string) error {
	msg, err := p.convertToMessage(ProductCommandDelete, &model.Product{ProductID: productID}, 0)
	if err != nil {
		return err
	}

	return p.producer.Produce(ctx, []message.Message{msg})
}

func (p *ProductProducer) convertToMessage(cmd ProductCommand, product *model.Product, timestamp int64) (message.Message, error) {
	productValue, err := json.Marshal(product)
	if err != nil {
		return message.Message{}, err
	}

	msg := message.Message{
		Key:   []byte(product.ProductID),
		Value: productValue,
		Headers: []message.Header{
			{
				Key:   "command_type",
				Value: []byte(cmd),
			},
		},
	}

	if timestamp != 0 {
		msg.Headers = append(msg.Headers, message.Header{
			Key: "timestamp",
			Value: func() []byte {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(timestamp))
				return b
			}(),
		})
	}

	return msg, nil
}
