package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Payment errors.
var (
	ErrPaymentAlreadySucceeded = errors.New("payment already succeeded")
	ErrPaymentNotSucceeded     = errors.New("payment is not succeeded")
	ErrInvalidRefundAmount     = errors.New("invalid refund amount")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

// Payment represents a payment aggregate root.
type Payment struct {
	id                    uuid.UUID
	orderID               uuid.UUID
	userID                uuid.UUID
	amount                int64
	currency              string
	method                PaymentMethod
	status                PaymentStatus
	provider              string
	stripePaymentIntentID string
	stripeChargeID        string
	tradeNo               string  // Provider's trade number (Alipay/WeChat)
	payerID               string  // Payer's ID (openid for WeChat, buyer_id for Alipay)
	failureCode           *string
	failureMessage        *string
	refundedAmount        int64
	succeededAt           *time.Time
	failedAt              *time.Time
	createdAt             time.Time
	updatedAt             time.Time
}

// NewPayment creates a new Payment.
func NewPayment(
	orderID, userID uuid.UUID,
	amount int64,
	currency string,
	method PaymentMethod,
	provider string,
) *Payment {
	now := time.Now()
	return &Payment{
		id:        uuid.New(),
		orderID:   orderID,
		userID:    userID,
		amount:    amount,
		currency:  currency,
		method:    method,
		status:    StatusPending,
		provider:  provider,
		createdAt: now,
		updatedAt: now,
	}
}

// RestorePayment recreates a Payment from persisted data.
func RestorePayment(
	id, orderID, userID uuid.UUID,
	amount int64,
	currency string,
	method PaymentMethod,
	status PaymentStatus,
	provider string,
	stripePaymentIntentID, stripeChargeID string,
	tradeNo, payerID string,
	failureCode, failureMessage *string,
	refundedAmount int64,
	succeededAt, failedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Payment {
	return &Payment{
		id:                    id,
		orderID:               orderID,
		userID:                userID,
		amount:                amount,
		currency:              currency,
		method:                method,
		status:                status,
		provider:              provider,
		stripePaymentIntentID: stripePaymentIntentID,
		stripeChargeID:        stripeChargeID,
		tradeNo:               tradeNo,
		payerID:               payerID,
		failureCode:           failureCode,
		failureMessage:        failureMessage,
		refundedAmount:        refundedAmount,
		succeededAt:           succeededAt,
		failedAt:              failedAt,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
	}
}

// --- Getters ---

func (p *Payment) ID() uuid.UUID                    { return p.id }
func (p *Payment) OrderID() uuid.UUID               { return p.orderID }
func (p *Payment) UserID() uuid.UUID                { return p.userID }
func (p *Payment) Amount() int64                    { return p.amount }
func (p *Payment) Currency() string                 { return p.currency }
func (p *Payment) Method() PaymentMethod            { return p.method }
func (p *Payment) Status() PaymentStatus            { return p.status }
func (p *Payment) Provider() string                 { return p.provider }
func (p *Payment) StripePaymentIntentID() string    { return p.stripePaymentIntentID }
func (p *Payment) StripeChargeID() string           { return p.stripeChargeID }
func (p *Payment) TradeNo() string                  { return p.tradeNo }
func (p *Payment) PayerID() string                  { return p.payerID }
func (p *Payment) FailureCode() *string             { return p.failureCode }
func (p *Payment) FailureMessage() *string          { return p.failureMessage }
func (p *Payment) RefundedAmount() int64            { return p.refundedAmount }
func (p *Payment) SucceededAt() *time.Time          { return p.succeededAt }
func (p *Payment) FailedAt() *time.Time             { return p.failedAt }
func (p *Payment) CreatedAt() time.Time             { return p.createdAt }
func (p *Payment) UpdatedAt() time.Time             { return p.updatedAt }

// IsSucceeded returns true if the payment succeeded.
func (p *Payment) IsSucceeded() bool {
	return p.status.IsSucceeded()
}

// --- Setters for non-critical fields ---

func (p *Payment) SetStripePaymentIntentID(id string) {
	p.stripePaymentIntentID = id
	p.updatedAt = time.Now()
}

func (p *Payment) SetStripeChargeID(id string) {
	p.stripeChargeID = id
	p.updatedAt = time.Now()
}

func (p *Payment) SetTradeNo(tradeNo string) {
	p.tradeNo = tradeNo
	p.updatedAt = time.Now()
}

func (p *Payment) SetPayerID(payerID string) {
	p.payerID = payerID
	p.updatedAt = time.Now()
}

// --- Domain Methods ---

// MarkAsSucceeded marks the payment as succeeded.
func (p *Payment) MarkAsSucceeded(chargeID string) error {
	if p.status.IsSucceeded() {
		return ErrPaymentAlreadySucceeded
	}

	if !p.status.CanTransitionTo(StatusSucceeded) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.status = StatusSucceeded
	p.stripeChargeID = chargeID
	p.succeededAt = &now
	p.updatedAt = now
	return nil
}

// MarkAsSucceededNative marks a native payment (Alipay/WeChat) as succeeded.
func (p *Payment) MarkAsSucceededNative(tradeNo, payerID string) error {
	if p.status.IsSucceeded() {
		return ErrPaymentAlreadySucceeded
	}

	if !p.status.CanTransitionTo(StatusSucceeded) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.status = StatusSucceeded
	p.tradeNo = tradeNo
	p.payerID = payerID
	p.succeededAt = &now
	p.updatedAt = now
	return nil
}

// MarkAsFailed marks the payment as failed.
func (p *Payment) MarkAsFailed(failureCode, failureMessage string) error {
	if !p.status.CanTransitionTo(StatusFailed) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.status = StatusFailed
	p.failureCode = &failureCode
	p.failureMessage = &failureMessage
	p.failedAt = &now
	p.updatedAt = now
	return nil
}

// MarkAsCanceled marks the payment as canceled.
func (p *Payment) MarkAsCanceled() error {
	if !p.status.CanTransitionTo(StatusCanceled) {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	p.status = StatusCanceled
	p.failedAt = &now
	p.updatedAt = now
	return nil
}

// Refund processes a refund for the payment.
// If amount is 0, refunds the full remaining amount.
func (p *Payment) Refund(amount int64) (int64, error) {
	if !p.IsSucceeded() && p.status != StatusRefunded {
		return 0, ErrPaymentNotSucceeded
	}

	refundAmount := amount
	if refundAmount == 0 {
		refundAmount = p.amount - p.refundedAmount
	}

	if refundAmount <= 0 || refundAmount > (p.amount-p.refundedAmount) {
		return 0, ErrInvalidRefundAmount
	}

	p.refundedAmount += refundAmount
	if p.refundedAmount >= p.amount {
		p.status = StatusRefunded
	}
	p.updatedAt = time.Now()

	return refundAmount, nil
}
