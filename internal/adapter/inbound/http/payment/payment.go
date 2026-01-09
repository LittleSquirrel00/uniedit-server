package paymenthttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// PaymentHandler handles payment HTTP requests.
type PaymentHandler struct {
	domain payment.PaymentDomain
}

// NewPaymentHandler creates a new payment handler.
func NewPaymentHandler(domain payment.PaymentDomain) *PaymentHandler {
	return &PaymentHandler{domain: domain}
}

// RegisterRoutes registers payment routes.
func (h *PaymentHandler) RegisterRoutes(r *gin.RouterGroup) {
	payments := r.Group("/payments")
	{
		payments.POST("/intent", h.CreatePaymentIntent)
		payments.POST("/native", h.CreateNativePayment)
		payments.GET("/:id", h.GetPayment)
		payments.GET("", h.ListPayments)
		payments.GET("/methods", h.ListPaymentMethods)
	}
}

// CreatePaymentIntent handles POST /payments/intent.
func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
	var req model.CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userID := mustGetUserID(c)

	resp, err := h.domain.CreatePaymentIntent(c.Request.Context(), req.OrderID, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateNativePayment handles POST /payments/native.
func (h *PaymentHandler) CreateNativePayment(c *gin.Context) {
	var req model.CreateNativePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userID := mustGetUserID(c)

	resp, err := h.domain.CreateNativePayment(c.Request.Context(), &req, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPayment handles GET /payments/:id.
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid payment ID",
		})
		return
	}

	userID := mustGetUserID(c)

	p, err := h.domain.GetPayment(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	if p == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    "not_found",
			Message: "Payment not found",
		})
		return
	}

	// Verify ownership
	if p.UserID != userID {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    "forbidden",
			Message: "Forbidden",
		})
		return
	}

	c.JSON(http.StatusOK, p)
}

// ListPayments handles GET /payments.
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	userID := mustGetUserID(c)

	var filter model.PaymentFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	// Force user filter for non-admin
	filter.UserID = &userID

	payments, total, err := h.domain.ListPayments(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}

	filter.DefaultPagination()
	c.JSON(http.StatusOK, model.PaginatedResponse[*model.Payment]{
		Data:       payments,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
	})
}

// ListPaymentMethods handles GET /payments/methods.
func (h *PaymentHandler) ListPaymentMethods(c *gin.Context) {
	userID := mustGetUserID(c)

	methods, err := h.domain.ListPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, methods)
}

// Compile-time check
var _ inbound.PaymentHttpPort = (*PaymentHandler)(nil)
