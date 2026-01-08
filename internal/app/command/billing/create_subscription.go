package billing

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
)

// CreateSubscriptionCommand represents a command to create a subscription.
type CreateSubscriptionCommand struct {
	UserID uuid.UUID
	PlanID string
}

// CreateSubscriptionResult is the result of creating a subscription.
type CreateSubscriptionResult struct {
	Subscription *billing.Subscription
}

// CreateSubscriptionHandler handles CreateSubscriptionCommand.
type CreateSubscriptionHandler struct {
	repo billing.Repository
}

// NewCreateSubscriptionHandler creates a new handler.
func NewCreateSubscriptionHandler(repo billing.Repository) *CreateSubscriptionHandler {
	return &CreateSubscriptionHandler{repo: repo}
}

// Handle executes the command.
func (h *CreateSubscriptionHandler) Handle(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
	// Check if subscription already exists
	existing, err := h.repo.GetSubscription(ctx, cmd.UserID)
	if err == nil && existing != nil {
		return nil, billing.ErrSubscriptionExists
	}

	// Get the plan
	plan, err := h.repo.GetPlan(ctx, cmd.PlanID)
	if err != nil {
		return nil, err
	}

	if !plan.Active() {
		return nil, billing.ErrPlanNotActive
	}

	// Create subscription
	sub, err := billing.NewSubscription(cmd.UserID, cmd.PlanID)
	if err != nil {
		return nil, err
	}

	if err := h.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return &CreateSubscriptionResult{Subscription: sub}, nil
}
