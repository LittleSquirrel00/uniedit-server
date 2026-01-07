package order

import (
	"context"

	"github.com/uniedit/server/internal/module/order/domain"
	"github.com/uniedit/server/internal/shared/events"
	"go.uber.org/zap"
)

// EventHandler handles payment-related events for the order module.
type EventHandler struct {
	repo   Repository
	logger *zap.Logger
}

// NewEventHandler creates a new order event handler.
func NewEventHandler(repo Repository, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		repo:   repo,
		logger: logger,
	}
}

// Handles returns the list of event types this handler can process.
func (h *EventHandler) Handles() []string {
	return []string{
		events.PaymentSucceededType,
		events.PaymentFailedType,
	}
}

// Handle processes the given event.
func (h *EventHandler) Handle(event events.Event) error {
	switch e := event.(type) {
	case *events.PaymentSucceededEvent:
		return h.handlePaymentSucceeded(e)
	case *events.PaymentFailedEvent:
		return h.handlePaymentFailed(e)
	default:
		h.logger.Warn("unhandled event type",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}
}

// handlePaymentSucceeded updates order status to paid.
func (h *EventHandler) handlePaymentSucceeded(event *events.PaymentSucceededEvent) error {
	ctx := context.Background()

	order, err := h.repo.GetOrder(ctx, event.OrderID)
	if err != nil {
		h.logger.Error("failed to get order",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	// Idempotency check: if already paid, skip
	if order.Status() == domain.StatusPaid {
		h.logger.Info("order already paid, skipping",
			zap.String("order_id", event.OrderID.String()),
		)
		return nil
	}

	// Use domain method for state transition
	if err := order.MarkAsPaid(); err != nil {
		h.logger.Error("failed to transition order to paid",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	if err := h.repo.UpdateOrder(ctx, order); err != nil {
		h.logger.Error("failed to update order",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("order marked as paid",
		zap.String("order_id", event.OrderID.String()),
		zap.String("payment_id", event.PaymentID.String()),
	)

	return nil
}

// handlePaymentFailed updates order status to failed.
func (h *EventHandler) handlePaymentFailed(event *events.PaymentFailedEvent) error {
	ctx := context.Background()

	order, err := h.repo.GetOrder(ctx, event.OrderID)
	if err != nil {
		h.logger.Error("failed to get order",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	// Idempotency check: if already failed, skip
	if order.Status() == domain.StatusFailed {
		h.logger.Info("order already failed, skipping",
			zap.String("order_id", event.OrderID.String()),
		)
		return nil
	}

	// Use domain method for state transition
	if err := order.MarkAsFailed(); err != nil {
		h.logger.Error("failed to transition order to failed",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	if err := h.repo.UpdateOrder(ctx, order); err != nil {
		h.logger.Error("failed to update order",
			zap.String("order_id", event.OrderID.String()),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("order marked as failed",
		zap.String("order_id", event.OrderID.String()),
		zap.String("failure_code", event.FailureCode),
	)

	return nil
}

// Compile-time check that EventHandler implements events.Handler.
var _ events.Handler = (*EventHandler)(nil)
