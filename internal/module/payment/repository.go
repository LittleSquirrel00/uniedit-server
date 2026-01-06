package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for payment data access.
type Repository interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Payment, error)
	UpdatePayment(ctx context.Context, payment *Payment) error
	ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*Payment, error)

	// Stripe webhook event operations
	CreateWebhookEvent(ctx context.Context, event *StripeWebhookEvent) error
	WebhookEventExists(ctx context.Context, eventID string) (bool, error)
	MarkWebhookEventProcessed(ctx context.Context, eventID string, err error) error

	// Native payment webhook event operations (Alipay/WeChat)
	CreatePaymentWebhookEvent(ctx context.Context, event *PaymentWebhookEvent) error
	MarkPaymentWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, err error) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new payment repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Payment Operations ---

func (r *repository) CreatePayment(ctx context.Context, payment *Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *repository) GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error) {
	var payment Payment
	err := r.db.WithContext(ctx).First(&payment, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *repository) GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Payment, error) {
	var payment Payment
	err := r.db.WithContext(ctx).First(&payment, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *repository) UpdatePayment(ctx context.Context, payment *Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

func (r *repository) ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*Payment, error) {
	var payments []*Payment
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

// --- Webhook Event Operations ---

func (r *repository) CreateWebhookEvent(ctx context.Context, event *StripeWebhookEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *repository) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&StripeWebhookEvent{}).
		Where("event_id = ?", eventID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) MarkWebhookEventProcessed(ctx context.Context, eventID string, processErr error) error {
	updates := map[string]interface{}{
		"processed":    true,
		"processed_at": gorm.Expr("NOW()"),
	}
	if processErr != nil {
		errStr := processErr.Error()
		updates["error"] = errStr
	}
	return r.db.WithContext(ctx).
		Model(&StripeWebhookEvent{}).
		Where("event_id = ?", eventID).
		Updates(updates).Error
}

// --- Native Payment Webhook Event Operations ---

func (r *repository) CreatePaymentWebhookEvent(ctx context.Context, event *PaymentWebhookEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *repository) MarkPaymentWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, processErr error) error {
	updates := map[string]interface{}{
		"processed":    true,
		"processed_at": gorm.Expr("NOW()"),
	}
	if processErr != nil {
		errStr := processErr.Error()
		updates["error"] = errStr
	}
	return r.db.WithContext(ctx).
		Model(&PaymentWebhookEvent{}).
		Where("id = ?", eventID).
		Updates(updates).Error
}
