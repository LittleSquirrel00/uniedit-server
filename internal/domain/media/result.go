package media

// GeneratedImage represents a single generated image.
type GeneratedImage struct {
	url           string
	b64JSON       string
	revisedPrompt string
}

// NewGeneratedImage creates a new generated image.
func NewGeneratedImage(url, b64JSON, revisedPrompt string) *GeneratedImage {
	return &GeneratedImage{
		url:           url,
		b64JSON:       b64JSON,
		revisedPrompt: revisedPrompt,
	}
}

// URL returns the image URL.
func (i *GeneratedImage) URL() string { return i.url }

// B64JSON returns the base64-encoded image.
func (i *GeneratedImage) B64JSON() string { return i.b64JSON }

// RevisedPrompt returns the revised prompt.
func (i *GeneratedImage) RevisedPrompt() string { return i.revisedPrompt }

// ImageUsage represents usage for image generation.
type ImageUsage struct {
	totalImages int
	costUSD     float64
}

// NewImageUsage creates a new image usage.
func NewImageUsage(totalImages int, costUSD float64) *ImageUsage {
	return &ImageUsage{
		totalImages: totalImages,
		costUSD:     costUSD,
	}
}

// TotalImages returns the total number of images.
func (u *ImageUsage) TotalImages() int { return u.totalImages }

// CostUSD returns the cost in USD.
func (u *ImageUsage) CostUSD() float64 { return u.costUSD }

// ImageResult represents the result of image generation.
type ImageResult struct {
	images    []*GeneratedImage
	model     string
	usage     *ImageUsage
	createdAt int64
}

// NewImageResult creates a new image result.
func NewImageResult(images []*GeneratedImage, model string, usage *ImageUsage, createdAt int64) *ImageResult {
	return &ImageResult{
		images:    images,
		model:     model,
		usage:     usage,
		createdAt: createdAt,
	}
}

// Images returns the generated images.
func (r *ImageResult) Images() []*GeneratedImage { return r.images }

// Model returns the model used.
func (r *ImageResult) Model() string { return r.model }

// Usage returns the usage information.
func (r *ImageResult) Usage() *ImageUsage { return r.usage }

// CreatedAt returns the creation timestamp.
func (r *ImageResult) CreatedAt() int64 { return r.createdAt }

// GeneratedVideo represents a single generated video.
type GeneratedVideo struct {
	url      string
	duration int
	width    int
	height   int
	fps      int
	fileSize int64
	format   string
}

// NewGeneratedVideo creates a new generated video.
func NewGeneratedVideo(url string, duration, width, height, fps int, fileSize int64, format string) *GeneratedVideo {
	return &GeneratedVideo{
		url:      url,
		duration: duration,
		width:    width,
		height:   height,
		fps:      fps,
		fileSize: fileSize,
		format:   format,
	}
}

// URL returns the video URL.
func (v *GeneratedVideo) URL() string { return v.url }

// Duration returns the duration in seconds.
func (v *GeneratedVideo) Duration() int { return v.duration }

// Width returns the video width.
func (v *GeneratedVideo) Width() int { return v.width }

// Height returns the video height.
func (v *GeneratedVideo) Height() int { return v.height }

// FPS returns the frames per second.
func (v *GeneratedVideo) FPS() int { return v.fps }

// FileSize returns the file size in bytes.
func (v *GeneratedVideo) FileSize() int64 { return v.fileSize }

// Format returns the video format.
func (v *GeneratedVideo) Format() string { return v.format }

// VideoUsage represents usage for video generation.
type VideoUsage struct {
	durationSeconds int
	costUSD         float64
}

// NewVideoUsage creates a new video usage.
func NewVideoUsage(durationSeconds int, costUSD float64) *VideoUsage {
	return &VideoUsage{
		durationSeconds: durationSeconds,
		costUSD:         costUSD,
	}
}

// DurationSeconds returns the duration in seconds.
func (u *VideoUsage) DurationSeconds() int { return u.durationSeconds }

// CostUSD returns the cost in USD.
func (u *VideoUsage) CostUSD() float64 { return u.costUSD }

// VideoState represents the state of a video generation.
type VideoState string

const (
	VideoStatePending    VideoState = "pending"
	VideoStateProcessing VideoState = "processing"
	VideoStateCompleted  VideoState = "completed"
	VideoStateFailed     VideoState = "failed"
)

// String returns the string representation.
func (s VideoState) String() string {
	return string(s)
}
