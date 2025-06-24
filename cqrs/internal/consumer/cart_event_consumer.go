package consumer

import (
	"encoding/json"

	"github.com/RoyceAzure/lab/cqrs/internal/event"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/event/handler"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type CartEventConsumer struct {
	*eventHandlerAdapter
}

func NewCartEventConsumer(consumer consumer.Consumer, cartEventHandler event_handler.Handler) IBaseConsumer {
	return newBaseConsumer(consumer, &CartEventConsumer{newEventHandlerAdapter(cartEventHandler)})
}

func (c *CartEventConsumer) transformData(msg message.Message) (ConsumeData, error) {
	headers := msg.Headers
	var eventType event.EventType
	for _, header := range headers {
		if header.Key == "event_type" {
			err := json.Unmarshal(header.Value, &eventType)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	var evt event.Event
	switch eventType {
	case event.CartCreatedEventName:
		evt = &event.CartCreatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return nil, err
		}
	case event.CartUpdatedEventName:
		evt = &event.CartUpdatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return nil, err
		}
	case event.CartDeletedEventName:
		evt = &event.CartDeletedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return nil, err
		}
	}

	return newEventToConsumeDataAdapter(evt), nil
}
