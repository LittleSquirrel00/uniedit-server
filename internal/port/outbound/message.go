package outbound

import "context"

// EventPublisherPort defines event publishing operations.
type EventPublisherPort interface {
	// Publish publishes a domain event.
	Publish(ctx context.Context, event interface{}) error
}

// MessagePort defines message queue operations.
type MessagePort interface {
	// Publish publishes a message to a topic.
	Publish(ctx context.Context, topic string, message []byte) error

	// Subscribe subscribes to a topic with a handler.
	Subscribe(ctx context.Context, topic string, handler func([]byte) error) error
}
