package model

import (
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
)

type CartCreatedEvent struct {
	BaseEvent
	UserID int
	Items  []model.OrderItemData
}

func NewCartCreatedEvent(aggregateID string, userID int, items []model.OrderItemData) *CartCreatedEvent {
	return &CartCreatedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, CartCreatedEventName),
		UserID:    userID,
		Items:     items,
	}
}

func (e *CartCreatedEvent) Type() EventType {
	return CartCreatedEventName
}

type CartFailedEvent struct {
	BaseEvent
	UserID  int
	Message string
}

func NewCartFailedEvent(aggregateID string, userID int, message string) *CartFailedEvent {
	return &CartFailedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, CartFailedEventName),
		UserID:    userID,
		Message:   message,
	}
}

func (e *CartFailedEvent) Type() EventType {
	return CartFailedEventName
}

type CartUpdatedEvent struct {
	BaseEvent
	UserID  int
	Details []cmd_model.CartUpdatedDetial
}

func NewCartUpdatedEvent(aggregateID string, userID int, details []cmd_model.CartUpdatedDetial) *CartUpdatedEvent {
	return &CartUpdatedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, CartUpdatedEventName),
		UserID:    userID,
		Details:   details,
	}
}

func (e *CartUpdatedEvent) Type() EventType {
	return CartUpdatedEventName
}

type CartDeletedEvent struct {
	BaseEvent
	UserID int
}

func NewCartDeletedEvent(aggregateID string, userID int) *CartDeletedEvent {
	return &CartDeletedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, CartDeletedEventName),
		UserID:    userID,
	}
}

func (e *CartDeletedEvent) Type() EventType {
	return CartDeletedEventName
}

type CartConfirmedEvent struct {
	BaseEvent
	UserID int
}

func NewCartConfirmedEvent(aggregateID string, userID int) *CartConfirmedEvent {
	return &CartConfirmedEvent{
		BaseEvent: *NewBaseEvent(aggregateID, CartConfirmedEventName),
		UserID:    userID,
	}
}

func (e *CartConfirmedEvent) Type() EventType {
	return CartConfirmedEventName
}
