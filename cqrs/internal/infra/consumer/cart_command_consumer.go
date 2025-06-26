package consumer

import (
	"encoding/json"

	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	handler "github.com/RoyceAzure/lab/cqrs/internal/handler/command"
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
	var commandType cmd_model.CommandType
	for _, header := range headers {
		if header.Key == "command_type" {
			commandType = cmd_model.CommandType(header.Value)
			break
		}
	}

	var cmd cmd_model.Command
	var zero consumeData
	switch commandType {
	case cmd_model.CartCreatedCommandName:
		cmd = &cmd_model.CartCreatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	case cmd_model.CartUpdatedCommandName:
		cmd = &cmd_model.CartUpdatedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	case cmd_model.CartDeletedCommandName:
		cmd = &cmd_model.CartDeletedCommand{}
		err := json.Unmarshal(msg.Value, &cmd)
		if err != nil {
			return zero, err
		}
	default:
		return zero, ErrUnknownCommandFormat
	}

	return consumeData{cmd, nil}, nil
}
