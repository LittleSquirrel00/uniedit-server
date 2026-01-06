package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for orders.
type Handler struct {
	service *Service
}

// NewHandler creates a new order handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the order routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	orders := r.Group("/orders")
	{
		orders.POST("/subscription", h.CreateSubscriptionOrder)
		orders.POST("/topup", h.CreateTopupOrder)
		orders.GET("", h.ListOrders)
		orders.GET("/:id", h.GetOrder)
		orders.POST("/:id/cancel", h.CancelOrder)
	}

	invoices := r.Group("/invoices")
	{
		invoices.GET("", h.ListInvoices)
		invoices.GET("/:id", h.GetInvoice)
	}
}

// CreateSubscriptionOrder creates a subscription order.
func (h *Handler) CreateSubscriptionOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSubscriptionOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.CreateSubscriptionOrder(c.Request.Context(), userID, req.PlanID)
	if err != nil {
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order": order.ToResponse(),
	})
}

// CreateTopupOrder creates a top-up order.
func (h *Handler) CreateTopupOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateTopupOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.CreateTopupOrder(c.Request.Context(), userID, req.Amount)
	if err != nil {
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order": order.ToResponse(),
	})
}

// ListOrders returns orders for the current user.
func (h *Handler) ListOrders(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var filter OrderFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pagination := NewPagination()
	if err := c.ShouldBindQuery(pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orders, total, err := h.service.ListOrders(c.Request.Context(), userID, &filter, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	responses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = order.ToResponse()
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, OrderListResponse{
		Orders:     responses,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	})
}

// GetOrder returns a single order.
func (h *Handler) GetOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	order, err := h.service.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		handleOrderError(c, err)
		return
	}

	// Check ownership
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, order.ToResponse())
}

// CancelOrder cancels a pending order.
func (h *Handler) CancelOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	// Check ownership first
	order, err := h.service.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		handleOrderError(c, err)
		return
	}
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.service.CancelOrder(c.Request.Context(), orderID, "user requested"); err != nil {
		handleOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order canceled"})
}

// ListInvoices returns invoices for the current user.
func (h *Handler) ListInvoices(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	invoices, err := h.service.ListInvoices(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}

	responses := make([]*InvoiceResponse, len(invoices))
	for i, invoice := range invoices {
		responses[i] = invoice.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"invoices": responses})
}

// GetInvoice returns a single invoice.
func (h *Handler) GetInvoice(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	invoice, err := h.service.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		handleOrderError(c, err)
		return
	}

	// Check ownership
	if invoice.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, invoice.ToResponse())
}

// --- Helpers ---

func getUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

func handleOrderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order_not_found"})
	case errors.Is(err, ErrOrderNotPending):
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_not_pending"})
	case errors.Is(err, ErrOrderNotCancelable):
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_not_cancelable"})
	case errors.Is(err, ErrOrderNotRefundable):
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_not_refundable"})
	case errors.Is(err, ErrInvalidTransition):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state_transition"})
	case errors.Is(err, ErrInvoiceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice_not_found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
