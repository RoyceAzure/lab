package consumer

import (
	"encoding/json"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	"github.com/RoyceAzure/lab/cqrs/internal/command/handler"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type CartCommandConsumer struct {
	*handlerAdapter
}

func NewCartCommandConsumer(consumer consumer.Consumer, commandHandler handler.Handler) IBaseConsumer {
	return newBaseConsumer(consumer, &CartCommandConsumer{newHandlerAdapter(commandHandler, nil)})
}

func (c *CartCommandConsumer) transformData(msg message.Message) (consumeData, error) {
	headers := msg.Headers
	var commandType command.CommandType
	for _, header := range headers {
		if header.Key == "command_type" {
			commandType = command.CommandType(header.Value)
			break
		}
	}

	var cmd command.Command
	var zero consumeData
	switch commandType {
	case command.CartCreatedCommandName:
		cmd = &command.CartCreatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	case command.CartUpdatedCommandName:
		cmd = &command.CartUpdatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	case command.CartDeletedCommandName:
		cmd = &command.CartDeletedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	default:
		return zero, ErrUnknownCommandFormat
	}

	return consumeData{cmd, nil}, nil
}
