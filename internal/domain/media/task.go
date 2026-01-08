package media

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a media generation task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// String returns the string representation of the task status.
func (s TaskStatus) String() string {
	return string(s)
}

// IsTerminal returns whether the status is terminal.
func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// TaskType represents the type of media generation task.
type TaskType string

const (
	TaskTypeImage TaskType = "image_generation"
	TaskTypeVideo TaskType = "video_generation"
	TaskTypeAudio TaskType = "audio_generation"
)

// String returns the string representation of the task type.
func (t TaskType) String() string {
	return string(t)
}

// TaskError represents a task error.
type TaskError struct {
	code    string
	message string
}

// NewTaskError creates a new task error.
func NewTaskError(code, message string) *TaskError {
	return &TaskError{
		code:    code,
		message: message,
	}
}

// Code returns the error code.
func (e *TaskError) Code() string { return e.code }

// Message returns the error message.
func (e *TaskError) Message() string { return e.message }

// Task represents a media generation task.
type Task struct {
	id        uuid.UUID
	ownerID   uuid.UUID
	taskType  TaskType
	status    TaskStatus
	progress  int
	input     map[string]any
	output    map[string]any
	err       *TaskError
	createdAt time.Time
	updatedAt time.Time
}

// NewTask creates a new media generation task.
func NewTask(ownerID uuid.UUID, taskType TaskType, input map[string]any) *Task {
	now := time.Now()
	return &Task{
		id:        uuid.New(),
		ownerID:   ownerID,
		taskType:  taskType,
		status:    TaskStatusPending,
		progress:  0,
		input:     input,
		createdAt: now,
		updatedAt: now,
	}
}

// ReconstructTask reconstructs a task from persistence.
func ReconstructTask(
	id uuid.UUID,
	ownerID uuid.UUID,
	taskType TaskType,
	status TaskStatus,
	progress int,
	input map[string]any,
	output map[string]any,
	err *TaskError,
	createdAt time.Time,
	updatedAt time.Time,
) *Task {
	return &Task{
		id:        id,
		ownerID:   ownerID,
		taskType:  taskType,
		status:    status,
		progress:  progress,
		input:     input,
		output:    output,
		err:       err,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ID returns the task ID.
func (t *Task) ID() uuid.UUID { return t.id }

// OwnerID returns the owner ID.
func (t *Task) OwnerID() uuid.UUID { return t.ownerID }

// TaskType returns the task type.
func (t *Task) TaskType() TaskType { return t.taskType }

// Status returns the task status.
func (t *Task) Status() TaskStatus { return t.status }

// Progress returns the progress (0-100).
func (t *Task) Progress() int { return t.progress }

// Input returns the input data.
func (t *Task) Input() map[string]any { return t.input }

// Output returns the output data.
func (t *Task) Output() map[string]any { return t.output }

// Error returns the task error if any.
func (t *Task) Error() *TaskError { return t.err }

// CreatedAt returns the creation time.
func (t *Task) CreatedAt() time.Time { return t.createdAt }

// UpdatedAt returns the update time.
func (t *Task) UpdatedAt() time.Time { return t.updatedAt }

// BelongsTo checks if the task belongs to the given user.
func (t *Task) BelongsTo(userID uuid.UUID) bool {
	return t.ownerID == userID
}

// Start starts the task.
func (t *Task) Start() {
	t.status = TaskStatusRunning
	t.updatedAt = time.Now()
}

// UpdateProgress updates the task progress.
func (t *Task) UpdateProgress(progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	t.progress = progress
	t.updatedAt = time.Now()
}

// Complete completes the task with output.
func (t *Task) Complete(output map[string]any) {
	t.status = TaskStatusCompleted
	t.progress = 100
	t.output = output
	t.updatedAt = time.Now()
}

// Fail marks the task as failed.
func (t *Task) Fail(code, message string) {
	t.status = TaskStatusFailed
	t.err = NewTaskError(code, message)
	t.updatedAt = time.Now()
}

// Cancel cancels the task.
func (t *Task) Cancel() {
	t.status = TaskStatusCancelled
	t.updatedAt = time.Now()
}

// IsTerminal returns whether the task has reached a terminal state.
func (t *Task) IsTerminal() bool {
	return t.status.IsTerminal()
}
