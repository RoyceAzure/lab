package handler

import (
	"context"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/eventdb"
)

type OrderEventHandler struct {
	eventDao *eventdb.EventDao
}

func NewOrderEventHandler(eventDao *eventdb.EventDao) *OrderEventHandler {
	return &OrderEventHandler{eventDao: eventDao}
}

func (h *OrderEventHandler) HandleEvent(ctx context.Context, evt event.Event) error {
	switch evt.Type() {
	case event.OrderCreatedEventName:
		// 處理 OrderConfirmedEvent
		if e, ok := evt.(*event.OrderCreatedEvent); ok {
			fmt.Printf("Handling OrderConfirmedEvent: OrderID=%d, UserID=%d\n", e.OrderID, e.UserID)
			return h.eventDao.AppendEvent(ctx, e.AggregateID, string(event.OrderCreatedEventName), e)
		}
	case event.OrderShippedEventName:
		// 處理 OrderShippedEvent
		if e, ok := evt.(*event.OrderShippedEvent); ok {
			fmt.Printf("Handling OrderShippedEvent: OrderID=%d, TrackingCode=%s\n", e.OrderID, e.TrackingCode)
			return h.eventDao.AppendEvent(ctx, e.AggregateID, string(event.OrderShippedEventName), e)
		}
	case event.OrderCancelledEventName:
		// 處理 OrderCancelledEvent
		if e, ok := evt.(*event.OrderCancelledEvent); ok {
			fmt.Printf("Handling OrderCancelledEvent: OrderID=%d, Message=%s\n", e.OrderID, e.Message)
			return h.eventDao.AppendEvent(ctx, e.AggregateID, string(event.OrderCancelledEventName), e)
		}
	case event.OrderRefundedEventName:
		// 處理 OrderRefundedEvent
		if e, ok := evt.(*event.OrderRefundedEvent); ok {
			fmt.Printf("Handling OrderRefundedEvent: OrderID=%d, Amount=%s\n", e.OrderID, e.Amount)
			return h.eventDao.AppendEvent(ctx, e.AggregateID, string(event.OrderRefundedEventName), e)
		}
	default:
		fmt.Printf("Unknown event type: %v\n", evt.Type())
	}
	return nil
}
