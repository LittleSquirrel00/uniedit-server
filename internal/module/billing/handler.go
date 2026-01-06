package billing

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for billing.
type Handler struct {
	service ServiceInterface
}

// NewHandler creates a new billing handler.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the billing routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/plans", h.ListPlans)
		billing.GET("/subscription", h.GetSubscription)
		billing.POST("/subscription/cancel", h.CancelSubscription)
		billing.GET("/quota", h.GetQuotaStatus)
		billing.GET("/usage", h.GetUsageStats)
		billing.GET("/balance", h.GetBalance)
	}
}

// ListPlans returns all available plans.
func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.service.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list plans"})
		return
	}

	responses := make([]*PlanResponse, len(plans))
	for i, plan := range plans {
		responses[i] = plan.ToResponse()
	}

	c.JSON(http.StatusOK, GetPlansResponse{Plans: responses})
}

// GetSubscription returns the user's subscription.
func (h *Handler) GetSubscription(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.service.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	// Get quota status
	quota, err := h.service.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		quota = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub.ToResponse(),
		"quota":        quota,
	})
}

// CancelSubscription cancels the user's subscription.
func (h *Handler) CancelSubscription(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to cancel at period end
		req.Immediately = false
	}

	sub, err := h.service.CancelSubscription(c.Request.Context(), userID, req.Immediately)
	if err != nil {
		handleBillingError(c, err)
		return
	}

	c.JSON(http.StatusOK, sub.ToResponse())
}

// GetQuotaStatus returns the user's quota status.
func (h *Handler) GetQuotaStatus(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status, err := h.service.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		handleBillingError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetUsageStats returns the user's usage statistics.
func (h *Handler) GetUsageStats(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UsageStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetUsageStats(c.Request.Context(), userID, req.Period, req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetBalance returns the user's credits balance.
func (h *Handler) GetBalance(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	balance, err := h.service.GetBalance(c.Request.Context(), userID)
	if err != nil {
		handleBillingError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"credits_balance": balance})
}

// --- Helpers ---

func getUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

func handleBillingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPlanNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "plan_not_found"})
	case errors.Is(err, ErrPlanNotActive):
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan_not_active"})
	case errors.Is(err, ErrSubscriptionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription_not_found"})
	case errors.Is(err, ErrSubscriptionExists):
		c.JSON(http.StatusConflict, gin.H{"error": "subscription_exists"})
	case errors.Is(err, ErrSubscriptionCanceled):
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription_already_canceled"})
	case errors.Is(err, ErrQuotaExceeded):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "quota_exceeded"})
	case errors.Is(err, ErrInsufficientCredits):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "insufficient_credits"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
