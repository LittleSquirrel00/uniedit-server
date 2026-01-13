package outbound

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

var ErrCacheMiss = errors.New("cache miss")

// PlanDatabasePort defines plan persistence operations.
type PlanDatabasePort interface {
	// ListActive lists all active plans.
	ListActive(ctx context.Context) ([]*model.Plan, error)

	// GetByID gets a plan by ID.
	GetByID(ctx context.Context, id string) (*model.Plan, error)

	// Create creates a new plan.
	Create(ctx context.Context, plan *model.Plan) error

	// Update updates a plan.
	Update(ctx context.Context, plan *model.Plan) error
}

// SubscriptionDatabasePort defines subscription persistence operations.
type SubscriptionDatabasePort interface {
	// Create creates a new subscription.
	Create(ctx context.Context, sub *model.Subscription) error

	// GetByUserID gets a subscription by user ID.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Subscription, error)

	// GetByUserIDWithPlan gets a subscription by user ID with plan loaded.
	GetByUserIDWithPlan(ctx context.Context, userID uuid.UUID) (*model.Subscription, error)

	// GetByStripeID gets a subscription by Stripe subscription ID.
	GetByStripeID(ctx context.Context, stripeSubID string) (*model.Subscription, error)

	// Update updates a subscription.
	Update(ctx context.Context, sub *model.Subscription) error

	// UpdateCredits updates the credits balance.
	UpdateCredits(ctx context.Context, userID uuid.UUID, amount int64) error

	// TryDeductCredits deducts credits if balance is sufficient (atomic).
	// Returns (true, nil) if deducted successfully, (false, nil) if insufficient.
	TryDeductCredits(ctx context.Context, userID uuid.UUID, amount int64) (bool, error)
}

// UsageRecordDatabasePort defines usage record persistence operations.
type UsageRecordDatabasePort interface {
	// Create creates a new usage record.
	Create(ctx context.Context, record *model.UsageRecord) error

	// GetStats gets aggregated usage statistics.
	GetStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*model.UsageStats, error)

	// GetMonthlyTokens gets total tokens used in a period.
	GetMonthlyTokens(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error)

	// GetMonthlyTokensByTaskType gets total tokens used for a task type in a period.
	GetMonthlyTokensByTaskType(ctx context.Context, userID uuid.UUID, start time.Time, taskType string) (int64, error)

	// GetDailyRequests gets request count for a day.
	GetDailyRequests(ctx context.Context, userID uuid.UUID, date time.Time) (int, error)

	// GetMonthlyUnitsByTaskType gets task-specific units used in a period.
	// For non-token tasks (e.g., image/video), units are stored in UsageRecord.InputTokens.
	GetMonthlyUnitsByTaskType(ctx context.Context, userID uuid.UUID, start time.Time, taskType string) (int64, error)
}

// QuotaCachePort defines quota caching operations (Redis).
type QuotaCachePort interface {
	// GetTokensUsed gets tokens used from cache.
	GetTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error)

	// IncrementTokens increments token counter.
	IncrementTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) (int64, error)

	// GetRequestsToday gets today's request count.
	GetRequestsToday(ctx context.Context, userID uuid.UUID) (int, error)

	// IncrementRequests increments request counter.
	IncrementRequests(ctx context.Context, userID uuid.UUID) (int, error)

	// ResetTokens resets token counter for a period.
	ResetTokens(ctx context.Context, userID uuid.UUID, periodStart time.Time) error

	// GetMediaUnitsUsed gets monthly units used for a media task type (e.g. image/video).
	GetMediaUnitsUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time, taskType string) (int64, error)

	// IncrementMediaUnits increments monthly units for a media task type.
	IncrementMediaUnits(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, taskType string, units int64) (int64, error)

	// ResetMediaUnits resets monthly units for a media task type.
	ResetMediaUnits(ctx context.Context, userID uuid.UUID, periodStart time.Time, taskType string) error
}

// StripePort defines Stripe payment operations.
type StripePort interface {
	// CreateCustomer creates a new Stripe customer.
	CreateCustomer(ctx context.Context, email, name string) (string, error)

	// CreateSubscription creates a new Stripe subscription.
	CreateSubscription(ctx context.Context, customerID, priceID string) (string, error)

	// CancelSubscription cancels a Stripe subscription.
	CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

	// CreateCheckoutSession creates a checkout session for subscription.
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error)

	// CreatePortalSession creates a customer portal session.
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)

	// GetSubscription gets subscription details from Stripe.
	GetSubscription(ctx context.Context, subscriptionID string) (*StripeSubscriptionInfo, error)
}

// StripeSubscriptionInfo represents Stripe subscription information.
type StripeSubscriptionInfo struct {
	ID                 string
	Status             string
	CustomerID         string
	PriceID            string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
}
