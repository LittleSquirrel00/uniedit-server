package orderhttp

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// OrderHandler handles order HTTP requests.
type OrderHandler struct {
	orderDomain order.OrderDomain
}

// NewOrderHandler creates a new order handler.
func NewOrderHandler(orderDomain order.OrderDomain) *OrderHandler {
	return &OrderHandler{orderDomain: orderDomain}
}

// RegisterRoutes registers order routes.
func (h *OrderHandler) RegisterRoutes(r *gin.RouterGroup) {
	orders := r.Group("/orders")
	{
		orders.POST("/subscription", h.CreateSubscriptionOrder)
		orders.POST("/topup", h.CreateTopupOrder)
		orders.GET("", h.ListOrders)
		orders.GET("/:id", h.GetOrder)
		orders.POST("/:id/cancel", h.CancelOrder)
	}
}

// CreateSubscriptionOrder handles POST /orders/subscription.
func (h *OrderHandler) CreateSubscriptionOrder(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ord, err := h.orderDomain.CreateSubscriptionOrder(c.Request.Context(), userID, req.PlanID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ord.ToResponse())
}

// CreateTopupOrder handles POST /orders/topup.
func (h *OrderHandler) CreateTopupOrder(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		Amount int64 `json:"amount" binding:"required,min=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ord, err := h.orderDomain.CreateTopupOrder(c.Request.Context(), userID, req.Amount)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ord.ToResponse())
}

// GetOrder handles GET /orders/:id.
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	ord, err := h.orderDomain.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check ownership
	if ord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, ord.ToResponse())
}

// ListOrders handles GET /orders.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Parse filters
	var filter *model.OrderFilter
	if statusStr := c.Query("status"); statusStr != "" {
		status := model.OrderStatus(statusStr)
		if status.IsValid() {
			filter = &model.OrderFilter{Status: &status}
		}
	}
	if typeStr := c.Query("type"); typeStr != "" {
		orderType := model.OrderType(typeStr)
		if orderType.IsValid() {
			if filter == nil {
				filter = &model.OrderFilter{}
			}
			filter.Type = &orderType
		}
	}

	orders, total, err := h.orderDomain.ListOrders(c.Request.Context(), userID, filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	responses := make([]*model.OrderResponse, len(orders))
	for i, ord := range orders {
		responses[i] = ord.ToResponse()
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, model.OrderListResponse{
		Orders:     responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// CancelOrder handles POST /orders/:id/cancel.
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	// Verify ownership first
	ord, err := h.orderDomain.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}
	if ord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.orderDomain.CancelOrder(c.Request.Context(), orderID, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order canceled"})
}

// Compile-time check
var _ inbound.OrderHttpPort = (*OrderHandler)(nil)
