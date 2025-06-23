package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	"github.com/RoyceAzure/lab/cqrs/internal/command/handler"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type ConsumerError error

var (
	ErrConsumerClosed      = errors.New("consumer closed")
	ErrCommandTypeNotFound = errors.New("command type not found")
)

type CartCommandConsumer struct {
	consumer       consumer.Consumer
	commandHandler handler.Handler
	closeChan      chan struct{}
}

func NewCartCommandConsumer(consumer consumer.Consumer, commandHandler handler.Handler) *CartCommandConsumer {
	return &CartCommandConsumer{consumer: consumer, commandHandler: commandHandler, closeChan: make(chan struct{})}
}

func (c *CartCommandConsumer) checkIsClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *CartCommandConsumer) Start(ctx context.Context) error {
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
				cmd, err := c.transformCommand(msg)
				if err != nil {
					log.Println("error", err)
					continue
				}

				err = c.commandHandler.HandleCommand(ctx, cmd)
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

func (c *CartCommandConsumer) transformCommand(msg message.Message) (command.Command, error) {
	headers := msg.Headers
	var commandType command.CommandType
	for _, header := range headers {
		if header.Key == "command_type" {
			err := json.Unmarshal(header.Value, &commandType)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	var cmd command.Command
	switch commandType {
	case command.CartCreatedCommandName:
		cmd = &command.CartCreatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return nil, err
		}
	case command.CartUpdatedCommandName:
		cmd = &command.CartUpdatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return nil, err
		}
	case command.CartDeletedCommandName:
		cmd = &command.CartDeletedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

func (c *CartCommandConsumer) Stop() {
	if c.checkIsClosed() {
		return
	}

	close(c.closeChan)
	c.consumer.Close()
}
