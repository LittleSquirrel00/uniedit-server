package paymenthttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// RefundHandler handles refund HTTP requests.
type RefundHandler struct {
	domain payment.PaymentDomain
}

// NewRefundHandler creates a new refund handler.
func NewRefundHandler(domain payment.PaymentDomain) *RefundHandler {
	return &RefundHandler{domain: domain}
}

// RegisterRoutes registers refund routes (admin only).
func (h *RefundHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/payments/:id/refund", h.CreateRefund)
}

// CreateRefund handles POST /payments/:id/refund.
func (h *RefundHandler) CreateRefund(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid payment ID",
		})
		return
	}

	var req model.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := h.domain.CreateRefund(c.Request.Context(), id, req.Amount, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "refund created"})
}

// Compile-time check
var _ inbound.RefundHttpPort = (*RefundHandler)(nil)
