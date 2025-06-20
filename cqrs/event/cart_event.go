package event

import (
	"github.com/RoyceAzure/lab/cqrs/command"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
)

type CartCreatedEvent struct {
	BaseEvent
	UserID int
	Items  []model.OrderItemData
}

type CartCommandFailedEvent struct {
	BaseEvent
	UserID  int
	Message string
}

type CartUpdatedEvent struct {
	BaseEvent
	UserID  int
	Details []command.CartUpdatedDetial
}

func (e *CartCommandFailedEvent) Type() EventType {
	return CartFailedEventName
}

func (e *CartCreatedEvent) Type() EventType {
	return CartCreatedEventName
}

type CartDeletedEvent struct {
	BaseEvent
	UserID int
}

func (e *CartDeletedEvent) Type() EventType {
	return CartDeletedEventName
}
