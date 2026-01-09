package mediahttp

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// Handler handles media HTTP requests.
type Handler struct {
	domain *media.Domain
}

// NewHandler creates a new media handler.
func NewHandler(domain *media.Domain) *Handler {
	return &Handler{domain: domain}
}

// RegisterRoutes registers media routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
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
func (h *Handler) RegisterAdminRoutes(r *gin.RouterGroup) {
	mediaGroup := r.Group("/media")
	{
		mediaGroup.GET("/providers", h.ListProviders)
		mediaGroup.GET("/providers/:id", h.GetProvider)
		mediaGroup.GET("/models", h.ListModels)
	}
}

// GenerateImage handles image generation requests.
func (h *Handler) GenerateImage(c *gin.Context) {
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
func (h *Handler) GenerateVideo(c *gin.Context) {
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
func (h *Handler) GetVideoStatus(c *gin.Context) {
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
func (h *Handler) GetTask(c *gin.Context) {
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
func (h *Handler) ListTasks(c *gin.Context) {
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
func (h *Handler) CancelTask(c *gin.Context) {
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
func (h *Handler) ListProviders(c *gin.Context) {
	providers, err := h.domain.ListProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, providers)
}

// GetProvider handles provider retrieval requests.
func (h *Handler) GetProvider(c *gin.Context) {
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
func (h *Handler) ListModels(c *gin.Context) {
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
func (h *Handler) GetModelsByCapability(c *gin.Context) {
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

func getUserID(c *gin.Context) uuid.UUID {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// Compile-time interface checks
var (
	_ inbound.MediaHttpPort      = (*Handler)(nil)
	_ inbound.MediaAdminHttpPort = (*Handler)(nil)
)
