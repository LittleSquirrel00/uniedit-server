package events

// Handler is the interface for event handlers.
type Handler interface {
	// Handles returns the list of event types this handler can process.
	Handles() []string

	// Handle processes the given event.
	// Implementations should be idempotent - handling the same event twice
	// should not produce duplicate side effects.
	Handle(event Event) error
}

// HandlerFunc is a function type that implements Handler for a single event type.
type HandlerFunc struct {
	eventTypes []string
	fn         func(Event) error
}

// NewHandlerFunc creates a new HandlerFunc.
func NewHandlerFunc(eventTypes []string, fn func(Event) error) *HandlerFunc {
	return &HandlerFunc{
		eventTypes: eventTypes,
		fn:         fn,
	}
}

// Handles returns the list of event types this handler can process.
func (h *HandlerFunc) Handles() []string {
	return h.eventTypes
}

// Handle processes the given event.
func (h *HandlerFunc) Handle(event Event) error {
	return h.fn(event)
}
