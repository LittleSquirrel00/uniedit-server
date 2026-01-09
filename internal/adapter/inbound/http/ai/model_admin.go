package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// ModelAdminHandler implements inbound.AIModelAdminHttpPort.
type ModelAdminHandler struct {
	domain ai.AIDomain
}

// NewModelAdminHandler creates a new model admin handler.
func NewModelAdminHandler(domain ai.AIDomain) *ModelAdminHandler {
	return &ModelAdminHandler{domain: domain}
}

// ListModels handles GET /admin/ai/models.
func (h *ModelAdminHandler) ListModels(c *gin.Context) {
	models, err := h.domain.ListModels(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// GetModel handles GET /admin/ai/models/:id.
func (h *ModelAdminHandler) GetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	m, err := h.domain.GetModel(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, m)
}

// CreateModelRequest represents a model creation request.
type CreateModelRequest struct {
	ID              string               `json:"id" binding:"required"`
	ProviderID      uuid.UUID            `json:"provider_id" binding:"required"`
	Name            string               `json:"name" binding:"required"`
	Capabilities    []model.AICapability `json:"capabilities" binding:"required"`
	ContextWindow   int                  `json:"context_window,omitempty"`
	MaxOutputTokens int                  `json:"max_output_tokens,omitempty"`
	InputCostPer1K  float64              `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K float64              `json:"output_cost_per_1k,omitempty"`
	Options         map[string]any       `json:"options,omitempty"`
	Enabled         bool                 `json:"enabled"`
}

// CreateModel handles POST /admin/ai/models.
func (h *ModelAdminHandler) CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert capabilities
	caps := make(pq.StringArray, len(req.Capabilities))
	for i, cap := range req.Capabilities {
		caps[i] = string(cap)
	}

	m := &model.AIModel{
		ID:              req.ID,
		ProviderID:      req.ProviderID,
		Name:            req.Name,
		Capabilities:    caps,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		InputCostPer1K:  req.InputCostPer1K,
		OutputCostPer1K: req.OutputCostPer1K,
		Options:         req.Options,
		Enabled:         req.Enabled,
	}

	if err := h.domain.CreateModel(c.Request.Context(), m); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, m)
}

// UpdateModelRequest represents a model update request.
type UpdateModelRequest struct {
	Name            *string               `json:"name,omitempty"`
	Capabilities    *[]model.AICapability `json:"capabilities,omitempty"`
	ContextWindow   *int                  `json:"context_window,omitempty"`
	MaxOutputTokens *int                  `json:"max_output_tokens,omitempty"`
	InputCostPer1K  *float64              `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K *float64              `json:"output_cost_per_1k,omitempty"`
	Options         *map[string]any       `json:"options,omitempty"`
	Enabled         *bool                 `json:"enabled,omitempty"`
}

// UpdateModel handles PUT /admin/ai/models/:id.
func (h *ModelAdminHandler) UpdateModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m, err := h.domain.GetModel(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Capabilities != nil {
		caps := make(pq.StringArray, len(*req.Capabilities))
		for i, cap := range *req.Capabilities {
			caps[i] = string(cap)
		}
		m.Capabilities = caps
	}
	if req.ContextWindow != nil {
		m.ContextWindow = *req.ContextWindow
	}
	if req.MaxOutputTokens != nil {
		m.MaxOutputTokens = *req.MaxOutputTokens
	}
	if req.InputCostPer1K != nil {
		m.InputCostPer1K = *req.InputCostPer1K
	}
	if req.OutputCostPer1K != nil {
		m.OutputCostPer1K = *req.OutputCostPer1K
	}
	if req.Options != nil {
		m.Options = *req.Options
	}
	if req.Enabled != nil {
		m.Enabled = *req.Enabled
	}

	if err := h.domain.UpdateModel(c.Request.Context(), m); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, m)
}

// DeleteModel handles DELETE /admin/ai/models/:id.
func (h *ModelAdminHandler) DeleteModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	if err := h.domain.DeleteModel(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
}

// Compile-time interface check
var _ inbound.AIModelAdminHttpPort = (*ModelAdminHandler)(nil)
