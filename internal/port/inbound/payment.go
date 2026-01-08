package inbound

import "github.com/gin-gonic/gin"

// PaymentHttpPort defines HTTP handler interface for payment operations.
type PaymentHttpPort interface {
	// CreatePaymentIntent handles POST /payments/intent
	// Creates a Stripe PaymentIntent for an order.
	CreatePaymentIntent(c *gin.Context)

	// CreateNativePayment handles POST /payments/native
	// Creates a native payment (Alipay/WeChat) for an order.
	CreateNativePayment(c *gin.Context)

	// GetPayment handles GET /payments/:id
	// Returns payment details by ID.
	GetPayment(c *gin.Context)

	// ListPayments handles GET /payments
	// Lists payments for the current user.
	ListPayments(c *gin.Context)

	// ListPaymentMethods handles GET /payments/methods
	// Lists saved payment methods for the current user.
	ListPaymentMethods(c *gin.Context)
}

// RefundHttpPort defines HTTP handler interface for refund operations.
type RefundHttpPort interface {
	// CreateRefund handles POST /payments/:id/refund
	// Creates a refund for a payment (admin only).
	CreateRefund(c *gin.Context)
}

// WebhookHttpPort defines HTTP handler interface for webhook operations.
type WebhookHttpPort interface {
	// HandleStripeWebhook handles POST /webhooks/stripe
	// Processes Stripe webhook events.
	HandleStripeWebhook(c *gin.Context)

	// HandleAlipayWebhook handles POST /webhooks/alipay
	// Processes Alipay webhook notifications.
	HandleAlipayWebhook(c *gin.Context)

	// HandleWechatWebhook handles POST /webhooks/wechat
	// Processes WeChat Pay webhook notifications.
	HandleWechatWebhook(c *gin.Context)
}
