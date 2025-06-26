package consumer

import (
	"encoding/json"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/event"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type CartEventConsumer struct {
	*handlerAdapter
}

func NewCartEventConsumer(consumer consumer.Consumer, cartEventHandler event_handler.Handler) IBaseConsumer {
	return newBaseConsumer(consumer, &CartEventConsumer{newHandlerAdapter(nil, cartEventHandler)})
}

func (c *CartEventConsumer) transformData(msg message.Message) (consumeData, error) {
	headers := msg.Headers
	var eventType evt_model.EventType
	for _, header := range headers {
		if header.Key == "event_type" {
			eventType = evt_model.EventType(header.Value)
			break
		}
	}

	var evt evt_model.Event
	var zero consumeData
	switch eventType {
	case evt_model.CartCreatedEventName:
		evt = &evt_model.CartCreatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.CartUpdatedEventName:
		evt = &evt_model.CartUpdatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.CartDeletedEventName:
		evt = &evt_model.CartDeletedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.CartFailedEventName:
		evt = &evt_model.CartFailedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	default:
		fmt.Printf("unknown event type: %s", eventType)
		return zero, ErrUnknownEventFormat
	}

	return consumeData{nil, evt}, nil
}
