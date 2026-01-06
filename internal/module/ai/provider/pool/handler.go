package pool

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for provider account pool.
type Handler struct {
	manager *Manager
	logger  *zap.Logger
}

// NewHandler creates a new handler.
func NewHandler(manager *Manager, logger *zap.Logger) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes registers the handler routes.
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	// Use /account-pool to avoid conflict with /providers/:id
	pool := admin.Group("/account-pool")
	{
		pool.POST("/providers/:provider_id/accounts", h.AddAccount)
		pool.GET("/providers/:provider_id/accounts", h.ListAccounts)
		pool.GET("/accounts/:account_id", h.GetAccount)
		pool.PATCH("/accounts/:account_id", h.UpdateAccount)
		pool.DELETE("/accounts/:account_id", h.DeleteAccount)
		pool.GET("/accounts/:account_id/stats", h.GetAccountStats)
		pool.POST("/accounts/:account_id/check-health", h.CheckHealth)
	}
}

// AddAccount adds an account to the pool.
//
//	@Summary		Add provider account
//	@Description	Add a new account to an AI provider's pool
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string					true	"Provider ID"
//	@Param			request		body		CreateAccountRequest	true	"Account data"
//	@Success		201			{object}	AccountResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		409			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts [post]
func (h *Handler) AddAccount(c *gin.Context) {
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

	account, err := h.manager.AddAccount(c.Request.Context(), providerID, req.Name, req.APIKey, weight, req.Priority)
	if err != nil {
		if err == ErrDuplicateAccount {
			c.JSON(http.StatusConflict, gin.H{"error": "account with this name already exists"})
			return
		}
		h.logger.Error("failed to add account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add account"})
		return
	}

	// Update rate limits if provided
	if req.RateLimitRPM > 0 || req.RateLimitTPM > 0 || req.DailyLimit > 0 {
		account.RateLimitRPM = req.RateLimitRPM
		account.RateLimitTPM = req.RateLimitTPM
		account.DailyLimit = req.DailyLimit
		if err := h.manager.UpdateAccount(c.Request.Context(), account); err != nil {
			h.logger.Warn("failed to update rate limits", zap.Error(err))
		}
	}

	c.JSON(http.StatusCreated, account.ToResponse())
}

// ListAccounts lists all accounts for a provider.
//
//	@Summary		List provider accounts
//	@Description	Get all accounts for an AI provider
//	@Tags			AI
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string	true	"Provider ID"
//	@Success		200			{object}	ListAccountsResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts [get]
func (h *Handler) ListAccounts(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("provider_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider_id"})
		return
	}

	accounts, err := h.manager.ListAccounts(c.Request.Context(), providerID)
	if err != nil {
		h.logger.Error("failed to list accounts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
		return
	}

	responses := make([]*AccountResponse, len(accounts))
	for i, acc := range accounts {
		responses[i] = acc.ToResponse()
	}

	c.JSON(http.StatusOK, ListAccountsResponse{
		Accounts: responses,
		Total:    len(accounts),
	})
}

// GetAccount gets an account by ID.
//
//	@Summary		Get provider account
//	@Description	Get details of a specific provider account
//	@Tags			AI
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string	true	"Provider ID"
//	@Param			account_id	path		string	true	"Account ID"
//	@Success		200			{object}	AccountResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts/{account_id} [get]
func (h *Handler) GetAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	account, err := h.manager.repo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		if err == ErrAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		h.logger.Error("failed to get account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get account"})
		return
	}

	c.JSON(http.StatusOK, account.ToResponse())
}

// UpdateAccount updates an account.
//
//	@Summary		Update provider account
//	@Description	Update a provider account's settings
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string					true	"Provider ID"
//	@Param			account_id	path		string					true	"Account ID"
//	@Param			request		body		UpdateAccountRequest	true	"Update data"
//	@Success		200			{object}	AccountResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts/{account_id} [patch]
func (h *Handler) UpdateAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.manager.repo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		if err == ErrAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		h.logger.Error("failed to get account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get account"})
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

	if err := h.manager.UpdateAccount(c.Request.Context(), account); err != nil {
		h.logger.Error("failed to update account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update account"})
		return
	}

	c.JSON(http.StatusOK, account.ToResponse())
}

// DeleteAccount removes an account from the pool.
//
//	@Summary		Delete provider account
//	@Description	Remove an account from a provider's pool
//	@Tags			AI
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path	string	true	"Provider ID"
//	@Param			account_id	path	string	true	"Account ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts/{account_id} [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	if err := h.manager.RemoveAccount(c.Request.Context(), accountID); err != nil {
		if err == ErrAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		h.logger.Error("failed to delete account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete account"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAccountStats gets usage statistics for an account.
//
//	@Summary		Get account statistics
//	@Description	Get usage statistics for a provider account
//	@Tags			AI
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string	true	"Provider ID"
//	@Param			account_id	path		string	true	"Account ID"
//	@Success		200			{object}	AccountStatsResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts/{account_id}/stats [get]
func (h *Handler) GetAccountStats(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	stats, err := h.manager.GetAccountStats(c.Request.Context(), accountID)
	if err != nil {
		if err == ErrAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		h.logger.Error("failed to get account stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get account stats"})
		return
	}

	c.JSON(http.StatusOK, stats.ToResponse())
}

// CheckHealth triggers a health check for an account.
//
//	@Summary		Check account health
//	@Description	Trigger a health check for a provider account
//	@Tags			AI
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider_id	path		string	true	"Provider ID"
//	@Param			account_id	path		string	true	"Account ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/admin/providers/{provider_id}/accounts/{account_id}/check-health [post]
func (h *Handler) CheckHealth(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	account, err := h.manager.repo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		if err == ErrAccountNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		h.logger.Error("failed to get account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get account"})
		return
	}

	// Reset health to trigger re-evaluation
	if err := h.manager.RefreshHealth(c.Request.Context(), account.ProviderID); err != nil {
		h.logger.Error("failed to refresh health", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh health"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "health check triggered",
		"health_status": account.HealthStatus,
	})
}
