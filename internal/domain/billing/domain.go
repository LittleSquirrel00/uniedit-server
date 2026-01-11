package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// BillingDomain defines billing domain service interface.
type BillingDomain interface {
	// Plan operations
	ListPlans(ctx context.Context) ([]*model.Plan, error)
	GetPlan(ctx context.Context, planID string) (*model.Plan, error)

	// Subscription operations
	GetSubscription(ctx context.Context, userID uuid.UUID) (*model.Subscription, error)
	CreateSubscription(ctx context.Context, userID uuid.UUID, planID string) (*model.Subscription, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID, immediately bool) (*model.Subscription, error)

	// Quota operations
	GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*model.QuotaStatus, error)
	CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error
	ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error

	// Usage operations
	GetUsageStats(ctx context.Context, userID uuid.UUID, period string, start, end *time.Time) (*model.UsageStats, error)
	RecordUsage(ctx context.Context, userID uuid.UUID, record *RecordUsageInput) error

	// Credits operations
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
	DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error

	// Stripe sync
	UpdateSubscriptionFromStripe(ctx context.Context, stripeSubID string, status model.SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) error
}

// RecordUsageInput represents input for recording usage.
type RecordUsageInput struct {
	RequestID    string
	TaskType     string
	ProviderID   uuid.UUID
	ModelID      string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	LatencyMs    int
	Success      bool
}

// billingDomain implements BillingDomain.
type billingDomain struct {
	planDB         outbound.PlanDatabasePort
	subscriptionDB outbound.SubscriptionDatabasePort
	usageDB        outbound.UsageRecordDatabasePort
	quotaCache     outbound.QuotaCachePort
	logger         *zap.Logger
}

// NewBillingDomain creates a new billing domain service.
func NewBillingDomain(
	planDB outbound.PlanDatabasePort,
	subscriptionDB outbound.SubscriptionDatabasePort,
	usageDB outbound.UsageRecordDatabasePort,
	quotaCache outbound.QuotaCachePort,
	logger *zap.Logger,
) BillingDomain {
	return &billingDomain{
		planDB:         planDB,
		subscriptionDB: subscriptionDB,
		usageDB:        usageDB,
		quotaCache:     quotaCache,
		logger:         logger,
	}
}

// --- Plan Operations ---

func (d *billingDomain) ListPlans(ctx context.Context) ([]*model.Plan, error) {
	return d.planDB.ListActive(ctx)
}

func (d *billingDomain) GetPlan(ctx context.Context, planID string) (*model.Plan, error) {
	plan, err := d.planDB.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// --- Subscription Operations ---

func (d *billingDomain) GetSubscription(ctx context.Context, userID uuid.UUID) (*model.Subscription, error) {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (d *billingDomain) CreateSubscription(ctx context.Context, userID uuid.UUID, planID string) (*model.Subscription, error) {
	// Check if subscription already exists
	existing, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err == nil && existing != nil {
		return nil, ErrSubscriptionExists
	}

	// Get plan to verify it exists and is active
	plan, err := d.planDB.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	if !plan.Active {
		return nil, ErrPlanNotActive
	}

	// Create subscription
	now := time.Now()
	periodEnd := endOfMonth(now)

	sub := &model.Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             planID,
		Status:             model.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := d.subscriptionDB.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	// Reload with plan for response
	return d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
}

func (d *billingDomain) CancelSubscription(ctx context.Context, userID uuid.UUID, immediately bool) (*model.Subscription, error) {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}

	if sub.IsCanceled() {
		return nil, ErrSubscriptionCanceled
	}

	now := time.Now()
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if immediately {
		sub.Status = model.SubscriptionStatusCanceled
		sub.PlanID = "free" // Downgrade to free plan
	} else {
		sub.CancelAtPeriodEnd = true
	}

	if err := d.subscriptionDB.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	return sub, nil
}

// --- Quota Operations ---

func (d *billingDomain) GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*model.QuotaStatus, error) {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}

	plan := sub.Plan
	if plan == nil {
		return nil, fmt.Errorf("subscription has no plan loaded")
	}

	// Get current usage from cache
	tokensUsed, err := d.quotaCache.GetTokensUsed(ctx, userID, sub.CurrentPeriodStart)
	if err != nil {
		d.logger.Warn("failed to get tokens from cache, using DB", zap.Error(err))
		tokensUsed, _ = d.usageDB.GetMonthlyTokens(ctx, userID, sub.CurrentPeriodStart)
	}

	requestsToday, err := d.quotaCache.GetRequestsToday(ctx, userID)
	if err != nil {
		d.logger.Warn("failed to get requests from cache, using DB", zap.Error(err))
		requestsToday, _ = d.usageDB.GetDailyRequests(ctx, userID, time.Now().UTC())
	}

	tokenLimit := plan.MonthlyTokens
	tokensRemaining := tokenLimit - tokensUsed
	if tokenLimit == -1 {
		tokensRemaining = -1
	} else if tokensRemaining < 0 {
		tokensRemaining = 0
	}

	return &model.QuotaStatus{
		Plan:            plan.Name,
		TokensUsed:      tokensUsed,
		TokensLimit:     tokenLimit,
		TokensRemaining: tokensRemaining,
		RequestsToday:   requestsToday,
		RequestsLimit:   plan.DailyRequests,
		ResetAt:         sub.CurrentPeriodEnd,
	}, nil
}

