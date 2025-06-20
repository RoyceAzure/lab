package event

import (
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/shopspring/decimal"
)

type CartCreatedEvent struct {
	BaseEvent
	UserID int
	Items  []model.OrderItemData
	Amount decimal.Decimal
}

type CartCreatedFailedEvent struct {
	BaseEvent
	UserID  int
	Message string
}

func (e *CartCreatedFailedEvent) Type() EventType {
	return CartCreatedFailedEventName
}

func (e *CartCreatedEvent) Type() EventType {
	return CartCreatedEventName
}
