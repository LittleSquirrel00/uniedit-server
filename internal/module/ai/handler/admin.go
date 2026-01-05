package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/provider"
)

// AdminHandler handles admin API requests for AI configuration.
type AdminHandler struct {
	providerRepo provider.Repository
	groupRepo    group.Repository
	registry     *provider.Registry
	groupManager *group.Manager
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	providerRepo provider.Repository,
	groupRepo group.Repository,
	registry *provider.Registry,
	groupManager *group.Manager,
) *AdminHandler {
	return &AdminHandler{
		providerRepo: providerRepo,
		groupRepo:    groupRepo,
		registry:     registry,
		groupManager: groupManager,
	}
}

// RegisterRoutes registers admin routes.
func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Provider routes
	r.GET("/providers", h.ListProviders)
	r.POST("/providers", h.CreateProvider)
	r.GET("/providers/:id", h.GetProvider)
	r.PUT("/providers/:id", h.UpdateProvider)
	r.DELETE("/providers/:id", h.DeleteProvider)

	// Model routes
	r.GET("/models", h.ListModels)
	r.POST("/models", h.CreateModel)
	r.GET("/models/:id", h.GetModel)
	r.PUT("/models/:id", h.UpdateModel)
	r.DELETE("/models/:id", h.DeleteModel)

	// Group routes
	r.GET("/groups", h.ListGroups)
	r.POST("/groups", h.CreateGroup)
	r.GET("/groups/:id", h.GetGroup)
	r.PUT("/groups/:id", h.UpdateGroup)
	r.DELETE("/groups/:id", h.DeleteGroup)

	// Registry management
	r.POST("/refresh", h.RefreshRegistry)
}

// Provider handlers

// ListProviders lists all providers.
// GET /api/v1/admin/ai/providers
func (h *AdminHandler) ListProviders(c *gin.Context) {
	providers, err := h.providerRepo.ListProviders(c.Request.Context(), false)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   providers,
	})
}

// CreateProviderRequest represents a provider creation request.
type CreateProviderRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Type     provider.ProviderType  `json:"type" binding:"required"`
	BaseURL  string                 `json:"base_url" binding:"required"`
	APIKey   string                 `json:"api_key" binding:"required"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Weight   int                    `json:"weight"`
	Priority int                    `json:"priority"`
	Enabled  bool                   `json:"enabled"`
}

// CreateProvider creates a new provider.
// POST /api/v1/admin/ai/providers
func (h *AdminHandler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p := &provider.Provider{
		ID:       uuid.New(),
		Name:     req.Name,
		Type:     req.Type,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Options:  req.Options,
		Weight:   req.Weight,
		Priority: req.Priority,
		Enabled:  req.Enabled,
	}

	if err := h.providerRepo.CreateProvider(c.Request.Context(), p); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusCreated, p)
}

// GetProvider gets a provider by ID.
// GET /api/v1/admin/ai/providers/:id
func (h *AdminHandler) GetProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	p, err := h.providerRepo.GetProvider(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, p)
}

// UpdateProviderRequest represents a provider update request.
type UpdateProviderRequest struct {
	Name     *string                 `json:"name,omitempty"`
	BaseURL  *string                 `json:"base_url,omitempty"`
	APIKey   *string                 `json:"api_key,omitempty"`
	Options  *map[string]interface{} `json:"options,omitempty"`
	Weight   *int                    `json:"weight,omitempty"`
	Priority *int                    `json:"priority,omitempty"`
	Enabled  *bool                   `json:"enabled,omitempty"`
}

// UpdateProvider updates a provider.
// PUT /api/v1/admin/ai/providers/:id
func (h *AdminHandler) UpdateProvider(c *gin.Context) {
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

	p, err := h.providerRepo.GetProvider(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.BaseURL != nil {
		p.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		p.APIKey = *req.APIKey
	}
	if req.Options != nil {
		p.Options = *req.Options
	}
	if req.Weight != nil {
		p.Weight = *req.Weight
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}

	if err := h.providerRepo.UpdateProvider(c.Request.Context(), p); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, p)
}

// DeleteProvider deletes a provider.
// DELETE /api/v1/admin/ai/providers/:id
func (h *AdminHandler) DeleteProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	if err := h.providerRepo.DeleteProvider(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "provider deleted"})
}

// Model handlers

// ListModels lists all models.
// GET /api/v1/admin/ai/models
func (h *AdminHandler) ListModels(c *gin.Context) {
	models, err := h.providerRepo.ListModels(c.Request.Context(), nil, false)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// CreateModelRequest represents a model creation request.
type CreateModelRequest struct {
	ID              string                 `json:"id" binding:"required"`
	ProviderID      uuid.UUID              `json:"provider_id" binding:"required"`
	Name            string                 `json:"name" binding:"required"`
	Capabilities    []provider.Capability  `json:"capabilities" binding:"required"`
	InputCostPer1K  float64                `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K float64                `json:"output_cost_per_1k,omitempty"`
	ContextWindow   int                    `json:"context_window,omitempty"`
	MaxOutputTokens int                    `json:"max_output_tokens,omitempty"`
	Options         map[string]interface{} `json:"options,omitempty"`
	Enabled         bool                   `json:"enabled"`
}

