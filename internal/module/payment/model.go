package payment

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// PaymentMethod represents a payment method type.
type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodAlipay PaymentMethod = "alipay"
	PaymentMethodWechat PaymentMethod = "wechat"
)

// Payment represents a payment record.
type Payment struct {
	ID                    uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID               uuid.UUID     `json:"order_id" gorm:"type:uuid;not null;index"`
	UserID                uuid.UUID     `json:"user_id" gorm:"type:uuid;not null;index"`
	Amount                int64         `json:"amount"` // In cents
	Currency              string        `json:"currency" gorm:"default:usd"`
	Method                PaymentMethod `json:"method"`
	Status                PaymentStatus `json:"status" gorm:"not null;default:pending"`
	Provider              string        `json:"provider" gorm:"default:stripe"`
	StripePaymentIntentID string        `json:"-"`
	StripeChargeID        string        `json:"-"`
	// China payment fields
	TradeNo        string  `json:"-" gorm:"index"` // Provider's trade number (Alipay/WeChat)
	PayerID        string  `json:"-"`              // Payer's ID (openid for WeChat, buyer_id for Alipay)
	FailureCode    *string `json:"failure_code,omitempty"`
	FailureMessage *string `json:"failure_message,omitempty"`
	RefundedAmount int64   `json:"refunded_amount" gorm:"default:0"`
	SucceededAt    *time.Time `json:"succeeded_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName returns the database table name.
func (Payment) TableName() string {
	return "payments"
}

// IsSucceeded returns true if the payment succeeded.
func (p *Payment) IsSucceeded() bool {
	return p.Status == PaymentStatusSucceeded
}

// StripeWebhookEvent represents a stored Stripe webhook event.
type StripeWebhookEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventID     string     `gorm:"uniqueIndex;not null"`
	Type        string     `gorm:"not null"`
	Data        string     `gorm:"type:jsonb"`
	Processed   bool       `gorm:"default:false"`
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

// TableName returns the database table name.
func (StripeWebhookEvent) TableName() string {
	return "stripe_webhook_events"
}

// PaymentWebhookEvent represents a stored payment webhook event (for Alipay/WeChat).
type PaymentWebhookEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Provider    string     `gorm:"not null;index"`           // alipay, wechat
	EventID     string     `gorm:"uniqueIndex:idx_provider_event;not null"` // out_trade_no or transaction_id
	EventType   string     `gorm:"not null"`                 // payment, refund
	TradeNo     string     `gorm:"index"`                    // Provider's trade number
	OutTradeNo  string     `gorm:"index"`                    // Our order/payment ID
	Data        string     `gorm:"type:jsonb"`               // Raw notification data
	Processed   bool       `gorm:"default:false"`
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

// TableName returns the database table name.
func (PaymentWebhookEvent) TableName() string {
	return "payment_webhook_events"
}
