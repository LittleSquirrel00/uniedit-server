package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/uniedit/server/api/pb/billing"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// Domain implements billing business logic.
// Inbound request/response uses pb types; DB read/write uses internal/model only at the persistence boundary.
type Domain struct {
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
) *Domain {
	return &Domain{
		planDB:         planDB,
		subscriptionDB: subscriptionDB,
		usageDB:        usageDB,
		quotaCache:     quotaCache,
		logger:         logger,
	}
}

// Compile-time interface check
var _ inbound.BillingDomain = (*Domain)(nil)

// --- Plan Operations ---

func (d *Domain) ListPlans(ctx context.Context) (*billingv1.ListPlansResponse, error) {
	plans, err := d.planDB.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*billingv1.Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, toPlanPB(p))
	}
	return &billingv1.ListPlansResponse{Plans: out}, nil
}

func (d *Domain) GetPlan(ctx context.Context, in *billingv1.GetByIDRequest) (*billingv1.Plan, error) {
	if in == nil || strings.TrimSpace(in.GetId()) == "" {
		return nil, ErrInvalidRequest
	}

	plan, err := d.planDB.GetByID(ctx, in.GetId())
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return toPlanPB(plan), nil
}

// --- Subscription Operations ---

func (d *Domain) GetSubscription(ctx context.Context, userID uuid.UUID) (*billingv1.Subscription, error) {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return toSubscriptionPB(sub), nil
}

func (d *Domain) CreateSubscription(ctx context.Context, userID uuid.UUID, in *billingv1.CreateSubscriptionRequest) (*billingv1.Subscription, error) {
	if in == nil || strings.TrimSpace(in.GetPlanId()) == "" {
		return nil, ErrInvalidRequest
	}

	// Check if subscription already exists
	existing, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err == nil && existing != nil {
		return nil, ErrSubscriptionExists
	}

	// Get plan to verify it exists and is active
	plan, err := d.planDB.GetByID(ctx, in.GetPlanId())
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
		PlanID:             in.GetPlanId(),
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
	reloaded, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if reloaded == nil {
		return nil, ErrSubscriptionNotFound
	}
	return toSubscriptionPB(reloaded), nil
}

func (d *Domain) CancelSubscription(ctx context.Context, userID uuid.UUID, in *billingv1.CancelSubscriptionRequest) (*billingv1.Subscription, error) {
	immediately := false
	if in != nil {
		immediately = in.GetImmediately()
	}

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

	return toSubscriptionPB(sub), nil
}

// --- Quota Operations ---

func (d *Domain) GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*billingv1.QuotaStatus, error) {
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

	return &billingv1.QuotaStatus{
		Plan:            plan.Name,
		TokensUsed:      tokensUsed,
		TokensLimit:     tokenLimit,
		TokensRemaining: tokensRemaining,
		RequestsToday:   int32(requestsToday),
		RequestsLimit:   int32(plan.DailyRequests),
		ResetAt:         sub.CurrentPeriodEnd.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (d *Domain) CheckQuota(ctx context.Context, userID uuid.UUID, in *billingv1.CheckQuotaRequest) error {
	_ = in // task_type currently not used for billing checks

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

func (d *Domain) ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error {
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

func (d *Domain) GetUsageStats(ctx context.Context, userID uuid.UUID, in *billingv1.GetUsageStatsRequest) (*billingv1.UsageStats, error) {
	period := "month"
	if in != nil && strings.TrimSpace(in.GetPeriod()) != "" {
		period = in.GetPeriod()
	}

	now := time.Now().UTC()
	var startTime, endTime time.Time

	var startStr, endStr string
	if in != nil {
		startStr, endStr = in.GetStart(), in.GetEnd()
	}
	startParsed, endParsed := parseRFC3339OrNil(startStr), parseRFC3339OrNil(endStr)
	if startParsed != nil && endParsed != nil {
		startTime = *startParsed
		endTime = *endParsed
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

	stats, err := d.usageDB.GetStats(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	return toUsageStatsPB(stats), nil
}

func (d *Domain) RecordUsage(ctx context.Context, userID uuid.UUID, in *billingv1.RecordUsageRequest) (*commonv1.MessageResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	providerID, err := uuid.Parse(in.GetProviderId())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid provider_id", ErrInvalidRequest)
	}

	record := &model.UsageRecord{
		UserID:       userID,
		Timestamp:    time.Now(),
		RequestID:    in.GetRequestId(),
		TaskType:     in.GetTaskType(),
		ProviderID:   providerID,
		ModelID:      in.GetModelId(),
		InputTokens:  int(in.GetInputTokens()),
		OutputTokens: int(in.GetOutputTokens()),
		TotalTokens:  int(in.GetInputTokens()) + int(in.GetOutputTokens()),
		CostUSD:      in.GetCostUsd(),
		LatencyMs:    int(in.GetLatencyMs()),
		Success:      in.GetSuccess(),
	}

	if err := d.usageDB.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create usage record: %w", err)
	}

	// Update quota counters if successful
	if in.GetSuccess() {
		if err := d.ConsumeQuota(ctx, userID, record.TotalTokens); err != nil {
			d.logger.Error("failed to consume quota", zap.Error(err))
		}
	}

	return &commonv1.MessageResponse{Message: "recorded"}, nil
}

// --- Credits Operations ---

func (d *Domain) GetBalance(ctx context.Context, userID uuid.UUID) (*billingv1.GetBalanceResponse, error) {
	sub, err := d.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	return &billingv1.GetBalanceResponse{Balance: sub.CreditsBalance}, nil
}

func (d *Domain) AddCredits(ctx context.Context, in *billingv1.AddCreditsRequest) (*commonv1.MessageResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	targetUserID, err := uuid.Parse(in.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id", ErrInvalidRequest)
	}
	if strings.TrimSpace(in.GetSource()) == "" {
		return nil, ErrInvalidRequest
	}

	if err := d.addCredits(ctx, targetUserID, in.GetAmount(), in.GetSource()); err != nil {
		return nil, err
	}

	return &commonv1.MessageResponse{Message: "credits added"}, nil
}

func (d *Domain) addCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
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

func (d *Domain) DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error {
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

func (d *Domain) UpdateSubscriptionFromStripe(ctx context.Context, stripeSubID string, status model.SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) error {
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

func parseRFC3339OrNil(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
