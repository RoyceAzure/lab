package model

import (
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/shopspring/decimal"
)

type OrderState uint

const (
	OrderStateCreated OrderState = iota
	OrderStateConfirmed
	OrderStateShipped
	OrderStateCancelled
)

// 訂單聚合
// 訂單階段  OrderItems不會變動
// 訂單階段  只有state 會變動
type OrderAggregate struct {
	OrderID      string                `json:"order_id"`
	UserID       int                   `json:"user_id"`
	OrderItems   []model.OrderItemData `json:"items"`
	Amount       decimal.Decimal       `json:"amount"`
	OrderDate    time.Time             `json:"order_date"`
	State        uint                  `json:"state"`
	CreatedAt    time.Time             `json:"created_at"`
	TrackingCode string                `json:"tracking_code"`
	Carrier      string                `json:"carrier"`
	Message      string                `json:"message"` // 取消原因
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

func NewOrderCreatedEvent(aggregateID string, userID int, orderID string, orderDate time.Time, items []model.OrderItemData, amount decimal.Decimal, toState uint) *OrderCreatedEvent {
	return &OrderCreatedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, OrderCreatedEventName),
		UserID:    userID,
		OrderID:   orderID,
		OrderDate: orderDate,
		Items:     items,
		Amount:    amount,
		ToState:   toState,
	}
}

func (e *OrderCreatedEvent) Type() EventType {
	return OrderCreatedEventName
}

type OrderConfirmedEvent struct {
	BaseEvent
	FromState uint `json:"from_state"`
	ToState   uint `json:"to_state"`
}

func NewOrderConfirmedEvent(aggregateID string, fromState uint, toState uint) *OrderConfirmedEvent {
	return &OrderConfirmedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, OrderConfirmedEventName),
		FromState: fromState,
		ToState:   toState,
	}
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

func NewOrderShippedEvent(aggregateID string, trackingCode string, carrier string, fromState uint, toState uint) *OrderShippedEvent {
	return &OrderShippedEvent{
		BaseEvent:    *NewBaseEvent(aggregateID, OrderShippedEventName),
		TrackingCode: trackingCode,
		Carrier:      carrier,
		FromState:    fromState,
		ToState:      toState,
	}
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

func NewOrderCancelledEvent(aggregateID string, message string, fromState uint, toState uint) *OrderCancelledEvent {
	return &OrderCancelledEvent{
		BaseEvent: *NewBaseEvent(aggregateID, OrderCancelledEventName),
		Message:   message,
		FromState: fromState,
		ToState:   toState,
	}
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

func NewOrderRefundedEvent(aggregateID string, amount decimal.Decimal, fromState uint, toState uint) *OrderRefundedEvent {
	return &OrderRefundedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, OrderRefundedEventName),
		Amount:    amount,
		FromState: fromState,
		ToState:   toState,
	}
}
func (e *OrderRefundedEvent) Type() EventType {
	return OrderRefundedEventName
}
