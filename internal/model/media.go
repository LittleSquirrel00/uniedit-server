package model

import (
	"time"

	"github.com/google/uuid"
)

// MediaProviderType represents the type of media provider.
type MediaProviderType string

const (
	MediaProviderTypeOpenAI    MediaProviderType = "openai"
	MediaProviderTypeAnthropic MediaProviderType = "anthropic"
	MediaProviderTypeGeneric   MediaProviderType = "generic"
)

// MediaCapability represents a media generation capability.
type MediaCapability string

const (
	MediaCapabilityImage MediaCapability = "image"
	MediaCapabilityVideo MediaCapability = "video"
	MediaCapabilityAudio MediaCapability = "audio"
)

// MediaProvider represents a media provider configuration.
type MediaProvider struct {
	ID           uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string            `json:"name"`
	Type         MediaProviderType `json:"type"`
	BaseURL      string            `json:"base_url"`
	EncryptedKey string            `json:"-"` // Never expose in JSON
	Enabled      bool              `json:"enabled"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// TableName returns the table name.
func (MediaProvider) TableName() string {
	return "media_providers"
}

// MediaModel represents a media generation model.
type MediaModel struct {
	ID           string            `json:"id" gorm:"primaryKey"`
	ProviderID   uuid.UUID         `json:"provider_id"`
	Name         string            `json:"name"`
	Capabilities []MediaCapability `json:"capabilities" gorm:"type:jsonb;serializer:json"`
	Enabled      bool              `json:"enabled"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// TableName returns the table name.
func (MediaModel) TableName() string {
	return "media_models"
}

// HasCapability checks if the model has a specific capability.
func (m *MediaModel) HasCapability(cap MediaCapability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// MediaTaskStatus represents the status of a media task.
type MediaTaskStatus string

const (
	MediaTaskStatusPending   MediaTaskStatus = "pending"
	MediaTaskStatusRunning   MediaTaskStatus = "running"
	MediaTaskStatusCompleted MediaTaskStatus = "completed"
	MediaTaskStatusFailed    MediaTaskStatus = "failed"
	MediaTaskStatusCancelled MediaTaskStatus = "cancelled"
)

// MediaTask represents an async media generation task.
type MediaTask struct {
	ID        uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	OwnerID   uuid.UUID       `json:"owner_id" gorm:"type:uuid;index"`
	Type      string          `json:"type"` // image_generation, video_generation
	Status    MediaTaskStatus `json:"status"`
	Progress  int             `json:"progress"`
	Input     string          `json:"input" gorm:"type:jsonb"`    // JSON serialized
	Output    string          `json:"output" gorm:"type:jsonb"`   // JSON serialized
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TableName returns the table name.
func (MediaTask) TableName() string {
	return "media_tasks"
}

// ImageRequest represents an image generation request.
type ImageRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"` // url or b64_json
	Model          string `json:"model,omitempty"`
}

// ImageResponse represents an image generation response.
type ImageResponse struct {
	Images    []*GeneratedImage `json:"images"`
	Model     string            `json:"model"`
	Usage     *ImageUsage       `json:"usage,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// GeneratedImage represents a single generated image.
type GeneratedImage struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageUsage represents token/credit usage for image generation.
type ImageUsage struct {
	TotalImages int     `json:"total_images"`
	CostUSD     float64 `json:"cost_usd,omitempty"`
}

// VideoState represents the state of a video generation task.
type VideoState string

const (
	VideoStatePending    VideoState = "pending"
	VideoStateProcessing VideoState = "processing"
	VideoStateCompleted  VideoState = "completed"
	VideoStateFailed     VideoState = "failed"
)

// VideoRequest represents a video generation request.
type VideoRequest struct {
	Prompt      string `json:"prompt,omitempty"`
	InputImage  string `json:"input_image,omitempty"`  // URL or base64
	InputVideo  string `json:"input_video,omitempty"`  // URL or base64
	Duration    int    `json:"duration,omitempty"`     // seconds
	AspectRatio string `json:"aspect_ratio,omitempty"` // 16:9, 9:16, 1:1
	Resolution  string `json:"resolution,omitempty"`   // 720p, 1080p
	FPS         int    `json:"fps,omitempty"`
	Model       string `json:"model,omitempty"`
	Seed        *int64 `json:"seed,omitempty"`
}

// VideoResponse represents a video generation response.
type VideoResponse struct {
	TaskID    string          `json:"task_id"`
	Status    VideoState      `json:"status"`
	Video     *GeneratedVideo `json:"video,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
}

// VideoStatus represents the status of a video generation task.
type VideoStatus struct {
	TaskID    string          `json:"task_id"`
	Status    VideoState      `json:"status"`
	Progress  int             `json:"progress"`
	Video     *GeneratedVideo `json:"video,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// GeneratedVideo represents a single generated video.
type GeneratedVideo struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FPS      int    `json:"fps"`
	FileSize int64  `json:"file_size,omitempty"`
	Format   string `json:"format,omitempty"`
}

// VideoUsage represents usage for video generation.
type VideoUsage struct {
	DurationSeconds int     `json:"duration_seconds"`
	CostUSD         float64 `json:"cost_usd,omitempty"`
}
