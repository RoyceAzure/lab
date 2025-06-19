package event

import (
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/shopspring/decimal"
)

type OrderCreatedEvent struct {
	BaseEvent
	OrderID string
	UserID  uint
	Items   []model.OrderItemData
	Amount  decimal.Decimal
	State   uint
}

func (e *OrderCreatedEvent) Type() EventType {
	return OrderCreatedEventName
}

type OrderConfirmedEvent struct {
	BaseEvent
	OrderID string
	UserID  uint
	State   uint
}

func (e *OrderConfirmedEvent) Type() EventType {
	return OrderConfirmedEventName
}

type OrderShippedEvent struct {
	BaseEvent
	OrderID      string
	UserID       uint
	TrackingCode string // 物流追蹤號
	Carrier      string // 物流商
	State        uint
}

func (e *OrderShippedEvent) Type() EventType {
	return OrderShippedEventName
}

type OrderCancelledEvent struct {
	BaseEvent
	OrderID string
	UserID  uint
	Message string
	State   uint
}

func (e *OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventName
}

type OrderRefundedEvent struct {
	BaseEvent
	OrderID string
	UserID  uint
	Amount  decimal.Decimal
	State   uint
}

func (e *OrderRefundedEvent) Type() EventType {
	return OrderRefundedEventName
}
