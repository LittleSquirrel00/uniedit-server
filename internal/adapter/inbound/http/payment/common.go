package paymenthttp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/utils/middleware"
)

// getUserID returns the user ID from context.
func getUserID(c *gin.Context) uuid.UUID {
	return middleware.GetUserID(c)
}

// mustGetUserID returns the user ID from context, panics if not found.
func mustGetUserID(c *gin.Context) uuid.UUID {
	userID := getUserID(c)
	if userID == uuid.Nil {
		panic("user_id not found in context")
	}
	return userID
}

// handleError maps payment domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		statusCode = http.StatusNotFound
		errorCode = "payment_not_found"
		message = "Payment not found"

	case errors.Is(err, payment.ErrForbidden):
		statusCode = http.StatusForbidden
		errorCode = "forbidden"
		message = "Forbidden"

	case errors.Is(err, payment.ErrOrderNotPending):
		statusCode = http.StatusBadRequest
		errorCode = "order_not_pending"
		message = "Order is not pending"

	case errors.Is(err, payment.ErrPaymentNotSucceeded):
		statusCode = http.StatusBadRequest
		errorCode = "payment_not_succeeded"
		message = "Payment is not succeeded"

	case errors.Is(err, payment.ErrInvalidRefundAmount):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_refund_amount"
		message = "Invalid refund amount"

	case errors.Is(err, payment.ErrProviderNotAvailable):
		statusCode = http.StatusServiceUnavailable
		errorCode = "provider_unavailable"
		message = "Payment provider not available"

	default:
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "Internal server error"
	}

	c.JSON(statusCode, model.ErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}
