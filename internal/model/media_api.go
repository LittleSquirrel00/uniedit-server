package model

import "github.com/google/uuid"

// MediaImageGenerationInput represents an image generation request.
type MediaImageGenerationInput struct {
	Prompt         string `json:"prompt"`
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
	Images    []*GeneratedImage `json:"images"`
	Model     string            `json:"model"`
	Usage     *ImageUsage       `json:"usage,omitempty"`
	CreatedAt int64             `json:"created_at"`
	TaskID    string            `json:"task_id,omitempty"`
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
	TaskID    string          `json:"task_id"`
	Status    VideoState      `json:"status"`
	Progress  int             `json:"progress"`
	Video     *GeneratedVideo `json:"video,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
}

// MediaTaskOutput represents a task response.
type MediaTaskOutput struct {
	ID        uuid.UUID       `json:"id"`
	OwnerID   uuid.UUID       `json:"owner_id"`
	Type      string          `json:"type"`
	Status    MediaTaskStatus `json:"status"`
	Progress  int             `json:"progress"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}
