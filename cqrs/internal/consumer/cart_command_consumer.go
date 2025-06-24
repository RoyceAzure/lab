package consumer

import (
	"encoding/json"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	"github.com/RoyceAzure/lab/cqrs/internal/command/handler"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type CartCommandConsumer struct {
	*commandHandlerAdapter
}

func NewCartCommandConsumer(consumer consumer.Consumer, commandHandler handler.Handler) IBaseConsumer {
	return newBaseConsumer(consumer, &CartCommandConsumer{newCommandHandlerAdapter(commandHandler)})
}

func (c *CartCommandConsumer) transformData(msg message.Message) (ConsumeData, error) {
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

	return newCommandToConsumeDataAdapter(cmd), nil
}
