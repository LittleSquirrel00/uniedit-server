package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for payments.
type Handler struct {
	service *Service
}

// NewHandler creates a new payment handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the payment routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	payments := r.Group("/payments")
	{
		payments.POST("/intent", h.CreatePaymentIntent)
		payments.POST("/native", h.CreateNativePayment)
		payments.GET("/methods", h.ListPaymentMethods)
		payments.GET("/:id", h.GetPayment)
	}
}

// CreatePaymentIntent creates a payment intent for an order.
func (h *Handler) CreatePaymentIntent(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreatePaymentIntent(c.Request.Context(), req.OrderID, userID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateNativePayment creates a native payment (Alipay/WeChat) for an order.
func (h *Handler) CreateNativePayment(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateNativePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateNativePayment(c.Request.Context(), &req, userID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListPaymentMethods returns the user's saved payment methods.
func (h *Handler) ListPaymentMethods(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	methods, err := h.service.ListPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list payment methods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": methods})
}

// GetPayment returns a payment by ID.
func (h *Handler) GetPayment(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	payment, err := h.service.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}

	// Check ownership
	if payment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, payment.ToResponse())
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

func handlePaymentError(c *gin.Context, err error) {
	switch err.Error() {
	case "forbidden":
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case "order is not pending":
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_not_pending"})
	case "payment is not succeeded":
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_not_succeeded"})
	case "no charge ID for refund":
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_charge_for_refund"})
	default:
		if err.Error() == "not found" || err.Error() == "payment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
