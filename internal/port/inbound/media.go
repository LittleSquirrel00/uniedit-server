package inbound

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// --- Request/Response Types ---

// MediaImageGenerationInput represents an image generation request.
type MediaImageGenerationInput struct {
	Prompt         string `json:"prompt" binding:"required"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	Model          string `json:"model,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// MediaImageGenerationOutput represents an image generation response.
type MediaImageGenerationOutput struct {
	Images    []*model.GeneratedImage `json:"images"`
	Model     string                  `json:"model"`
	Usage     *model.ImageUsage       `json:"usage,omitempty"`
	CreatedAt int64                   `json:"created_at"`
	TaskID    string                  `json:"task_id,omitempty"`
}

// MediaVideoGenerationInput represents a video generation request.
type MediaVideoGenerationInput struct {
	Prompt      string `json:"prompt,omitempty"`
	InputImage  string `json:"input_image,omitempty"`
	InputVideo  string `json:"input_video,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	FPS         int    `json:"fps,omitempty"`
	Model       string `json:"model,omitempty"`
}

// MediaVideoGenerationOutput represents a video generation response.
type MediaVideoGenerationOutput struct {
	TaskID    string                `json:"task_id"`
	Status    model.VideoState      `json:"status"`
	Progress  int                   `json:"progress"`
	Video     *model.GeneratedVideo `json:"video,omitempty"`
	Error     string                `json:"error,omitempty"`
	CreatedAt int64                 `json:"created_at"`
}

// MediaTaskOutput represents a task response.
type MediaTaskOutput struct {
	ID        uuid.UUID             `json:"id"`
	OwnerID   uuid.UUID             `json:"owner_id"`
	Type      string                `json:"type"`
	Status    model.MediaTaskStatus `json:"status"`
	Progress  int                   `json:"progress"`
	Error     string                `json:"error,omitempty"`
	CreatedAt int64                 `json:"created_at"`
	UpdatedAt int64                 `json:"updated_at"`
}

// --- Domain Interface ---

// MediaDomain defines the media domain service interface.
type MediaDomain interface {
	// GenerateImage generates images synchronously.
	GenerateImage(ctx context.Context, userID uuid.UUID, input *MediaImageGenerationInput) (*MediaImageGenerationOutput, error)

	// GenerateVideo generates videos (async via task).
	GenerateVideo(ctx context.Context, userID uuid.UUID, input *MediaVideoGenerationInput) (*MediaVideoGenerationOutput, error)

	// GetVideoStatus returns the status of a video generation task.
	GetVideoStatus(ctx context.Context, userID uuid.UUID, taskID string) (*MediaVideoGenerationOutput, error)

	// GetTask returns a task by ID.
	GetTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*MediaTaskOutput, error)

	// ListTasks lists tasks for a user.
	ListTasks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*MediaTaskOutput, error)

	// CancelTask cancels a task.
	CancelTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error

	// --- Provider management (admin) ---

	// GetProvider returns a provider by ID.
	GetProvider(ctx context.Context, id uuid.UUID) (*model.MediaProvider, error)

	// ListProviders lists all providers.
	ListProviders(ctx context.Context) ([]*model.MediaProvider, error)

	// --- Model management (admin) ---

	// GetModel returns a model by ID.
	GetModel(ctx context.Context, id string) (*model.MediaModel, error)

	// ListModelsByCapability lists models with a capability.
	ListModelsByCapability(ctx context.Context, cap model.MediaCapability) ([]*model.MediaModel, error)
}

// --- HTTP Port Interfaces ---

// MediaHttpPort defines media HTTP handlers.
type MediaHttpPort interface {
	// GenerateImage handles image generation requests.
	GenerateImage(c *gin.Context)

	// GenerateVideo handles video generation requests.
	GenerateVideo(c *gin.Context)

	// GetVideoStatus handles video status requests.
	GetVideoStatus(c *gin.Context)

	// GetTask handles task retrieval requests.
	GetTask(c *gin.Context)

	// ListTasks handles task listing requests.
	ListTasks(c *gin.Context)

	// CancelTask handles task cancellation requests.
	CancelTask(c *gin.Context)
}

// MediaAdminHttpPort defines media admin HTTP handlers.
type MediaAdminHttpPort interface {
	// ListProviders lists all providers.
	ListProviders(c *gin.Context)

	// GetProvider gets a provider by ID.
	GetProvider(c *gin.Context)

	// ListModels lists all models.
	ListModels(c *gin.Context)

	// GetModelsByCapability gets models by capability.
	GetModelsByCapability(c *gin.Context)
}
