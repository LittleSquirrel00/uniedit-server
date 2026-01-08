package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	mediaCmd "github.com/uniedit/server/internal/app/command/media"
	mediaQuery "github.com/uniedit/server/internal/app/query/media"
)

// MediaHandler handles media HTTP requests.
type MediaHandler struct {
	generateImage  *mediaCmd.GenerateImageHandler
	submitVideo    *mediaCmd.SubmitVideoHandler
	cancelTask     *mediaCmd.CancelTaskHandler
	getTask        *mediaQuery.GetTaskHandler
	listTasks      *mediaQuery.ListTasksHandler
	getVideoStatus *mediaQuery.GetVideoStatusHandler
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(
	generateImage *mediaCmd.GenerateImageHandler,
	submitVideo *mediaCmd.SubmitVideoHandler,
	cancelTask *mediaCmd.CancelTaskHandler,
	getTask *mediaQuery.GetTaskHandler,
	listTasks *mediaQuery.ListTasksHandler,
	getVideoStatus *mediaQuery.GetVideoStatusHandler,
) *MediaHandler {
	return &MediaHandler{
		generateImage:  generateImage,
		submitVideo:    submitVideo,
		cancelTask:     cancelTask,
		getTask:        getTask,
		listTasks:      listTasks,
		getVideoStatus: getVideoStatus,
	}
}

// RegisterRoutes registers media routes.
func (h *MediaHandler) RegisterRoutes(r *gin.RouterGroup) {
	media := r.Group("/media")
	{
		// Image generation
		media.POST("/images/generations", h.GenerateImage)

		// Video generation
		media.POST("/videos/generations", h.GenerateVideo)
		media.GET("/videos/generations/:task_id", h.GetVideoStatus)

		// Tasks
		media.GET("/tasks", h.ListTasks)
		media.GET("/tasks/:task_id", h.GetTask)
		media.POST("/tasks/:task_id/cancel", h.CancelTask)
	}
}

// GenerateImage handles image generation requests.
// @Summary Generate images
// @Tags Media
// @Accept json
// @Produce json
// @Param request body GenerateImageRequest true "Image generation request"
// @Success 200 {object} mediaCmd.GenerateImageResult
// @Router /api/v1/media/images/generations [post]
func (h *MediaHandler) GenerateImage(c *gin.Context) {
	userID := getUserID(c)

	var req GenerateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.generateImage.Handle(c.Request.Context(), mediaCmd.GenerateImageCommand{
		UserID:         userID,
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		Model:          req.Model,
		ResponseFormat: req.ResponseFormat,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateVideo handles video generation requests.
// @Summary Generate videos
// @Tags Media
// @Accept json
// @Produce json
// @Param request body GenerateVideoRequest true "Video generation request"
// @Success 200 {object} mediaCmd.SubmitVideoResult
// @Router /api/v1/media/videos/generations [post]
func (h *MediaHandler) GenerateVideo(c *gin.Context) {
	userID := getUserID(c)

	var req GenerateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.submitVideo.Handle(c.Request.Context(), mediaCmd.SubmitVideoCommand{
		UserID:      userID,
		Prompt:      req.Prompt,
		InputImage:  req.InputImage,
		InputVideo:  req.InputVideo,
		Duration:    req.Duration,
		AspectRatio: req.AspectRatio,
		Resolution:  req.Resolution,
		FPS:         req.FPS,
		Model:       req.Model,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetVideoStatus handles video status requests.
// @Summary Get video generation status
// @Tags Media
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} mediaQuery.VideoStatusDTO
// @Router /api/v1/media/videos/generations/{task_id} [get]
func (h *MediaHandler) GetVideoStatus(c *gin.Context) {
	userID := getUserID(c)

	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	result, err := h.getVideoStatus.Handle(c.Request.Context(), mediaQuery.GetVideoStatusQuery{
		UserID: userID,
		TaskID: taskID,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListTasks handles task listing requests.
// @Summary List media tasks
// @Tags Media
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} mediaQuery.TaskDTO
// @Router /api/v1/media/tasks [get]
func (h *MediaHandler) ListTasks(c *gin.Context) {
	userID := getUserID(c)

	var req ListTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.listTasks.Handle(c.Request.Context(), mediaQuery.ListTasksQuery{
		UserID: userID,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": result})
}

// GetTask handles task retrieval requests.
// @Summary Get media task
// @Tags Media
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} mediaQuery.TaskDTO
// @Router /api/v1/media/tasks/{task_id} [get]
func (h *MediaHandler) GetTask(c *gin.Context) {
	userID := getUserID(c)

	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	result, err := h.getTask.Handle(c.Request.Context(), mediaQuery.GetTaskQuery{
		UserID: userID,
		TaskID: taskID,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CancelTask handles task cancellation requests.
// @Summary Cancel media task
// @Tags Media
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/media/tasks/{task_id}/cancel [post]
func (h *MediaHandler) CancelTask(c *gin.Context) {
	userID := getUserID(c)

	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	_, err = h.cancelTask.Handle(c.Request.Context(), mediaCmd.CancelTaskCommand{
		UserID: userID,
		TaskID: taskID,
	})
	if err != nil {
		handleMediaError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

// Request DTOs

// GenerateImageRequest represents an image generation request.
type GenerateImageRequest struct {
	Prompt         string `json:"prompt" binding:"required"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	Model          string `json:"model,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// GenerateVideoRequest represents a video generation request.
type GenerateVideoRequest struct {
	Prompt      string `json:"prompt,omitempty"`
	InputImage  string `json:"input_image,omitempty"`
	InputVideo  string `json:"input_video,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	FPS         int    `json:"fps,omitempty"`
	Model       string `json:"model,omitempty"`
}

// ListTasksRequest represents a task list request.
type ListTasksRequest struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
}

// Helper functions

func handleMediaError(c *gin.Context, err error) {
	// Map domain errors to HTTP status codes
	switch err.Error() {
	case "task not found", "model not found", "provider not found":
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case "task not owned by user":
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case "prompt is required", "prompt, input_image, or input_video is required",
		"invalid request", "task already completed or cancelled":
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case "provider is unhealthy", "no healthy provider available":
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
