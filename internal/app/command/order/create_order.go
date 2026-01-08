package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/shared/random"
)

// CreateOrderCommand represents a command to create an order.
type CreateOrderCommand struct {
	UserID        uuid.UUID
	Type          order.OrderType
	Currency      string
	PlanID        string // For subscription orders
	CreditsAmount int64  // For topup orders
	Items         []OrderItemInput
}

// OrderItemInput represents an order item input.
type OrderItemInput struct {
	Description string
	Quantity    int
	UnitPrice   int64
}

// CreateOrderResult is the result of creating an order.
type CreateOrderResult struct {
	Order *order.Order
}

// CreateOrderHandler handles CreateOrderCommand.
type CreateOrderHandler struct {
	repo order.Repository
}

// NewCreateOrderHandler creates a new handler.
func NewCreateOrderHandler(repo order.Repository) *CreateOrderHandler {
	return &CreateOrderHandler{repo: repo}
}

// Handle executes the command.
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (*CreateOrderResult, error) {
	// Generate order number
	orderNo := fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), random.UpperAlphaNum(8))

	// Create order
	ord, err := order.NewOrder(cmd.UserID, orderNo, cmd.Type, cmd.Currency)
	if err != nil {
		return nil, err
	}

	// Set type-specific fields
	if cmd.PlanID != "" {
		ord.SetPlanID(cmd.PlanID)
	}
	if cmd.CreditsAmount > 0 {
		ord.SetCreditsAmount(cmd.CreditsAmount)
	}

	// Add items
	for _, item := range cmd.Items {
		orderItem, err := order.NewOrderItem(
			item.Description,
			item.Quantity,
			order.NewMoney(item.UnitPrice, cmd.Currency),
		)
		if err != nil {
			return nil, err
		}
		if err := ord.AddItem(orderItem); err != nil {
			return nil, err
		}
	}

	// Set expiration (24 hours)
	ord.SetExpiration(time.Now().Add(24 * time.Hour))

	// Persist
	if err := h.repo.CreateOrder(ctx, ord); err != nil {
		return nil, err
	}

	return &CreateOrderResult{Order: ord}, nil
}
