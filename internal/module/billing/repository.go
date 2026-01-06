package billing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for billing data access.
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

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new billing repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Plan Operations ---

func (r *repository) ListActivePlans(ctx context.Context) ([]*Plan, error) {
	var plans []*Plan
	err := r.db.WithContext(ctx).
		Where("active = ?", true).
		Order("display_order ASC").
		Find(&plans).Error
	return plans, err
}

func (r *repository) GetPlan(ctx context.Context, id string) (*Plan, error) {
	var plan Plan
	err := r.db.WithContext(ctx).First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

// --- Subscription Operations ---

func (r *repository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *repository) GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) GetSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ?", userID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *repository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", stripeSubID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

// --- Usage Operations ---

func (r *repository) CreateUsageRecord(ctx context.Context, record *UsageRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
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
		Model(&UsageRecord{}).
		Select("COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Scan(&totals).Error
	if err != nil {
		return nil, err
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
		Model(&UsageRecord{}).
		Select("model_id, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Group("model_id").
		Scan(&modelStats).Error
	if err != nil {
		return nil, err
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
		Model(&UsageRecord{}).
		Select("DATE(timestamp) as date, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Group("DATE(timestamp)").
		Order("DATE(timestamp) ASC").
		Scan(&dailyStats).Error
	if err != nil {
		return nil, err
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
		Model(&UsageRecord{}).
		Select("COALESCE(SUM(total_tokens), 0)").
		Where("user_id = ? AND timestamp >= ? AND success = true", userID, start).
		Scan(&total).Error
	return total, err
}

func (r *repository) GetDailyRequestCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	var count int64
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := r.db.WithContext(ctx).
		Model(&UsageRecord{}).
		Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, startOfDay, endOfDay).
		Count(&count).Error
	return int(count), err
}
