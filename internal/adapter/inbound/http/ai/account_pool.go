package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// AccountPoolHandler implements inbound.AIAccountPoolHttpPort.
type AccountPoolHandler struct {
	domain ai.AIDomain
}

// NewAccountPoolHandler creates a new account pool handler.
func NewAccountPoolHandler(domain ai.AIDomain) *AccountPoolHandler {
	return &AccountPoolHandler{domain: domain}
}

// ListAccounts handles GET /admin/ai/providers/:provider_id/accounts.
func (h *AccountPoolHandler) ListAccounts(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("provider_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider_id"})
		return
	}

	accounts, err := h.domain.ListAccounts(c.Request.Context(), providerID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response format (hide sensitive data)
	responses := make([]*AccountResponse, len(accounts))
	for i, acc := range accounts {
		responses[i] = toAccountResponse(acc)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   responses,
		"total":  len(accounts),
	})
}

// GetAccount handles GET /admin/ai/accounts/:id.
func (h *AccountPoolHandler) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	account, err := h.domain.GetAccount(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(account))
}

// CreateAccountRequest represents an account creation request.
type CreateAccountRequest struct {
	Name         string `json:"name" binding:"required"`
	APIKey       string `json:"api_key" binding:"required"`
	Weight       int    `json:"weight"`
	Priority     int    `json:"priority"`
	RateLimitRPM int    `json:"rate_limit_rpm"`
	RateLimitTPM int    `json:"rate_limit_tpm"`
	DailyLimit   int    `json:"daily_limit"`
}

// CreateAccount handles POST /admin/ai/providers/:provider_id/accounts.
func (h *AccountPoolHandler) CreateAccount(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("provider_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider_id"})
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	weight := req.Weight
	if weight <= 0 {
		weight = 1
	}

	// Extract key prefix (first 8 chars or less)
	keyPrefix := req.APIKey
	if len(keyPrefix) > 8 {
		keyPrefix = keyPrefix[:8]
	}

	account := &model.AIProviderAccount{
		ID:           uuid.New(),
		ProviderID:   providerID,
		Name:         req.Name,
		KeyPrefix:    keyPrefix + "...",
		Weight:       weight,
		Priority:     req.Priority,
		IsActive:     true,
		HealthStatus: model.AIHealthStatusHealthy,
		RateLimitRPM: req.RateLimitRPM,
		RateLimitTPM: req.RateLimitTPM,
		DailyLimit:   req.DailyLimit,
	}

	// The domain will handle encryption
	if err := h.domain.CreateAccount(c.Request.Context(), account, req.APIKey); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toAccountResponse(account))
}

// UpdateAccountRequest represents an account update request.
type UpdateAccountRequest struct {
	Name         *string `json:"name,omitempty"`
	Weight       *int    `json:"weight,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
	RateLimitRPM *int    `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM *int    `json:"rate_limit_tpm,omitempty"`
	DailyLimit   *int    `json:"daily_limit,omitempty"`
}

// UpdateAccount handles PUT /admin/ai/accounts/:id.
func (h *AccountPoolHandler) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.domain.GetAccount(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		account.Name = *req.Name
	}
	if req.Weight != nil {
		account.Weight = *req.Weight
	}
	if req.Priority != nil {
		account.Priority = *req.Priority
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.RateLimitRPM != nil {
		account.RateLimitRPM = *req.RateLimitRPM
	}
	if req.RateLimitTPM != nil {
		account.RateLimitTPM = *req.RateLimitTPM
	}
	if req.DailyLimit != nil {
		account.DailyLimit = *req.DailyLimit
	}

	if err := h.domain.UpdateAccount(c.Request.Context(), account); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(account))
}

// DeleteAccount handles DELETE /admin/ai/accounts/:id.
func (h *AccountPoolHandler) DeleteAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	if err := h.domain.DeleteAccount(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAccountStats handles GET /admin/ai/accounts/:id/stats.
func (h *AccountPoolHandler) GetAccountStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	stats, err := h.domain.GetAccountStats(c.Request.Context(), id, 30)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": id,
		"stats":      stats,
	})
}

// ResetAccountHealth handles POST /admin/ai/accounts/:id/reset-health.
func (h *AccountPoolHandler) ResetAccountHealth(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	if err := h.domain.ResetAccountHealth(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "health reset",
		"health_status": model.AIHealthStatusHealthy,
	})
}

// AccountResponse represents an account in API responses.
type AccountResponse struct {
	ID                  uuid.UUID            `json:"id"`
	ProviderID          uuid.UUID            `json:"provider_id"`
	Name                string               `json:"name"`
	KeyPrefix           string               `json:"key_prefix"`
	Weight              int                  `json:"weight"`
	Priority            int                  `json:"priority"`
	IsActive            bool                 `json:"is_active"`
	HealthStatus        model.AIHealthStatus `json:"health_status"`
	ConsecutiveFailures int                  `json:"consecutive_failures"`
	RateLimitRPM        int                  `json:"rate_limit_rpm"`
	RateLimitTPM        int                  `json:"rate_limit_tpm"`
	DailyLimit          int                  `json:"daily_limit"`
	TotalRequests       int64                `json:"total_requests"`
	TotalTokens         int64                `json:"total_tokens"`
	TotalCostUSD        float64              `json:"total_cost_usd"`
}

func toAccountResponse(acc *model.AIProviderAccount) *AccountResponse {
	return &AccountResponse{
		ID:                  acc.ID,
		ProviderID:          acc.ProviderID,
		Name:                acc.Name,
		KeyPrefix:           acc.KeyPrefix,
		Weight:              acc.Weight,
		Priority:            acc.Priority,
		IsActive:            acc.IsActive,
		HealthStatus:        acc.HealthStatus,
		ConsecutiveFailures: acc.ConsecutiveFailures,
		RateLimitRPM:        acc.RateLimitRPM,
		RateLimitTPM:        acc.RateLimitTPM,
		DailyLimit:          acc.DailyLimit,
		TotalRequests:       acc.TotalRequests,
		TotalTokens:         acc.TotalTokens,
		TotalCostUSD:        acc.TotalCostUSD,
	}
}

// Compile-time interface check
var _ inbound.AIAccountPoolHttpPort = (*AccountPoolHandler)(nil)
