package billinghttp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
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

// handleError maps billing domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, billing.ErrPlanNotFound):
		statusCode = http.StatusNotFound
		errorCode = "plan_not_found"
		message = "Plan not found"

	case errors.Is(err, billing.ErrPlanNotActive):
		statusCode = http.StatusBadRequest
		errorCode = "plan_not_active"
		message = "Plan is not active"

	case errors.Is(err, billing.ErrSubscriptionNotFound):
		statusCode = http.StatusNotFound
		errorCode = "subscription_not_found"
		message = "Subscription not found"

	case errors.Is(err, billing.ErrSubscriptionExists):
		statusCode = http.StatusConflict
		errorCode = "subscription_exists"
		message = "Subscription already exists"

	case errors.Is(err, billing.ErrSubscriptionCanceled):
		statusCode = http.StatusBadRequest
		errorCode = "subscription_canceled"
		message = "Subscription is already canceled"

	case errors.Is(err, billing.ErrQuotaExceeded):
		statusCode = http.StatusForbidden
		errorCode = "quota_exceeded"
		message = "Quota exceeded"

	case errors.Is(err, billing.ErrTokenLimitReached):
		statusCode = http.StatusForbidden
		errorCode = "token_limit_reached"
		message = "Token limit reached"

	case errors.Is(err, billing.ErrRequestLimitReached):
		statusCode = http.StatusForbidden
		errorCode = "request_limit_reached"
		message = "Request limit reached"

	case errors.Is(err, billing.ErrInvalidCreditsAmount):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_credits_amount"
		message = "Invalid credits amount"

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
