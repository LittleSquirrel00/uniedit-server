package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QuotaManager defines the interface for quota management.
type QuotaManager interface {
	GetTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error)
	GetRequestsToday(ctx context.Context, userID uuid.UUID) (int, error)
	IncrementTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) (int64, error)
	IncrementRequests(ctx context.Context, userID uuid.UUID) (int, error)
	ResetTokens(ctx context.Context, userID uuid.UUID, periodStart time.Time) error
	CheckQuota(ctx context.Context, userID uuid.UUID, periodStart time.Time, tokenLimit int64, requestLimit int) error
}

// ServiceInterface defines the billing service interface.
type ServiceInterface interface {
	// Plan operations
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, planID string) (*Plan, error)

	// Subscription operations
	GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	CreateSubscription(ctx context.Context, userID uuid.UUID, planID string) (*Subscription, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID, immediately bool) (*Subscription, error)

	// Quota operations
	GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*QuotaStatus, error)
	CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error
	ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error

	// Usage operations
	GetUsageStats(ctx context.Context, userID uuid.UUID, period UsagePeriod, start, end *time.Time) (*UsageStats, error)
	RecordUsage(ctx context.Context, userID uuid.UUID, req *RecordUsageRequest) error

	// Credits operations
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
	DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error

	// Stripe sync
	UpdateSubscriptionFromStripe(ctx context.Context, stripeSubID string, status SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) error
}

// Service implements billing operations.
type Service struct {
	repo         Repository
	quotaManager QuotaManager
	logger       *zap.Logger
}

// NewService creates a new billing service.
func NewService(repo Repository, quotaManager QuotaManager, logger *zap.Logger) *Service {
	return &Service{
		repo:         repo,
		quotaManager: quotaManager,
		logger:       logger,
	}
}

// --- Plan Operations ---

func (s *Service) ListPlans(ctx context.Context) ([]*Plan, error) {
	return s.repo.ListActivePlans(ctx)
}

func (s *Service) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return s.repo.GetPlan(ctx, planID)
}

// --- Subscription Operations ---

func (s *Service) GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	return s.repo.GetSubscriptionWithPlan(ctx, userID)
}

func (s *Service) CreateSubscription(ctx context.Context, userID uuid.UUID, planID string) (*Subscription, error) {
	// Check if subscription already exists
	existing, err := s.repo.GetSubscription(ctx, userID)
	if err == nil && existing != nil {
		return nil, ErrSubscriptionExists
	}
	if err != nil && err != ErrSubscriptionNotFound {
		return nil, fmt.Errorf("check existing subscription: %w", err)
	}

	// Get plan
	plan, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !plan.Active {
		return nil, ErrPlanNotActive
	}

	// Create subscription
	now := time.Now()
	periodEnd := endOfMonth(now)

	sub := &Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             planID,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	// Load plan for response
	sub.Plan = plan
	return sub, nil
}

func (s *Service) CancelSubscription(ctx context.Context, userID uuid.UUID, immediately bool) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}

	if sub.IsCanceled() {
		return nil, ErrSubscriptionCanceled
	}

	now := time.Now()
	if immediately {
		sub.Status = SubscriptionStatusCanceled
		sub.CanceledAt = &now
		// Downgrade to free plan
		sub.PlanID = "free"
	} else {
		sub.CancelAtPeriodEnd = true
		sub.CanceledAt = &now
	}

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	return sub, nil
}

// --- Quota Operations ---

