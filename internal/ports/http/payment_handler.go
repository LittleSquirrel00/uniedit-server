package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	paymentCmd "github.com/uniedit/server/internal/app/command/payment"
	paymentQuery "github.com/uniedit/server/internal/app/query/payment"
	"github.com/uniedit/server/internal/domain/payment"
)

// PaymentHandler handles HTTP requests for payments using CQRS pattern.
type PaymentHandler struct {
	createPayment       *paymentCmd.CreatePaymentHandler
	markSucceeded       *paymentCmd.MarkPaymentSucceededHandler
	refundPayment       *paymentCmd.RefundPaymentHandler
	getPayment          *paymentQuery.GetPaymentHandler
	listPaymentsByOrder *paymentQuery.ListPaymentsByOrderHandler
	errorHandler        *ErrorHandler
}

// NewPaymentHandler creates a new payment handler.
func NewPaymentHandler(
	createPayment *paymentCmd.CreatePaymentHandler,
	markSucceeded *paymentCmd.MarkPaymentSucceededHandler,
	refundPayment *paymentCmd.RefundPaymentHandler,
	getPayment *paymentQuery.GetPaymentHandler,
	listPaymentsByOrder *paymentQuery.ListPaymentsByOrderHandler,
) *PaymentHandler {
	return &PaymentHandler{
		createPayment:       createPayment,
		markSucceeded:       markSucceeded,
		refundPayment:       refundPayment,
		getPayment:          getPayment,
		listPaymentsByOrder: listPaymentsByOrder,
		errorHandler:        NewErrorHandler(),
	}
}

// RegisterProtectedRoutes registers protected payment routes.
func (h *PaymentHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	payments := r.Group("/payments")
	{
		payments.POST("", h.CreatePayment)
		payments.GET("/:id", h.GetPayment)
		payments.GET("/order/:order_id", h.ListPaymentsByOrder)
	}
}

// RegisterAdminRoutes registers admin payment routes.
func (h *PaymentHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	payments := r.Group("/payments")
	{
		payments.POST("/:id/refund", h.RefundPayment)
	}
}

// CreatePaymentRequest represents a request to create a payment.
type CreatePaymentRequest struct {
	OrderID  string `json:"order_id" binding:"required,uuid"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
	Currency string `json:"currency" binding:"required"`
	Method   string `json:"method" binding:"required,oneof=card alipay_web alipay_h5 alipay_app alipay_native wechat_native wechat_h5 wechat_app wechat_mini wechat_jsapi"`
	Provider string `json:"provider" binding:"required,oneof=stripe alipay wechat"`
}

// CreatePayment handles POST /payments
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	orderID, _ := uuid.Parse(req.OrderID)

	result, err := h.createPayment.Handle(c.Request.Context(), paymentCmd.CreatePaymentCommand{
		OrderID:  orderID,
		UserID:   userID,
		Amount:   req.Amount,
		Currency: req.Currency,
		Method:   payment.PaymentMethod(req.Method),
		Provider: req.Provider,
	})
	if err != nil {
		h.errorHandler.HandlePaymentError(c, err)
		return
	}

	respondCreated(c, gin.H{
		"payment": paymentToResponse(result.Payment),
	})
}

// GetPayment handles GET /payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid payment ID format")
		return
	}

	result, err := h.getPayment.Handle(c.Request.Context(), paymentQuery.GetPaymentQuery{
		PaymentID: paymentID,
		UserID:    userID,
	})
	if err != nil {
		h.errorHandler.HandlePaymentError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"payment": paymentToResponse(result.Payment),
	})
}

// ListPaymentsByOrder handles GET /payments/order/:order_id
func (h *PaymentHandler) ListPaymentsByOrder(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid order ID format")
		return
	}

	result, err := h.listPaymentsByOrder.Handle(c.Request.Context(), paymentQuery.ListPaymentsByOrderQuery{
		OrderID: orderID,
		UserID:  userID,
	})
	if err != nil {
		h.errorHandler.HandlePaymentError(c, err)
		return
	}

	payments := make([]gin.H, len(result.Payments))
	for i, p := range result.Payments {
		payments[i] = paymentToResponse(p)
	}

	respondSuccess(c, gin.H{
		"payments": payments,
	})
}

// RefundPaymentRequest represents a request to refund a payment.
type RefundPaymentRequest struct {
	Amount int64 `json:"amount"` // 0 means full refund
}

// RefundPayment handles POST /payments/:id/refund (admin only)
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid payment ID format")
		return
	}

	var req RefundPaymentRequest
	_ = c.ShouldBindJSON(&req) // Optional, defaults to 0 (full refund)

	result, err := h.refundPayment.Handle(c.Request.Context(), paymentCmd.RefundPaymentCommand{
		PaymentID: paymentID,
		Amount:    req.Amount,
	})
	if err != nil {
		h.errorHandler.HandlePaymentError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"payment":         paymentToResponse(result.Payment),
		"refunded_amount": result.RefundedAmount,
	})
}

// Response helpers

func paymentToResponse(p *payment.Payment) gin.H {
	resp := gin.H{
		"id":         p.ID(),
		"order_id":   p.OrderID(),
		"amount":     p.Amount(),
		"currency":   p.Currency(),
		"method":     p.Method(),
		"status":     p.Status(),
		"provider":   p.Provider(),
		"created_at": p.CreatedAt(),
		"updated_at": p.UpdatedAt(),
	}

	if p.StripePaymentIntentID() != "" {
		resp["stripe_payment_intent_id"] = p.StripePaymentIntentID()
	}
	if p.StripeChargeID() != "" {
		resp["stripe_charge_id"] = p.StripeChargeID()
	}
	if p.TradeNo() != "" {
		resp["trade_no"] = p.TradeNo()
	}
	if p.RefundedAmount() > 0 {
		resp["refunded_amount"] = p.RefundedAmount()
	}
	if p.SucceededAt() != nil {
		resp["succeeded_at"] = p.SucceededAt()
	}
	if p.FailedAt() != nil {
		resp["failed_at"] = p.FailedAt()
	}
	if p.FailureCode() != nil {
		resp["failure_code"] = *p.FailureCode()
	}
	if p.FailureMessage() != nil {
		resp["failure_message"] = *p.FailureMessage()
	}

	return resp
}
