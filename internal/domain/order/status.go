package order

import "fmt"

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusCanceled  OrderStatus = "canceled"
	StatusRefunded  OrderStatus = "refunded"
	StatusFailed    OrderStatus = "failed"
)

// String returns the string representation of the status.
func (s OrderStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid order status.
func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusPaid, StatusCanceled, StatusRefunded, StatusFailed:
		return true
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func (s OrderStatus) IsTerminal() bool {
	return s == StatusCanceled || s == StatusRefunded
}

// OrderType represents the type of order.
type OrderType string

const (
	TypeSubscription OrderType = "subscription"
	TypeTopup        OrderType = "topup"
	TypeUpgrade      OrderType = "upgrade"
)

// String returns the string representation of the order type.
func (t OrderType) String() string {
	return string(t)
}

// IsValid checks if the type is a valid order type.
func (t OrderType) IsValid() bool {
	switch t {
	case TypeSubscription, TypeTopup, TypeUpgrade:
		return true
	}
	return false
}

// transitions defines valid state transitions.
var transitions = map[OrderStatus][]OrderStatus{
	StatusPending:  {StatusPaid, StatusCanceled, StatusFailed},
	StatusPaid:     {StatusRefunded},
	StatusCanceled: {}, // Terminal state
	StatusRefunded: {}, // Terminal state
	StatusFailed:   {StatusPending}, // Can retry
}

// CanTransitionTo checks if a transition from the current status to target is valid.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	allowed, ok := transitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

// AllowedTransitions returns all allowed transitions from the current status.
func (s OrderStatus) AllowedTransitions() []OrderStatus {
	allowed, ok := transitions[s]
	if !ok {
		return []OrderStatus{}
	}
	result := make([]OrderStatus, len(allowed))
	copy(result, allowed)
	return result
}

// ErrInvalidTransition is returned when a state transition is not allowed.
var ErrInvalidTransition = fmt.Errorf("invalid state transition")
