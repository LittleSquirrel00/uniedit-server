package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/payment/domain"
	"github.com/uniedit/server/internal/module/payment/provider"
	"github.com/uniedit/server/internal/shared/events"
	"go.uber.org/zap"
)

// Service implements payment operations.
type Service struct {
	repo          Repository
	orderReader   OrderReader
	billingReader BillingReader
	eventBus      EventPublisher
	registry      *ProviderRegistry
	notifyBaseURL string // Base URL for payment notifications
	logger        *zap.Logger
}

// NewService creates a new payment service.
func NewService(
	repo Repository,
	orderReader OrderReader,
	billingReader BillingReader,
	eventBus EventPublisher,
	registry *ProviderRegistry,
	notifyBaseURL string,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:          repo,
		orderReader:   orderReader,
		billingReader: billingReader,
		eventBus:      eventBus,
		registry:      registry,
		notifyBaseURL: notifyBaseURL,
		logger:        logger,
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent for an order.
func (s *Service) CreatePaymentIntent(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*PaymentIntentResponse, error) {
	// Get order
	ord, err := s.orderReader.GetOrder(ctx, orderID)
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
	sub, err := s.billingReader.GetSubscription(ctx, userID)
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
	if err := s.orderReader.SetStripePaymentIntentID(ctx, orderID, pi.ID); err != nil {
		s.logger.Error("failed to set payment intent ID on order", zap.Error(err))
	}

	// Create payment record using domain factory
	payment := domain.NewPayment(orderID, userID, ord.Total, ord.Currency, domain.MethodCard, stripeProvider.Name())
	payment.SetStripePaymentIntentID(pi.ID)

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
	ord, err := s.orderReader.GetOrder(ctx, req.OrderID)
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

	// Create payment using domain factory
	payment := domain.NewPayment(req.OrderID, userID, ord.Total, currency, domain.PaymentMethod(req.Method), nativeProvider.Name())

	// Create native payment order
	nativeOrder, err := nativeProvider.CreateNativePayment(
		ctx,
		req.Scene,
		payment.ID().String(), // Use payment ID as order ID for provider
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

	// Set trade number
	payment.SetTradeNo(nativeOrder.TradeNo)

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment record: %w", err)
	}

	return &NativePaymentResponse{
		PaymentID:   payment.ID(),
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
	ord, err := s.orderReader.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// Update payment record
	payment, err := s.repo.GetPaymentByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		s.logger.Warn("payment record not found, creating new one", zap.String("payment_intent_id", paymentIntentID))
		// Create new payment using domain factory
		payment = domain.NewPayment(ord.ID, ord.UserID, ord.Total, ord.Currency, domain.MethodCard, "stripe")
		payment.SetStripePaymentIntentID(paymentIntentID)
	}

	// Use domain method to mark as succeeded
	if err := payment.MarkAsSucceeded(chargeID); err != nil {
		if err == domain.ErrPaymentAlreadySucceeded {
			s.logger.Info("payment already succeeded", zap.String("payment_intent_id", paymentIntentID))
			return nil
		}
		return fmt.Errorf("mark payment succeeded: %w", err)
	}

	// Save payment
	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Publish PaymentSucceeded event
	// Event handlers will:
	// - Update order status to "paid"
	// - Add credits for topup orders
	// - Handle subscription activation
	if s.eventBus != nil {
		event := events.NewPaymentSucceededEvent(
			payment.ID(),
			ord.ID,
			ord.UserID,
			payment.Amount(),
			payment.Currency(),
			payment.Provider(),
			ord.Type,
			ord.CreditsAmount,
			ord.PlanID,
		)
		s.eventBus.Publish(event)
	} else {
		// Fallback: direct handling if no event bus (for backward compatibility)
		if err := s.orderReader.UpdateOrderStatus(ctx, ord.ID, "paid"); err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}
		if ord.Type == OrderTypeTopup {
			if err := s.billingReader.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
				s.logger.Error("failed to add credits", zap.Error(err))
			}
		}
	}

	return nil
}

