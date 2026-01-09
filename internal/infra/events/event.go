package events

import (
	"time"

	"github.com/google/uuid"
)

// Event is the interface that all domain events must implement.
type Event interface {
	// EventID returns the unique identifier for this event instance.
	EventID() uuid.UUID

	// EventType returns the type name of the event (e.g., "PaymentSucceeded").
	EventType() string

	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time

	// AggregateID returns the ID of the aggregate that produced this event.
	AggregateID() uuid.UUID

	// AggregateType returns the type of aggregate (e.g., "Payment", "Order").
	AggregateType() string
}

// BaseEvent provides a base implementation of the Event interface.
// Embed this struct in your domain events to inherit common fields.
type BaseEvent struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	AggregateUUID uuid.UUID `json:"aggregate_id"`
	AggregateName string    `json:"aggregate_type"`
}

// EventID returns the unique identifier for this event instance.
func (e BaseEvent) EventID() uuid.UUID {
	return e.ID
}

// EventType returns the type name of the event.
func (e BaseEvent) EventType() string {
	return e.Type
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that produced this event.
func (e BaseEvent) AggregateID() uuid.UUID {
	return e.AggregateUUID
}

// AggregateType returns the type of aggregate.
func (e BaseEvent) AggregateType() string {
	return e.AggregateName
}

// NewBaseEvent creates a new BaseEvent with the given parameters.
func NewBaseEvent(eventType string, aggregateID uuid.UUID, aggregateType string) BaseEvent {
	return BaseEvent{
		ID:            uuid.New(),
		Type:          eventType,
		Timestamp:     time.Now(),
		AggregateUUID: aggregateID,
		AggregateName: aggregateType,
	}
}
