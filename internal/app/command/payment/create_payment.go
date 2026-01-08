package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
)

// CreatePaymentCommand represents a command to create a payment record.
type CreatePaymentCommand struct {
	OrderID  uuid.UUID
	UserID   uuid.UUID
	Amount   int64
	Currency string
	Method   payment.PaymentMethod
	Provider string
}

// CreatePaymentResult is the result of creating a payment.
type CreatePaymentResult struct {
	Payment *payment.Payment
}

// CreatePaymentHandler handles CreatePaymentCommand.
type CreatePaymentHandler struct {
	repo payment.Repository
}

// NewCreatePaymentHandler creates a new handler.
func NewCreatePaymentHandler(repo payment.Repository) *CreatePaymentHandler {
	return &CreatePaymentHandler{repo: repo}
}

// Handle executes the command.
func (h *CreatePaymentHandler) Handle(ctx context.Context, cmd CreatePaymentCommand) (*CreatePaymentResult, error) {
	p := payment.NewPayment(cmd.OrderID, cmd.UserID, cmd.Amount, cmd.Currency, cmd.Method, cmd.Provider)

	if err := h.repo.CreatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	return &CreatePaymentResult{Payment: p}, nil
}