// HandlePaymentFailed handles a failed payment.
func (s *Service) HandlePaymentFailed(ctx context.Context, paymentIntentID, failureCode, failureMessage string) error {
	payment, err := s.repo.GetPaymentByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	// Use domain method
	if err := payment.MarkAsFailed(failureCode, failureMessage); err != nil {
		return fmt.Errorf("mark payment failed: %w", err)
	}

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Publish PaymentFailed event
	if s.eventBus != nil {
		event := events.NewPaymentFailedEvent(
			payment.ID(),
			payment.OrderID(),
			payment.UserID(),
			failureCode,
			failureMessage,
			payment.Provider(),
		)
		s.eventBus.Publish(event)
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

	// Use domain method to calculate refund amount
	refundAmount, err := payment.Refund(amount)
	if err != nil {
		return fmt.Errorf("refund: %w", err)
	}

	// Handle refund based on provider
	switch payment.Provider() {
	case "stripe":
		if payment.StripeChargeID() == "" {
			return fmt.Errorf("no charge ID for refund")
		}
		stripeProvider, err := s.registry.Get("stripe")
		if err != nil {
			return fmt.Errorf("stripe provider not available: %w", err)
		}
		_, err = stripeProvider.CreateRefund(ctx, payment.StripeChargeID(), refundAmount, reason)
		if err != nil {
			return fmt.Errorf("create stripe refund: %w", err)
		}

	case "alipay", "wechat":
		nativeProvider, err := s.registry.GetNative(payment.Provider())
		if err != nil {
			return fmt.Errorf("native provider not available: %w", err)
		}
		refundID := uuid.New().String()
		_, err = nativeProvider.RefundPayment(ctx, payment.ID().String(), payment.TradeNo(), refundID, refundAmount, payment.Amount(), reason)
		if err != nil {
			return fmt.Errorf("create native refund: %w", err)
		}

	default:
		return fmt.Errorf("unsupported provider for refund: %s", payment.Provider())
	}

	// Update payment
	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// If full refund, mark order as refunded
	if payment.Status() == domain.StatusRefunded {
		if err := s.orderReader.UpdateOrderStatus(ctx, payment.OrderID(), "refunded"); err != nil {
			s.logger.Error("failed to mark order as refunded", zap.Error(err))
		}
	}

	return nil
}

// GetPayment returns a payment by ID.
func (s *Service) GetPayment(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	return s.repo.GetPayment(ctx, paymentID)
}

// ListPaymentMethods returns payment methods for a user.
func (s *Service) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*PaymentMethodInfo, error) {
	sub, err := s.billingReader.GetSubscription(ctx, userID)
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
func (s *Service) CreateWebhookEvent(ctx context.Context, eventID, eventType, data string) error {
	event := domain.NewStripeWebhookEvent(eventID, eventType, data)
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
	if payment.Status() == domain.StatusSucceeded {
		s.logger.Info("payment already succeeded, skipping",
			zap.String("payment_id", payment.ID().String()),
		)
		return result.SuccessResp, nil
	}

	// Store webhook event for idempotency
	eventID := fmt.Sprintf("%s:%s", providerName, result.TradeNo)
	webhookEvent := domain.NewPaymentWebhookEvent(
		providerName,
		eventID,
		"payment",
		result.TradeNo,
		result.OutTradeNo,
		result.RawData,
	)
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
			zap.String("payment_id", payment.ID().String()),
		)
	}

	// Mark event as processed
	if markErr := s.repo.MarkPaymentWebhookEventProcessed(ctx, webhookEvent.ID(), processErr); markErr != nil {
		s.logger.Error("failed to mark webhook event processed", zap.Error(markErr))
	}

	if processErr != nil {
		return "", processErr
	}

	return result.SuccessResp, nil
}

// handleNativePaymentSuccess handles a successful native payment.
func (s *Service) handleNativePaymentSuccess(ctx context.Context, payment *domain.Payment, result *provider.NotifyResult) error {
	// Use domain method
	if err := payment.MarkAsSucceededNative(result.TradeNo, result.PayerID); err != nil {
		if err == domain.ErrPaymentAlreadySucceeded {
			return nil
		}
		return fmt.Errorf("mark payment succeeded: %w", err)
	}

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Get order for event
	ord, err := s.orderReader.GetOrder(ctx, payment.OrderID())
	if err != nil {
		s.logger.Error("failed to get order for event", zap.Error(err))
		return nil // Don't fail the webhook for this
	}

	// Publish PaymentSucceeded event
	if s.eventBus != nil {
		event := events.NewPaymentSucceededEvent(
			payment.ID(),
			ord.ID,
			ord.UserID,
			payment.Amount(),
			payment.Currency(),
			payment.Provider(),
			ord.Type,
			ord.CreditsAmount,
			ord.PlanID,
		)
		s.eventBus.Publish(event)
	} else {
		// Fallback: direct handling if no event bus
		if err := s.orderReader.UpdateOrderStatus(ctx, payment.OrderID(), "paid"); err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}
		if ord.Type == OrderTypeTopup {
			if err := s.billingReader.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
				s.logger.Error("failed to add credits", zap.Error(err))
			}
		}
	}

	return nil
}

// handleNativePaymentClosed handles a closed/cancelled native payment.
func (s *Service) handleNativePaymentClosed(ctx context.Context, payment *domain.Payment) error {
	// Use domain method
	if err := payment.MarkAsCanceled(); err != nil {
		return fmt.Errorf("mark payment canceled: %w", err)
	}

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	return nil
}

// GetProviderRegistry returns the provider registry.
func (s *Service) GetProviderRegistry() *ProviderRegistry {
	return s.registry
}
