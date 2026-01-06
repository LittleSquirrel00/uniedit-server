package media

// ImageRequest represents an image generation request.
type ImageRequest struct {
	Prompt         string      `json:"prompt"`
	NegativePrompt string      `json:"negative_prompt,omitempty"`
	N              int         `json:"n,omitempty"`
	Size           string      `json:"size,omitempty"`
	Quality        string      `json:"quality,omitempty"`
	Style          string      `json:"style,omitempty"`
	ResponseFormat string      `json:"response_format,omitempty"` // url or b64_json
	Model          string      `json:"model,omitempty"`
	Extra          interface{} `json:"extra,omitempty"`
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

// VideoRequest represents a video generation request.
type VideoRequest struct {
	Prompt      string      `json:"prompt,omitempty"`
	InputImage  string      `json:"input_image,omitempty"`  // URL or base64
	InputVideo  string      `json:"input_video,omitempty"`  // URL or base64
	Duration    int         `json:"duration,omitempty"`     // seconds
	AspectRatio string      `json:"aspect_ratio,omitempty"` // 16:9, 9:16, 1:1
	Resolution  string      `json:"resolution,omitempty"`   // 720p, 1080p
	FPS         int         `json:"fps,omitempty"`
	Model       string      `json:"model,omitempty"`
	Seed        *int64      `json:"seed,omitempty"`
	Extra       interface{} `json:"extra,omitempty"`
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

// VideoState represents the state of a video generation task.
type VideoState string

const (
	VideoStatePending    VideoState = "pending"
	VideoStateProcessing VideoState = "processing"
	VideoStateCompleted  VideoState = "completed"
	VideoStateFailed     VideoState = "failed"
)

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

// ImageGenerationRequest represents an image generation API request.
type ImageGenerationRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	Model          string `json:"model,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// ImageGenerationResponse represents an image generation API response.
type ImageGenerationResponse struct {
	*ImageResponse
	TaskID string `json:"task_id,omitempty"`
}

// VideoGenerationRequest represents a video generation API request.
type VideoGenerationRequest struct {
	Prompt      string `json:"prompt,omitempty"`
	InputImage  string `json:"input_image,omitempty"`
	InputVideo  string `json:"input_video,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	FPS         int    `json:"fps,omitempty"`
	Model       string `json:"model,omitempty"`
	Async       bool   `json:"async,omitempty"`
}

// VideoGenerationResponse represents a video generation API response.
type VideoGenerationResponse struct {
	TaskID    string          `json:"task_id"`
	Status    VideoState      `json:"status"`
	Progress  int             `json:"progress"`
	Video     *GeneratedVideo `json:"video,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
}
