package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	orderCmd "github.com/uniedit/server/internal/app/command/order"
	orderQuery "github.com/uniedit/server/internal/app/query/order"
	"github.com/uniedit/server/internal/domain/order"
)

// OrderHandler handles HTTP requests for orders using CQRS pattern.
type OrderHandler struct {
	createOrder  *orderCmd.CreateOrderHandler
	cancelOrder  *orderCmd.CancelOrderHandler
	getOrder     *orderQuery.GetOrderHandler
	listOrders   *orderQuery.ListOrdersHandler
	errorHandler *ErrorHandler
}

// NewOrderHandler creates a new order handler.
func NewOrderHandler(
	createOrder *orderCmd.CreateOrderHandler,
	cancelOrder *orderCmd.CancelOrderHandler,
	getOrder *orderQuery.GetOrderHandler,
	listOrders *orderQuery.ListOrdersHandler,
) *OrderHandler {
	return &OrderHandler{
		createOrder:  createOrder,
		cancelOrder:  cancelOrder,
		getOrder:     getOrder,
		listOrders:   listOrders,
		errorHandler: NewErrorHandler(),
	}
}

// RegisterProtectedRoutes registers protected order routes.
func (h *OrderHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	orders := r.Group("/orders")
	{
		orders.POST("", h.CreateOrder)
		orders.GET("", h.ListOrders)
		orders.GET("/:id", h.GetOrder)
		orders.POST("/:id/cancel", h.CancelOrder)
	}
}

// CreateOrderRequest represents a request to create an order.
type CreateOrderRequest struct {
	Type          string           `json:"type" binding:"required,oneof=subscription topup credits"`
	Currency      string           `json:"currency"`
	PlanID        string           `json:"plan_id"`
	CreditsAmount int64            `json:"credits_amount"`
	Items         []OrderItemInput `json:"items"`
}

// OrderItemInput represents an order item in a request.
type OrderItemInput struct {
	Description string `json:"description" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	UnitPrice   int64  `json:"unit_price" binding:"required,min=0"`
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Convert items
	items := make([]orderCmd.OrderItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = orderCmd.OrderItemInput{
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		}
	}

	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	result, err := h.createOrder.Handle(c.Request.Context(), orderCmd.CreateOrderCommand{
		UserID:        userID,
		Type:          order.OrderType(req.Type),
		Currency:      currency,
		PlanID:        req.PlanID,
		CreditsAmount: req.CreditsAmount,
		Items:         items,
	})
	if err != nil {
		h.errorHandler.HandleOrderError(c, err)
		return
	}

	respondCreated(c, gin.H{
		"order": orderToResponse(result.Order),
	})
}

// ListOrdersRequest represents query parameters for listing orders.
type ListOrdersRequest struct {
	Status   string `form:"status"`
	Type     string `form:"type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Set defaults
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	result, err := h.listOrders.Handle(c.Request.Context(), orderQuery.ListOrdersQuery{
		UserID:   userID,
		Status:   req.Status,
		Type:     req.Type,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.errorHandler.HandleOrderError(c, err)
		return
	}

	orders := make([]gin.H, len(result.Orders))
	for i, ord := range result.Orders {
		orders[i] = orderToResponse(ord)
	}

	respondPaginated(c, orders, result.Total, page, pageSize)
}

// GetOrder handles GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid order ID format")
		return
	}

	result, err := h.getOrder.Handle(c.Request.Context(), orderQuery.GetOrderQuery{
		OrderID: orderID,
		UserID:  userID,
	})
	if err != nil {
		h.errorHandler.HandleOrderError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"order": orderToResponse(result.Order),
	})
}

// CancelOrder handles POST /orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid order ID format")
		return
	}

	result, err := h.cancelOrder.Handle(c.Request.Context(), orderCmd.CancelOrderCommand{
		OrderID: orderID,
		UserID:  userID,
	})
	if err != nil {
		h.errorHandler.HandleOrderError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"order": orderToResponse(result.Order),
	})
}

// Response helpers

func orderToResponse(ord *order.Order) gin.H {
	items := make([]gin.H, len(ord.Items()))
	for i, item := range ord.Items() {
		items[i] = gin.H{
			"id":          item.ID(),
			"description": item.Description(),
			"quantity":    item.Quantity(),
			"unit_price":  item.UnitPrice().Amount(),
			"amount":      item.Amount().Amount(),
		}
	}

	resp := gin.H{
		"id":         ord.ID(),
		"order_no":   ord.OrderNo(),
		"type":       ord.Type(),
		"status":     ord.Status(),
		"subtotal":   ord.Subtotal().Amount(),
		"discount":   ord.Discount().Amount(),
		"tax":        ord.Tax().Amount(),
		"total":      ord.Total().Amount(),
		"currency":   ord.Currency(),
		"items":      items,
		"created_at": ord.CreatedAt(),
		"updated_at": ord.UpdatedAt(),
	}

	if ord.PlanID() != nil {
		resp["plan_id"] = *ord.PlanID()
	}
	if ord.CreditsAmount() > 0 {
		resp["credits_amount"] = ord.CreditsAmount()
	}
	if ord.PaidAt() != nil {
		resp["paid_at"] = ord.PaidAt()
	}
	if ord.CanceledAt() != nil {
		resp["canceled_at"] = ord.CanceledAt()
	}
	if ord.ExpiresAt() != nil {
		resp["expires_at"] = ord.ExpiresAt()
	}

	return resp
}
