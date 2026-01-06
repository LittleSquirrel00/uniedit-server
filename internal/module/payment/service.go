package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"github.com/uniedit/server/internal/module/order"
	"github.com/uniedit/server/internal/module/payment/provider"
	"go.uber.org/zap"
)

// Service implements payment operations.
type Service struct {
	repo           Repository
	orderService   *order.Service
	billingService billing.ServiceInterface
	registry       *ProviderRegistry
	notifyBaseURL  string // Base URL for payment notifications
	logger         *zap.Logger
}

// NewService creates a new payment service.
func NewService(
	repo Repository,
	orderService *order.Service,
	billingService billing.ServiceInterface,
	registry *ProviderRegistry,
	notifyBaseURL string,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		orderService:   orderService,
		billingService: billingService,
		registry:       registry,
		notifyBaseURL:  notifyBaseURL,
		logger:         logger,
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent for an order.
func (s *Service) CreatePaymentIntent(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*PaymentIntentResponse, error) {
	// Get order
	ord, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID != userID {
		return nil, fmt.Errorf("forbidden")
	}

	// Check if order is pending
	if !ord.IsPending() {
		return nil, fmt.Errorf("order is not pending")
	}

	// Get Stripe provider
	stripeProvider, err := s.registry.Get("stripe")
	if err != nil {
		return nil, fmt.Errorf("stripe provider not available: %w", err)
	}

	// Get or create Stripe customer
	sub, err := s.billingService.GetSubscription(ctx, userID)
	var customerID string
	if err == nil && sub != nil {
		customerID = ""
	}

	// Create payment intent
	metadata := map[string]string{
		"order_id": orderID.String(),
		"user_id":  userID.String(),
	}

	pi, err := stripeProvider.CreatePaymentIntent(ctx, ord.Total, ord.Currency, customerID, metadata)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}

	// Store PaymentIntent ID on order
	if err := s.orderService.SetStripePaymentIntentID(ctx, orderID, pi.ID); err != nil {
		s.logger.Error("failed to set payment intent ID on order", zap.Error(err))
	}

	// Create payment record
	payment := &Payment{
		ID:                    uuid.New(),
		OrderID:               orderID,
		UserID:                userID,
		Amount:                ord.Total,
		Currency:              ord.Currency,
		Method:                PaymentMethodCard,
		Status:                PaymentStatusPending,
		Provider:              stripeProvider.Name(),
		StripePaymentIntentID: pi.ID,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		s.logger.Error("failed to create payment record", zap.Error(err))
	}

	return &PaymentIntentResponse{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
		Amount:          pi.Amount,
		Currency:        pi.Currency,
	}, nil
}

// CreateNativePayment creates a native payment (Alipay/WeChat) for an order.
func (s *Service) CreateNativePayment(ctx context.Context, req *CreateNativePaymentRequest, userID uuid.UUID) (*NativePaymentResponse, error) {
	// Get order
	ord, err := s.orderService.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID != userID {
		return nil, fmt.Errorf("forbidden")
	}

	// Check if order is pending
	if !ord.IsPending() {
		return nil, fmt.Errorf("order is not pending")
	}

	// Get native provider
	nativeProvider, err := s.registry.GetNativeByMethod(req.Method)
	if err != nil {
		return nil, fmt.Errorf("provider not available: %w", err)
	}

	// Build notify URL
	notifyURL := fmt.Sprintf("%s/webhooks/%s", s.notifyBaseURL, nativeProvider.Name())

	// Build metadata
	metadata := map[string]string{
		"order_id": req.OrderID.String(),
		"user_id":  userID.String(),
	}
	if req.OpenID != "" {
		metadata["openid"] = req.OpenID
	}

	// Determine currency based on provider
	currency := "CNY"

	// Create payment ID upfront
	paymentID := uuid.New()

	// Create native payment order
	nativeOrder, err := nativeProvider.CreateNativePayment(
		ctx,
		req.Scene,
		paymentID.String(), // Use payment ID as order ID for provider
		ord.Total,
		fmt.Sprintf("Order #%s", ord.ID.String()[:8]),
		"",
		notifyURL,
		req.ReturnURL,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("create native payment: %w", err)
	}

	// Create payment record
	payment := &Payment{
		ID:       paymentID,
		OrderID:  req.OrderID,
		UserID:   userID,
		Amount:   ord.Total,
		Currency: currency,
		Method:   req.Method,
		Status:   PaymentStatusPending,
		Provider: nativeProvider.Name(),
		TradeNo:  nativeOrder.TradeNo,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment record: %w", err)
	}

	return &NativePaymentResponse{
		PaymentID:   paymentID,
		OrderID:     req.OrderID,
		Method:      req.Method,
		PayURL:      nativeOrder.PayURL,
		QRCode:      nativeOrder.QRCode,
		AppPayData:  nativeOrder.AppPayData,
		MiniPayData: nativeOrder.MiniPayData,
		Amount:      nativeOrder.Amount,
		Currency:    nativeOrder.Currency,
		ExpireTime:  nativeOrder.ExpireTime,
	}, nil
}

