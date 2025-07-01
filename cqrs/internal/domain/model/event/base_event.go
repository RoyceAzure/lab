package model

import (
	"time"

	"github.com/google/uuid"
)

type BaseEvent struct {
	EventID     uuid.UUID `json:"eventId"`
	AggregateID string    `json:"aggregateId"`
	CreatedAt   time.Time `json:"createdAt"`
	EventType   EventType `json:"eventType"`
}

func NewBaseEvent(aggregateID string, eventType EventType) *BaseEvent {
	return &BaseEvent{
		EventID:     uuid.New(),
		AggregateID: aggregateID,
		CreatedAt:   time.Now().UTC(),
		EventType:   EventType(eventType),
	}
}

func (e *BaseEvent) GetID() string {
	return e.EventID.String()
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
	CartConfirmedEventName  EventType = "CartConfirmed"
)

type Event interface {
	Type() EventType
	GetID() string
}
