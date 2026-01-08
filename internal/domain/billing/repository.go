package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UsageStats represents usage statistics for a user.
type UsageStats struct {
	TotalTokens      int64
	TotalRequests    int
	TotalChatTokens  int64
	TotalImageCount  int
	TotalVideoMinutes int
	TotalEmbeddingTokens int64
}

// Repository defines the interface for billing data access.
// This interface is defined in the domain layer (Port) and implemented in infra layer (Adapter).
type Repository interface {
	// Plan operations
	ListActivePlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)

	// Subscription operations
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	GetSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error)

	// Usage operations
	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*UsageStats, error)
	GetMonthlyTokenUsage(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error)
	GetDailyRequestCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error)
}