// HandlePaymentSucceeded handles a successful payment.
func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID, chargeID string) error {
	// Get order
	ord, err := s.orderService.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// Update payment record
	payment, err := s.repo.GetPaymentByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		s.logger.Warn("payment record not found, creating new one", zap.String("payment_intent_id", paymentIntentID))
		payment = &Payment{
			ID:                    uuid.New(),
			OrderID:               ord.ID,
			UserID:                ord.UserID,
			Amount:                ord.Total,
			Currency:              ord.Currency,
			Method:                PaymentMethodCard,
			Provider:              "stripe",
			StripePaymentIntentID: paymentIntentID,
		}
	}

	now := time.Now()
	payment.Status = PaymentStatusSucceeded
	payment.StripeChargeID = chargeID
	payment.SucceededAt = &now

	if payment.ID == uuid.Nil {
		payment.ID = uuid.New()
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return fmt.Errorf("create payment: %w", err)
		}
	} else {
		if err := s.repo.UpdatePayment(ctx, payment); err != nil {
			return fmt.Errorf("update payment: %w", err)
		}
	}

	// Mark order as paid
	if err := s.orderService.MarkAsPaid(ctx, ord.ID); err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	// Fulfill order based on type
	switch ord.Type {
	case order.OrderTypeTopup:
		if err := s.billingService.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
			s.logger.Error("failed to add credits", zap.Error(err))
		}
	case order.OrderTypeSubscription:
		// Subscription is handled by Stripe webhook for subscription.created
		s.logger.Info("subscription order paid, waiting for subscription webhook",
			zap.String("order_id", ord.ID.String()),
		)
	}

	return nil
}

// HandlePaymentFailed handles a failed payment.
func (s *Service) HandlePaymentFailed(ctx context.Context, paymentIntentID, failureCode, failureMessage string) error {
	payment, err := s.repo.GetPaymentByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	now := time.Now()
	payment.Status = PaymentStatusFailed
	payment.FailureCode = &failureCode
	payment.FailureMessage = &failureMessage
	payment.FailedAt = &now

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	return nil
}

// CreateRefund creates a refund for a payment.
func (s *Service) CreateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}

	if !payment.IsSucceeded() {
		return fmt.Errorf("payment is not succeeded")
	}

	// Determine refund amount
	refundAmount := amount
	if refundAmount == 0 {
		refundAmount = payment.Amount - payment.RefundedAmount
	}

	// Handle refund based on provider
	switch payment.Provider {
	case "stripe":
		if payment.StripeChargeID == "" {
			return fmt.Errorf("no charge ID for refund")
		}
		stripeProvider, err := s.registry.Get("stripe")
		if err != nil {
			return fmt.Errorf("stripe provider not available: %w", err)
		}
		_, err = stripeProvider.CreateRefund(ctx, payment.StripeChargeID, refundAmount, reason)
		if err != nil {
			return fmt.Errorf("create stripe refund: %w", err)
		}

	case "alipay", "wechat":
		nativeProvider, err := s.registry.GetNative(payment.Provider)
		if err != nil {
			return fmt.Errorf("native provider not available: %w", err)
		}
		refundID := uuid.New().String()
		_, err = nativeProvider.RefundPayment(ctx, payment.ID.String(), payment.TradeNo, refundID, refundAmount, payment.Amount, reason)
		if err != nil {
			return fmt.Errorf("create native refund: %w", err)
		}

	default:
		return fmt.Errorf("unsupported provider for refund: %s", payment.Provider)
	}

	// Update payment
	payment.RefundedAmount += refundAmount
	if payment.RefundedAmount >= payment.Amount {
		payment.Status = PaymentStatusRefunded
	}

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// If full refund, mark order as refunded
	if payment.RefundedAmount >= payment.Amount {
		if err := s.orderService.MarkAsRefunded(ctx, payment.OrderID); err != nil {
			s.logger.Error("failed to mark order as refunded", zap.Error(err))
		}
	}

	return nil
}

// GetPayment returns a payment by ID.
func (s *Service) GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	return s.repo.GetPayment(ctx, paymentID)
}

// ListPaymentMethods returns payment methods for a user.
func (s *Service) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*PaymentMethodInfo, error) {
	sub, err := s.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Note: In a real implementation, we'd use the Stripe customer ID from subscription
	_ = sub

	// For now, return empty list
	return []*PaymentMethodInfo{}, nil
}

