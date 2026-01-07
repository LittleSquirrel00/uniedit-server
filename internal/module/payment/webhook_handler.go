package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
	"github.com/uniedit/server/internal/module/billing"
	"go.uber.org/zap"
)

// WebhookHandler handles Stripe webhook events.
type WebhookHandler struct {
	paymentService *Service
	billingService billing.ServiceInterface
	logger         *zap.Logger
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(
	paymentService *Service,
	billingService billing.ServiceInterface,
	logger *zap.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		billingService: billingService,
		logger:         logger,
	}
}

// RegisterRoutes registers the webhook routes.
func (h *WebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/stripe", h.HandleStripeWebhook)
	r.POST("/alipay", h.HandleAlipayWebhook)
	r.POST("/wechat", h.HandleWechatWebhook)
}

// HandleStripeWebhook handles incoming Stripe webhook events.
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	// Read raw body for signature verification
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Verify signature
	signature := c.GetHeader("Stripe-Signature")
	if err := h.paymentService.VerifyWebhookSignature(payload, signature); err != nil {
		h.logger.Warn("invalid webhook signature", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	// Parse event
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		h.logger.Error("failed to parse webhook event", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event"})
		return
	}

	ctx := c.Request.Context()

	// Idempotency check
	exists, err := h.paymentService.WebhookEventExists(ctx, event.ID)
	if err != nil {
		h.logger.Error("failed to check event existence", zap.Error(err))
		// Continue processing - better to process twice than miss
	}
	if exists {
		h.logger.Info("webhook event already processed", zap.String("event_id", event.ID))
		c.JSON(http.StatusOK, gin.H{"status": "already_processed"})
		return
	}

	// Store event
	if err := h.paymentService.CreateWebhookEvent(ctx, event.ID, string(event.Type), string(payload)); err != nil {
		h.logger.Error("failed to store webhook event", zap.Error(err))
	}

	// Process event
	var processErr error
	switch event.Type {
	case "payment_intent.succeeded":
		processErr = h.handlePaymentIntentSucceeded(ctx, &event)
	case "payment_intent.payment_failed":
		processErr = h.handlePaymentIntentFailed(ctx, &event)
	case "customer.subscription.created":
		processErr = h.handleSubscriptionCreated(ctx, &event)
	case "customer.subscription.updated":
		processErr = h.handleSubscriptionUpdated(ctx, &event)
	case "customer.subscription.deleted":
		processErr = h.handleSubscriptionDeleted(ctx, &event)
	case "invoice.paid":
		processErr = h.handleInvoicePaid(ctx, &event)
	case "invoice.payment_failed":
		processErr = h.handleInvoicePaymentFailed(ctx, &event)
	default:
		h.logger.Debug("unhandled webhook event type", zap.String("type", string(event.Type)))
	}

	// Mark event as processed
	if err := h.paymentService.MarkWebhookEventProcessed(ctx, event.ID, processErr); err != nil {
		h.logger.Error("failed to mark event processed", zap.Error(err))
	}

	if processErr != nil {
		h.logger.Error("failed to process webhook event",
			zap.String("event_id", event.ID),
			zap.String("type", string(event.Type)),
			zap.Error(processErr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "processing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(ctx context.Context, event *stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return fmt.Errorf("unmarshal payment intent: %w", err)
	}

	h.logger.Info("payment intent succeeded",
		zap.String("payment_intent_id", pi.ID),
		zap.Int64("amount", pi.Amount),
	)

	// Get charge ID
	var chargeID string
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}

	return h.paymentService.HandlePaymentSucceeded(ctx, pi.ID, chargeID)
}

func (h *WebhookHandler) handlePaymentIntentFailed(ctx context.Context, event *stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return fmt.Errorf("unmarshal payment intent: %w", err)
	}

	h.logger.Warn("payment intent failed",
		zap.String("payment_intent_id", pi.ID),
	)

	var failureCode, failureMessage string
	if pi.LastPaymentError != nil {
		failureCode = string(pi.LastPaymentError.Code)
		failureMessage = pi.LastPaymentError.Msg
	}

	return h.paymentService.HandlePaymentFailed(ctx, pi.ID, failureCode, failureMessage)
}

func (h *WebhookHandler) handleSubscriptionCreated(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	h.logger.Info("subscription created",
		zap.String("subscription_id", sub.ID),
		zap.String("customer_id", sub.Customer.ID),
	)

	// Get plan ID from metadata or price lookup
	var planID string
	if len(sub.Items.Data) > 0 && sub.Items.Data[0].Price != nil {
		// Use price lookup_key or product ID as plan mapping
		if sub.Items.Data[0].Price.LookupKey != "" {
			planID = sub.Items.Data[0].Price.LookupKey
		}
	}

	if planID == "" {
		h.logger.Warn("no plan ID found in subscription", zap.String("subscription_id", sub.ID))
		return nil
	}

	// Note: The actual subscription creation in our system happens during order fulfillment.
	// This webhook confirms the Stripe subscription was created successfully.
	// We may want to sync the Stripe subscription ID to our subscription record here.

	return nil
}

func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	h.logger.Info("subscription updated",
		zap.String("subscription_id", sub.ID),
		zap.String("status", string(sub.Status)),
	)

	// Handle status changes
	switch sub.Status {
	case stripe.SubscriptionStatusCanceled:
		// Subscription was canceled
		h.logger.Info("subscription canceled via Stripe",
			zap.String("subscription_id", sub.ID),
		)
	case stripe.SubscriptionStatusPastDue:
		// Payment failed, subscription is past due
		h.logger.Warn("subscription past due",
			zap.String("subscription_id", sub.ID),
		)
	case stripe.SubscriptionStatusUnpaid:
		// Multiple payment failures
		h.logger.Warn("subscription unpaid",
			zap.String("subscription_id", sub.ID),
		)
	}

	// Note: In a complete implementation, we would update our local subscription record
	// to reflect the Stripe subscription status.

	return nil
}

