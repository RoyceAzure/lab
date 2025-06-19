package event

import (
	"github.com/shopspring/decimal"
)

type EventType string

const (
	OrderCreatedEventName   EventType = "OrderCreated"
	OrderConfirmedEventName EventType = "OrderConfirmed"
	OrderShippedEventName   EventType = "OrderShipped"
	OrderCancelledEventName EventType = "OrderCancelled"
	OrderRefundedEventName  EventType = "OrderRefunded"
)

type Event interface {
	Type() EventType
}

type OrderCreatedEvent struct {
	BaseEvent
	OrderID uint
	UserID  uint
	Items   []OrderItemData
}

func (e *OrderCreatedEvent) Type() EventType {
	return OrderCreatedEventName
}

type OrderItemData struct {
	BaseEvent
	ProductID   uint
	Quantity    int
	Price       decimal.Decimal
	Amount      decimal.Decimal
	ProductName string
}

type OrderConfirmedEvent struct {
	BaseEvent
	OrderID uint
	UserID  uint
}

func (e *OrderConfirmedEvent) Type() EventType {
	return OrderConfirmedEventName
}

type OrderShippedEvent struct {
	BaseEvent
	OrderID      uint
	UserID       uint
	TrackingCode string // 物流追蹤號
	Carrier      string // 物流商
}

func (e *OrderShippedEvent) Type() EventType {
	return OrderShippedEventName
}

type OrderCancelledEvent struct {
	BaseEvent
	OrderID uint
	UserID  uint
	Message string
}

func (e *OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventName
}

type OrderRefundedEvent struct {
	BaseEvent
	OrderID uint
	UserID  uint
	Amount  decimal.Decimal
}

func (e *OrderRefundedEvent) Type() EventType {
	return OrderRefundedEventName
}
