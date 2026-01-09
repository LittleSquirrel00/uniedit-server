package events

import (
	"sync"

	"go.uber.org/zap"
)

// Bus is a simple synchronous event bus for domain events.
// It dispatches events to registered handlers synchronously.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   *zap.Logger
}

// NewBus creates a new event bus.
func NewBus(logger *zap.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

// Register registers a handler for the events it handles.
func (b *Bus) Register(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, eventType := range handler.Handles() {
		b.handlers[eventType] = append(b.handlers[eventType], handler)
		b.logger.Debug("registered event handler",
			zap.String("event_type", eventType),
		)
	}
}

// Publish dispatches an event to all registered handlers.
// Handlers are called synchronously in registration order.
// If a handler fails, the error is logged but other handlers continue processing.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.EventType()]
	b.mu.RUnlock()

	if len(handlers) == 0 {
		b.logger.Debug("no handlers registered for event",
			zap.String("event_type", event.EventType()),
			zap.String("event_id", event.EventID().String()),
		)
		return
	}

	b.logger.Info("publishing event",
		zap.String("event_type", event.EventType()),
		zap.String("event_id", event.EventID().String()),
		zap.String("aggregate_id", event.AggregateID().String()),
		zap.Int("handler_count", len(handlers)),
	)

	for _, handler := range handlers {
		if err := handler.Handle(event); err != nil {
			// Log error but continue with other handlers (error isolation)
			b.logger.Error("event handler failed",
				zap.String("event_type", event.EventType()),
				zap.String("event_id", event.EventID().String()),
				zap.Error(err),
			)
		}
	}
}

// PublishAll dispatches multiple events.
func (b *Bus) PublishAll(events []Event) {
	for _, event := range events {
		b.Publish(event)
	}
}