func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	h.logger.Info("subscription deleted",
		zap.String("subscription_id", sub.ID),
	)

	// Mark local subscription as canceled if not already
	// This handles cases where the subscription was canceled directly in Stripe

	return nil
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, event *stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	h.logger.Info("invoice paid",
		zap.String("invoice_id", inv.ID),
		zap.Int64("amount_paid", inv.AmountPaid),
	)

	// For subscription renewals, this is where we would:
	// 1. Extend the subscription period
	// 2. Reset quotas for the new period
	// 3. Create an invoice record in our system

	return nil
}

func (h *WebhookHandler) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	h.logger.Warn("invoice payment failed",
		zap.String("invoice_id", inv.ID),
		zap.String("customer_id", inv.Customer.ID),
	)

	// Handle failed recurring payment:
	// 1. Notify user
	// 2. Update subscription status if needed
	// 3. Consider grace period before suspending service

	return nil
}

// WebhookConfig holds webhook configuration.
type WebhookConfig struct {
	EndpointSecret string
	// Timeout for webhook processing
	Timeout time.Duration
}

// HandleAlipayWebhook handles incoming Alipay webhook notifications.
func (h *WebhookHandler) HandleAlipayWebhook(c *gin.Context) {
	// Read raw body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read alipay webhook body", zap.Error(err))
		c.String(http.StatusBadRequest, "fail")
		return
	}

	h.logger.Debug("received alipay webhook", zap.String("body", string(payload)))

	// Process notification
	resp, err := h.paymentService.HandleNativePaymentNotify(
		c.Request.Context(),
		"alipay",
		payload,
		nil, // Alipay doesn't use headers for verification
	)
	if err != nil {
		h.logger.Error("failed to process alipay webhook",
			zap.Error(err),
		)
		c.String(http.StatusInternalServerError, "fail")
		return
	}

	// Alipay expects "success" as response
	c.String(http.StatusOK, resp)
}

// HandleWechatWebhook handles incoming WeChat Pay webhook notifications.
func (h *WebhookHandler) HandleWechatWebhook(c *gin.Context) {
	// Read raw body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read wechat webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "failed to read body"})
		return
	}

	h.logger.Debug("received wechat webhook", zap.String("body", string(payload)))

	// Extract headers for signature verification
	headers := map[string]string{
		"Wechatpay-Timestamp": c.GetHeader("Wechatpay-Timestamp"),
		"Wechatpay-Nonce":     c.GetHeader("Wechatpay-Nonce"),
		"Wechatpay-Signature": c.GetHeader("Wechatpay-Signature"),
		"Wechatpay-Serial":    c.GetHeader("Wechatpay-Serial"),
	}

	// Process notification
	resp, err := h.paymentService.HandleNativePaymentNotify(
		c.Request.Context(),
		"wechat",
		payload,
		headers,
	)
	if err != nil {
		h.logger.Error("failed to process wechat webhook",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}

	// WeChat expects JSON response
	c.Data(http.StatusOK, "application/json", []byte(resp))
}
