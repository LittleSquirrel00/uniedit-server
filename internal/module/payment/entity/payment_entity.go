package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/payment/domain"
)

// PaymentEntity is the GORM entity for Payment.
type PaymentEntity struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID               uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID                uuid.UUID `gorm:"type:uuid;not null;index"`
	Amount                int64
	Currency              string `gorm:"default:usd"`
	Method                string
	Status                string `gorm:"not null;default:pending"`
	Provider              string `gorm:"default:stripe"`
	StripePaymentIntentID string
	StripeChargeID        string
	TradeNo               string     `gorm:"index"`
	PayerID               string
	FailureCode           *string
	FailureMessage        *string
	RefundedAmount        int64      `gorm:"default:0"`
	SucceededAt           *time.Time
	FailedAt              *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TableName returns the database table name.
func (PaymentEntity) TableName() string {
	return "payments"
}

// ToDomain converts entity to domain Payment.
func (e *PaymentEntity) ToDomain() *domain.Payment {
	return domain.RestorePayment(
		e.ID,
		e.OrderID,
		e.UserID,
		e.Amount,
		e.Currency,
		domain.PaymentMethod(e.Method),
		domain.PaymentStatus(e.Status),
		e.Provider,
		e.StripePaymentIntentID,
		e.StripeChargeID,
		e.TradeNo,
		e.PayerID,
		e.FailureCode,
		e.FailureMessage,
		e.RefundedAmount,
		e.SucceededAt,
		e.FailedAt,
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// FromDomainPayment converts domain Payment to entity.
func FromDomainPayment(p *domain.Payment) *PaymentEntity {
	return &PaymentEntity{
		ID:                    p.ID(),
		OrderID:               p.OrderID(),
		UserID:                p.UserID(),
		Amount:                p.Amount(),
		Currency:              p.Currency(),
		Method:                string(p.Method()),
		Status:                string(p.Status()),
		Provider:              p.Provider(),
		StripePaymentIntentID: p.StripePaymentIntentID(),
		StripeChargeID:        p.StripeChargeID(),
		TradeNo:               p.TradeNo(),
		PayerID:               p.PayerID(),
		FailureCode:           p.FailureCode(),
		FailureMessage:        p.FailureMessage(),
		RefundedAmount:        p.RefundedAmount(),
		SucceededAt:           p.SucceededAt(),
		FailedAt:              p.FailedAt(),
		CreatedAt:             p.CreatedAt(),
		UpdatedAt:             p.UpdatedAt(),
	}
}

// StripeWebhookEventEntity is the GORM entity for StripeWebhookEvent.
type StripeWebhookEventEntity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventID     string    `gorm:"uniqueIndex;not null"`
	Type        string    `gorm:"not null"`
	Data        string    `gorm:"type:jsonb"`
	Processed   bool      `gorm:"default:false"`
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

// TableName returns the database table name.
func (StripeWebhookEventEntity) TableName() string {
	return "stripe_webhook_events"
}

// ToDomain converts entity to domain StripeWebhookEvent.
func (e *StripeWebhookEventEntity) ToDomain() *domain.StripeWebhookEvent {
	return domain.RestoreStripeWebhookEvent(
		e.ID,
		e.EventID,
		e.Type,
		e.Data,
		e.Processed,
		e.ProcessedAt,
		e.Error,
		e.CreatedAt,
	)
}

// FromDomainStripeWebhookEvent converts domain StripeWebhookEvent to entity.
func FromDomainStripeWebhookEvent(e *domain.StripeWebhookEvent) *StripeWebhookEventEntity {
	return &StripeWebhookEventEntity{
		ID:          e.ID(),
		EventID:     e.EventID(),
		Type:        e.EventType(),
		Data:        e.Data(),
		Processed:   e.Processed(),
		ProcessedAt: e.ProcessedAt(),
		Error:       e.Error(),
		CreatedAt:   e.CreatedAt(),
	}
}

// PaymentWebhookEventEntity is the GORM entity for PaymentWebhookEvent.
type PaymentWebhookEventEntity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Provider    string    `gorm:"not null;index"`
	EventID     string    `gorm:"uniqueIndex:idx_provider_event;not null"`
	EventType   string    `gorm:"not null"`
	TradeNo     string    `gorm:"index"`
	OutTradeNo  string    `gorm:"index"`
	Data        string    `gorm:"type:jsonb"`
	Processed   bool      `gorm:"default:false"`
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

// TableName returns the database table name.
func (PaymentWebhookEventEntity) TableName() string {
	return "payment_webhook_events"
}

// ToDomain converts entity to domain PaymentWebhookEvent.
func (e *PaymentWebhookEventEntity) ToDomain() *domain.PaymentWebhookEvent {
	return domain.RestorePaymentWebhookEvent(
		e.ID,
		e.Provider,
		e.EventID,
		e.EventType,
		e.TradeNo,
		e.OutTradeNo,
		e.Data,
		e.Processed,
		e.ProcessedAt,
		e.Error,
		e.CreatedAt,
	)
}

// FromDomainPaymentWebhookEvent converts domain PaymentWebhookEvent to entity.
func FromDomainPaymentWebhookEvent(e *domain.PaymentWebhookEvent) *PaymentWebhookEventEntity {
	return &PaymentWebhookEventEntity{
		ID:          e.ID(),
		Provider:    e.Provider(),
		EventID:     e.EventID(),
		EventType:   e.EventType(),
		TradeNo:     e.TradeNo(),
		OutTradeNo:  e.OutTradeNo(),
		Data:        e.Data(),
		Processed:   e.Processed(),
		ProcessedAt: e.ProcessedAt(),
		Error:       e.Error(),
		CreatedAt:   e.CreatedAt(),
	}
}
