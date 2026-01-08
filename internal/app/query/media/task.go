package media

import (
	"context"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/media"
)

// GetTaskQuery represents a query to get a task.
type GetTaskQuery struct {
	UserID uuid.UUID
	TaskID uuid.UUID
}

// TaskDTO represents a task response.
type TaskDTO struct {
	ID        string         `json:"id"`
	OwnerID   string         `json:"owner_id"`
	TaskType  string         `json:"task_type"`
	Status    string         `json:"status"`
	Progress  int            `json:"progress"`
	Input     map[string]any `json:"input,omitempty"`
	Output    map[string]any `json:"output,omitempty"`
	Error     *TaskErrorDTO  `json:"error,omitempty"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// TaskErrorDTO represents a task error.
type TaskErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetTaskHandler handles task retrieval.
type GetTaskHandler struct {
	taskRepo media.TaskRepository
}

// NewGetTaskHandler creates a new handler.
func NewGetTaskHandler(taskRepo media.TaskRepository) *GetTaskHandler {
	return &GetTaskHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the query.
func (h *GetTaskHandler) Handle(ctx context.Context, q GetTaskQuery) (*TaskDTO, error) {
	task, err := h.taskRepo.GetByID(ctx, q.TaskID)
	if err != nil {
		return nil, err
	}

	if !task.BelongsTo(q.UserID) {
		return nil, media.ErrTaskNotOwned
	}

	return taskToDTO(task), nil
}

// ListTasksQuery represents a query to list tasks.
type ListTasksQuery struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

// ListTasksHandler handles task listing.
type ListTasksHandler struct {
	taskRepo media.TaskRepository
}

// NewListTasksHandler creates a new handler.
func NewListTasksHandler(taskRepo media.TaskRepository) *ListTasksHandler {
	return &ListTasksHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the query.
func (h *ListTasksHandler) Handle(ctx context.Context, q ListTasksQuery) ([]*TaskDTO, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	tasks, err := h.taskRepo.ListByOwner(ctx, q.UserID, limit, q.Offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*TaskDTO, len(tasks))
	for i, task := range tasks {
		dtos[i] = taskToDTO(task)
	}

	return dtos, nil
}

// GetVideoStatusQuery represents a query to get video generation status.
type GetVideoStatusQuery struct {
	UserID uuid.UUID
	TaskID uuid.UUID
}

// VideoStatusDTO represents video generation status.
type VideoStatusDTO struct {
	TaskID    string          `json:"task_id"`
	Status    string          `json:"status"`
	Progress  int             `json:"progress"`
	Video     *GeneratedVideoDTO `json:"video,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
}

// GeneratedVideoDTO represents a generated video.
type GeneratedVideoDTO struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FPS      int    `json:"fps"`
	FileSize int64  `json:"file_size,omitempty"`
	Format   string `json:"format,omitempty"`
}

// GetVideoStatusHandler handles video status queries.
type GetVideoStatusHandler struct {
	taskRepo media.TaskRepository
}

// NewGetVideoStatusHandler creates a new handler.
func NewGetVideoStatusHandler(taskRepo media.TaskRepository) *GetVideoStatusHandler {
	return &GetVideoStatusHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the query.
func (h *GetVideoStatusHandler) Handle(ctx context.Context, q GetVideoStatusQuery) (*VideoStatusDTO, error) {
	task, err := h.taskRepo.GetByID(ctx, q.TaskID)
	if err != nil {
		return nil, err
	}

	if !task.BelongsTo(q.UserID) {
		return nil, media.ErrTaskNotOwned
	}

	dto := &VideoStatusDTO{
		TaskID:    task.ID().String(),
		Status:    taskStatusToVideoState(task.Status()),
		Progress:  task.Progress(),
		CreatedAt: task.CreatedAt().Unix(),
	}

	// Parse output if completed
	if task.Status() == media.TaskStatusCompleted && task.Output() != nil {
		dto.Video = parseVideoFromOutput(task.Output())
	}

	// Include error if failed
	if task.Error() != nil {
		dto.Error = task.Error().Message()
	}

	return dto, nil
}

// Helper functions

func taskToDTO(t *media.Task) *TaskDTO {
	dto := &TaskDTO{
		ID:        t.ID().String(),
		OwnerID:   t.OwnerID().String(),
		TaskType:  t.TaskType().String(),
		Status:    t.Status().String(),
		Progress:  t.Progress(),
		Input:     t.Input(),
		Output:    t.Output(),
		CreatedAt: t.CreatedAt().Unix(),
		UpdatedAt: t.UpdatedAt().Unix(),
	}

	if t.Error() != nil {
		dto.Error = &TaskErrorDTO{
			Code:    t.Error().Code(),
			Message: t.Error().Message(),
		}
	}

	return dto
}

func taskStatusToVideoState(status media.TaskStatus) string {
	switch status {
	case media.TaskStatusPending:
		return "pending"
	case media.TaskStatusRunning:
		return "processing"
	case media.TaskStatusCompleted:
		return "completed"
	case media.TaskStatusFailed, media.TaskStatusCancelled:
		return "failed"
	default:
		return "pending"
	}
}

func parseVideoFromOutput(output map[string]any) *GeneratedVideoDTO {
	if output == nil {
		return nil
	}

	video := &GeneratedVideoDTO{}

	if url, ok := output["url"].(string); ok {
		video.URL = url
	}
	if duration, ok := output["duration"].(float64); ok {
		video.Duration = int(duration)
	}
	if width, ok := output["width"].(float64); ok {
		video.Width = int(width)
	}
	if height, ok := output["height"].(float64); ok {
		video.Height = int(height)
	}
	if fps, ok := output["fps"].(float64); ok {
		video.FPS = int(fps)
	}
	if fileSize, ok := output["file_size"].(float64); ok {
		video.FileSize = int64(fileSize)
	}
	if format, ok := output["format"].(string); ok {
		video.Format = format
	}

	if video.URL == "" {
		return nil
	}

	return video
}
