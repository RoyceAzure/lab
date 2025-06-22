package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
)

type HandlerError error

var (
	errHandlerNotFound HandlerError = errors.New("handler not found")
)

type HandlerFunc func(ctx context.Context, evt event.Event) error

func (f HandlerFunc) HandleEvent(ctx context.Context, evt event.Event) error {
	return f(ctx, evt)
}

type Handler interface {
	HandleEvent(ctx context.Context, evt event.Event) error
}

type HandlerDispatcher struct {
	handlers   map[event.EventType]Handler
	eventCache cache.Cache
}

func NewHandlerDispatcher(handlers map[event.EventType]Handler, eventCache cache.Cache) *HandlerDispatcher {
	return &HandlerDispatcher{handlers: handlers, eventCache: eventCache}
}

func (d *HandlerDispatcher) HandleEvent(ctx context.Context, evt event.Event) error {
	// 檢查事件是否已經處理過
	eventKey := fmt.Sprintf("%s:%s", evt.Type(), evt.GetID())
	_, err := d.eventCache.Get(ctx, eventKey)
	if err != nil {
		return err
	}
	handler, ok := d.handlers[evt.Type()]
	if !ok {
		return errHandlerNotFound
	}
	return handler.HandleEvent(ctx, evt)
}

func NewOrderHandler(orderEventHandler *OrderEventHandler) Handler {
	return &HandlerDispatcher{
		handlers: map[event.EventType]Handler{
			event.OrderCreatedEventName:   HandlerFunc(orderEventHandler.HandleOrderCreated),
			event.OrderConfirmedEventName: HandlerFunc(orderEventHandler.HandleOrderConfirmed),
			event.OrderShippedEventName:   HandlerFunc(orderEventHandler.HandleOrderShipped),
			event.OrderCancelledEventName: HandlerFunc(orderEventHandler.HandleOrderCancelled),
			event.OrderRefundedEventName:  HandlerFunc(orderEventHandler.HandleOrderRefunded),
		},
	}
}
