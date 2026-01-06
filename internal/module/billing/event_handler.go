package billing

import (
	"context"

	"github.com/uniedit/server/internal/shared/events"
	"go.uber.org/zap"
)

// EventHandler handles payment-related events for the billing module.
type EventHandler struct {
	service ServiceInterface
	logger  *zap.Logger
}

// NewEventHandler creates a new billing event handler.
func NewEventHandler(service ServiceInterface, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		service: service,
		logger:  logger,
	}
}

// Handles returns the list of event types this handler can process.
func (h *EventHandler) Handles() []string {
	return []string{
		events.PaymentSucceededType,
	}
}

// Handle processes the given event.
func (h *EventHandler) Handle(event events.Event) error {
	switch e := event.(type) {
	case *events.PaymentSucceededEvent:
		return h.handlePaymentSucceeded(e)
	default:
		h.logger.Warn("unhandled event type",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}
}

// handlePaymentSucceeded adds credits for topup orders.
func (h *EventHandler) handlePaymentSucceeded(event *events.PaymentSucceededEvent) error {
	// Only process topup orders
	if event.OrderType != events.OrderTypeTopup {
		h.logger.Debug("skipping non-topup order",
			zap.String("order_type", event.OrderType),
			zap.String("order_id", event.OrderID.String()),
		)
		return nil
	}

	if event.CreditsAmount <= 0 {
		h.logger.Warn("topup order with zero credits",
			zap.String("order_id", event.OrderID.String()),
		)
		return nil
	}

	ctx := context.Background()

	// Add credits (service should handle idempotency)
	if err := h.service.AddCredits(ctx, event.UserID, event.CreditsAmount, "topup"); err != nil {
		h.logger.Error("failed to add credits",
			zap.String("user_id", event.UserID.String()),
			zap.Int64("amount", event.CreditsAmount),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("credits added for topup",
		zap.String("user_id", event.UserID.String()),
		zap.Int64("amount", event.CreditsAmount),
		zap.String("order_id", event.OrderID.String()),
	)

	return nil
}

// Compile-time check that EventHandler implements events.Handler.
var _ events.Handler = (*EventHandler)(nil)
