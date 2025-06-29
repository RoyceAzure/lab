package eventdb

import (
	"context"
	"errors"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
)

var ErrNonOrderEvent EventFormatError = errors.New("non order event")

// 處理order相關事件儲存到eventdb
// order聚合的還原  projection
// 後續應該統一給app層 order service 處理

func (dao *EventDao) SaveOrderCreatedEvent(ctx context.Context, data *evt_model.OrderCreatedEvent) error {
	isOrderEvent, err := dao.checkBaseOrderEvent(*data)
	if err != nil {
		return err
	}

	if !isOrderEvent {
		return ErrNonOrderEvent
	}

	return dao.AppendEvent(ctx, data.AggregateID, string(data.Type()), data)
}

func (dao *EventDao) SaveOrderConfirmedEvent(ctx context.Context, data *evt_model.OrderConfirmedEvent) error {
	isOrderEvent, err := dao.checkBaseOrderEvent(*data)
	if err != nil {
		return err
	}

	if !isOrderEvent {
		return ErrNonOrderEvent
	}

	return dao.AppendEvent(ctx, data.AggregateID, string(data.Type()), data)
}

func (dao *EventDao) SaveOrderShippedEvent(ctx context.Context, data *evt_model.OrderShippedEvent) error {
	isOrderEvent, err := dao.checkBaseOrderEvent(*data)
	if err != nil {
		return err
	}

	if !isOrderEvent {
		return ErrNonOrderEvent
	}

	return dao.AppendEvent(ctx, data.AggregateID, string(data.Type()), data)
}

func (dao *EventDao) SaveOrderCancelledEvent(ctx context.Context, data *evt_model.OrderCancelledEvent) error {
	isOrderEvent, err := dao.checkBaseOrderEvent(*data)
	if err != nil {
		return err
	}

	if !isOrderEvent {
		return ErrNonOrderEvent
	}

	return dao.AppendEvent(ctx, data.AggregateID, string(data.Type()), data)
}

func (dao *EventDao) SaveOrderRefundedEvent(ctx context.Context, data *evt_model.OrderRefundedEvent) error {
	isOrderEvent, err := dao.checkBaseOrderEvent(*data)
	if err != nil {
		return err
	}

	if !isOrderEvent {
		return ErrNonOrderEvent
	}

	return dao.AppendEvent(ctx, data.AggregateID, string(data.Type()), data)
}

func (dao *EventDao) checkBaseOrderEvent(base any) (bool, error) {
	baseEvent, ok := base.(evt_model.BaseEvent)
	if !ok {
		return false, fmt.Errorf("%w: event_id is required", ErrEventFormat)
	}

	if baseEvent.AggregateID == "" {
		return false, fmt.Errorf("%w: aggregate_id is required", ErrEventFormat)
	}

	if baseEvent.CreatedAt.IsZero() {
		return false, fmt.Errorf("%w: created_at is required", ErrEventFormat)
	}

	orderEventTypes := []evt_model.EventType{
		evt_model.OrderCreatedEventName,
		evt_model.OrderConfirmedEventName,
		evt_model.OrderShippedEventName,
		evt_model.OrderCancelledEventName,
		evt_model.OrderRefundedEventName,
	}

	var isOrderEvent bool
	for _, eventType := range orderEventTypes {
		if eventType == baseEvent.EventType {
			isOrderEvent = true
		}
	}

	if !isOrderEvent {
		return false, ErrNonOrderEvent
	}

	return true, nil

}
