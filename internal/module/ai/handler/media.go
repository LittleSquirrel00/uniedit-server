package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/module/media"
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
//
//	@Summary		Generate image
//	@Description	Generate an image from a text prompt using AI models
//	@Tags			AI Media
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		media.ImageGenerationRequest	true	"Image generation request"
//	@Success		200		{object}	media.ImageGenerationResponse
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded or quota exceeded"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/images/generations [post]
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
//
//	@Summary		Generate video
//	@Description	Generate a video from text prompt, image, or video input. Returns a task for async processing.
//	@Tags			AI Media
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		media.VideoGenerationRequest	true	"Video generation request"
//	@Success		202		{object}	media.VideoGenerationResponse	"Task created for async processing"
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded or quota exceeded"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/videos/generations [post]
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
//
//	@Summary		Get video generation status
//	@Description	Get the status and result of a video generation task
//	@Tags			AI Media
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		string	true	"Task ID"
//	@Success		200		{object}	media.VideoGenerationResponse
//	@Failure		400		{object}	map[string]string	"Invalid task ID"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		404		{object}	map[string]string	"Task not found"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/videos/{task_id} [get]
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
