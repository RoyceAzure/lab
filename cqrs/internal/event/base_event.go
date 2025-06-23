package event

import "time"

type BaseEvent struct {
	EventID     string    `json:"event_id"`
	AggregateID string    `json:"aggregate_id"`
	CreatedAt   time.Time `json:"created_at"`
	EventType   EventType `json:"event_type"`
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
