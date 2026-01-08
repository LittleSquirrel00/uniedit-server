package billing

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
)

// CancelSubscriptionCommand represents a command to cancel a subscription.
type CancelSubscriptionCommand struct {
	UserID      uuid.UUID
	Immediately bool
}

// CancelSubscriptionResult is the result of canceling a subscription.
type CancelSubscriptionResult struct {
	Subscription *billing.Subscription
}

// CancelSubscriptionHandler handles CancelSubscriptionCommand.
type CancelSubscriptionHandler struct {
	repo billing.Repository
}

// NewCancelSubscriptionHandler creates a new handler.
func NewCancelSubscriptionHandler(repo billing.Repository) *CancelSubscriptionHandler {
	return &CancelSubscriptionHandler{repo: repo}
}

// Handle executes the command.
func (h *CancelSubscriptionHandler) Handle(ctx context.Context, cmd CancelSubscriptionCommand) (*CancelSubscriptionResult, error) {
	sub, err := h.repo.GetSubscription(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	// Use domain method to cancel
	if err := sub.Cancel(cmd.Immediately); err != nil {
		return nil, err
	}

	if err := h.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return &CancelSubscriptionResult{Subscription: sub}, nil
}
