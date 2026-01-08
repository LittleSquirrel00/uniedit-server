package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// subscriptionAdapter implements outbound.SubscriptionDatabasePort.
type subscriptionAdapter struct {
	db *gorm.DB
}

// NewSubscriptionAdapter creates a new subscription database adapter.
func NewSubscriptionAdapter(db *gorm.DB) outbound.SubscriptionDatabasePort {
	return &subscriptionAdapter{db: db}
}

func (a *subscriptionAdapter) Create(ctx context.Context, sub *model.Subscription) error {
	return a.db.WithContext(ctx).Create(sub).Error
}

func (a *subscriptionAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Subscription, error) {
	var sub model.Subscription
	err := a.db.WithContext(ctx).First(&sub, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (a *subscriptionAdapter) GetByUserIDWithPlan(ctx context.Context, userID uuid.UUID) (*model.Subscription, error) {
	var sub model.Subscription
	err := a.db.WithContext(ctx).
		Preload("Plan").
		First(&sub, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (a *subscriptionAdapter) GetByStripeID(ctx context.Context, stripeSubID string) (*model.Subscription, error) {
	var sub model.Subscription
	err := a.db.WithContext(ctx).First(&sub, "stripe_subscription_id = ?", stripeSubID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (a *subscriptionAdapter) Update(ctx context.Context, sub *model.Subscription) error {
	return a.db.WithContext(ctx).Save(sub).Error
}

func (a *subscriptionAdapter) UpdateCredits(ctx context.Context, userID uuid.UUID, amount int64) error {
	return a.db.WithContext(ctx).
		Model(&model.Subscription{}).
		Where("user_id = ?", userID).
		UpdateColumn("credits_balance", gorm.Expr("credits_balance + ?", amount)).
		Error
}

// Compile-time check
var _ outbound.SubscriptionDatabasePort = (*subscriptionAdapter)(nil)
