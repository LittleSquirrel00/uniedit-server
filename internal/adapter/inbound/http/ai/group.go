package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// GroupHandler implements inbound.AIModelGroupHttpPort.
type GroupHandler struct {
	domain ai.AIDomain
}

// NewGroupHandler creates a new group handler.
func NewGroupHandler(domain ai.AIDomain) *GroupHandler {
	return &GroupHandler{domain: domain}
}

// ListGroups handles GET /admin/ai/groups.
func (h *GroupHandler) ListGroups(c *gin.Context) {
	groups, err := h.domain.ListGroups(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   groups,
	})
}

// GetGroup handles GET /admin/ai/groups/:id.
func (h *GroupHandler) GetGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group id required"})
		return
	}

	group, err := h.domain.GetGroup(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// CreateGroupRequest represents a group creation request.
type CreateGroupRequest struct {
	ID                   string                   `json:"id" binding:"required"`
	Name                 string                   `json:"name" binding:"required"`
	TaskType             model.AITaskType         `json:"task_type" binding:"required"`
	Models               []string                 `json:"models" binding:"required"`
	Strategy             *model.AIStrategyConfig  `json:"strategy,omitempty"`
	Fallback             *model.AIFallbackConfig  `json:"fallback,omitempty"`
	RequiredCapabilities []string                 `json:"required_capabilities,omitempty"`
	Enabled              bool                     `json:"enabled"`
}

// CreateGroup handles POST /admin/ai/groups.
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := &model.AIModelGroup{
		ID:                   req.ID,
		Name:                 req.Name,
		TaskType:             req.TaskType,
		Models:               pq.StringArray(req.Models),
		Strategy:             req.Strategy,
		Fallback:             req.Fallback,
		RequiredCapabilities: pq.StringArray(req.RequiredCapabilities),
		Enabled:              req.Enabled,
	}

	// Set default strategy
	if group.Strategy == nil {
		group.Strategy = &model.AIStrategyConfig{Type: model.AIStrategyPriority}
	}

	if err := h.domain.CreateGroup(c.Request.Context(), group); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateGroupRequest represents a group update request.
type UpdateGroupRequest struct {
	Name                 *string                  `json:"name,omitempty"`
	Models               *[]string                `json:"models,omitempty"`
	Strategy             *model.AIStrategyConfig  `json:"strategy,omitempty"`
	Fallback             *model.AIFallbackConfig  `json:"fallback,omitempty"`
	RequiredCapabilities *[]string                `json:"required_capabilities,omitempty"`
	Enabled              *bool                    `json:"enabled,omitempty"`
}

// UpdateGroup handles PUT /admin/ai/groups/:id.
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
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

	group, err := h.domain.GetGroup(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Models != nil {
		group.Models = pq.StringArray(*req.Models)
	}
	if req.Strategy != nil {
		group.Strategy = req.Strategy
	}
	if req.Fallback != nil {
		group.Fallback = req.Fallback
	}
	if req.RequiredCapabilities != nil {
		group.RequiredCapabilities = pq.StringArray(*req.RequiredCapabilities)
	}
	if req.Enabled != nil {
		group.Enabled = *req.Enabled
	}

	if err := h.domain.UpdateGroup(c.Request.Context(), group); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup handles DELETE /admin/ai/groups/:id.
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group id required"})
		return
	}

	if err := h.domain.DeleteGroup(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

// Compile-time interface check
var _ inbound.AIModelGroupHttpPort = (*GroupHandler)(nil)
