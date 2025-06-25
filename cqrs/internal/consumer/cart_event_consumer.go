package consumer

import (
	"encoding/json"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/event"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/event/handler"
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
	var eventType event.EventType
	for _, header := range headers {
		if header.Key == "event_type" {
			eventType = event.EventType(header.Value)
			break
		}
	}

	var evt event.Event
	var zero consumeData
	switch eventType {
	case event.CartCreatedEventName:
		evt = &event.CartCreatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case event.CartUpdatedEventName:
		evt = &event.CartUpdatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case event.CartDeletedEventName:
		evt = &event.CartDeletedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case event.CartFailedEventName:
		evt = &event.CartFailedEvent{}
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
