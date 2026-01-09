package billinghttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// SubscriptionHandler handles subscription HTTP requests.
type SubscriptionHandler struct {
	billingDomain billing.BillingDomain
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(billingDomain billing.BillingDomain) *SubscriptionHandler {
	return &SubscriptionHandler{billingDomain: billingDomain}
}

// RegisterRoutes registers subscription routes.
func (h *SubscriptionHandler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/plans", h.ListPlans)
		billing.GET("/plans/:id", h.GetPlan)
		billing.GET("/subscription", h.GetSubscription)
		billing.POST("/subscription", h.CreateSubscription)
		billing.POST("/subscription/cancel", h.CancelSubscription)
		billing.POST("/checkout", h.CreateCheckoutSession)
		billing.POST("/portal", h.CreatePortalSession)
	}
}

// ListPlans handles GET /billing/plans.
func (h *SubscriptionHandler) ListPlans(c *gin.Context) {
	plans, err := h.billingDomain.ListPlans(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

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

// GetPlan handles GET /billing/plans/:id.
func (h *SubscriptionHandler) GetPlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan ID is required"})
		return
	}

	plan, err := h.billingDomain.GetPlan(c.Request.Context(), planID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan.ToResponse())
}

// GetSubscription handles GET /billing/subscription.
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	sub, err := h.billingDomain.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, sub.ToResponse())
}

// CreateSubscription handles POST /billing/subscription.
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
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
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, sub.ToResponse())
}

// CancelSubscription handles POST /billing/subscription/cancel.
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		Immediately bool `json:"immediately"`
	}
	_ = c.ShouldBindJSON(&req)

	sub, err := h.billingDomain.CancelSubscription(c.Request.Context(), userID, req.Immediately)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, sub.ToResponse())
}

// CreateCheckoutSession handles POST /billing/checkout.
func (h *SubscriptionHandler) CreateCheckoutSession(c *gin.Context) {
	// TODO: Implement Stripe checkout session creation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// CreatePortalSession handles POST /billing/portal.
func (h *SubscriptionHandler) CreatePortalSession(c *gin.Context) {
	// TODO: Implement Stripe portal session creation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// QuotaHandler handles quota HTTP requests.
type QuotaHandler struct {
	billingDomain billing.BillingDomain
}

// NewQuotaHandler creates a new quota handler.
func NewQuotaHandler(billingDomain billing.BillingDomain) *QuotaHandler {
	return &QuotaHandler{billingDomain: billingDomain}
}

// RegisterRoutes registers quota routes.
func (h *QuotaHandler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/quota", h.GetQuotaStatus)
		billing.POST("/quota/check", h.CheckQuota)
	}
}

// GetQuotaStatus handles GET /billing/quota.
func (h *QuotaHandler) GetQuotaStatus(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	status, err := h.billingDomain.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// CheckQuota handles POST /billing/quota/check.
func (h *QuotaHandler) CheckQuota(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		TaskType string `json:"task_type"`
	}
	_ = c.ShouldBindJSON(&req)

	err := h.billingDomain.CheckQuota(c.Request.Context(), userID, req.TaskType)
	if err != nil {
		handleError(c, err)
		c.JSON(http.StatusForbidden, gin.H{"allowed": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": true})
}

// Compile-time checks
var _ inbound.BillingHttpPort = (*SubscriptionHandler)(nil)
var _ inbound.QuotaHttpPort = (*QuotaHandler)(nil)
