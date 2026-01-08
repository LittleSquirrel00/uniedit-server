package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
)

// ErrorHandler provides centralized error handling for HTTP responses.
type ErrorHandler struct{}

// NewErrorHandler creates a new error handler.
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleBillingError handles billing domain errors.
func (h *ErrorHandler) HandleBillingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, billing.ErrPlanNotFound):
		respondError(c, http.StatusNotFound, "plan_not_found", "Plan not found")
	case errors.Is(err, billing.ErrPlanNotActive):
		respondError(c, http.StatusBadRequest, "plan_not_active", "Plan is not active")
	case errors.Is(err, billing.ErrSubscriptionNotFound):
		respondError(c, http.StatusNotFound, "subscription_not_found", "Subscription not found")
	case errors.Is(err, billing.ErrSubscriptionExists):
		respondError(c, http.StatusConflict, "subscription_exists", "Subscription already exists")
	case errors.Is(err, billing.ErrSubscriptionCanceled):
		respondError(c, http.StatusBadRequest, "subscription_canceled", "Subscription is already canceled")
	case errors.Is(err, billing.ErrQuotaExceeded):
		respondError(c, http.StatusPaymentRequired, "quota_exceeded", "Quota exceeded")
	case errors.Is(err, billing.ErrInsufficientCredits):
		respondError(c, http.StatusPaymentRequired, "insufficient_credits", "Insufficient credits")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

// HandleOrderError handles order domain errors.
func (h *ErrorHandler) HandleOrderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		respondError(c, http.StatusNotFound, "order_not_found", "Order not found")
	case errors.Is(err, order.ErrOrderNotPending):
		respondError(c, http.StatusBadRequest, "order_not_pending", "Order is not pending")
	case errors.Is(err, order.ErrOrderNotCancelable):
		respondError(c, http.StatusBadRequest, "order_not_cancelable", "Order cannot be canceled")
	case errors.Is(err, order.ErrInvalidTransition):
		respondError(c, http.StatusBadRequest, "invalid_state_transition", "Invalid order state transition")
	case errors.Is(err, order.ErrInvoiceNotFound):
		respondError(c, http.StatusNotFound, "invoice_not_found", "Invoice not found")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

// HandlePaymentError handles payment domain errors.
func (h *ErrorHandler) HandlePaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		respondError(c, http.StatusNotFound, "payment_not_found", "Payment not found")
	case errors.Is(err, payment.ErrInvalidPayment):
		respondError(c, http.StatusBadRequest, "invalid_payment", "Invalid payment")
	case errors.Is(err, payment.ErrRefundFailed):
		respondError(c, http.StatusBadRequest, "refund_failed", "Refund failed")
	case errors.Is(err, payment.ErrWebhookDuplicate):
		respondError(c, http.StatusOK, "webhook_duplicate", "Webhook event already processed")
	default:
		// Check for specific error messages
		errMsg := err.Error()
		switch errMsg {
		case "forbidden":
			respondError(c, http.StatusForbidden, "forbidden", "Access denied")
		case "order is not pending":
			respondError(c, http.StatusBadRequest, "order_not_pending", "Order is not pending")
		case "payment is not succeeded":
			respondError(c, http.StatusBadRequest, "payment_not_succeeded", "Payment has not succeeded")
		case "no charge ID for refund":
			respondError(c, http.StatusBadRequest, "no_charge_for_refund", "No charge ID available for refund")
		default:
			respondError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
	}
}

// HandleError handles a generic error based on its type.
func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	// Try billing errors first
	if errors.Is(err, billing.ErrPlanNotFound) ||
		errors.Is(err, billing.ErrSubscriptionNotFound) ||
		errors.Is(err, billing.ErrQuotaExceeded) {
		h.HandleBillingError(c, err)
		return
	}

	// Try order errors
	if errors.Is(err, order.ErrOrderNotFound) ||
		errors.Is(err, order.ErrInvoiceNotFound) ||
		errors.Is(err, order.ErrInvalidTransition) {
		h.HandleOrderError(c, err)
		return
	}

	// Try payment errors
	if errors.Is(err, payment.ErrPaymentNotFound) ||
		errors.Is(err, payment.ErrInvalidPayment) {
		h.HandlePaymentError(c, err)
		return
	}

	// Default to internal error
	respondError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
}
