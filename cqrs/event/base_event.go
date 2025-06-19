package event

import "time"

type BaseEvent struct {
	EventID     string    `json:"event_id"`
	AggregateID string    `json:"aggregate_id"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}
