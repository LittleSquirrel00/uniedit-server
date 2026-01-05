package task

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the status of an AI task.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Type represents the type of AI task.
type Type string

const (
	TypeChat            Type = "chat"
	TypeImageGeneration Type = "image_generation"
	TypeVideoGeneration Type = "video_generation"
	TypeAudioGeneration Type = "audio_generation"
	TypeEmbedding       Type = "embedding"
)

// Error represents a task error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Task represents an AI task.
type Task struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Type           Type           `json:"type" gorm:"not null"`
	Status         Status         `json:"status" gorm:"not null"`
	Progress       int            `json:"progress" gorm:"default:0"`
	Input          map[string]any `json:"input" gorm:"type:jsonb;serializer:json;not null"`
	Output         map[string]any `json:"output,omitempty" gorm:"type:jsonb;serializer:json"`
	Error          *Error         `json:"error,omitempty" gorm:"type:jsonb;serializer:json"`
	ExternalTaskID string         `json:"external_task_id,omitempty" gorm:"column:external_task_id"`
	ProviderID     *uuid.UUID     `json:"provider_id,omitempty" gorm:"type:uuid"`
	ModelID        string         `json:"model_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

// TableName returns the table name for Task.
func (Task) TableName() string {
	return "ai_tasks"
}

// IsTerminal checks if the task is in a terminal state.
func (t *Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
}

// IsPending checks if the task is pending.
func (t *Task) IsPending() bool {
	return t.Status == StatusPending
}

// IsRunning checks if the task is running.
func (t *Task) IsRunning() bool {
	return t.Status == StatusRunning
}

// Filter represents task filter options.
type Filter struct {
	UserID   *uuid.UUID
	Type     *Type
	Status   *Status
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string
}

// Input represents task input configuration.
type Input struct {
	Type     Type           `json:"type"`
	Payload  map[string]any `json:"payload"`
	Priority int            `json:"priority,omitempty"`
	Timeout  time.Duration  `json:"timeout,omitempty"`
	Retry    *RetryConfig   `json:"retry,omitempty"`
}

// RetryConfig represents retry configuration.
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
}
