package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	billingCmd "github.com/uniedit/server/internal/app/command/billing"
	billingQuery "github.com/uniedit/server/internal/app/query/billing"
	"github.com/uniedit/server/internal/domain/billing"
)

// BillingHandler handles HTTP requests for billing using CQRS pattern.
type BillingHandler struct {
	createSubscription *billingCmd.CreateSubscriptionHandler
	cancelSubscription *billingCmd.CancelSubscriptionHandler
	getSubscription    *billingQuery.GetSubscriptionHandler
	listPlans          *billingQuery.ListPlansHandler
	getUsageStats      *billingQuery.GetUsageStatsHandler
	errorHandler       *ErrorHandler
}

// NewBillingHandler creates a new billing handler.
func NewBillingHandler(
	createSubscription *billingCmd.CreateSubscriptionHandler,
	cancelSubscription *billingCmd.CancelSubscriptionHandler,
	getSubscription *billingQuery.GetSubscriptionHandler,
	listPlans *billingQuery.ListPlansHandler,
	getUsageStats *billingQuery.GetUsageStatsHandler,
) *BillingHandler {
	return &BillingHandler{
		createSubscription: createSubscription,
		cancelSubscription: cancelSubscription,
		getSubscription:    getSubscription,
		listPlans:          listPlans,
		getUsageStats:      getUsageStats,
		errorHandler:       NewErrorHandler(),
	}
}

// RegisterRoutes registers public billing routes.
func (h *BillingHandler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/plans", h.ListPlans)
	}
}

// RegisterProtectedRoutes registers protected billing routes.
func (h *BillingHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/subscription", h.GetSubscription)
		billing.POST("/subscription", h.CreateSubscription)
		billing.POST("/subscription/cancel", h.CancelSubscription)
		billing.GET("/usage", h.GetUsageStats)
	}
}

// CreateSubscriptionRequest represents a request to create a subscription.
type CreateSubscriptionRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CreateSubscription handles POST /billing/subscription
func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.createSubscription.Handle(c.Request.Context(), billingCmd.CreateSubscriptionCommand{
		UserID: userID,
		PlanID: req.PlanID,
	})
	if err != nil {
		h.errorHandler.HandleBillingError(c, err)
		return
	}

	respondCreated(c, gin.H{
		"subscription": subscriptionToResponse(result.Subscription),
	})
}

// CancelSubscriptionRequest represents a request to cancel a subscription.
type CancelSubscriptionRequest struct {
	Immediately bool `json:"immediately"`
}

// CancelSubscription handles POST /billing/subscription/cancel
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CancelSubscriptionRequest
	_ = c.ShouldBindJSON(&req) // Optional, defaults to false

	result, err := h.cancelSubscription.Handle(c.Request.Context(), billingCmd.CancelSubscriptionCommand{
		UserID:      userID,
		Immediately: req.Immediately,
	})
	if err != nil {
		h.errorHandler.HandleBillingError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"subscription": subscriptionToResponse(result.Subscription),
	})
}

// GetSubscription handles GET /billing/subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	result, err := h.getSubscription.Handle(c.Request.Context(), billingQuery.GetSubscriptionQuery{
		UserID: userID,
	})
	if err != nil {
		h.errorHandler.HandleBillingError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"subscription": subscriptionToResponse(result.Subscription),
	})
}

// ListPlans handles GET /billing/plans
func (h *BillingHandler) ListPlans(c *gin.Context) {
	result, err := h.listPlans.Handle(c.Request.Context(), billingQuery.ListPlansQuery{})
	if err != nil {
		h.errorHandler.HandleBillingError(c, err)
		return
	}

	plans := make([]gin.H, len(result.Plans))
	for i, plan := range result.Plans {
		plans[i] = planToResponse(plan)
	}

	respondSuccess(c, gin.H{
		"plans": plans,
	})
}

// GetUsageStatsRequest represents a request to get usage stats.
type GetUsageStatsRequest struct {
	Period    string `form:"period"`     // day, week, month
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
}

// GetUsageStats handles GET /billing/usage
func (h *BillingHandler) GetUsageStats(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req GetUsageStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Calculate date range
	var startDate, endDate time.Time
	now := time.Now()

	switch req.Period {
	case "day":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate = startDate.Add(24 * time.Hour)
	case "week":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	default:
		// Default to current month
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	}

	result, err := h.getUsageStats.Handle(c.Request.Context(), billingQuery.GetUsageStatsQuery{
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		h.errorHandler.HandleBillingError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"usage": result.Stats,
	})
}

// Response helpers

func subscriptionToResponse(sub *billing.Subscription) gin.H {
	return gin.H{
		"id":                     sub.ID(),
		"user_id":                sub.UserID(),
		"plan_id":                sub.PlanID(),
		"status":                 sub.Status(),
		"current_period_start":   sub.CurrentPeriodStart(),
		"current_period_end":     sub.CurrentPeriodEnd(),
		"cancel_at_period_end":   sub.CancelAtPeriodEnd(),
		"credits_balance":        sub.CreditsBalance(),
		"created_at":             sub.CreatedAt(),
	}
}

func planToResponse(plan *billing.Plan) gin.H {
	return gin.H{
		"id":             plan.ID(),
		"name":           plan.Name(),
		"description":    plan.Description(),
		"type":           plan.Type(),
		"billing_cycle":  plan.BillingCycle(),
		"price_usd":      plan.PriceUSD(),
		"monthly_tokens": plan.MonthlyTokens(),
		"daily_requests": plan.DailyRequests(),
		"max_api_keys":   plan.MaxAPIKeys(),
		"features":       plan.Features(),
		"active":         plan.Active(),
	}
}
