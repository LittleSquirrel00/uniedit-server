package payment

import (
	"time"

	"github.com/google/uuid"
)

// StripeWebhookEvent represents a stored Stripe webhook event.
type StripeWebhookEvent struct {
	id          uuid.UUID
	eventID     string
	eventType   string
	data        string
	processed   bool
	processedAt *time.Time
	err         *string
	createdAt   time.Time
}

// NewStripeWebhookEvent creates a new Stripe webhook event.
func NewStripeWebhookEvent(eventID, eventType, data string) *StripeWebhookEvent {
	return &StripeWebhookEvent{
		id:        uuid.New(),
		eventID:   eventID,
		eventType: eventType,
		data:      data,
		processed: false,
		createdAt: time.Now(),
	}
}

// RestoreStripeWebhookEvent recreates a StripeWebhookEvent from persisted data.
func RestoreStripeWebhookEvent(
	id uuid.UUID,
	eventID, eventType, data string,
	processed bool,
	processedAt *time.Time,
	err *string,
	createdAt time.Time,
) *StripeWebhookEvent {
	return &StripeWebhookEvent{
		id:          id,
		eventID:     eventID,
		eventType:   eventType,
		data:        data,
		processed:   processed,
		processedAt: processedAt,
		err:         err,
		createdAt:   createdAt,
	}
}

func (e *StripeWebhookEvent) ID() uuid.UUID        { return e.id }
func (e *StripeWebhookEvent) EventID() string      { return e.eventID }
func (e *StripeWebhookEvent) EventType() string    { return e.eventType }
func (e *StripeWebhookEvent) Data() string         { return e.data }
func (e *StripeWebhookEvent) Processed() bool      { return e.processed }
func (e *StripeWebhookEvent) ProcessedAt() *time.Time { return e.processedAt }
func (e *StripeWebhookEvent) Error() *string       { return e.err }
func (e *StripeWebhookEvent) CreatedAt() time.Time { return e.createdAt }

// MarkProcessed marks the event as processed.
func (e *StripeWebhookEvent) MarkProcessed(err error) {
	now := time.Now()
	e.processed = true
	e.processedAt = &now
	if err != nil {
		errStr := err.Error()
		e.err = &errStr
	}
}

// PaymentWebhookEvent represents a stored payment webhook event (for Alipay/WeChat).
type PaymentWebhookEvent struct {
	id          uuid.UUID
	provider    string
	eventID     string
	eventType   string
	tradeNo     string
	outTradeNo  string
	data        string
	processed   bool
	processedAt *time.Time
	err         *string
	createdAt   time.Time
}

// NewPaymentWebhookEvent creates a new payment webhook event.
func NewPaymentWebhookEvent(
	provider, eventID, eventType string,
	tradeNo, outTradeNo, data string,
) *PaymentWebhookEvent {
	return &PaymentWebhookEvent{
		id:         uuid.New(),
		provider:   provider,
		eventID:    eventID,
		eventType:  eventType,
		tradeNo:    tradeNo,
		outTradeNo: outTradeNo,
		data:       data,
		processed:  false,
		createdAt:  time.Now(),
	}
}

// RestorePaymentWebhookEvent recreates a PaymentWebhookEvent from persisted data.
func RestorePaymentWebhookEvent(
	id uuid.UUID,
	provider, eventID, eventType string,
	tradeNo, outTradeNo, data string,
	processed bool,
	processedAt *time.Time,
	err *string,
	createdAt time.Time,
) *PaymentWebhookEvent {
	return &PaymentWebhookEvent{
		id:          id,
		provider:    provider,
		eventID:     eventID,
		eventType:   eventType,
		tradeNo:     tradeNo,
		outTradeNo:  outTradeNo,
		data:        data,
		processed:   processed,
		processedAt: processedAt,
		err:         err,
		createdAt:   createdAt,
	}
}

func (e *PaymentWebhookEvent) ID() uuid.UUID          { return e.id }
func (e *PaymentWebhookEvent) Provider() string       { return e.provider }
func (e *PaymentWebhookEvent) EventID() string        { return e.eventID }
func (e *PaymentWebhookEvent) EventType() string      { return e.eventType }
func (e *PaymentWebhookEvent) TradeNo() string        { return e.tradeNo }
func (e *PaymentWebhookEvent) OutTradeNo() string     { return e.outTradeNo }
func (e *PaymentWebhookEvent) Data() string           { return e.data }
func (e *PaymentWebhookEvent) Processed() bool        { return e.processed }
func (e *PaymentWebhookEvent) ProcessedAt() *time.Time { return e.processedAt }
func (e *PaymentWebhookEvent) Error() *string         { return e.err }
func (e *PaymentWebhookEvent) CreatedAt() time.Time   { return e.createdAt }

// MarkProcessed marks the event as processed.
func (e *PaymentWebhookEvent) MarkProcessed(err error) {
	now := time.Now()
	e.processed = true
	e.processedAt = &now
	if err != nil {
		errStr := err.Error()
		e.err = &errStr
	}
}
