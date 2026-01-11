package order

import "errors"

var (
	// ErrOrderNotFound is returned when an order is not found.
	ErrOrderNotFound = errors.New("order not found")

	// ErrOrderNotPending is returned when an order is not in pending status.
	ErrOrderNotPending = errors.New("order is not pending")

	// ErrOrderNotPaid is returned when an order is not in paid status.
	ErrOrderNotPaid = errors.New("order is not paid")

	// ErrOrderNotCancelable is returned when an order cannot be canceled.
	ErrOrderNotCancelable = errors.New("order cannot be canceled")

	// ErrOrderNotRefundable is returned when an order cannot be refunded.
	ErrOrderNotRefundable = errors.New("order cannot be refunded")

	// ErrOrderExpired is returned when an order has expired.
	ErrOrderExpired = errors.New("order has expired")

	// ErrInvalidTransition is returned when a state transition is not allowed.
	ErrInvalidTransition = errors.New("invalid state transition")

	// ErrInvoiceNotFound is returned when an invoice is not found.
	ErrInvoiceNotFound = errors.New("invoice not found")

	// ErrInvoiceAlreadyExists is returned when an invoice already exists for an order.
	ErrInvoiceAlreadyExists = errors.New("invoice already exists for this order")

	// ErrInvalidOrderType is returned when an order type is invalid.
	ErrInvalidOrderType = errors.New("invalid order type")

	// ErrMinimumTopupAmount is returned when a top-up amount is below minimum.
	ErrMinimumTopupAmount = errors.New("minimum top-up is $1.00 (100 cents)")

	// ErrFreePlanNotOrderable is returned when trying to create an order for a free plan.
	ErrFreePlanNotOrderable = errors.New("cannot create order for free plan")
)
