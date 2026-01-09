package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// MediaHandler handles media HTTP requests.
type MediaHandler struct {
	domain *media.Domain
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(domain *media.Domain) *MediaHandler {
	return &MediaHandler{domain: domain}
}

// RegisterRoutes registers media routes.
func (h *MediaHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	mediaGroup := r.Group("/media")
	mediaGroup.Use(authMiddleware)
	{
		// Image generation
		mediaGroup.POST("/images/generations", h.GenerateImage)

		// Video generation
		mediaGroup.POST("/videos/generations", h.GenerateVideo)
		mediaGroup.GET("/videos/generations/:task_id", h.GetVideoStatus)

		// Tasks
		mediaGroup.GET("/tasks", h.ListTasks)
		mediaGroup.GET("/tasks/:task_id", h.GetTask)
		mediaGroup.DELETE("/tasks/:task_id", h.CancelTask)
	}
}

// RegisterAdminRoutes registers media admin routes.
func (h *MediaHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	mediaGroup := r.Group("/media")
	{
		mediaGroup.GET("/providers", h.ListProviders)
		mediaGroup.GET("/providers/:id", h.GetProvider)
		mediaGroup.GET("/models", h.ListModels)
	}
}

// GenerateImage handles image generation requests.
// @Summary Generate images
// @Tags Media
// @Accept json
// @Produce json
// @Param request body inbound.MediaImageGenerationInput true "Image generation request"
// @Success 200 {object} inbound.MediaImageGenerationOutput
// @Router /media/images/generations [post]
func (h *MediaHandler) GenerateImage(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input inbound.MediaImageGenerationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.domain.GenerateImage(c.Request.Context(), userID, &input)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// GenerateVideo handles video generation requests.
// @Summary Generate videos
// @Tags Media
// @Accept json
// @Produce json
// @Param request body inbound.MediaVideoGenerationInput true "Video generation request"
// @Success 200 {object} inbound.MediaVideoGenerationOutput
// @Router /media/videos/generations [post]
func (h *MediaHandler) GenerateVideo(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input inbound.MediaVideoGenerationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.domain.GenerateVideo(c.Request.Context(), userID, &input)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// GetVideoStatus handles video status requests.
// @Summary Get video generation status
// @Tags Media
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} inbound.MediaVideoGenerationOutput
// @Router /media/videos/generations/{task_id} [get]
func (h *MediaHandler) GetVideoStatus(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := c.Param("task_id")
	output, err := h.domain.GetVideoStatus(c.Request.Context(), userID, taskID)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// GetTask handles task retrieval requests.
// @Summary Get task
// @Tags Media
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} inbound.MediaTaskOutput
// @Router /media/tasks/{task_id} [get]
func (h *MediaHandler) GetTask(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskIDStr := c.Param("task_id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	output, err := h.domain.GetTask(c.Request.Context(), userID, taskID)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// ListTasks handles task listing requests.
// @Summary List tasks
// @Tags Media
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} inbound.MediaTaskOutput
// @Router /media/tasks [get]
func (h *MediaHandler) ListTasks(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, err := h.domain.ListTasks(c.Request.Context(), userID, limit, offset)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// CancelTask handles task cancellation requests.
// @Summary Cancel task
// @Tags Media
// @Param task_id path string true "Task ID"
// @Success 204 "No Content"
// @Router /media/tasks/{task_id} [delete]
func (h *MediaHandler) CancelTask(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskIDStr := c.Param("task_id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.domain.CancelTask(c.Request.Context(), userID, taskID); err != nil {
		handleMediaError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListProviders handles provider listing requests.
// @Summary List providers (admin)
// @Tags Media Admin
// @Produce json
// @Success 200 {array} model.MediaProvider
// @Router /admin/media/providers [get]
func (h *MediaHandler) ListProviders(c *gin.Context) {
	providers, err := h.domain.ListProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, providers)
}

// GetProvider handles provider retrieval requests.
// @Summary Get provider (admin)
// @Tags Media Admin
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} model.MediaProvider
// @Router /admin/media/providers/{id} [get]
func (h *MediaHandler) GetProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	provider, err := h.domain.GetProvider(c.Request.Context(), id)
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// ListModels handles model listing requests.
// @Summary List models (admin)
// @Tags Media Admin
// @Produce json
// @Param capability query string false "Filter by capability"
// @Success 200 {array} model.MediaModel
// @Router /admin/media/models [get]
func (h *MediaHandler) ListModels(c *gin.Context) {
	capability := c.Query("capability")

	if capability != "" {
		models, err := h.domain.ListModelsByCapability(c.Request.Context(), model.MediaCapability(capability))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, models)
		return
	}

	// List all - not implemented in domain yet, return empty for now
	c.JSON(http.StatusOK, []model.MediaModel{})
}

// GetModelsByCapability handles model listing by capability.
// @Summary Get models by capability (admin)
// @Tags Media Admin
// @Produce json
// @Param capability path string true "Capability"
// @Success 200 {array} model.MediaModel
// @Router /admin/media/models/capability/{capability} [get]
func (h *MediaHandler) GetModelsByCapability(c *gin.Context) {
	capability := c.Param("capability")
	if capability == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "capability is required"})
		return
	}

	models, err := h.domain.ListModelsByCapability(c.Request.Context(), model.MediaCapability(capability))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models)
}

// handleMediaError handles media domain errors.
func handleMediaError(c *gin.Context, err error) {
	switch err {
	case media.ErrProviderNotFound, media.ErrModelNotFound, media.ErrTaskNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case media.ErrTaskNotOwned:
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case media.ErrInvalidInput:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case media.ErrNoAdapterFound, media.ErrProviderUnhealthy, media.ErrNoHealthyProvider:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case media.ErrCapabilityNotSupported:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case media.ErrTaskAlreadyCompleted, media.ErrTaskAlreadyCancelled:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Compile-time interface checks
var (
	_ inbound.MediaHttpPort      = (*MediaHandler)(nil)
	_ inbound.MediaAdminHttpPort = (*MediaHandler)(nil)
)
