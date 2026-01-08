package payment

import "errors"

var (
	// ErrPaymentNotFound is returned when a payment is not found.
	ErrPaymentNotFound = errors.New("payment not found")

	// ErrPaymentAlreadySucceeded is returned when trying to mark
	// an already succeeded payment as succeeded.
	ErrPaymentAlreadySucceeded = errors.New("payment already succeeded")

	// ErrPaymentNotSucceeded is returned when an operation requires
	// a succeeded payment but the payment is not succeeded.
	ErrPaymentNotSucceeded = errors.New("payment is not succeeded")

	// ErrInvalidRefundAmount is returned when the refund amount is invalid.
	ErrInvalidRefundAmount = errors.New("invalid refund amount")

	// ErrInvalidStatusTransition is returned when a status transition is invalid.
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrOrderNotPending is returned when an order is not in pending status.
	ErrOrderNotPending = errors.New("order is not pending")

	// ErrForbidden is returned when the user is not authorized.
	ErrForbidden = errors.New("forbidden")

	// ErrProviderNotAvailable is returned when a payment provider is not available.
	ErrProviderNotAvailable = errors.New("provider not available")

	// ErrNoChargeID is returned when no charge ID is available for refund.
	ErrNoChargeID = errors.New("no charge ID for refund")
)
