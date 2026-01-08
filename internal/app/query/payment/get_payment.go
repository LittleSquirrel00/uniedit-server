package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
)

// GetPaymentQuery represents a query to get a payment.
type GetPaymentQuery struct {
	PaymentID uuid.UUID
	UserID    uuid.UUID // For ownership verification
}

// GetPaymentResult is the result of getting a payment.
type GetPaymentResult struct {
	Payment *payment.Payment
}

// GetPaymentHandler handles GetPaymentQuery.
type GetPaymentHandler struct {
	repo payment.Repository
}

// NewGetPaymentHandler creates a new handler.
func NewGetPaymentHandler(repo payment.Repository) *GetPaymentHandler {
	return &GetPaymentHandler{repo: repo}
}

// Handle executes the query.
func (h *GetPaymentHandler) Handle(ctx context.Context, query GetPaymentQuery) (*GetPaymentResult, error) {
	p, err := h.repo.GetPayment(ctx, query.PaymentID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if p.UserID() != query.UserID {
		return nil, payment.ErrPaymentNotFound // Hide existence from unauthorized users
	}

	return &GetPaymentResult{Payment: p}, nil
}

// ListPaymentsByOrderQuery represents a query to list payments for an order.
type ListPaymentsByOrderQuery struct {
	OrderID uuid.UUID
	UserID  uuid.UUID // For ownership verification
}

// ListPaymentsByOrderResult is the result of listing payments.
type ListPaymentsByOrderResult struct {
	Payments []*payment.Payment
}

// ListPaymentsByOrderHandler handles ListPaymentsByOrderQuery.
type ListPaymentsByOrderHandler struct {
	repo payment.Repository
}

// NewListPaymentsByOrderHandler creates a new handler.
func NewListPaymentsByOrderHandler(repo payment.Repository) *ListPaymentsByOrderHandler {
	return &ListPaymentsByOrderHandler{repo: repo}
}

// Handle executes the query.
func (h *ListPaymentsByOrderHandler) Handle(ctx context.Context, query ListPaymentsByOrderQuery) (*ListPaymentsByOrderResult, error) {
	payments, err := h.repo.ListPaymentsByOrder(ctx, query.OrderID)
	if err != nil {
		return nil, err
	}

	// Filter by ownership
	var result []*payment.Payment
	for _, p := range payments {
		if p.UserID() == query.UserID {
			result = append(result, p)
		}
	}

	return &ListPaymentsByOrderResult{Payments: result}, nil
}
