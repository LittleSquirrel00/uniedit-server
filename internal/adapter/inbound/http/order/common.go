package orderhttp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/utils/middleware"
)

// getUserID returns the user ID from context.
func getUserID(c *gin.Context) uuid.UUID {
	return middleware.GetUserID(c)
}

// getUserIDFromContext returns the user ID from context with error response.
func getUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return uuid.Nil, false
	}
	return userID, true
}

// handleError maps order domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		statusCode = http.StatusNotFound
		errorCode = "order_not_found"
		message = "Order not found"

	case errors.Is(err, order.ErrOrderNotCancelable):
		statusCode = http.StatusBadRequest
		errorCode = "order_not_cancelable"
		message = "Order cannot be canceled"

	case errors.Is(err, order.ErrOrderNotRefundable):
		statusCode = http.StatusBadRequest
		errorCode = "order_not_refundable"
		message = "Order cannot be refunded"

	case errors.Is(err, order.ErrMinimumTopupAmount):
		statusCode = http.StatusBadRequest
		errorCode = "minimum_topup_amount"
		message = "Minimum top-up is $1.00"

	case errors.Is(err, order.ErrFreePlanNotOrderable):
		statusCode = http.StatusBadRequest
		errorCode = "free_plan_not_orderable"
		message = "Cannot create order for free plan"

	case errors.Is(err, order.ErrInvoiceNotFound):
		statusCode = http.StatusNotFound
		errorCode = "invoice_not_found"
		message = "Invoice not found"

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
