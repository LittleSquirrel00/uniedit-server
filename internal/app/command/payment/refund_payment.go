package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
)

// RefundPaymentCommand represents a command to refund a payment.
type RefundPaymentCommand struct {
	PaymentID uuid.UUID
	Amount    int64  // 0 for full refund
	Reason    string
}

// RefundPaymentResult is the result of refunding a payment.
type RefundPaymentResult struct {
	Payment       *payment.Payment
	RefundedAmount int64
}

// RefundPaymentHandler handles RefundPaymentCommand.
type RefundPaymentHandler struct {
	repo payment.Repository
}

// NewRefundPaymentHandler creates a new handler.
func NewRefundPaymentHandler(repo payment.Repository) *RefundPaymentHandler {
	return &RefundPaymentHandler{repo: repo}
}

// Handle executes the command.
func (h *RefundPaymentHandler) Handle(ctx context.Context, cmd RefundPaymentCommand) (*RefundPaymentResult, error) {
	p, err := h.repo.GetPayment(ctx, cmd.PaymentID)
	if err != nil {
		return nil, err
	}

	// Use domain method to calculate and execute refund
	refundedAmount, err := p.Refund(cmd.Amount)
	if err != nil {
		return nil, fmt.Errorf("refund: %w", err)
	}

	if err := h.repo.UpdatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	return &RefundPaymentResult{
		Payment:       p,
		RefundedAmount: refundedAmount,
	}, nil
}
