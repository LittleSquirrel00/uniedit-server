package billing

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
)

// GetSubscriptionQuery represents a query to get a subscription.
type GetSubscriptionQuery struct {
	UserID uuid.UUID
}

// GetSubscriptionResult is the result of getting a subscription.
type GetSubscriptionResult struct {
	Subscription *billing.Subscription
}

// GetSubscriptionHandler handles GetSubscriptionQuery.
type GetSubscriptionHandler struct {
	repo billing.Repository
}

// NewGetSubscriptionHandler creates a new handler.
func NewGetSubscriptionHandler(repo billing.Repository) *GetSubscriptionHandler {
	return &GetSubscriptionHandler{repo: repo}
}

// Handle executes the query.
func (h *GetSubscriptionHandler) Handle(ctx context.Context, query GetSubscriptionQuery) (*GetSubscriptionResult, error) {
	sub, err := h.repo.GetSubscriptionWithPlan(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	return &GetSubscriptionResult{Subscription: sub}, nil
}

// ListPlansQuery represents a query to list all active plans.
type ListPlansQuery struct{}

// ListPlansResult is the result of listing plans.
type ListPlansResult struct {
	Plans []*billing.Plan
}

// ListPlansHandler handles ListPlansQuery.
type ListPlansHandler struct {
	repo billing.Repository
}

// NewListPlansHandler creates a new handler.
func NewListPlansHandler(repo billing.Repository) *ListPlansHandler {
	return &ListPlansHandler{repo: repo}
}

// Handle executes the query.
func (h *ListPlansHandler) Handle(ctx context.Context, _ ListPlansQuery) (*ListPlansResult, error) {
	plans, err := h.repo.ListActivePlans(ctx)
	if err != nil {
		return nil, err
	}

	return &ListPlansResult{Plans: plans}, nil
}
