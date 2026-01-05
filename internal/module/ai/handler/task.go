package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/task"
)

// TaskHandler handles task API requests.
type TaskHandler struct {
	taskManager *task.Manager
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(taskManager *task.Manager) *TaskHandler {
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
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	Progress    int         `json:"progress"`
	Error       *task.Error `json:"error,omitempty"`
	CreatedAt   int64       `json:"created_at"`
	UpdatedAt   int64       `json:"updated_at"`
	CompletedAt *int64      `json:"completed_at,omitempty"`
}

// List handles task list requests.
// GET /api/v1/ai/tasks
func (h *TaskHandler) List(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse query parameters
	filter := &task.Filter{}
	if status := c.Query("status"); status != "" {
		s := task.Status(status)
		filter.Status = &s
	}
	if taskType := c.Query("type"); taskType != "" {
		t := task.Type(taskType)
		filter.Type = &t
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
// GET /api/v1/ai/tasks/:id
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
	if t.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, taskToResponse(t))
}

// Cancel handles task cancel requests.
// DELETE /api/v1/ai/tasks/:id
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
	if t.UserID != userID {
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
func taskToResponse(t *task.Task) *TaskResponse {
	resp := &TaskResponse{
		ID:        t.ID.String(),
		Type:      string(t.Type),
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
