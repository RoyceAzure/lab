package event

import (
	"github.com/RoyceAzure/lab/cqrs/command"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
)

type CartCreatedEvent struct {
	BaseEvent
	UserID int
	Items  []model.OrderItemData
}

func (e *CartCreatedEvent) Type() EventType {
	return CartCreatedEventName
}

type CartFailedEvent struct {
	BaseEvent
	UserID  int
	Message string
}

func (e *CartFailedEvent) Type() EventType {
	return CartFailedEventName
}

type CartUpdatedEvent struct {
	BaseEvent
	UserID  int
	Details []command.CartUpdatedDetial
}

func (e *CartUpdatedEvent) Type() EventType {
	return CartUpdatedEventName
}

type CartDeletedEvent struct {
	BaseEvent
	UserID int
}

func (e *CartDeletedEvent) Type() EventType {
	return CartDeletedEventName
}
