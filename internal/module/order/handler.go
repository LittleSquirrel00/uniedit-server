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

// RegisterRoutes registers public order routes (currently none, placeholder for future).
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// No public routes for orders
}

// RegisterProtectedRoutes registers order routes that require authentication.
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
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
//
//	@Summary		Create subscription order
//	@Description	Create a new order for a subscription plan
//	@Tags			Order
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateSubscriptionOrderRequest	true	"Create order request"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/orders/subscription [post]
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
		"order": OrderToResponse(order),
	})
}

// CreateTopupOrder creates a top-up order.
//
//	@Summary		Create top-up order
//	@Description	Create a new order to top up credits
//	@Tags			Order
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateTopupOrderRequest	true	"Create order request"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/orders/topup [post]
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
		"order": OrderToResponse(order),
	})
}

// ListOrders returns orders for the current user.
//
//	@Summary		List orders
//	@Description	Get all orders for the current user
//	@Tags			Order
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"Filter by status"
//	@Param			type		query		string	false	"Filter by order type"
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size"		default(20)
//	@Success		200			{object}	OrderListResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Router			/orders [get]
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
		responses[i] = OrderToResponse(order)
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
//
//	@Summary		Get order
//	@Description	Get details of a specific order
//	@Tags			Order
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	OrderResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/orders/{id} [get]
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
	if order.UserID() != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, OrderToResponse(order))
}

// CancelOrder cancels a pending order.
//
//	@Summary		Cancel order
//	@Description	Cancel a pending order
//	@Tags			Order
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/orders/{id}/cancel [post]
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
	if order.UserID() != userID {
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
//
//	@Summary		List invoices
//	@Description	Get all invoices for the current user
//	@Tags			Order
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/invoices [get]
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
		responses[i] = InvoiceToResponse(invoice)
	}

	c.JSON(http.StatusOK, gin.H{"invoices": responses})
}

// GetInvoice returns a single invoice.
//
//	@Summary		Get invoice
//	@Description	Get details of a specific invoice
//	@Tags			Order
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Invoice ID"
//	@Success		200	{object}	InvoiceResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/invoices/{id} [get]
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

	c.JSON(http.StatusOK, InvoiceToResponse(invoice))
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
