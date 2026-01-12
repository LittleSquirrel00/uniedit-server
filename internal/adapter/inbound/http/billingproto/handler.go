package billingproto

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	billingv1 "github.com/uniedit/server/api/pb/billing"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	billingDomain inbound.BillingDomain
}

func NewHandler(billingDomain inbound.BillingDomain) *Handler {
	return &Handler{billingDomain: billingDomain}
}

func (h *Handler) ListPlans(c *gin.Context, _ *commonv1.Empty) (*billingv1.ListPlansResponse, error) {
	out, err := h.billingDomain.ListPlans(c.Request.Context())
	if err != nil {
		return nil, mapBillingError(err)
	}
	return out, nil
}

func (h *Handler) GetPlan(c *gin.Context, in *billingv1.GetByIDRequest) (*billingv1.Plan, error) {
	plan, err := h.billingDomain.GetPlan(c.Request.Context(), in)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return plan, nil
}

func (h *Handler) GetSubscription(c *gin.Context, _ *commonv1.Empty) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	sub, err := h.billingDomain.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return sub, nil
}

func (h *Handler) CreateSubscription(c *gin.Context, in *billingv1.CreateSubscriptionRequest) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	sub, err := h.billingDomain.CreateSubscription(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapBillingError(err)
	}

	c.Status(http.StatusCreated)
	return sub, nil
}

func (h *Handler) CancelSubscription(c *gin.Context, in *billingv1.CancelSubscriptionRequest) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	sub, err := h.billingDomain.CancelSubscription(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return sub, nil
}

func (h *Handler) GetQuotaStatus(c *gin.Context, _ *commonv1.Empty) (*billingv1.QuotaStatus, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	status, err := h.billingDomain.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return status, nil
}

func (h *Handler) CheckQuota(c *gin.Context, in *billingv1.CheckQuotaRequest) (*billingv1.CheckQuotaResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	if err := h.billingDomain.CheckQuota(c.Request.Context(), userID, in); err != nil {
		c.Status(http.StatusForbidden)
		return &billingv1.CheckQuotaResponse{Allowed: false}, nil
	}

	return &billingv1.CheckQuotaResponse{Allowed: true}, nil
}

func (h *Handler) GetBalance(c *gin.Context, _ *commonv1.Empty) (*billingv1.GetBalanceResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.billingDomain.GetBalance(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return out, nil
}

func (h *Handler) GetUsageStats(c *gin.Context, in *billingv1.GetUsageStatsRequest) (*billingv1.UsageStats, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	stats, err := h.billingDomain.GetUsageStats(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return stats, nil
}

func (h *Handler) RecordUsage(c *gin.Context, in *billingv1.RecordUsageRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.billingDomain.RecordUsage(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return out, nil
}

func (h *Handler) AddCredits(c *gin.Context, in *billingv1.AddCreditsRequest) (*commonv1.MessageResponse, error) {
	out, err := h.billingDomain.AddCredits(c.Request.Context(), in)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return out, nil
}

func requireUserID(c *gin.Context) (uuid.UUID, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.Nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}
	return userID, nil
}

func mapBillingError(err error) error {
	switch {
	case errors.Is(err, billing.ErrInvalidRequest):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: "Invalid request", Err: err}
	case errors.Is(err, billing.ErrPlanNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "plan_not_found", Message: "Plan not found", Err: err}
	case errors.Is(err, billing.ErrPlanNotActive):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "plan_not_active", Message: "Plan is not active", Err: err}
	case errors.Is(err, billing.ErrSubscriptionNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "subscription_not_found", Message: "Subscription not found", Err: err}
	case errors.Is(err, billing.ErrSubscriptionExists):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "subscription_exists", Message: "Subscription already exists", Err: err}
	case errors.Is(err, billing.ErrSubscriptionCanceled):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "subscription_canceled", Message: "Subscription already canceled", Err: err}
	case errors.Is(err, billing.ErrQuotaExceeded),
		errors.Is(err, billing.ErrTokenLimitReached),
		errors.Is(err, billing.ErrRequestLimitReached):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "quota_exceeded", Message: err.Error(), Err: err}
	case errors.Is(err, billing.ErrInsufficientCredits):
		return &protohttp.HTTPError{Status: http.StatusPaymentRequired, Code: "insufficient_credits", Message: "Insufficient credits", Err: err}
	case errors.Is(err, billing.ErrInvalidCreditsAmount):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_credits_amount", Message: "Invalid credits amount", Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}
