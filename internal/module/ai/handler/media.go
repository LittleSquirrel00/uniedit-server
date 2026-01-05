package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/module/ai/media"
)

// MediaHandler handles media generation API requests.
type MediaHandler struct {
	mediaService *media.Service
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(mediaService *media.Service) *MediaHandler {
	return &MediaHandler{
		mediaService: mediaService,
	}
}

// RegisterRoutes registers media routes.
func (h *MediaHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/images/generations", h.GenerateImage)
	r.POST("/videos/generations", h.GenerateVideo)
	r.GET("/videos/:task_id", h.GetVideoStatus)
}

// GenerateImage handles image generation requests.
// POST /api/v1/ai/images/generations
func (h *MediaHandler) GenerateImage(c *gin.Context) {
	var req media.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Validate request
	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt required"})
		return
	}

	// Execute
	resp, err := h.mediaService.GenerateImage(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GenerateVideo handles video generation requests.
// POST /api/v1/ai/videos/generations
func (h *MediaHandler) GenerateVideo(c *gin.Context) {
	var req media.VideoGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Validate request
	if req.Prompt == "" && req.InputImage == "" && req.InputVideo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt, input_image, or input_video required"})
		return
	}

	// Execute
	resp, err := h.mediaService.GenerateVideo(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// GetVideoStatus handles video status requests.
// GET /api/v1/ai/videos/:task_id
func (h *MediaHandler) GetVideoStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id required"})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get status
	resp, err := h.mediaService.GetVideoStatus(c.Request.Context(), userID, taskID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
