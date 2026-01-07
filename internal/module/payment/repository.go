package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/payment/domain"
	"github.com/uniedit/server/internal/module/payment/entity"
	"gorm.io/gorm"
)

// Repository defines the interface for payment data access.
type Repository interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment *domain.Payment) error
	GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Payment, error)
	UpdatePayment(ctx context.Context, payment *domain.Payment) error
	ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*domain.Payment, error)

	// Stripe webhook event operations
	CreateWebhookEvent(ctx context.Context, event *domain.StripeWebhookEvent) error
	WebhookEventExists(ctx context.Context, eventID string) (bool, error)
	MarkWebhookEventProcessed(ctx context.Context, eventID string, err error) error

	// Native payment webhook event operations (Alipay/WeChat)
	CreatePaymentWebhookEvent(ctx context.Context, event *domain.PaymentWebhookEvent) error
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

func (r *repository) CreatePayment(ctx context.Context, payment *domain.Payment) error {
	ent := entity.FromDomainPayment(payment)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *repository) GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var ent entity.PaymentEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Payment, error) {
	var ent entity.PaymentEntity
	err := r.db.WithContext(ctx).First(&ent, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment by payment intent id: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) UpdatePayment(ctx context.Context, payment *domain.Payment) error {
	ent := entity.FromDomainPayment(payment)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

func (r *repository) ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*domain.Payment, error) {
	var entities []*entity.PaymentEntity
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list payments by order: %w", err)
	}

	payments := make([]*domain.Payment, len(entities))
	for i, ent := range entities {
		payments[i] = ent.ToDomain()
	}
	return payments, nil
}

// --- Webhook Event Operations ---

func (r *repository) CreateWebhookEvent(ctx context.Context, event *domain.StripeWebhookEvent) error {
	ent := entity.FromDomainStripeWebhookEvent(event)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create webhook event: %w", err)
	}
	return nil
}

func (r *repository) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.StripeWebhookEventEntity{}).
		Where("event_id = ?", eventID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check webhook event exists: %w", err)
	}
	return count > 0, nil
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
	err := r.db.WithContext(ctx).
		Model(&entity.StripeWebhookEventEntity{}).
		Where("event_id = ?", eventID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("mark webhook event processed: %w", err)
	}
	return nil
}

// --- Native Payment Webhook Event Operations ---

func (r *repository) CreatePaymentWebhookEvent(ctx context.Context, event *domain.PaymentWebhookEvent) error {
	ent := entity.FromDomainPaymentWebhookEvent(event)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create payment webhook event: %w", err)
	}
	return nil
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
	err := r.db.WithContext(ctx).
		Model(&entity.PaymentWebhookEventEntity{}).
		Where("id = ?", eventID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("mark payment webhook event processed: %w", err)
	}
	return nil
}
