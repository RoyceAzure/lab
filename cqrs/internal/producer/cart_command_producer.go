package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	"github.com/RoyceAzure/lab/cqrs/internal/model"
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
	ProduceCartUpdatedCommand(ctx context.Context, userID int, details []command.CartUpdatedDetial) error
	ProduceCartDeletedCommand(ctx context.Context, userID int) error
}

func NewCartCommandProducer(producer producer.Producer) *CartCommandProducer {
	return &CartCommandProducer{producer: producer}
}

func (c *CartCommandProducer) ProduceCartCreatedCommand(ctx context.Context, userID int, items []model.CartItem) error {
	command := command.CartCreatedCommand{
		UserID: userID,
		Items:  items,
	}

	msg, err := c.convertToMessage(userID, &command)
	if err != nil {
		return err
	}

	return c.producer.Produce(ctx, []message.Message{msg})
}

func (c *CartCommandProducer) ProduceCartUpdatedCommand(ctx context.Context, userID int, details []command.CartUpdatedDetial) error {
	command := command.CartUpdatedCommand{
		UserID:  userID,
		Details: details,
	}

	msg, err := c.convertToMessage(userID, &command)
	if err != nil {
		return err
	}

	return c.producer.Produce(ctx, []message.Message{msg})
}

func (c *CartCommandProducer) ProduceCartDeletedCommand(ctx context.Context, userID int) error {
	command := command.CartDeletedCommand{
		UserID: userID,
	}

	msg, err := c.convertToMessage(userID, &command)
	if err != nil {
		return err
	}

	return c.producer.Produce(ctx, []message.Message{msg})
}

func (c *CartCommandProducer) convertToMessage(userID int, cmd command.Command) (message.Message, error) {
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
