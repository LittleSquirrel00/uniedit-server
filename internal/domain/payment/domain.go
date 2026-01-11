package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// PaymentDomain defines payment domain service interface.
type PaymentDomain interface {
	// CreatePaymentIntent creates a Stripe PaymentIntent for an order.
	CreatePaymentIntent(ctx context.Context, orderID, userID uuid.UUID) (*model.PaymentIntentResponse, error)

	// CreateNativePayment creates a native payment (Alipay/WeChat) for an order.
	CreateNativePayment(ctx context.Context, req *model.CreateNativePaymentRequest, userID uuid.UUID) (*model.NativePaymentResponse, error)

	// HandlePaymentSucceeded handles a successful Stripe payment.
	HandlePaymentSucceeded(ctx context.Context, paymentIntentID, chargeID string) error

	// HandlePaymentFailed handles a failed payment.
	HandlePaymentFailed(ctx context.Context, paymentIntentID, failureCode, failureMessage string) error

	// HandleNativePaymentNotify handles a payment notification from Alipay/WeChat.
	HandleNativePaymentNotify(ctx context.Context, providerName string, body []byte, headers map[string]string) (string, error)

	// CreateRefund creates a refund for a payment.
	CreateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error

	// GetPayment returns a payment by ID.
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error)

	// ListPayments lists payments by filter.
	ListPayments(ctx context.Context, filter model.PaymentFilter) ([]*model.Payment, int64, error)

	// ListPaymentMethods returns payment methods for a user.
	ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*model.PaymentMethodInfo, error)

	// VerifyWebhookSignature verifies a Stripe webhook signature.
	VerifyWebhookSignature(payload []byte, signature string) error

	// WebhookEventExists checks if a webhook event has been processed.
	WebhookEventExists(ctx context.Context, provider, eventID string) (bool, error)

	// CreateWebhookEvent stores a webhook event.
	CreateWebhookEvent(ctx context.Context, provider, eventID, eventType, data string) (*model.WebhookEvent, error)

	// MarkWebhookEventProcessed marks a webhook event as processed.
	MarkWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, err error) error
}

// paymentDomain implements PaymentDomain.
type paymentDomain struct {
	paymentDB      outbound.PaymentDatabasePort
	webhookDB      outbound.WebhookEventDatabasePort
	providerReg    outbound.PaymentProviderRegistryPort
	orderReader    outbound.OrderReaderPort
	billingReader  outbound.BillingReaderPort
	eventPublisher outbound.EventPublisherPort
	notifyBaseURL  string
	logger         *zap.Logger
}

// NewPaymentDomain creates a new payment domain service.
func NewPaymentDomain(
	paymentDB outbound.PaymentDatabasePort,
	webhookDB outbound.WebhookEventDatabasePort,
	providerReg outbound.PaymentProviderRegistryPort,
	orderReader outbound.OrderReaderPort,
	billingReader outbound.BillingReaderPort,
	eventPublisher outbound.EventPublisherPort,
	notifyBaseURL string,
	logger *zap.Logger,
) PaymentDomain {
	return &paymentDomain{
		paymentDB:      paymentDB,
		webhookDB:      webhookDB,
		providerReg:    providerReg,
		orderReader:    orderReader,
		billingReader:  billingReader,
		eventPublisher: eventPublisher,
		notifyBaseURL:  notifyBaseURL,
		logger:         logger,
	}
}

