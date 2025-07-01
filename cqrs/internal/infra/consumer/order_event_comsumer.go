package consumer

import (
	"encoding/json"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/event"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type OrderEventConsumer struct {
	*handlerAdapter
}

// topic: order-event
// 分區: userID
func NewOrderEventConsumer(consumer consumer.Consumer, orderEventHandler event_handler.Handler) IBaseConsumer {
	return newBaseConsumer(consumer, &OrderEventConsumer{newHandlerAdapter(nil, orderEventHandler)})
}

func (c *OrderEventConsumer) transformData(msg message.Message) (consumeData, error) {
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
	case evt_model.OrderCreatedEventName:
		evt = &evt_model.OrderCreatedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.OrderConfirmedEventName:
		evt = &evt_model.OrderConfirmedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.OrderShippedEventName:
		evt = &evt_model.OrderShippedEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.OrderCancelledEventName:
		evt = &evt_model.OrderCancelledEvent{}
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			return zero, err
		}
	case evt_model.OrderRefundedEventName:
		evt = &evt_model.OrderRefundedEvent{}
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
