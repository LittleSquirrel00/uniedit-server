package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// creditsHandler implements inbound.CreditsHttpPort.
type creditsHandler struct {
	billingDomain billing.BillingDomain
}

// NewCreditsHandler creates a new credits HTTP handler.
func NewCreditsHandler(billingDomain billing.BillingDomain) inbound.CreditsHttpPort {
	return &creditsHandler{billingDomain: billingDomain}
}

func (h *creditsHandler) GetBalance(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		return
	}

	balance, err := h.billingDomain.GetBalance(c.Request.Context(), userID)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *creditsHandler) AddCredits(c *gin.Context) {
	// Admin-only endpoint
	var req struct {
		UserID uuid.UUID `json:"user_id" binding:"required"`
		Amount int64     `json:"amount" binding:"required,gt=0"`
		Source string    `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.billingDomain.AddCredits(c.Request.Context(), req.UserID, req.Amount, req.Source); err != nil {
		switch err {
		case billing.ErrSubscriptionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		case billing.ErrInvalidCreditsAmount:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credits amount"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add credits"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "credits added"})
}

// Compile-time check
var _ inbound.CreditsHttpPort = (*creditsHandler)(nil)
