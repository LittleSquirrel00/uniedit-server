package paymenthttp

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// WebhookHandler handles payment webhook HTTP requests.
type WebhookHandler struct {
	domain payment.PaymentDomain
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(domain payment.PaymentDomain) *WebhookHandler {
	return &WebhookHandler{domain: domain}
}

// RegisterRoutes registers webhook routes.
func (h *WebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/stripe", h.HandleStripeWebhook)
		webhooks.POST("/alipay", h.HandleAlipayWebhook)
		webhooks.POST("/wechat", h.HandleWechatWebhook)
	}
}

// HandleStripeWebhook handles POST /webhooks/stripe.
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_payload",
			Message: "Failed to read request body",
		})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if err := h.domain.VerifyWebhookSignature(payload, signature); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_signature",
			Message: "Webhook signature verification failed",
		})
		return
	}

	// TODO: Parse Stripe event and handle accordingly
	// This would typically use stripe.webhook.ConstructEvent

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// HandleAlipayWebhook handles POST /webhooks/alipay.
func (h *WebhookHandler) HandleAlipayWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := h.domain.HandleNativePaymentNotify(c.Request.Context(), "alipay", body, headers)
	if err != nil {
		c.String(http.StatusInternalServerError, "fail")
		return
	}

	c.String(http.StatusOK, resp)
}

// HandleWechatWebhook handles POST /webhooks/wechat.
func (h *WebhookHandler) HandleWechatWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "failed to read body"})
		return
	}

	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := h.domain.HandleNativePaymentNotify(c.Request.Context(), "wechat", body, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}

	// WeChat expects JSON response
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": resp})
}

// Compile-time check
var _ inbound.WebhookHttpPort = (*WebhookHandler)(nil)