func (d *paymentDomain) CreatePaymentIntent(ctx context.Context, orderID, userID uuid.UUID) (*model.PaymentIntentResponse, error) {
	// Get order
	ord, err := d.orderReader.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID != userID {
		return nil, ErrForbidden
	}

	// Check if order is pending
	if !ord.IsPending() {
		return nil, ErrOrderNotPending
	}

	// Get Stripe provider
	stripeProvider, err := d.providerReg.Get("stripe")
	if err != nil {
		return nil, fmt.Errorf("%w: stripe", ErrProviderNotAvailable)
	}

	// Get customer ID from subscription if available
	var customerID string
	if sub, err := d.billingReader.GetSubscription(ctx, userID); err == nil && sub != nil {
		customerID = sub.StripeCustomerID
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
	if err := d.orderReader.SetStripePaymentIntentID(ctx, orderID, pi.ID); err != nil {
		d.logger.Error("failed to set payment intent ID on order", zap.Error(err))
	}

	// Create payment record
	now := time.Now()
	payment := &model.Payment{
		ID:                    uuid.New(),
		OrderID:               orderID,
		UserID:                userID,
		Amount:                ord.Total,
		Currency:              ord.Currency,
		Method:                model.PaymentMethodCard,
		Status:                model.PaymentStatusPending,
		Provider:              stripeProvider.Name(),
		StripePaymentIntentID: pi.ID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := d.paymentDB.Create(ctx, payment); err != nil {
		d.logger.Error("failed to create payment record", zap.Error(err))
	}

	return &model.PaymentIntentResponse{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
		Amount:          pi.Amount,
		Currency:        pi.Currency,
	}, nil
}

func (d *paymentDomain) CreateNativePayment(ctx context.Context, req *model.CreateNativePaymentRequest, userID uuid.UUID) (*model.NativePaymentResponse, error) {
	// Get order
	ord, err := d.orderReader.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID != userID {
		return nil, ErrForbidden
	}

	// Check if order is pending
	if !ord.IsPending() {
		return nil, ErrOrderNotPending
	}

	// Get native provider
	nativeProvider, err := d.providerReg.GetNativeByMethod(req.Method)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotAvailable, req.Method)
	}

	// Build notify URL
	notifyURL := fmt.Sprintf("%s/webhooks/%s", d.notifyBaseURL, nativeProvider.Name())

	// Create payment record first to get payment ID
	now := time.Now()
	payment := &model.Payment{
		ID:        uuid.New(),
		OrderID:   req.OrderID,
		UserID:    userID,
		Amount:    ord.Total,
		Currency:  "CNY",
		Method:    model.PaymentMethod(req.Method),
		Status:    model.PaymentStatusPending,
		Provider:  nativeProvider.Name(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Build metadata
	metadata := map[string]string{
		"order_id": req.OrderID.String(),
		"user_id":  userID.String(),
	}
	if req.OpenID != "" {
		metadata["openid"] = req.OpenID
	}

	// Create native payment order
	nativeOrder, err := nativeProvider.CreateNativePayment(
		ctx,
		req.Scene,
		payment.ID.String(),
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
	payment.TradeNo = nativeOrder.TradeNo

	if err := d.paymentDB.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment record: %w", err)
	}

	return &model.NativePaymentResponse{
		PaymentID:   payment.ID,
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

func (d *paymentDomain) HandlePaymentSucceeded(ctx context.Context, paymentIntentID, chargeID string) error {
	// Get order
	ord, err := d.orderReader.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// Get payment record
	payment, err := d.paymentDB.FindByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		d.logger.Warn("payment record not found, creating new one",
			zap.String("payment_intent_id", paymentIntentID))
		// Create new payment
		now := time.Now()
		payment = &model.Payment{
			ID:                    uuid.New(),
			OrderID:               ord.ID,
			UserID:                ord.UserID,
			Amount:                ord.Total,
			Currency:              ord.Currency,
			Method:                model.PaymentMethodCard,
			Status:                model.PaymentStatusPending,
			Provider:              "stripe",
			StripePaymentIntentID: paymentIntentID,
			CreatedAt:             now,
			UpdatedAt:             now,
		}
	}

	// Check if already succeeded
	if payment.Status.IsSucceeded() {
		d.logger.Info("payment already succeeded",
			zap.String("payment_intent_id", paymentIntentID))
		return nil
	}

	// Check status transition
	if !payment.Status.CanTransitionTo(model.PaymentStatusSucceeded) {
		return ErrInvalidStatusTransition
	}

	// Update payment
	now := time.Now()
	payment.Status = model.PaymentStatusSucceeded
	payment.StripeChargeID = chargeID
	payment.SucceededAt = &now
	payment.UpdatedAt = now

	if err := d.paymentDB.Update(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Publish event or handle directly
	if d.eventPublisher != nil {
		// Publish PaymentSucceeded event
		event := &PaymentSucceededEvent{
			PaymentID:     payment.ID,
			OrderID:       ord.ID,
			UserID:        ord.UserID,
			Amount:        payment.Amount,
			Currency:      payment.Currency,
			Provider:      payment.Provider,
			OrderType:     ord.Type,
			CreditsAmount: ord.CreditsAmount,
			PlanID:        ord.PlanID,
		}
		if err := d.eventPublisher.Publish(ctx, event); err != nil {
			d.logger.Error("failed to publish payment succeeded event", zap.Error(err))
		}
	} else {
		// Fallback: direct handling
		if err := d.orderReader.UpdateOrderStatus(ctx, ord.ID, "paid"); err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}
		if ord.Type == "topup" {
			if err := d.billingReader.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
				d.logger.Error("failed to add credits", zap.Error(err))
			}
		}
	}

	return nil
}

func (d *paymentDomain) HandlePaymentFailed(ctx context.Context, paymentIntentID, failureCode, failureMessage string) error {
	payment, err := d.paymentDB.FindByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	// Check status transition
	if !payment.Status.CanTransitionTo(model.PaymentStatusFailed) {
		return ErrInvalidStatusTransition
	}

	// Update payment
	now := time.Now()
	payment.Status = model.PaymentStatusFailed
	payment.FailureCode = &failureCode
	payment.FailureMessage = &failureMessage
	payment.FailedAt = &now
	payment.UpdatedAt = now

	if err := d.paymentDB.Update(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Publish event
	if d.eventPublisher != nil {
		event := &PaymentFailedEvent{
			PaymentID:      payment.ID,
			OrderID:        payment.OrderID,
			UserID:         payment.UserID,
			FailureCode:    failureCode,
			FailureMessage: failureMessage,
			Provider:       payment.Provider,
		}
		if err := d.eventPublisher.Publish(ctx, event); err != nil {
			d.logger.Error("failed to publish payment failed event", zap.Error(err))
		}
	}

	return nil
}

func (d *paymentDomain) HandleNativePaymentNotify(ctx context.Context, providerName string, body []byte, headers map[string]string) (string, error) {
	// Get native provider
	nativeProvider, err := d.providerReg.GetNative(providerName)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrProviderNotAvailable, providerName)
	}

	// Parse and verify notification
	result, err := nativeProvider.ParseNotify(ctx, body, headers)
	if err != nil {
		return "", fmt.Errorf("parse notify: %w", err)
	}

	// Get payment by our payment ID (out_trade_no is our payment ID)
	paymentID, err := uuid.Parse(result.OutTradeNo)
	if err != nil {
		return "", fmt.Errorf("invalid payment ID: %w", err)
	}

	payment, err := d.paymentDB.FindByID(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("get payment: %w", err)
	}

	// Check idempotency - if already processed, return success
	if payment.Status == model.PaymentStatusSucceeded {
		d.logger.Info("payment already succeeded, skipping",
			zap.String("payment_id", payment.ID.String()))
		return result.SuccessResp, nil
	}

	// Store webhook event for idempotency
	eventID := fmt.Sprintf("%s:%s", providerName, result.TradeNo)
	webhookEvent, err := d.CreateWebhookEvent(ctx, providerName, eventID, "payment", result.RawData)
	if err != nil {
		// If duplicate, skip processing
		d.logger.Info("webhook event already exists, skipping",
			zap.String("event_id", eventID))
		return result.SuccessResp, nil
	}

	// Process based on status
	var processErr error
	switch result.Status {
	case "success":
		processErr = d.handleNativePaymentSuccess(ctx, payment, result)
	case "closed":
		processErr = d.handleNativePaymentClosed(ctx, payment)
	default:
		d.logger.Info("ignoring notification with status",
			zap.String("status", result.Status),
			zap.String("payment_id", payment.ID.String()))
	}

	// Mark event as processed
	if markErr := d.webhookDB.MarkProcessed(ctx, webhookEvent.ID, processErr); markErr != nil {
		d.logger.Error("failed to mark webhook event processed", zap.Error(markErr))
	}

	if processErr != nil {
		return "", processErr
	}

	return result.SuccessResp, nil
}

func (d *paymentDomain) handleNativePaymentSuccess(ctx context.Context, payment *model.Payment, result *model.ProviderNotifyResult) error {
	// Check status transition
	if payment.Status.IsSucceeded() {
		return nil
	}
	if !payment.Status.CanTransitionTo(model.PaymentStatusSucceeded) {
		return ErrInvalidStatusTransition
	}

	// Update payment
	now := time.Now()
	payment.Status = model.PaymentStatusSucceeded
	payment.TradeNo = result.TradeNo
	payment.PayerID = result.PayerID
	payment.SucceededAt = &now
	payment.UpdatedAt = now

	if err := d.paymentDB.Update(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// Get order for event
	ord, err := d.orderReader.GetOrder(ctx, payment.OrderID)
	if err != nil {
		d.logger.Error("failed to get order for event", zap.Error(err))
		return nil
	}

	// Publish event or handle directly
	if d.eventPublisher != nil {
		event := &PaymentSucceededEvent{
			PaymentID:     payment.ID,
			OrderID:       ord.ID,
			UserID:        ord.UserID,
			Amount:        payment.Amount,
			Currency:      payment.Currency,
			Provider:      payment.Provider,
			OrderType:     ord.Type,
			CreditsAmount: ord.CreditsAmount,
			PlanID:        ord.PlanID,
		}
		if err := d.eventPublisher.Publish(ctx, event); err != nil {
			d.logger.Error("failed to publish payment succeeded event", zap.Error(err))
		}
	} else {
		// Fallback: direct handling
		if err := d.orderReader.UpdateOrderStatus(ctx, payment.OrderID, "paid"); err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}
		if ord.Type == "topup" {
			if err := d.billingReader.AddCredits(ctx, ord.UserID, ord.CreditsAmount, "topup"); err != nil {
				d.logger.Error("failed to add credits", zap.Error(err))
			}
		}
	}

	return nil
}

func (d *paymentDomain) handleNativePaymentClosed(ctx context.Context, payment *model.Payment) error {
	// Check status transition
	if !payment.Status.CanTransitionTo(model.PaymentStatusCanceled) {
		return ErrInvalidStatusTransition
	}

	// Update payment
	now := time.Now()
	payment.Status = model.PaymentStatusCanceled
	payment.FailedAt = &now
	payment.UpdatedAt = now

	if err := d.paymentDB.Update(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	return nil
}

func (d *paymentDomain) CreateRefund(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) error {
	payment, err := d.paymentDB.FindByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if !payment.Status.IsSucceeded() && payment.Status != model.PaymentStatusRefunded {
		return ErrPaymentNotSucceeded
	}

	// Calculate refund amount
	refundAmount := amount
	if refundAmount == 0 {
		refundAmount = payment.Amount - payment.RefundedAmount
	}

	// Validate refund amount
	if refundAmount <= 0 || refundAmount > (payment.Amount-payment.RefundedAmount) {
		return ErrInvalidRefundAmount
	}

	// Handle refund based on provider
	switch payment.Provider {
	case "stripe":
		if payment.StripeChargeID == "" {
			return ErrNoChargeID
		}
		stripeProvider, err := d.providerReg.Get("stripe")
		if err != nil {
			return fmt.Errorf("%w: stripe", ErrProviderNotAvailable)
		}
		_, err = stripeProvider.CreateRefund(ctx, payment.StripeChargeID, refundAmount, reason)
		if err != nil {
			return fmt.Errorf("create stripe refund: %w", err)
		}

	case "alipay", "wechat":
		nativeProvider, err := d.providerReg.GetNative(payment.Provider)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrProviderNotAvailable, payment.Provider)
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
		payment.Status = model.PaymentStatusRefunded
	}
	payment.UpdatedAt = time.Now()

	if err := d.paymentDB.Update(ctx, payment); err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	// If full refund, mark order as refunded
	if payment.Status == model.PaymentStatusRefunded {
		if err := d.orderReader.UpdateOrderStatus(ctx, payment.OrderID, "refunded"); err != nil {
			d.logger.Error("failed to mark order as refunded", zap.Error(err))
		}
	}

	return nil
}

func (d *paymentDomain) GetPayment(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error) {
	return d.paymentDB.FindByID(ctx, paymentID)
}

func (d *paymentDomain) ListPayments(ctx context.Context, filter model.PaymentFilter) ([]*model.Payment, int64, error) {
	return d.paymentDB.FindByFilter(ctx, filter)
}

func (d *paymentDomain) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*model.PaymentMethodInfo, error) {
	sub, err := d.billingReader.GetSubscription(ctx, userID)
	if err != nil || sub == nil || sub.StripeCustomerID == "" {
		return []*model.PaymentMethodInfo{}, nil
	}

	stripeProvider, err := d.providerReg.Get("stripe")
	if err != nil {
		return []*model.PaymentMethodInfo{}, nil
	}

	return stripeProvider.ListPaymentMethods(ctx, sub.StripeCustomerID)
}

