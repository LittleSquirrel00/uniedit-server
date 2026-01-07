package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing/domain"
	"github.com/uniedit/server/internal/module/billing/entity"
	"gorm.io/gorm"
)

// Repository defines the interface for billing data access.
type Repository interface {
	// Plan operations
	ListActivePlans(ctx context.Context) ([]*domain.Plan, error)
	GetPlan(ctx context.Context, id string) (*domain.Plan, error)

	// Subscription operations
	CreateSubscription(ctx context.Context, sub *domain.Subscription) error
	GetSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	GetSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, sub *domain.Subscription) error
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*domain.Subscription, error)

	// Usage operations
	CreateUsageRecord(ctx context.Context, record *domain.UsageRecord) error
	GetUsageStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*UsageStats, error)
	GetMonthlyTokenUsage(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error)
	GetDailyRequestCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new billing repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Plan Operations ---

func (r *repository) ListActivePlans(ctx context.Context) ([]*domain.Plan, error) {
	var entities []*entity.PlanEntity
	err := r.db.WithContext(ctx).
		Where("active = ?", true).
		Order("display_order ASC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list active plans: %w", err)
	}

	plans := make([]*domain.Plan, len(entities))
	for i, ent := range entities {
		plans[i] = ent.ToDomain()
	}
	return plans, nil
}

func (r *repository) GetPlan(ctx context.Context, id string) (*domain.Plan, error) {
	var ent entity.PlanEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan: %w", err)
	}
	return ent.ToDomain(), nil
}

// --- Subscription Operations ---

func (r *repository) CreateSubscription(ctx context.Context, sub *domain.Subscription) error {
	ent := entity.FromDomainSubscription(sub)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}

func (r *repository) GetSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) GetSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ?", userID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription with plan: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) UpdateSubscription(ctx context.Context, sub *domain.Subscription) error {
	ent := entity.FromDomainSubscription(sub)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (r *repository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", stripeSubID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription by stripe id: %w", err)
	}
	return ent.ToDomain(), nil
}

// --- Usage Operations ---

func (r *repository) CreateUsageRecord(ctx context.Context, record *domain.UsageRecord) error {
	ent := entity.FromDomainUsageRecord(record)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create usage record: %w", err)
	}
	return nil
}

func (r *repository) GetUsageStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*UsageStats, error) {
	stats := &UsageStats{
		ByModel: make(map[string]*ModelUsage),
		ByDay:   make([]*DailyUsage, 0),
	}

	// Get totals
	var totals struct {
		TotalTokens   int64
		TotalRequests int64
		TotalCostUSD  float64
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Select("COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Scan(&totals).Error
	if err != nil {
		return nil, fmt.Errorf("get usage totals: %w", err)
	}
	stats.TotalTokens = totals.TotalTokens
	stats.TotalRequests = int(totals.TotalRequests)
	stats.TotalCostUSD = totals.TotalCostUSD

	// Get by model
	var modelStats []struct {
		ModelID       string
		TotalTokens   int64
		TotalRequests int64
		TotalCostUSD  float64
	}
	err = r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Select("model_id, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Group("model_id").
		Scan(&modelStats).Error
	if err != nil {
		return nil, fmt.Errorf("get usage by model: %w", err)
	}
	for _, m := range modelStats {
		stats.ByModel[m.ModelID] = &ModelUsage{
			ModelID:       m.ModelID,
			TotalTokens:   m.TotalTokens,
			TotalRequests: int(m.TotalRequests),
			TotalCostUSD:  m.TotalCostUSD,
		}
	}

	// Get by day
	var dailyStats []struct {
		Date          time.Time
		TotalTokens   int64
		TotalRequests int64
		TotalCostUSD  float64
	}
	err = r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Select("DATE(timestamp) as date, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Group("DATE(timestamp)").
		Order("DATE(timestamp) ASC").
		Scan(&dailyStats).Error
	if err != nil {
		return nil, fmt.Errorf("get usage by day: %w", err)
	}
	for _, d := range dailyStats {
		stats.ByDay = append(stats.ByDay, &DailyUsage{
			Date:          d.Date.Format("2006-01-02"),
			TotalTokens:   d.TotalTokens,
			TotalRequests: int(d.TotalRequests),
			TotalCostUSD:  d.TotalCostUSD,
		})
	}

	return stats, nil
}

func (r *repository) GetMonthlyTokenUsage(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Select("COALESCE(SUM(total_tokens), 0)").
		Where("user_id = ? AND timestamp >= ? AND success = true", userID, start).
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("get monthly token usage: %w", err)
	}
	return total, nil
}

func (r *repository) GetDailyRequestCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	var count int64
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, startOfDay, endOfDay).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("get daily request count: %w", err)
	}
	return int(count), nil
}
