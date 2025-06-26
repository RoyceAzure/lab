package model

import (
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/shopspring/decimal"
)

type OrderAggregate struct {
	OrderID    string
	UserID     int
	OrderItems []model.OrderItemData
	Amount     decimal.Decimal
	OrderDate  time.Time
	State      uint
	// 不需要 Version，EventStore 管理
}

type OrderCreatedEvent struct {
	BaseEvent
	Items     []model.OrderItemData
	Amount    decimal.Decimal
	FromState uint
	ToState   uint
}

func (e *OrderCreatedEvent) Type() EventType {
	return OrderCreatedEventName
}

type OrderConfirmedEvent struct {
	BaseEvent
	FromState uint
	ToState   uint
}

func (e *OrderConfirmedEvent) Type() EventType {
	return OrderConfirmedEventName
}

type OrderShippedEvent struct {
	BaseEvent
	TrackingCode string // 物流追蹤號
	Carrier      string // 物流商
	FromState    uint
	ToState      uint
}

func (e *OrderShippedEvent) Type() EventType {
	return OrderShippedEventName
}

type OrderCancelledEvent struct {
	BaseEvent
	Message   string
	Items     []model.OrderItemData
	FromState uint
	ToState   uint
}

func (e *OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventName
}

type OrderRefundedEvent struct {
	BaseEvent
	Amount    decimal.Decimal
	FromState uint
	ToState   uint
}

func (e *OrderRefundedEvent) Type() EventType {
	return OrderRefundedEventName
}
