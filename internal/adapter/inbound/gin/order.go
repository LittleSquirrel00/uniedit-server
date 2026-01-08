package gin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// orderHandler implements inbound.OrderHttpPort.
type orderHandler struct {
	orderDomain order.OrderDomain
}

// NewOrderHandler creates a new order HTTP handler.
func NewOrderHandler(orderDomain order.OrderDomain) inbound.OrderHttpPort {
	return &orderHandler{orderDomain: orderDomain}
}

func (h *orderHandler) CreateSubscriptionOrder(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ord.ToResponse())
}

func (h *orderHandler) CreateTopupOrder(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ord.ToResponse())
}

func (h *orderHandler) GetOrder(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		handleOrderError(c, err)
		return
	}

	// Check ownership
	if ord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, ord.ToResponse())
}

func (h *orderHandler) ListOrders(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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

	// Convert to response
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

func (h *orderHandler) CancelOrder(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		handleOrderError(c, err)
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
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order canceled"})
}

func handleOrderError(c *gin.Context, err error) {
	switch err {
	case order.ErrOrderNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case order.ErrOrderNotCancelable:
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be canceled"})
	case order.ErrOrderNotRefundable:
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be refunded"})
	case order.ErrMinimumTopupAmount:
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum top-up is $1.00"})
	case order.ErrFreePlanNotOrderable:
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot create order for free plan"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// Compile-time check
var _ inbound.OrderHttpPort = (*orderHandler)(nil)
