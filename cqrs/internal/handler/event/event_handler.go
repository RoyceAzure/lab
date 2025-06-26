package handler

import (
	"context"
	"errors"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/redis/go-redis/v9"
)

type HandlerError error

var (
	errHandlerNotFound    HandlerError = errors.New("handler not found")
	errUnknownEventFormat HandlerError = errors.New("unknown event format")
)

type HandlerFunc func(ctx context.Context, evt evt_model.Event) error

func (f HandlerFunc) HandleEvent(ctx context.Context, evt evt_model.Event) error {
	return f(ctx, evt)
}

type Handler interface {
	HandleEvent(ctx context.Context, evt evt_model.Event) error
}

type HandlerDispatcher struct {
	handlers   map[evt_model.EventType]Handler
	eventCache *redis.Client
}

func NewHandlerDispatcher(handlers map[evt_model.EventType]Handler, eventCache *redis.Client) *HandlerDispatcher {
	return &HandlerDispatcher{handlers: handlers, eventCache: eventCache}
}

func (d *HandlerDispatcher) HandleEvent(ctx context.Context, evt evt_model.Event) error {
	// 檢查事件是否已經處理過
	if d.eventCache != nil {
		eventKey := fmt.Sprintf("%s:%s", evt.Type(), evt.GetID())
		_, err := d.eventCache.Get(ctx, eventKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}
	}

	handler, ok := d.handlers[evt.Type()]
	if !ok {
		return errHandlerNotFound
	}
	return handler.HandleEvent(ctx, evt)
}

func NewOrderEventHandlerDispatcher(orderEventHandler *OrderEventHandler) Handler {
	return &HandlerDispatcher{
		handlers: map[evt_model.EventType]Handler{
			evt_model.OrderCreatedEventName:   HandlerFunc(orderEventHandler.HandleOrderCreated),
			evt_model.OrderConfirmedEventName: HandlerFunc(orderEventHandler.HandleOrderConfirmed),
			evt_model.OrderShippedEventName:   HandlerFunc(orderEventHandler.HandleOrderShipped),
		},
	}
}

func NewCartEventHandler(cartRepo *redis_repo.CartRepo) Handler {
	cartEventHandler := newCartEventHandler(cartRepo)
	return &HandlerDispatcher{
		handlers: map[evt_model.EventType]Handler{
			evt_model.CartCreatedEventName: HandlerFunc(cartEventHandler.HandleCartCreated),
			evt_model.CartFailedEventName:  HandlerFunc(cartEventHandler.HandleCartFailed),
			evt_model.CartUpdatedEventName: HandlerFunc(cartEventHandler.HandleCartUpdated),
			evt_model.CartDeletedEventName: HandlerFunc(cartEventHandler.HandleCartDeleted),
		},
	}
}