// VerifyWebhookSignature verifies a Stripe webhook signature.
func (s *Service) VerifyWebhookSignature(payload []byte, signature string) error {
	stripeProvider, err := s.registry.Get("stripe")
	if err != nil {
		return fmt.Errorf("stripe provider not available: %w", err)
	}
	return stripeProvider.VerifyWebhookSignature(payload, signature)
}

// WebhookEventExists checks if a webhook event has been processed.
func (s *Service) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
	return s.repo.WebhookEventExists(ctx, eventID)
}

// CreateWebhookEvent stores a webhook event.
func (s *Service) CreateWebhookEvent(ctx context.Context, event *StripeWebhookEvent) error {
	return s.repo.CreateWebhookEvent(ctx, event)
}

// MarkWebhookEventProcessed marks a webhook event as processed.
func (s *Service) MarkWebhookEventProcessed(ctx context.Context, eventID string, err error) error {
	return s.repo.MarkWebhookEventProcessed(ctx, eventID, err)
}

// HandleNativePaymentNotify handles a payment notification from Alipay/WeChat.
func (s *Service) HandleNativePaymentNotify(ctx context.Context, providerName string, body []byte, headers map[string]string) (string, error) {
	// Get native provider
	nativeProvider, err := s.registry.GetNative(providerName)
	if err != nil {
		return "", fmt.Errorf("native provider not available: %w", err)
	}

	// Parse and verify notification
	result, err := nativeProvider.ParseNotify(ctx, body, headers)
	if err != nil {
		return "", fmt.Errorf("parse notify: %w", err)
	}

	// Get payment by our order ID (out_trade_no is our payment ID)
	paymentID, err := uuid.Parse(result.OutTradeNo)
	if err != nil {
		return "", fmt.Errorf("invalid payment ID: %w", err)
	}

	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("get payment: %w", err)
	}

	// Check idempotency - if already processed, return success
	if payment.Status == PaymentStatusSucceeded {
		s.logger.Info("payment already succeeded, skipping",
			zap.String("payment_id", payment.ID.String()),
		)
		return result.SuccessResp, nil
	}

	// Store webhook event for idempotency
	eventID := fmt.Sprintf("%s:%s", providerName, result.TradeNo)
	webhookEvent := &PaymentWebhookEvent{
		Provider:   providerName,
		EventID:    eventID,
		EventType:  "payment",
		TradeNo:    result.TradeNo,
		OutTradeNo: result.OutTradeNo,
		Data:       result.RawData,
	}
	if err := s.repo.CreatePaymentWebhookEvent(ctx, webhookEvent); err != nil {
		// If duplicate, skip processing
		s.logger.Info("webhook event already exists, skipping",
			zap.String("event_id", eventID),
		)
		return result.SuccessResp, nil
	}

	// Process based on status
	var processErr error
	switch result.Status {
	case "success":
		processErr = s.handleNativePaymentSuccess(ctx, payment, result)
	case "closed":
		processErr = s.handleNativePaymentClosed(ctx, payment)
	default:
		s.logger.Info("ignoring notification with status",
			zap.String("status", result.Status),
			zap.String("payment_id", payment.ID.String()),
		)
	}

	// Mark event as processed
	if markErr := s.repo.MarkPaymentWebhookEventProcessed(ctx, webhookEvent.ID, processErr); markErr != nil {
		s.logger.Error("failed to mark webhook event processed", zap.Error(markErr))
	}

	if processErr != nil {
		return "", processErr
	}

	return result.SuccessResp, nil
}

// handleNativePaymentSuccess handles a successful native payment.
func (s *Service) handleNativePaymentSuccess(ctx context.Context, payment *Payment, result *provider.NotifyResult) error {
	// Update payment record
	now := time.Now()
	payment.Status = PaymentStatusSucceeded
	payment.TradeNo = result.TradeNo
	payment.PayerID = result.PayerID
	payment.SucceededAt = &now

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Mark order as paid
	if err := s.orderService.MarkAsPaid(ctx, payment.OrderID); err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	// Fulfill order
	ord, err := s.orderService.GetOrder(ctx, payment.OrderID)
	if err != nil {
		s.logger.Error("failed to get order for fulfillment", zap.Error(err))
		return nil // Don't fail the webhook for this
	}

	switch ord.Type {
	case order.OrderTypeTopup:
		if err := s.billingService.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
			s.logger.Error("failed to add credits", zap.Error(err))
		}
	}

	return nil
}

// handleNativePaymentClosed handles a closed/cancelled native payment.
func (s *Service) handleNativePaymentClosed(ctx context.Context, payment *Payment) error {
	now := time.Now()
	payment.Status = PaymentStatusCanceled
	payment.FailedAt = &now

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	return nil
}

// GetProviderRegistry returns the provider registry.
func (s *Service) GetProviderRegistry() *ProviderRegistry {
	return s.registry
}
