package order

import "errors"

// Domain errors for order.
var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNotPending     = errors.New("order is not pending")
	ErrOrderNotPaid        = errors.New("order is not paid")
	ErrOrderExpired        = errors.New("order has expired")
	ErrOrderNotCancelable  = errors.New("order cannot be canceled")
	ErrOrderNotRefundable  = errors.New("order cannot be refunded")
	ErrInvalidOrderState   = errors.New("invalid order state")
	ErrInvoiceNotFound     = errors.New("invoice not found")
	ErrInsufficientBalance = errors.New("insufficient balance for refund")
)

// Note: ErrInvalidTransition is defined in status.go
