package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
)

// CancelOrderCommand represents a command to cancel an order.
type CancelOrderCommand struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
	Reason  string
}

// CancelOrderResult is the result of canceling an order.
type CancelOrderResult struct {
	Order *order.Order
}

// CancelOrderHandler handles CancelOrderCommand.
type CancelOrderHandler struct {
	repo order.Repository
}

// NewCancelOrderHandler creates a new handler.
func NewCancelOrderHandler(repo order.Repository) *CancelOrderHandler {
	return &CancelOrderHandler{repo: repo}
}

// Handle executes the command.
func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) (*CancelOrderResult, error) {
	ord, err := h.repo.GetOrder(ctx, cmd.OrderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID() != cmd.UserID {
		return nil, order.ErrOrderNotFound // Hide existence from unauthorized users
	}

	// Use domain method to cancel
	if err := ord.Cancel(); err != nil {
		return nil, err
	}

	if err := h.repo.UpdateOrder(ctx, ord); err != nil {
		return nil, err
	}

	return &CancelOrderResult{Order: ord}, nil
}
