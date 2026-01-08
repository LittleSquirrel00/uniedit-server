package inbound

import "github.com/gin-gonic/gin"

// BillingHttpPort defines HTTP handler interface for billing operations.
type BillingHttpPort interface {
	// ListPlans handles GET /billing/plans
	ListPlans(c *gin.Context)

	// GetPlan handles GET /billing/plans/:id
	GetPlan(c *gin.Context)

	// GetSubscription handles GET /billing/subscription
	GetSubscription(c *gin.Context)

	// CreateSubscription handles POST /billing/subscription
	CreateSubscription(c *gin.Context)

	// CancelSubscription handles POST /billing/subscription/cancel
	CancelSubscription(c *gin.Context)

	// CreateCheckoutSession handles POST /billing/checkout
	CreateCheckoutSession(c *gin.Context)

	// CreatePortalSession handles POST /billing/portal
	CreatePortalSession(c *gin.Context)
}

// QuotaHttpPort defines HTTP handler interface for quota operations.
type QuotaHttpPort interface {
	// GetQuotaStatus handles GET /billing/quota
	GetQuotaStatus(c *gin.Context)

	// CheckQuota handles POST /billing/quota/check
	CheckQuota(c *gin.Context)
}

// UsageHttpPort defines HTTP handler interface for usage operations.
type UsageHttpPort interface {
	// GetUsageStats handles GET /billing/usage
	GetUsageStats(c *gin.Context)

	// RecordUsage handles POST /billing/usage (internal use)
	RecordUsage(c *gin.Context)
}

// CreditsHttpPort defines HTTP handler interface for credits operations.
type CreditsHttpPort interface {
	// GetBalance handles GET /billing/credits
	GetBalance(c *gin.Context)

	// AddCredits handles POST /billing/credits (admin only)
	AddCredits(c *gin.Context)
}

// StripeWebhookHttpPort defines HTTP handler interface for Stripe webhooks.
type StripeWebhookHttpPort interface {
	// HandleWebhook handles POST /webhooks/stripe
	HandleWebhook(c *gin.Context)
}
