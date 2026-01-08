package gin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/port/inbound"
)

// usageHandler implements inbound.UsageHttpPort.
type usageHandler struct {
	billingDomain billing.BillingDomain
}

// NewUsageHandler creates a new usage HTTP handler.
func NewUsageHandler(billingDomain billing.BillingDomain) inbound.UsageHttpPort {
	return &usageHandler{billingDomain: billingDomain}
}

func (h *usageHandler) GetUsageStats(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *usageHandler) RecordUsage(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

// Compile-time check
var _ inbound.UsageHttpPort = (*usageHandler)(nil)
