package billinghttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// CreditsHandler handles credits HTTP requests.
type CreditsHandler struct {
	billingDomain billing.BillingDomain
}

// NewCreditsHandler creates a new credits handler.
func NewCreditsHandler(billingDomain billing.BillingDomain) *CreditsHandler {
	return &CreditsHandler{billingDomain: billingDomain}
}

// RegisterRoutes registers credits routes.
func (h *CreditsHandler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/credits", h.GetBalance)
	}
}

// RegisterAdminRoutes registers admin credits routes.
func (h *CreditsHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	r.POST("/billing/credits", h.AddCredits)
}

// GetBalance handles GET /billing/credits.
func (h *CreditsHandler) GetBalance(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	balance, err := h.billingDomain.GetBalance(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

// AddCredits handles POST /admin/billing/credits.
func (h *CreditsHandler) AddCredits(c *gin.Context) {
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
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "credits added"})
}

// Compile-time check
var _ inbound.CreditsHttpPort = (*CreditsHandler)(nil)
