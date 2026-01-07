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

// RegisterRoutes registers public payment routes (currently none).
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// No public routes for payments
}

// RegisterProtectedRoutes registers payment routes that require authentication.
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	payments := r.Group("/payments")
	{
		payments.POST("/intent", h.CreatePaymentIntent)
		payments.POST("/native", h.CreateNativePayment)
		payments.GET("/methods", h.ListPaymentMethods)
		payments.GET("/:id", h.GetPayment)
	}
}

// CreatePaymentIntent creates a payment intent for an order.
//
//	@Summary		Create payment intent
//	@Description	Create a Stripe payment intent for an order
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreatePaymentIntentRequest	true	"Payment intent request"
//	@Success		200		{object}	PaymentIntentResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/payments/intent [post]
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
//
//	@Summary		Create native payment
//	@Description	Create an Alipay or WeChat Pay payment for an order
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateNativePaymentRequest	true	"Native payment request"
//	@Success		200		{object}	NativePaymentResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/payments/native [post]
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
//
//	@Summary		List payment methods
//	@Description	Get all saved payment methods for the current user
//	@Tags			Payment
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/payments/methods [get]
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
//
//	@Summary		Get payment
//	@Description	Get details of a specific payment
//	@Tags			Payment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Payment ID"
//	@Success		200	{object}	PaymentResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/payments/{id} [get]
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
	if payment.UserID() != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, PaymentToResponse(payment))
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
