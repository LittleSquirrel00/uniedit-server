package payment

import "errors"

// Module errors.
var (
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrWebhookEventExists     = errors.New("webhook event already processed")
	ErrPaymentAlreadyProcessed = errors.New("payment already processed")
	ErrRefundFailed           = errors.New("refund failed")
	ErrProviderNotFound       = errors.New("payment provider not found")
)
