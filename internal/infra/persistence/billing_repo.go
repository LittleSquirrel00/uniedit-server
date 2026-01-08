package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/infra/persistence/entity"
	"gorm.io/gorm"
)

// BillingRepository implements billing.Repository interface.
type BillingRepository struct {
	db *gorm.DB
}

// NewBillingRepository creates a new billing repository.
func NewBillingRepository(db *gorm.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

// --- Plan Operations ---

func (r *BillingRepository) ListActivePlans(ctx context.Context) ([]*billing.Plan, error) {
	var entities []*entity.PlanEntity
	err := r.db.WithContext(ctx).
		Where("active = ?", true).
		Order("display_order ASC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list active plans: %w", err)
	}

	plans := make([]*billing.Plan, len(entities))
	for i, ent := range entities {
		plans[i] = ent.ToDomain()
	}
	return plans, nil
}

func (r *BillingRepository) GetPlan(ctx context.Context, id string) (*billing.Plan, error) {
	var ent entity.PlanEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, billing.ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan: %w", err)
	}
	return ent.ToDomain(), nil
}

// --- Subscription Operations ---

func (r *BillingRepository) CreateSubscription(ctx context.Context, sub *billing.Subscription) error {
	ent := entity.FromDomainSubscription(sub)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}

func (r *BillingRepository) GetSubscription(ctx context.Context, userID uuid.UUID) (*billing.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, billing.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *BillingRepository) GetSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*billing.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ?", userID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, billing.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription with plan: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *BillingRepository) UpdateSubscription(ctx context.Context, sub *billing.Subscription) error {
	ent := entity.FromDomainSubscription(sub)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (r *BillingRepository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*billing.Subscription, error) {
	var ent entity.SubscriptionEntity
	err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", stripeSubID).
		First(&ent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, billing.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("get subscription by stripe id: %w", err)
	}
	return ent.ToDomain(), nil
}

// --- Usage Operations ---

func (r *BillingRepository) CreateUsageRecord(ctx context.Context, record *billing.UsageRecord) error {
	ent := entity.FromDomainUsageRecord(record)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create usage record: %w", err)
	}
	return nil
}

func (r *BillingRepository) GetUsageStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*billing.UsageStats, error) {
	// Get totals
	var totals struct {
		TotalTokens   int64
		TotalRequests int64
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UsageRecordEntity{}).
		Select("COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ? AND success = true", userID, start, end).
		Scan(&totals).Error
	if err != nil {
		return nil, fmt.Errorf("get usage totals: %w", err)
	}

	return &billing.UsageStats{
		TotalTokens:   totals.TotalTokens,
		TotalRequests: int(totals.TotalRequests),
	}, nil
}

func (r *BillingRepository) GetMonthlyTokenUsage(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error) {
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

func (r *BillingRepository) GetDailyRequestCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
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

// Ensure BillingRepository implements billing.Repository.
var _ billing.Repository = (*BillingRepository)(nil)
