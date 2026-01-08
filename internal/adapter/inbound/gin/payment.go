package gin

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// paymentAdapter implements inbound.PaymentHttpPort.
type paymentAdapter struct {
	domain payment.PaymentDomain
}

// NewPaymentAdapter creates a new payment HTTP adapter.
func NewPaymentAdapter(domain payment.PaymentDomain) inbound.PaymentHttpPort {
	return &paymentAdapter{domain: domain}
}

// RegisterPaymentRoutes registers payment routes.
func RegisterPaymentRoutes(r *gin.RouterGroup, adapter inbound.PaymentHttpPort) {
	payments := r.Group("/payments")
	{
		payments.POST("/intent", adapter.CreatePaymentIntent)
		payments.POST("/native", adapter.CreateNativePayment)
		payments.GET("/:id", adapter.GetPayment)
		payments.GET("", adapter.ListPayments)
		payments.GET("/methods", adapter.ListPaymentMethods)
	}
}

func (a *paymentAdapter) CreatePaymentIntent(c *gin.Context) {
	var req model.CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	resp, err := a.domain.CreatePaymentIntent(c.Request.Context(), req.OrderID, userID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (a *paymentAdapter) CreateNativePayment(c *gin.Context) {
	var req model.CreateNativePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	resp, err := a.domain.CreateNativePayment(c.Request.Context(), &req, userID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (a *paymentAdapter) GetPayment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "invalid payment ID",
		})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	p, err := a.domain.GetPayment(c.Request.Context(), id)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	if p == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    "not_found",
			Message: "payment not found",
		})
		return
	}

	// Verify ownership
	if p.UserID != userID {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    "forbidden",
			Message: "forbidden",
		})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (a *paymentAdapter) ListPayments(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var filter model.PaymentFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	// Force user filter for non-admin
	filter.UserID = &userID

	payments, total, err := a.domain.ListPayments(c.Request.Context(), filter)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	filter.DefaultPagination()
	c.JSON(http.StatusOK, model.PaginatedResponse[*model.Payment]{
		Data:       payments,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
	})
}

func (a *paymentAdapter) ListPaymentMethods(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	methods, err := a.domain.ListPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, methods)
}

// Compile-time check
var _ inbound.PaymentHttpPort = (*paymentAdapter)(nil)

// --- Refund Adapter ---

// refundAdapter implements inbound.RefundHttpPort.
type refundAdapter struct {
	domain payment.PaymentDomain
}

// NewRefundAdapter creates a new refund HTTP adapter.
func NewRefundAdapter(domain payment.PaymentDomain) inbound.RefundHttpPort {
	return &refundAdapter{domain: domain}
}

// RegisterRefundRoutes registers refund routes (admin only).
func RegisterRefundRoutes(r *gin.RouterGroup, adapter inbound.RefundHttpPort) {
	r.POST("/payments/:id/refund", adapter.CreateRefund)
}

func (a *refundAdapter) CreateRefund(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "invalid payment ID",
		})
		return
	}

	var req model.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.CreateRefund(c.Request.Context(), id, req.Amount, req.Reason); err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "refund created"})
}

// Compile-time check
var _ inbound.RefundHttpPort = (*refundAdapter)(nil)

// --- Webhook Adapter ---

// webhookAdapter implements inbound.WebhookHttpPort.
type webhookAdapter struct {
	domain payment.PaymentDomain
}

// NewWebhookAdapter creates a new webhook HTTP adapter.
func NewWebhookAdapter(domain payment.PaymentDomain) inbound.WebhookHttpPort {
	return &webhookAdapter{domain: domain}
}

// RegisterWebhookRoutes registers webhook routes.
func RegisterWebhookRoutes(r *gin.RouterGroup, adapter inbound.WebhookHttpPort) {
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/stripe", adapter.HandleStripeWebhook)
		webhooks.POST("/alipay", adapter.HandleAlipayWebhook)
		webhooks.POST("/wechat", adapter.HandleWechatWebhook)
	}
}

func (a *webhookAdapter) HandleStripeWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_payload",
			Message: "failed to read request body",
		})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if err := a.domain.VerifyWebhookSignature(payload, signature); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_signature",
			Message: "webhook signature verification failed",
		})
		return
	}

	// TODO: Parse Stripe event and handle accordingly
	// This would typically use stripe.webhook.ConstructEvent

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (a *webhookAdapter) HandleAlipayWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := a.domain.HandleNativePaymentNotify(c.Request.Context(), "alipay", body, headers)
	if err != nil {
		c.String(http.StatusInternalServerError, "fail")
		return
	}

	c.String(http.StatusOK, resp)
}

func (a *webhookAdapter) HandleWechatWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "failed to read body"})
		return
	}

	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := a.domain.HandleNativePaymentNotify(c.Request.Context(), "wechat", body, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}

	// WeChat expects JSON response
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": resp})
}

// Compile-time check
var _ inbound.WebhookHttpPort = (*webhookAdapter)(nil)

// --- Error Handler ---

func handlePaymentError(c *gin.Context, err error) {
	switch err {
	case payment.ErrPaymentNotFound:
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    "not_found",
			Message: "payment not found",
		})
	case payment.ErrForbidden:
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    "forbidden",
			Message: "forbidden",
		})
	case payment.ErrOrderNotPending:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "order_not_pending",
			Message: "order is not pending",
		})
	case payment.ErrPaymentNotSucceeded:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "payment_not_succeeded",
			Message: "payment is not succeeded",
		})
	case payment.ErrInvalidRefundAmount:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_refund_amount",
			Message: "invalid refund amount",
		})
	case payment.ErrProviderNotAvailable:
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    "provider_unavailable",
			Message: "payment provider not available",
		})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    "internal_error",
			Message: "internal error",
		})
	}
}