func (d *billingDomain) CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrQuotaExceeded
	}

	if !sub.IsActive() {
		return ErrQuotaExceeded
	}

	plan := sub.Plan
	if plan == nil {
		return fmt.Errorf("subscription has no plan loaded")
	}

	// Check token limit
	if !plan.IsUnlimitedTokens() {
		tokensUsed, err := d.quotaCache.GetTokensUsed(ctx, userID, sub.CurrentPeriodStart)
		if err != nil {
			tokensUsed, _ = d.usageDB.GetMonthlyTokens(ctx, userID, sub.CurrentPeriodStart)
		}
		if tokensUsed >= plan.MonthlyTokens {
			return ErrTokenLimitReached
		}
	}

	// Check request limit
	if !plan.IsUnlimitedRequests() {
		requestsToday, err := d.quotaCache.GetRequestsToday(ctx, userID)
		if err != nil {
			requestsToday, _ = d.usageDB.GetDailyRequests(ctx, userID, time.Now().UTC())
		}
		if requestsToday >= plan.DailyRequests {
			return ErrRequestLimitReached
		}
	}

	return nil
}

func (d *billingDomain) ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error {
	sub, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	// Increment token counter
	_, err = d.quotaCache.IncrementTokens(ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, int64(tokens))
	if err != nil {
		d.logger.Error("failed to increment tokens in cache", zap.Error(err))
	}

	// Increment request counter
	_, err = d.quotaCache.IncrementRequests(ctx, userID)
	if err != nil {
		d.logger.Error("failed to increment requests in cache", zap.Error(err))
	}

	return nil
}

// --- Usage Operations ---

func (d *billingDomain) GetUsageStats(ctx context.Context, userID uuid.UUID, period string, start, end *time.Time) (*model.UsageStats, error) {
	now := time.Now().UTC()
	var startTime, endTime time.Time

	if start != nil && end != nil {
		startTime = *start
		endTime = *end
	} else {
		switch period {
		case "day":
			startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			endTime = startTime.Add(24 * time.Hour)
		case "week":
			startTime = now.AddDate(0, 0, -7)
			endTime = now
		case "month":
			startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			endTime = startTime.AddDate(0, 1, 0)
		default:
			startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			endTime = startTime.AddDate(0, 1, 0)
		}
	}

	return d.usageDB.GetStats(ctx, userID, startTime, endTime)
}

func (d *billingDomain) RecordUsage(ctx context.Context, userID uuid.UUID, input *RecordUsageInput) error {
	record := &model.UsageRecord{
		UserID:       userID,
		Timestamp:    time.Now(),
		RequestID:    input.RequestID,
		TaskType:     input.TaskType,
		ProviderID:   input.ProviderID,
		ModelID:      input.ModelID,
		InputTokens:  input.InputTokens,
		OutputTokens: input.OutputTokens,
		TotalTokens:  input.InputTokens + input.OutputTokens,
		CostUSD:      input.CostUSD,
		LatencyMs:    input.LatencyMs,
		Success:      input.Success,
	}

	if err := d.usageDB.Create(ctx, record); err != nil {
		return fmt.Errorf("create usage record: %w", err)
	}

	// Update quota counters if successful
	if input.Success {
		if err := d.ConsumeQuota(ctx, userID, record.TotalTokens); err != nil {
			d.logger.Error("failed to consume quota", zap.Error(err))
		}
	}

	return nil
}

// --- Credits Operations ---

func (d *billingDomain) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	sub, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if sub == nil {
		return 0, ErrSubscriptionNotFound
	}
	return sub.CreditsBalance, nil
}

func (d *billingDomain) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
	if amount <= 0 {
		return ErrInvalidCreditsAmount
	}

	sub, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	sub.CreditsBalance += amount
	sub.UpdatedAt = time.Now()

	if err := d.subscriptionDB.Update(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	d.logger.Info("credits added",
		zap.String("user_id", userID.String()),
		zap.Int64("amount", amount),
		zap.String("source", source),
	)

	return nil
}

func (d *billingDomain) DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error {
	if amount <= 0 {
		return ErrInvalidCreditsAmount
	}

	sub, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	if sub.CreditsBalance < amount {
		return ErrInsufficientCredits
	}

	sub.CreditsBalance -= amount
	sub.UpdatedAt = time.Now()

	if err := d.subscriptionDB.Update(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	d.logger.Info("credits deducted",
		zap.String("user_id", userID.String()),
		zap.Int64("amount", amount),
		zap.String("reason", reason),
	)

	return nil
}

// --- Stripe Sync ---

func (d *billingDomain) UpdateSubscriptionFromStripe(ctx context.Context, stripeSubID string, status model.SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) error {
	sub, err := d.subscriptionDB.GetByStripeID(ctx, stripeSubID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	oldPeriodStart := sub.CurrentPeriodStart

	sub.Status = status
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd
	sub.CancelAtPeriodEnd = cancelAtPeriodEnd
	sub.UpdatedAt = time.Now()

	if err := d.subscriptionDB.Update(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	// Reset quota counters on period renewal
	if periodStart.After(oldPeriodStart) {
		if err := d.quotaCache.ResetTokens(ctx, sub.UserID, oldPeriodStart); err != nil {
			d.logger.Error("failed to reset tokens", zap.Error(err))
		}
	}

	return nil
}

// --- Helpers ---

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
}