func (s *Service) GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*QuotaStatus, error) {
	sub, err := s.repo.GetSubscriptionWithPlan(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current usage from Redis
	tokensUsed, err := s.quotaManager.GetTokensUsed(ctx, userID, sub.CurrentPeriodStart)
	if err != nil {
		s.logger.Warn("failed to get tokens from redis, using DB", zap.Error(err))
		tokensUsed, _ = s.repo.GetMonthlyTokenUsage(ctx, userID, sub.CurrentPeriodStart)
	}

	requestsToday, err := s.quotaManager.GetRequestsToday(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get requests from redis, using DB", zap.Error(err))
		requestsToday, _ = s.repo.GetDailyRequestCount(ctx, userID, time.Now().UTC())
	}

	tokenLimit := sub.Plan.MonthlyTokens
	tokensRemaining := tokenLimit - tokensUsed
	if tokenLimit == -1 {
		tokensRemaining = -1
	} else if tokensRemaining < 0 {
		tokensRemaining = 0
	}

	return &QuotaStatus{
		Plan:            sub.Plan.Name,
		TokensUsed:      tokensUsed,
		TokensLimit:     tokenLimit,
		TokensRemaining: tokensRemaining,
		RequestsToday:   requestsToday,
		RequestsLimit:   sub.Plan.DailyRequests,
		ResetAt:         sub.CurrentPeriodEnd,
	}, nil
}

func (s *Service) CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error {
	sub, err := s.repo.GetSubscriptionWithPlan(ctx, userID)
	if err != nil {
		if err == ErrSubscriptionNotFound {
			// No subscription = no quota
			return ErrQuotaExceeded
		}
		return err
	}

	if !sub.IsActive() {
		return ErrQuotaExceeded
	}

	return s.quotaManager.CheckQuota(
		ctx,
		userID,
		sub.CurrentPeriodStart,
		sub.Plan.MonthlyTokens,
		sub.Plan.DailyRequests,
	)
}

func (s *Service) ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}

	// Increment token counter
	_, err = s.quotaManager.IncrementTokens(ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, int64(tokens))
	if err != nil {
		s.logger.Error("failed to increment tokens in redis", zap.Error(err))
	}

	// Increment request counter
	_, err = s.quotaManager.IncrementRequests(ctx, userID)
	if err != nil {
		s.logger.Error("failed to increment requests in redis", zap.Error(err))
	}

	return nil
}

// --- Usage Operations ---

func (s *Service) GetUsageStats(ctx context.Context, userID uuid.UUID, period UsagePeriod, start, end *time.Time) (*UsageStats, error) {
	now := time.Now().UTC()
	var startTime, endTime time.Time

	if start != nil && end != nil {
		startTime = *start
		endTime = *end
	} else {
		switch period {
		case UsagePeriodDay:
			startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			endTime = startTime.Add(24 * time.Hour)
		case UsagePeriodWeek:
			startTime = now.AddDate(0, 0, -7)
			endTime = now
		case UsagePeriodMonth:
			startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			endTime = startTime.AddDate(0, 1, 0)
		}
	}

	return s.repo.GetUsageStats(ctx, userID, startTime, endTime)
}

func (s *Service) RecordUsage(ctx context.Context, userID uuid.UUID, req *RecordUsageRequest) error {
	record := &UsageRecord{
		UserID:       userID,
		Timestamp:    time.Now(),
		RequestID:    req.RequestID,
		TaskType:     req.TaskType,
		ProviderID:   req.ProviderID,
		ModelID:      req.ModelID,
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		TotalTokens:  req.InputTokens + req.OutputTokens,
		CostUSD:      req.CostUSD,
		LatencyMs:    req.LatencyMs,
		Success:      req.Success,
	}

	if err := s.repo.CreateUsageRecord(ctx, record); err != nil {
		return fmt.Errorf("create usage record: %w", err)
	}

	// Update quota counters if successful
	if req.Success {
		if err := s.ConsumeQuota(ctx, userID, record.TotalTokens); err != nil {
			s.logger.Error("failed to consume quota", zap.Error(err))
		}
	}

	return nil
}

// --- Credits Operations ---

func (s *Service) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return 0, err
	}
	return sub.CreditsBalance, nil
}

func (s *Service) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}

	sub.CreditsBalance += amount
	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	s.logger.Info("credits added",
		zap.String("user_id", userID.String()),
		zap.Int64("amount", amount),
		zap.String("source", source),
	)

	return nil
}

func (s *Service) DeductCredits(ctx context.Context, userID uuid.UUID, amount int64, reason string) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}

	if sub.CreditsBalance < amount {
		return ErrInsufficientCredits
	}

	sub.CreditsBalance -= amount
	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	s.logger.Info("credits deducted",
		zap.String("user_id", userID.String()),
		zap.Int64("amount", amount),
		zap.String("reason", reason),
	)

	return nil
}

// --- Stripe Sync ---

func (s *Service) UpdateSubscriptionFromStripe(ctx context.Context, stripeSubID string, status SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) error {
	sub, err := s.repo.GetSubscriptionByStripeID(ctx, stripeSubID)
	if err != nil {
		return err
	}

	sub.Status = status
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd
	sub.CancelAtPeriodEnd = cancelAtPeriodEnd

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	// Reset quota counters on period renewal
	if periodStart.After(sub.CurrentPeriodStart) {
		if err := s.quotaManager.ResetTokens(ctx, sub.UserID, sub.CurrentPeriodStart); err != nil {
			s.logger.Error("failed to reset tokens", zap.Error(err))
		}
	}

	return nil
}

// --- Helpers ---

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
}
