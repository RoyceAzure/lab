package model

import (
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/shopspring/decimal"
)

// 訂單聚合
// 訂單階段  OrderItems不會變動
// 訂單階段  只有state 會變動
type OrderAggregate struct {
	OrderID      string
	UserID       int
	OrderItems   []model.OrderItemData
	Amount       decimal.Decimal
	OrderDate    time.Time
	State        uint
	CreatedAt    time.Time
	TrackingCode string
	Carrier      string
	Message      string // 取消原因
	// 不需要 Version，EventStore 管理
}

type OrderCreatedEvent struct {
	BaseEvent
	UserID    int                   `json:"user_id"`
	OrderID   string                `json:"order_id"`
	OrderDate time.Time             `json:"order_date"`
	Items     []model.OrderItemData `json:"items"`
	Amount    decimal.Decimal       `json:"amount"`
	ToState   uint                  `json:"to_state"`
}

func (e *OrderCreatedEvent) Type() EventType {
	return OrderCreatedEventName
}

type OrderConfirmedEvent struct {
	BaseEvent
	FromState uint `json:"from_state"`
	ToState   uint `json:"to_state"`
}

func (e *OrderConfirmedEvent) Type() EventType {
	return OrderConfirmedEventName
}

type OrderShippedEvent struct {
	BaseEvent
	TrackingCode string `json:"tracking_code"`
	Carrier      string `json:"carrier"`
	FromState    uint   `json:"from_state"`
	ToState      uint   `json:"to_state"`
}

func (e *OrderShippedEvent) Type() EventType {
	return OrderShippedEventName
}

type OrderCancelledEvent struct {
	BaseEvent
	Message   string `json:"message"`
	FromState uint   `json:"from_state"`
	ToState   uint   `json:"to_state"`
}

func (e *OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventName
}

type OrderRefundedEvent struct {
	BaseEvent
	Amount    decimal.Decimal `json:"amount"`
	FromState uint            `json:"from_state"`
	ToState   uint            `json:"to_state"`
}

func (e *OrderRefundedEvent) Type() EventType {
	return OrderRefundedEventName
}
