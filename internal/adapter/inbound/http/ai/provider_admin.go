package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// ProviderAdminHandler implements inbound.AIProviderAdminHttpPort.
type ProviderAdminHandler struct {
	domain ai.AIDomain
}

// NewProviderAdminHandler creates a new provider admin handler.
func NewProviderAdminHandler(domain ai.AIDomain) *ProviderAdminHandler {
	return &ProviderAdminHandler{domain: domain}
}

// ListProviders handles GET /admin/ai/providers.
func (h *ProviderAdminHandler) ListProviders(c *gin.Context) {
	providers, err := h.domain.ListProviders(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   providers,
	})
}

// GetProvider handles GET /admin/ai/providers/:id.
func (h *ProviderAdminHandler) GetProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	provider, err := h.domain.GetProvider(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// CreateProviderRequest represents a provider creation request.
type CreateProviderRequest struct {
	Name      string                  `json:"name" binding:"required"`
	Type      model.AIProviderType    `json:"type" binding:"required"`
	BaseURL   string                  `json:"base_url" binding:"required"`
	APIKey    string                  `json:"api_key" binding:"required"`
	RateLimit *model.AIRateLimitConfig `json:"rate_limit,omitempty"`
	Options   map[string]any          `json:"options,omitempty"`
	Weight    int                     `json:"weight"`
	Priority  int                     `json:"priority"`
	Enabled   bool                    `json:"enabled"`
}

// CreateProvider handles POST /admin/ai/providers.
func (h *ProviderAdminHandler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider := &model.AIProvider{
		ID:        uuid.New(),
		Name:      req.Name,
		Type:      req.Type,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		RateLimit: req.RateLimit,
		Options:   req.Options,
		Weight:    req.Weight,
		Priority:  req.Priority,
		Enabled:   req.Enabled,
	}

	if err := h.domain.CreateProvider(c.Request.Context(), provider); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// UpdateProviderRequest represents a provider update request.
type UpdateProviderRequest struct {
	Name      *string                  `json:"name,omitempty"`
	BaseURL   *string                  `json:"base_url,omitempty"`
	APIKey    *string                  `json:"api_key,omitempty"`
	RateLimit *model.AIRateLimitConfig `json:"rate_limit,omitempty"`
	Options   *map[string]any          `json:"options,omitempty"`
	Weight    *int                     `json:"weight,omitempty"`
	Priority  *int                     `json:"priority,omitempty"`
	Enabled   *bool                    `json:"enabled,omitempty"`
}

// UpdateProvider handles PUT /admin/ai/providers/:id.
func (h *ProviderAdminHandler) UpdateProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.domain.GetProvider(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		provider.Name = *req.Name
	}
	if req.BaseURL != nil {
		provider.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		provider.APIKey = *req.APIKey
	}
	if req.RateLimit != nil {
		provider.RateLimit = req.RateLimit
	}
	if req.Options != nil {
		provider.Options = *req.Options
	}
	if req.Weight != nil {
		provider.Weight = *req.Weight
	}
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}

	if err := h.domain.UpdateProvider(c.Request.Context(), provider); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider handles DELETE /admin/ai/providers/:id.
func (h *ProviderAdminHandler) DeleteProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	if err := h.domain.DeleteProvider(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider deleted"})
}

// SyncModels handles POST /admin/ai/providers/:id/sync.
func (h *ProviderAdminHandler) SyncModels(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	if err := h.domain.SyncModels(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "models synced"})
}

// HealthCheck handles POST /admin/ai/providers/:id/health.
func (h *ProviderAdminHandler) HealthCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	healthy, err := h.domain.ProviderHealthCheck(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider_id": id,
		"healthy":     healthy,
	})
}

// Compile-time interface check
var _ inbound.AIProviderAdminHttpPort = (*ProviderAdminHandler)(nil)
