package media

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/media"
)

// SubmitVideoCommand represents a command to submit a video generation task.
type SubmitVideoCommand struct {
	UserID      uuid.UUID
	Prompt      string
	InputImage  string
	InputVideo  string
	Duration    int
	AspectRatio string
	Resolution  string
	FPS         int
	Model       string
}

// SubmitVideoResult is the result of video task submission.
type SubmitVideoResult struct {
	TaskID    string
	Status    string
	Progress  int
	CreatedAt int64
}

// SubmitVideoHandler handles video generation task submission.
type SubmitVideoHandler struct {
	taskRepo media.TaskRepository
}

// NewSubmitVideoHandler creates a new handler.
func NewSubmitVideoHandler(taskRepo media.TaskRepository) *SubmitVideoHandler {
	return &SubmitVideoHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the command.
func (h *SubmitVideoHandler) Handle(ctx context.Context, cmd SubmitVideoCommand) (*SubmitVideoResult, error) {
	// Validate
	if cmd.Prompt == "" && cmd.InputImage == "" && cmd.InputVideo == "" {
		return nil, media.ErrMissingInput
	}

	// Build input payload
	input := map[string]any{
		"prompt":       cmd.Prompt,
		"input_image":  cmd.InputImage,
		"input_video":  cmd.InputVideo,
		"duration":     cmd.Duration,
		"aspect_ratio": cmd.AspectRatio,
		"resolution":   cmd.Resolution,
		"fps":          cmd.FPS,
		"model":        cmd.Model,
	}

	// Create task
	task := media.NewTask(cmd.UserID, media.TaskTypeVideo, input)

	if err := h.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return &SubmitVideoResult{
		TaskID:    task.ID().String(),
		Status:    task.Status().String(),
		Progress:  task.Progress(),
		CreatedAt: task.CreatedAt().Unix(),
	}, nil
}

// CancelTaskCommand represents a command to cancel a task.
type CancelTaskCommand struct {
	UserID uuid.UUID
	TaskID uuid.UUID
}

// CancelTaskResult is the result of task cancellation.
type CancelTaskResult struct{}

// CancelTaskHandler handles task cancellation.
type CancelTaskHandler struct {
	taskRepo media.TaskRepository
}

// NewCancelTaskHandler creates a new handler.
func NewCancelTaskHandler(taskRepo media.TaskRepository) *CancelTaskHandler {
	return &CancelTaskHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the command.
func (h *CancelTaskHandler) Handle(ctx context.Context, cmd CancelTaskCommand) (*CancelTaskResult, error) {
	task, err := h.taskRepo.GetByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, err
	}

	if !task.BelongsTo(cmd.UserID) {
		return nil, media.ErrTaskNotOwned
	}

	if task.IsTerminal() {
		return nil, media.ErrTaskAlreadyDone
	}

	task.Cancel()

	if err := h.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return &CancelTaskResult{}, nil
}

// UpdateTaskProgressCommand represents a command to update task progress.
type UpdateTaskProgressCommand struct {
	TaskID   uuid.UUID
	Progress int
}

// UpdateTaskProgressResult is the result of progress update.
type UpdateTaskProgressResult struct{}

// UpdateTaskProgressHandler handles task progress updates.
type UpdateTaskProgressHandler struct {
	taskRepo media.TaskRepository
}

// NewUpdateTaskProgressHandler creates a new handler.
func NewUpdateTaskProgressHandler(taskRepo media.TaskRepository) *UpdateTaskProgressHandler {
	return &UpdateTaskProgressHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the command.
func (h *UpdateTaskProgressHandler) Handle(ctx context.Context, cmd UpdateTaskProgressCommand) (*UpdateTaskProgressResult, error) {
	task, err := h.taskRepo.GetByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, err
	}

	task.UpdateProgress(cmd.Progress)

	if err := h.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return &UpdateTaskProgressResult{}, nil
}

// CompleteTaskCommand represents a command to complete a task.
type CompleteTaskCommand struct {
	TaskID uuid.UUID
	Output map[string]any
}

// CompleteTaskResult is the result of task completion.
type CompleteTaskResult struct{}

// CompleteTaskHandler handles task completion.
type CompleteTaskHandler struct {
	taskRepo media.TaskRepository
}

// NewCompleteTaskHandler creates a new handler.
func NewCompleteTaskHandler(taskRepo media.TaskRepository) *CompleteTaskHandler {
	return &CompleteTaskHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the command.
func (h *CompleteTaskHandler) Handle(ctx context.Context, cmd CompleteTaskCommand) (*CompleteTaskResult, error) {
	task, err := h.taskRepo.GetByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, err
	}

	task.Complete(cmd.Output)

	if err := h.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return &CompleteTaskResult{}, nil
}

// FailTaskCommand represents a command to fail a task.
type FailTaskCommand struct {
	TaskID  uuid.UUID
	Code    string
	Message string
}

// FailTaskResult is the result of task failure.
type FailTaskResult struct{}

// FailTaskHandler handles task failure.
type FailTaskHandler struct {
	taskRepo media.TaskRepository
}

// NewFailTaskHandler creates a new handler.
func NewFailTaskHandler(taskRepo media.TaskRepository) *FailTaskHandler {
	return &FailTaskHandler{
		taskRepo: taskRepo,
	}
}

// Handle executes the command.
func (h *FailTaskHandler) Handle(ctx context.Context, cmd FailTaskCommand) (*FailTaskResult, error) {
	task, err := h.taskRepo.GetByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, err
	}

	task.Fail(cmd.Code, cmd.Message)

	if err := h.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return &FailTaskResult{}, nil
}
