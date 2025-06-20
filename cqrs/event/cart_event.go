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

type CartCreatedFailedEvent struct {
	BaseEvent
	UserID  int
	Message string
}

type CartUpdatedEvent struct {
	BaseEvent
	UserID  int
	Details []command.CartUpdatedDetial
}

func (e *CartCreatedFailedEvent) Type() EventType {
	return CartCreatedFailedEventName
}

func (e *CartCreatedEvent) Type() EventType {
	return CartCreatedEventName
}
