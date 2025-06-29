package model

import "time"

type BaseEvent struct {
	EventID     string    `json:"eventId"`
	AggregateID string    `json:"aggregateId"`
	CreatedAt   time.Time `json:"createdAt"`
	EventType   EventType `json:"eventType"`
}

func (e *BaseEvent) GetID() string {
	return e.EventID
}

type EventType string

const (
	OrderCreatedEventName   EventType = "OrderCreated"
	OrderConfirmedEventName EventType = "OrderConfirmed"
	OrderShippedEventName   EventType = "OrderShipped"
	OrderCancelledEventName EventType = "OrderCancelled"
	OrderRefundedEventName  EventType = "OrderRefunded"
	CartCreatedEventName    EventType = "CartCreated"
	CartFailedEventName     EventType = "CartActionFailed"
	CartUpdatedEventName    EventType = "CartUpdated"
	CartDeletedEventName    EventType = "CartDeleted"
)

type Event interface {
	Type() EventType
	GetID() string
}