func (d *paymentDomain) VerifyWebhookSignature(payload []byte, signature string) error {
	stripeProvider, err := d.providerReg.Get("stripe")
	if err != nil {
		return fmt.Errorf("%w: stripe", ErrProviderNotAvailable)
	}
	return stripeProvider.VerifyWebhookSignature(payload, signature)
}

func (d *paymentDomain) WebhookEventExists(ctx context.Context, provider, eventID string) (bool, error) {
	return d.webhookDB.Exists(ctx, provider, eventID)
}

func (d *paymentDomain) CreateWebhookEvent(ctx context.Context, provider, eventID, eventType, data string) (*model.WebhookEvent, error) {
	event := &model.WebhookEvent{
		ID:        uuid.New(),
		Provider:  provider,
		EventID:   eventID,
		EventType: eventType,
		Data:      data,
		Processed: false,
		CreatedAt: time.Now(),
	}
	if err := d.webhookDB.Create(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (d *paymentDomain) MarkWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, err error) error {
	return d.webhookDB.MarkProcessed(ctx, eventID, err)
}

// --- Domain Events ---

// PaymentSucceededEvent is published when a payment succeeds.
type PaymentSucceededEvent struct {
	PaymentID     uuid.UUID
	OrderID       uuid.UUID
	UserID        uuid.UUID
	Amount        int64
	Currency      string
	Provider      string
	OrderType     string
	CreditsAmount int64
	PlanID        string
}

// PaymentFailedEvent is published when a payment fails.
type PaymentFailedEvent struct {
	PaymentID      uuid.UUID
	OrderID        uuid.UUID
	UserID         uuid.UUID
	FailureCode    string
	FailureMessage string
	Provider       string
}
