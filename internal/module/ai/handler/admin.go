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
//
//	@Summary		List AI providers
//	@Description	List all configured AI providers (admin only)
//	@Tags			AI Admin - Providers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"List of providers"
//	@Failure		401	{object}	map[string]string		"Unauthorized"
//	@Failure		403	{object}	map[string]string		"Forbidden - Admin access required"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/admin/ai/providers [get]
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
//
//	@Summary		Create AI provider
//	@Description	Create a new AI provider configuration (admin only)
//	@Tags			AI Admin - Providers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateProviderRequest	true	"Provider configuration"
//	@Success		201		{object}	provider.Provider
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/providers [post]
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
//
//	@Summary		Get AI provider
//	@Description	Get details of a specific AI provider by ID (admin only)
//	@Tags			AI Admin - Providers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Provider ID (UUID)"
//	@Success		200	{object}	provider.Provider
//	@Failure		400	{object}	map[string]string	"Invalid provider ID"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Provider not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/providers/{id} [get]
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
//
//	@Summary		Update AI provider
//	@Description	Update an existing AI provider configuration (admin only)
//	@Tags			AI Admin - Providers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Provider ID (UUID)"
//	@Param			request	body		UpdateProviderRequest	true	"Updated provider configuration"
//	@Success		200		{object}	provider.Provider
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404		{object}	map[string]string	"Provider not found"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/providers/{id} [put]
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
//
//	@Summary		Delete AI provider
//	@Description	Delete an AI provider configuration (admin only)
//	@Tags			AI Admin - Providers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Provider ID (UUID)"
//	@Success		200	{object}	map[string]string	"Provider deleted successfully"
//	@Failure		400	{object}	map[string]string	"Invalid provider ID"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Provider not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/providers/{id} [delete]
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
//
//	@Summary		List AI models
//	@Description	List all configured AI models (admin only)
//	@Tags			AI Admin - Models
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"List of models"
//	@Failure		401	{object}	map[string]string		"Unauthorized"
//	@Failure		403	{object}	map[string]string		"Forbidden - Admin access required"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/admin/ai/models [get]
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
//
//	@Summary		Create AI model
//	@Description	Create a new AI model configuration (admin only)
//	@Tags			AI Admin - Models
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateModelRequest	true	"Model configuration"
//	@Success		201		{object}	provider.Model
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/models [post]
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
//
//	@Summary		Get AI model
//	@Description	Get details of a specific AI model by ID (admin only)
//	@Tags			AI Admin - Models
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Model ID"
//	@Success		200	{object}	provider.Model
//	@Failure		400	{object}	map[string]string	"Model ID required"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Model not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/models/{id} [get]
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
//
//	@Summary		Update AI model
//	@Description	Update an existing AI model configuration (admin only)
//	@Tags			AI Admin - Models
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Model ID"
//	@Param			request	body		UpdateModelRequest	true	"Updated model configuration"
//	@Success		200		{object}	provider.Model
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404		{object}	map[string]string	"Model not found"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/models/{id} [put]
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
//
//	@Summary		Delete AI model
//	@Description	Delete an AI model configuration (admin only)
//	@Tags			AI Admin - Models
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Model ID"
//	@Success		200	{object}	map[string]string	"Model deleted successfully"
//	@Failure		400	{object}	map[string]string	"Model ID required"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Model not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/models/{id} [delete]
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
//
//	@Summary		List AI groups
//	@Description	List all configured AI model groups (admin only)
//	@Tags			AI Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"List of groups"
//	@Failure		401	{object}	map[string]string		"Unauthorized"
//	@Failure		403	{object}	map[string]string		"Forbidden - Admin access required"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/admin/ai/groups [get]
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
//
//	@Summary		Create AI group
//	@Description	Create a new AI model group for routing (admin only)
//	@Tags			AI Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateGroupRequest	true	"Group configuration"
//	@Success		201		{object}	group.Group
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/groups [post]
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
//
//	@Summary		Get AI group
//	@Description	Get details of a specific AI model group by ID (admin only)
//	@Tags			AI Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{object}	group.Group
//	@Failure		400	{object}	map[string]string	"Group ID required"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Group not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/groups/{id} [get]
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
//
//	@Summary		Update AI group
//	@Description	Update an existing AI model group configuration (admin only)
//	@Tags			AI Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Group ID"
//	@Param			request	body		UpdateGroupRequest	true	"Updated group configuration"
//	@Success		200		{object}	group.Group
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		403		{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404		{object}	map[string]string	"Group not found"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/groups/{id} [put]
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
//
//	@Summary		Delete AI group
//	@Description	Delete an AI model group configuration (admin only)
//	@Tags			AI Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{object}	map[string]string	"Group deleted successfully"
//	@Failure		400	{object}	map[string]string	"Group ID required"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		404	{object}	map[string]string	"Group not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/groups/{id} [delete]
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
//
//	@Summary		Refresh AI registry
//	@Description	Refresh the provider registry and group manager to reload configurations (admin only)
//	@Tags			AI Admin - System
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]string	"Registry refreshed successfully"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		403	{object}	map[string]string	"Forbidden - Admin access required"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/admin/ai/refresh [post]
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
