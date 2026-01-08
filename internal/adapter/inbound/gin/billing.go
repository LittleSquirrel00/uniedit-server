package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// billingHandler implements inbound.BillingHttpPort and inbound.QuotaHttpPort.
type billingHandler struct {
	billingDomain billing.BillingDomain
}

// NewBillingHandler creates a new billing HTTP handler.
func NewBillingHandler(billingDomain billing.BillingDomain) inbound.BillingHttpPort {
	return &billingHandler{billingDomain: billingDomain}
}

// NewQuotaHandler creates a new quota HTTP handler.
func NewQuotaHandler(billingDomain billing.BillingDomain) inbound.QuotaHttpPort {
	return &billingHandler{billingDomain: billingDomain}
}

func (h *billingHandler) ListPlans(c *gin.Context) {
	plans, err := h.billingDomain.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list plans"})
		return
	}

	// Convert to response format
	response := make([]gin.H, len(plans))
	for i, plan := range plans {
		response[i] = gin.H{
			"id":                       plan.ID,
			"type":                     plan.Type,
			"name":                     plan.Name,
			"description":              plan.Description,
			"billing_cycle":            plan.BillingCycle,
			"price_usd":                plan.PriceUSD,
			"monthly_tokens":           plan.MonthlyTokens,
			"daily_requests":           plan.DailyRequests,
			"max_api_keys":             plan.MaxAPIKeys,
			"features":                 plan.Features,
			"monthly_chat_tokens":      plan.MonthlyChatTokens,
			"monthly_image_credits":    plan.MonthlyImageCredits,
			"monthly_video_minutes":    plan.MonthlyVideoMinutes,
			"monthly_embedding_tokens": plan.MonthlyEmbeddingTokens,
			"git_storage_mb":           plan.GitStorageMB,
			"lfs_storage_mb":           plan.LFSStorageMB,
			"max_team_members":         plan.MaxTeamMembers,
		}
	}

	c.JSON(http.StatusOK, gin.H{"plans": response})
}

func (h *billingHandler) GetPlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan ID is required"})
		return
	}

	plan, err := h.billingDomain.GetPlan(c.Request.Context(), planID)
	if err != nil {
		if err == billing.ErrPlanNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plan"})
		return
	}

	c.JSON(http.StatusOK, plan.ToResponse())
}

func (h *billingHandler) GetSubscription(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	sub, err := h.billingDomain.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	c.JSON(http.StatusOK, sub.ToResponse())
}

func (h *billingHandler) CreateSubscription(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.billingDomain.CreateSubscription(c.Request.Context(), userID, req.PlanID)
	if err != nil {
		switch err {
		case billing.ErrSubscriptionExists:
			c.JSON(http.StatusConflict, gin.H{"error": "subscription already exists"})
		case billing.ErrPlanNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		case billing.ErrPlanNotActive:
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan is not active"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		}
		return
	}

	c.JSON(http.StatusCreated, sub.ToResponse())
}

func (h *billingHandler) CancelSubscription(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		Immediately bool `json:"immediately"`
	}
	_ = c.ShouldBindJSON(&req)

	sub, err := h.billingDomain.CancelSubscription(c.Request.Context(), userID, req.Immediately)
	if err != nil {
		switch err {
		case billing.ErrSubscriptionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		case billing.ErrSubscriptionCanceled:
			c.JSON(http.StatusBadRequest, gin.H{"error": "subscription is already canceled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		}
		return
	}

	c.JSON(http.StatusOK, sub.ToResponse())
}

func (h *billingHandler) CreateCheckoutSession(c *gin.Context) {
	// TODO: Implement Stripe checkout session creation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *billingHandler) CreatePortalSession(c *gin.Context) {
	// TODO: Implement Stripe portal session creation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// QuotaHttpPort implementation

func (h *billingHandler) GetQuotaStatus(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	status, err := h.billingDomain.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *billingHandler) CheckQuota(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		TaskType string `json:"task_type"`
	}
	_ = c.ShouldBindJSON(&req)

	err := h.billingDomain.CheckQuota(c.Request.Context(), userID, req.TaskType)
	if err != nil {
		switch err {
		case billing.ErrQuotaExceeded:
			c.JSON(http.StatusForbidden, gin.H{"error": "quota exceeded", "allowed": false})
		case billing.ErrTokenLimitReached:
			c.JSON(http.StatusForbidden, gin.H{"error": "token limit reached", "allowed": false})
		case billing.ErrRequestLimitReached:
			c.JSON(http.StatusForbidden, gin.H{"error": "request limit reached", "allowed": false})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check quota"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": true})
}

// Compile-time checks
var _ inbound.BillingHttpPort = (*billingHandler)(nil)
var _ inbound.QuotaHttpPort = (*billingHandler)(nil)
