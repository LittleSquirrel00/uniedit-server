package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
)

// GetOrderQuery represents a query to get an order.
type GetOrderQuery struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
}

// GetOrderResult is the result of getting an order.
type GetOrderResult struct {
	Order *order.Order
}

// GetOrderHandler handles GetOrderQuery.
type GetOrderHandler struct {
	repo order.Repository
}

// NewGetOrderHandler creates a new handler.
func NewGetOrderHandler(repo order.Repository) *GetOrderHandler {
	return &GetOrderHandler{repo: repo}
}

// Handle executes the query.
func (h *GetOrderHandler) Handle(ctx context.Context, query GetOrderQuery) (*GetOrderResult, error) {
	ord, err := h.repo.GetOrderWithItems(ctx, query.OrderID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if ord.UserID() != query.UserID {
		return nil, order.ErrOrderNotFound
	}

	return &GetOrderResult{Order: ord}, nil
}

// ListOrdersQuery represents a query to list orders.
type ListOrdersQuery struct {
	UserID     uuid.UUID
	Status     string
	Type       string
	Page       int
	PageSize   int
}

// ListOrdersResult is the result of listing orders.
type ListOrdersResult struct {
	Orders []*order.Order
	Total  int64
}

// ListOrdersHandler handles ListOrdersQuery.
type ListOrdersHandler struct {
	repo order.Repository
}

// NewListOrdersHandler creates a new handler.
func NewListOrdersHandler(repo order.Repository) *ListOrdersHandler {
	return &ListOrdersHandler{repo: repo}
}

// Handle executes the query.
func (h *ListOrdersHandler) Handle(ctx context.Context, query ListOrdersQuery) (*ListOrdersResult, error) {
	filter := &order.OrderFilter{
		Status: query.Status,
		Type:   query.Type,
	}

	pagination := &order.Pagination{
		Page:     query.Page,
		PageSize: query.PageSize,
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 20
	}

	orders, total, err := h.repo.ListOrders(ctx, query.UserID, filter, pagination)
	if err != nil {
		return nil, err
	}

	return &ListOrdersResult{Orders: orders, Total: total}, nil
}
