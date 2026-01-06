// Package task provides a generic task management infrastructure for async operations.
// It supports task submission, execution, progress tracking, and external task polling.
package task

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the status of a task.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Error represents a task error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Task represents a generic async task.
type Task struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID        uuid.UUID      `json:"owner_id" gorm:"type:uuid;not null;index"`
	Type           string         `json:"type" gorm:"not null;index"`
	Status         Status         `json:"status" gorm:"not null;index"`
	Progress       int            `json:"progress" gorm:"default:0"`
	Input          map[string]any `json:"input" gorm:"type:jsonb;serializer:json;not null"`
	Output         map[string]any `json:"output,omitempty" gorm:"type:jsonb;serializer:json"`
	Error          *Error         `json:"error,omitempty" gorm:"type:jsonb;serializer:json"`
	ExternalTaskID string         `json:"external_task_id,omitempty" gorm:"column:external_task_id;index"`
	Metadata       map[string]any `json:"metadata,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

// TableName returns the table name for Task.
func (Task) TableName() string {
	return "tasks"
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
	OwnerID  *uuid.UUID
	Type     *string
	Status   *Status
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string
}

// SubmitRequest represents a task submission request.
type SubmitRequest struct {
	Type     string         `json:"type"`
	Payload  map[string]any `json:"payload"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Priority int            `json:"priority,omitempty"`
	Timeout  time.Duration  `json:"timeout,omitempty"`
}

// ExternalSubmitRequest represents a task submission for external async operations.
type ExternalSubmitRequest struct {
	SubmitRequest
	ExternalTaskID string `json:"external_task_id"`
}