// CreateModel creates a new model.
// POST /api/v1/admin/ai/models
func (h *AdminHandler) CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert capabilities
	caps := make([]string, len(req.Capabilities))
	for i, cap := range req.Capabilities {
		caps[i] = string(cap)
	}

	m := &provider.Model{
		ID:              req.ID,
		ProviderID:      req.ProviderID,
		Name:            req.Name,
		Capabilities:    caps,
		InputCostPer1K:  req.InputCostPer1K,
		OutputCostPer1K: req.OutputCostPer1K,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		Options:         req.Options,
		Enabled:         req.Enabled,
	}

	if err := h.providerRepo.CreateModel(c.Request.Context(), m); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusCreated, m)
}

// GetModel gets a model by ID.
// GET /api/v1/admin/ai/models/:id
func (h *AdminHandler) GetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	m, err := h.providerRepo.GetModel(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, m)
}

// UpdateModelRequest represents a model update request.
type UpdateModelRequest struct {
	Name            *string                 `json:"name,omitempty"`
	Capabilities    *[]provider.Capability  `json:"capabilities,omitempty"`
	InputCostPer1K  *float64                `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K *float64                `json:"output_cost_per_1k,omitempty"`
	ContextWindow   *int                    `json:"context_window,omitempty"`
	MaxOutputTokens *int                    `json:"max_output_tokens,omitempty"`
	Options         *map[string]interface{} `json:"options,omitempty"`
	Enabled         *bool                   `json:"enabled,omitempty"`
}

// UpdateModel updates a model.
// PUT /api/v1/admin/ai/models/:id
func (h *AdminHandler) UpdateModel(c *gin.Context) {
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

	m, err := h.providerRepo.GetModel(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Capabilities != nil {
		caps := make([]string, len(*req.Capabilities))
		for i, cap := range *req.Capabilities {
			caps[i] = string(cap)
		}
		m.Capabilities = caps
	}
	if req.InputCostPer1K != nil {
		m.InputCostPer1K = *req.InputCostPer1K
	}
	if req.OutputCostPer1K != nil {
		m.OutputCostPer1K = *req.OutputCostPer1K
	}
	if req.ContextWindow != nil {
		m.ContextWindow = *req.ContextWindow
	}
	if req.MaxOutputTokens != nil {
		m.MaxOutputTokens = *req.MaxOutputTokens
	}
	if req.Options != nil {
		m.Options = *req.Options
	}
	if req.Enabled != nil {
		m.Enabled = *req.Enabled
	}

	if err := h.providerRepo.UpdateModel(c.Request.Context(), m); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, m)
}

// DeleteModel deletes a model.
// DELETE /api/v1/admin/ai/models/:id
func (h *AdminHandler) DeleteModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	if err := h.providerRepo.DeleteModel(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	// Refresh registry
	_ = h.registry.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
}

// Group handlers

// ListGroups lists all groups.
// GET /api/v1/admin/ai/groups
func (h *AdminHandler) ListGroups(c *gin.Context) {
	groups, err := h.groupRepo.List(c.Request.Context(), true)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   groups,
	})
}

// CreateGroupRequest represents a group creation request.
type CreateGroupRequest struct {
	ID                   string               `json:"id" binding:"required"`
	Name                 string               `json:"name" binding:"required"`
	TaskType             group.TaskType       `json:"task_type" binding:"required"`
	Models               []string             `json:"models" binding:"required"`
	Strategy             *group.StrategyConfig `json:"strategy,omitempty"`
	Fallback             *group.FallbackConfig `json:"fallback,omitempty"`
	RequiredCapabilities []string             `json:"required_capabilities,omitempty"`
	Enabled              bool                 `json:"enabled"`
}

// CreateGroup creates a new group.
// POST /api/v1/admin/ai/groups
func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g := &group.Group{
		ID:                   req.ID,
		Name:                 req.Name,
		TaskType:             req.TaskType,
		Models:               req.Models,
		Strategy:             req.Strategy,
		Fallback:             req.Fallback,
		RequiredCapabilities: req.RequiredCapabilities,
		Enabled:              req.Enabled,
	}

	// Set default strategy
	if g.Strategy == nil {
		g.Strategy = &group.StrategyConfig{Type: group.StrategyPriority}
	}

	if err := h.groupRepo.Create(c.Request.Context(), g); err != nil {
		handleError(c, err)
		return
	}

	// Refresh group manager
	_ = h.groupManager.Refresh(c.Request.Context())

	c.JSON(http.StatusCreated, g)
}

// GetGroup gets a group by ID.
// GET /api/v1/admin/ai/groups/:id
func (h *AdminHandler) GetGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group id required"})
		return
	}

	g, err := h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, g)
}

// UpdateGroupRequest represents a group update request.
type UpdateGroupRequest struct {
	Name                 *string               `json:"name,omitempty"`
	Models               *[]string             `json:"models,omitempty"`
	Strategy             *group.StrategyConfig `json:"strategy,omitempty"`
	Fallback             *group.FallbackConfig `json:"fallback,omitempty"`
	RequiredCapabilities *[]string             `json:"required_capabilities,omitempty"`
	Enabled              *bool                 `json:"enabled,omitempty"`
}

// UpdateGroup updates a group.
// PUT /api/v1/admin/ai/groups/:id
func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group id required"})
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, err := h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		g.Name = *req.Name
	}
	if req.Models != nil {
		g.Models = *req.Models
	}
	if req.Strategy != nil {
		g.Strategy = req.Strategy
	}
	if req.Fallback != nil {
		g.Fallback = req.Fallback
	}
	if req.RequiredCapabilities != nil {
		g.RequiredCapabilities = *req.RequiredCapabilities
	}
	if req.Enabled != nil {
		g.Enabled = *req.Enabled
	}

	if err := h.groupRepo.Update(c.Request.Context(), g); err != nil {
		handleError(c, err)
		return
	}

	// Refresh group manager
	_ = h.groupManager.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, g)
}

// DeleteGroup deletes a group.
// DELETE /api/v1/admin/ai/groups/:id
func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group id required"})
		return
	}

	if err := h.groupRepo.Delete(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	// Refresh group manager
	_ = h.groupManager.Refresh(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

// RefreshRegistry refreshes the provider registry and group manager.
// POST /api/v1/admin/ai/refresh
func (h *AdminHandler) RefreshRegistry(c *gin.Context) {
	if err := h.registry.Refresh(c.Request.Context()); err != nil {
		handleError(c, err)
		return
	}

	if err := h.groupManager.Refresh(c.Request.Context()); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "registry refreshed"})
}
