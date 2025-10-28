package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
)

// 需要根據userid做balancer分區
// topic: 由producer創建時設置
type CartCommandProducer struct {
	producer producer.Producer
}

type ICartCommandProducer interface {
	ProduceCartCreatedCommand(ctx context.Context, userID int, items []model.CartItem) error
	ProduceCartUpdatedCommand(ctx context.Context, userID int, details []cmd_model.CartUpdatedDetial) error
	ProduceCartDeletedCommand(ctx context.Context, userID int) error
	ProduceCartConfirmedCommand(ctx context.Context, userID int) error
}

func NewCartCommandProducer(producer producer.Producer) *CartCommandProducer {
	return &CartCommandProducer{producer: producer}
}

func (c *CartCommandProducer) ProduceCartCreatedCommand(ctx context.Context, userID int, items []model.CartItem) error {
	command := cmd_model.NewCartCreatedCommand(userID, items)

	msg, err := c.convertToMessage(userID, command)
	if err != nil {
		return err
	}

	_, err = c.producer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return err
	}
	return nil
}

func (c *CartCommandProducer) ProduceCartUpdatedCommand(ctx context.Context, userID int, details []cmd_model.CartUpdatedDetial) error {
	command := cmd_model.NewCartUpdatedCommand(userID, details)

	msg, err := c.convertToMessage(userID, command)
	if err != nil {
		return err
	}

	_, err = c.producer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return err
	}
	return nil
}

func (c *CartCommandProducer) ProduceCartDeletedCommand(ctx context.Context, userID int) error {
	command := cmd_model.NewCartDeletedCommand(userID)

	msg, err := c.convertToMessage(userID, command)
	if err != nil {
		return err
	}

	_, err = c.producer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return err
	}
	return nil
}

func (c *CartCommandProducer) ProduceCartConfirmedCommand(ctx context.Context, userID int) error {
	command := cmd_model.NewCartConfirmedCommand(userID)

	msg, err := c.convertToMessage(userID, command)
	if err != nil {
		return err
	}

	_, err = c.producer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return err
	}
	return nil
}

func (c *CartCommandProducer) convertToMessage(userID int, cmd cmd_model.Command) (message.Message, error) {
	cmdValue, err := json.Marshal(cmd)
	if err != nil {
		return message.Message{}, err
	}

	msg := message.Message{
		Key:   []byte(fmt.Sprintf("%d", userID)),
		Value: cmdValue,
		Headers: []message.Header{
			{
				Key:   "command_type",
				Value: []byte(cmd.Type()),
			},
		},
	}

	return msg, nil
}
