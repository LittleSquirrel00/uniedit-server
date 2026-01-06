package order

import "errors"

// Module errors.
var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderNotPending      = errors.New("order is not pending")
	ErrOrderNotPaid         = errors.New("order is not paid")
	ErrOrderExpired         = errors.New("order has expired")
	ErrOrderNotCancelable   = errors.New("order cannot be canceled")
	ErrOrderNotRefundable   = errors.New("order cannot be refunded")
	ErrInvalidTransition    = errors.New("invalid order status transition")
	ErrInvoiceNotFound      = errors.New("invoice not found")
	ErrInsufficientBalance  = errors.New("insufficient balance for refund")
)
