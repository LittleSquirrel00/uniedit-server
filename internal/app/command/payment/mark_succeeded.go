package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
)

// MarkPaymentSucceededCommand represents a command to mark a payment as succeeded.
type MarkPaymentSucceededCommand struct {
	PaymentID uuid.UUID
	ChargeID  string // For Stripe payments
	TradeNo   string // For native payments (Alipay/WeChat)
	PayerID   string // For native payments
}

// MarkPaymentSucceededResult is the result of marking a payment as succeeded.
type MarkPaymentSucceededResult struct {
	Payment *payment.Payment
}

// MarkPaymentSucceededHandler handles MarkPaymentSucceededCommand.
type MarkPaymentSucceededHandler struct {
	repo payment.Repository
}

// NewMarkPaymentSucceededHandler creates a new handler.
func NewMarkPaymentSucceededHandler(repo payment.Repository) *MarkPaymentSucceededHandler {
	return &MarkPaymentSucceededHandler{repo: repo}
}

// Handle executes the command.
func (h *MarkPaymentSucceededHandler) Handle(ctx context.Context, cmd MarkPaymentSucceededCommand) (*MarkPaymentSucceededResult, error) {
	p, err := h.repo.GetPayment(ctx, cmd.PaymentID)
	if err != nil {
		return nil, err
	}

	// Use appropriate domain method based on payment type
	if cmd.ChargeID != "" {
		if err := p.MarkAsSucceeded(cmd.ChargeID); err != nil {
			return nil, fmt.Errorf("mark succeeded: %w", err)
		}
	} else {
		if err := p.MarkAsSucceededNative(cmd.TradeNo, cmd.PayerID); err != nil {
			return nil, fmt.Errorf("mark succeeded native: %w", err)
		}
	}

	if err := h.repo.UpdatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	return &MarkPaymentSucceededResult{Payment: p}, nil
}
