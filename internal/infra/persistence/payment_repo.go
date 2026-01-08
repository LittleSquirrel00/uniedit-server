package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/infra/persistence/entity"
	"gorm.io/gorm"
)

// PaymentRepository implements payment.Repository interface.
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new payment repository.
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// --- Payment Operations ---

func (r *PaymentRepository) CreatePayment(ctx context.Context, p *payment.Payment) error {
	ent := entity.FromDomainPayment(p)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetPayment(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	var ent entity.PaymentEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *PaymentRepository) GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.Payment, error) {
	var ent entity.PaymentEntity
	err := r.db.WithContext(ctx).First(&ent, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment by payment intent id: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *PaymentRepository) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	ent := entity.FromDomainPayment(p)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*payment.Payment, error) {
	var entities []*entity.PaymentEntity
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list payments by order: %w", err)
	}

	payments := make([]*payment.Payment, len(entities))
	for i, ent := range entities {
		payments[i] = ent.ToDomain()
	}
	return payments, nil
}

// --- Stripe Webhook Event Operations ---

func (r *PaymentRepository) CreateWebhookEvent(ctx context.Context, event *payment.StripeWebhookEvent) error {
	ent := entity.FromDomainStripeWebhookEvent(event)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create webhook event: %w", err)
	}
	return nil
}

func (r *PaymentRepository) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
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

func (r *PaymentRepository) MarkWebhookEventProcessed(ctx context.Context, eventID string, processErr error) error {
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

func (r *PaymentRepository) CreatePaymentWebhookEvent(ctx context.Context, event *payment.PaymentWebhookEvent) error {
	ent := entity.FromDomainPaymentWebhookEvent(event)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create payment webhook event: %w", err)
	}
	return nil
}

func (r *PaymentRepository) MarkPaymentWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, processErr error) error {
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

// Ensure PaymentRepository implements payment.Repository.
var _ payment.Repository = (*PaymentRepository)(nil)
