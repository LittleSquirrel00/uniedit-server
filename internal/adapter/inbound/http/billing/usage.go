package billinghttp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// UsageHandler handles usage HTTP requests.
type UsageHandler struct {
	billingDomain billing.BillingDomain
}

// NewUsageHandler creates a new usage handler.
func NewUsageHandler(billingDomain billing.BillingDomain) *UsageHandler {
	return &UsageHandler{billingDomain: billingDomain}
}

// RegisterRoutes registers usage routes.
func (h *UsageHandler) RegisterRoutes(r *gin.RouterGroup) {
	billing := r.Group("/billing")
	{
		billing.GET("/usage", h.GetUsageStats)
		billing.POST("/usage", h.RecordUsage)
	}
}

// GetUsageStats handles GET /billing/usage.
func (h *UsageHandler) GetUsageStats(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	period := c.DefaultQuery("period", "month")

	var start, end *time.Time
	if startStr := c.Query("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			start = &t
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			end = &t
		}
	}

	stats, err := h.billingDomain.GetUsageStats(c.Request.Context(), userID, period, start, end)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RecordUsage handles POST /billing/usage.
func (h *UsageHandler) RecordUsage(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req struct {
		RequestID    string    `json:"request_id" binding:"required"`
		TaskType     string    `json:"task_type" binding:"required"`
		ProviderID   uuid.UUID `json:"provider_id" binding:"required"`
		ModelID      string    `json:"model_id" binding:"required"`
		InputTokens  int       `json:"input_tokens"`
		OutputTokens int       `json:"output_tokens"`
		CostUSD      float64   `json:"cost_usd"`
		LatencyMs    int       `json:"latency_ms"`
		Success      bool      `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := &billing.RecordUsageInput{
		RequestID:    req.RequestID,
		TaskType:     req.TaskType,
		ProviderID:   req.ProviderID,
		ModelID:      req.ModelID,
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		CostUSD:      req.CostUSD,
		LatencyMs:    req.LatencyMs,
		Success:      req.Success,
	}

	if err := h.billingDomain.RecordUsage(c.Request.Context(), userID, input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

// Compile-time check
var _ inbound.UsageHttpPort = (*UsageHandler)(nil)
