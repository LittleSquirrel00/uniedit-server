package payment

import "errors"

// Domain errors for payment.
var (
	ErrPaymentNotFound   = errors.New("payment not found")
	ErrInvalidPayment    = errors.New("invalid payment")
	ErrRefundFailed      = errors.New("refund failed")
	ErrWebhookDuplicate  = errors.New("webhook event already processed")
)
