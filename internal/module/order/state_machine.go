package order

import "fmt"

// StateMachine validates and executes order state transitions.
type StateMachine struct {
	transitions map[OrderStatus][]OrderStatus
}

// NewStateMachine creates a new order state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		transitions: map[OrderStatus][]OrderStatus{
			OrderStatusPending:  {OrderStatusPaid, OrderStatusCanceled, OrderStatusFailed},
			OrderStatusPaid:     {OrderStatusRefunded},
			OrderStatusCanceled: {}, // Terminal state
			OrderStatusRefunded: {}, // Terminal state
			OrderStatusFailed:   {OrderStatusPending}, // Can retry
		},
	}
}

// CanTransition checks if a transition from `from` to `to` is valid.
func (sm *StateMachine) CanTransition(from, to OrderStatus) bool {
	allowed, ok := sm.transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// Transition attempts to transition an order to a new state.
func (sm *StateMachine) Transition(order *Order, to OrderStatus) error {
	if !sm.CanTransition(order.Status, to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, order.Status, to)
	}
	order.Status = to
	return nil
}

// GetAllowedTransitions returns all allowed transitions from the current state.
func (sm *StateMachine) GetAllowedTransitions(from OrderStatus) []OrderStatus {
	allowed, ok := sm.transitions[from]
	if !ok {
		return []OrderStatus{}
	}
	result := make([]OrderStatus, len(allowed))
	copy(result, allowed)
	return result
}
