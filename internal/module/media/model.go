// Package media provides media generation services (images, videos, audio).
// This module is independent from the AI/LLM module.
package media

import (
	"context"

	"github.com/google/uuid"
)

// ProviderType represents the type of media provider.
type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGeneric   ProviderType = "generic"
)

// Capability represents a media generation capability.
type Capability string

const (
	CapabilityImage Capability = "image"
	CapabilityVideo Capability = "video"
	CapabilityAudio Capability = "audio"
)

// Provider represents a media provider configuration.
type Provider struct {
	ID      uuid.UUID    `json:"id"`
	Name    string       `json:"name"`
	Type    ProviderType `json:"type"`
	BaseURL string       `json:"base_url"`
	APIKey  string       `json:"-"` // Never expose in JSON
	Enabled bool         `json:"enabled"`
}

// Model represents a media generation model.
type Model struct {
	ID           string       `json:"id"`
	ProviderID   uuid.UUID    `json:"provider_id"`
	Name         string       `json:"name"`
	Capabilities []Capability `json:"capabilities"`
	Enabled      bool         `json:"enabled"`
}

// HasCapability checks if the model has a specific capability.
func (m *Model) HasCapability(cap Capability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Adapter defines the interface for media generation adapters.
type Adapter interface {
	// Type returns the provider type this adapter supports.
	Type() ProviderType

	// GenerateImage generates images from a text prompt.
	GenerateImage(ctx context.Context, req *ImageRequest, model *Model, prov *Provider) (*ImageResponse, error)

	// GenerateVideo generates videos from input (text, image, or video).
	GenerateVideo(ctx context.Context, req *VideoRequest, model *Model, prov *Provider) (*VideoResponse, error)

	// GetVideoStatus checks the status of a video generation task.
	GetVideoStatus(ctx context.Context, taskID string, prov *Provider) (*VideoStatus, error)

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap Capability) bool

	// HealthCheck performs a health check on the provider.
	HealthCheck(ctx context.Context, prov *Provider) error
}

// ProviderRegistry provides access to media providers and models.
type ProviderRegistry interface {
	GetProvider(id uuid.UUID) (*Provider, bool)
	GetModelWithProvider(modelID string) (*Model, *Provider, bool)
	GetModelsByCapability(cap Capability) []*Model
}

// HealthChecker checks provider health status.
type HealthChecker interface {
	IsHealthy(providerID uuid.UUID) bool
}

// TaskManager manages async media generation tasks.
type TaskManager interface {
	Submit(ctx context.Context, ownerID uuid.UUID, req *TaskSubmitRequest) (*Task, error)
	Get(ctx context.Context, id uuid.UUID) (*Task, error)
}

// TaskSubmitRequest represents a task submission request.
type TaskSubmitRequest struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents an async media generation task.
type Task struct {
	ID        uuid.UUID      `json:"id"`
	OwnerID   uuid.UUID      `json:"owner_id"`
	Type      string         `json:"type"`
	Status    TaskStatus     `json:"status"`
	Progress  int            `json:"progress"`
	Input     map[string]any `json:"input"`
	Output    map[string]any `json:"output,omitempty"`
	Error     *TaskError     `json:"error,omitempty"`
	CreatedAt int64          `json:"created_at"`
}

// TaskError represents a task error.
type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
