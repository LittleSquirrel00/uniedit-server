package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sharedtask "github.com/uniedit/server/internal/shared/task"
)

// TaskHandler handles task API requests.
type TaskHandler struct {
	taskManager *sharedtask.Manager
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(taskManager *sharedtask.Manager) *TaskHandler {
	return &TaskHandler{
		taskManager: taskManager,
	}
}

// RegisterRoutes registers task routes.
func (h *TaskHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/tasks", h.List)
	r.GET("/tasks/:id", h.Get)
	r.DELETE("/tasks/:id", h.Cancel)
}

// TaskResponse represents a task API response.
type TaskResponse struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Progress    int               `json:"progress"`
	Error       *sharedtask.Error `json:"error,omitempty"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
	CompletedAt *int64            `json:"completed_at,omitempty"`
}

// List handles task list requests.
//
//	@Summary		List AI tasks
//	@Description	List all AI tasks for the current user with optional filtering
//	@Tags			AI Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"Filter by status (pending, processing, completed, failed, cancelled)"
//	@Param			type	query		string	false	"Filter by task type (video_generation, image_generation)"
//	@Success		200		{object}	map[string]interface{}	"List of tasks"
//	@Failure		401		{object}	map[string]string		"Unauthorized"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/ai/tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse query parameters
	filter := &sharedtask.Filter{}
	if status := c.Query("status"); status != "" {
		s := sharedtask.Status(status)
		filter.Status = &s
	}
	if taskType := c.Query("type"); taskType != "" {
		filter.Type = &taskType
	}

	// List tasks
	tasks, err := h.taskManager.List(c.Request.Context(), userID, filter)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response
	responses := make([]*TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = taskToResponse(t)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   responses,
	})
}

// Get handles task get requests.
//
//	@Summary		Get AI task
//	@Description	Get details of a specific AI task by ID
//	@Tags			AI Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Task ID (UUID)"
//	@Success		200	{object}	TaskResponse
//	@Failure		400	{object}	map[string]string	"Invalid task ID"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		404	{object}	map[string]string	"Task not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/ai/tasks/{id} [get]
func (h *TaskHandler) Get(c *gin.Context) {
	// Parse task ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get task
	t, err := h.taskManager.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check ownership
	if t.OwnerID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, taskToResponse(t))
}

// Cancel handles task cancel requests.
//
//	@Summary		Cancel AI task
//	@Description	Cancel a pending or processing AI task
//	@Tags			AI Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Task ID (UUID)"
//	@Success		200	{object}	map[string]string	"Task cancelled successfully"
//	@Failure		400	{object}	map[string]string	"Invalid task ID or task cannot be cancelled"
//	@Failure		401	{object}	map[string]string	"Unauthorized"
//	@Failure		404	{object}	map[string]string	"Task not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/ai/tasks/{id} [delete]
func (h *TaskHandler) Cancel(c *gin.Context) {
	// Parse task ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get task first to check ownership
	t, err := h.taskManager.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check ownership
	if t.OwnerID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// Cancel task
	if err := h.taskManager.Cancel(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

// taskToResponse converts a task to a response.
func taskToResponse(t *sharedtask.Task) *TaskResponse {
	resp := &TaskResponse{
		ID:        t.ID.String(),
		Type:      t.Type,
		Status:    string(t.Status),
		Progress:  t.Progress,
		Error:     t.Error,
		CreatedAt: t.CreatedAt.Unix(),
		UpdatedAt: t.UpdatedAt.Unix(),
	}

	if t.CompletedAt != nil {
		ts := t.CompletedAt.Unix()
		resp.CompletedAt = &ts
	}

	return resp
}
