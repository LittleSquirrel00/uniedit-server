package media

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/task"
)

// Service provides media generation operations.
type Service struct {
	registry        *provider.Registry
	healthMonitor   *provider.HealthMonitor
	adapterRegistry *Registry
	taskManager     *task.Manager
}

// NewService creates a new media service.
func NewService(
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	taskManager *task.Manager,
) *Service {
	svc := &Service{
		registry:        registry,
		healthMonitor:   healthMonitor,
		adapterRegistry: GetRegistry(),
		taskManager:     taskManager,
	}

	// Register task executors
	if taskManager != nil {
		taskManager.RegisterExecutor(task.TypeImageGeneration, svc.executeImageTask)
		taskManager.RegisterExecutor(task.TypeVideoGeneration, svc.executeVideoTask)
	}

	return svc
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

// GenerateImage generates images synchronously.
func (s *Service) GenerateImage(ctx context.Context, userID uuid.UUID, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	// Find image generation model
	model, prov, err := s.findModel(req.Model, provider.CapabilityImage)
	if err != nil {
		return nil, err
	}

	// Get adapter
	adapter, err := s.adapterRegistry.GetForProvider(prov)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &ImageRequest{
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
		Model:          model.ID,
	}

	// Execute
	resp, err := adapter.GenerateImage(ctx, adapterReq, model, prov)
	if err != nil {
		return nil, fmt.Errorf("generate image: %w", err)
	}

	return &ImageGenerationResponse{
		ImageResponse: resp,
	}, nil
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

// GenerateVideo generates videos (always async via task manager).
func (s *Service) GenerateVideo(ctx context.Context, userID uuid.UUID, req *VideoGenerationRequest) (*VideoGenerationResponse, error) {
	// Validate request
	if req.Prompt == "" && req.InputImage == "" && req.InputVideo == "" {
		return nil, fmt.Errorf("prompt, input_image, or input_video required")
	}

	// Convert request to map for task input
	inputPayload := map[string]any{
		"prompt":       req.Prompt,
		"input_image":  req.InputImage,
		"input_video":  req.InputVideo,
		"duration":     req.Duration,
		"aspect_ratio": req.AspectRatio,
		"resolution":   req.Resolution,
		"fps":          req.FPS,
		"model":        req.Model,
	}

	// Submit task
	t, err := s.taskManager.Submit(ctx, userID, &task.Input{
		Type:    task.TypeVideoGeneration,
		Payload: inputPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("submit task: %w", err)
	}

	return &VideoGenerationResponse{
		TaskID:    t.ID.String(),
		Status:    VideoStatePending,
		Progress:  0,
		CreatedAt: t.CreatedAt.Unix(),
	}, nil
}

// GetVideoStatus returns the status of a video generation task.
func (s *Service) GetVideoStatus(ctx context.Context, userID uuid.UUID, taskID string) (*VideoGenerationResponse, error) {
	id, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task id: %w", err)
	}

	t, err := s.taskManager.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Check ownership
	if t.UserID != userID {
		return nil, fmt.Errorf("task not found")
	}

	resp := &VideoGenerationResponse{
		TaskID:    t.ID.String(),
		Status:    taskStatusToVideoState(t.Status),
		Progress:  t.Progress,
		CreatedAt: t.CreatedAt.Unix(),
	}

	// Parse output if completed
	if t.Status == task.StatusCompleted && t.Output != nil {
		// Convert map to GeneratedVideo
		outputBytes, _ := json.Marshal(t.Output)
		var video GeneratedVideo
		if err := json.Unmarshal(outputBytes, &video); err == nil {
			resp.Video = &video
		}
	}

	// Include error if failed
	if t.Error != nil {
		resp.Error = t.Error.Message
	}

	return resp, nil
}

// executeImageTask executes an image generation task.
func (s *Service) executeImageTask(ctx context.Context, t *task.Task, onProgress func(int)) error {
	// Parse request from map
	inputBytes, err := json.Marshal(t.Input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}

	var req ImageGenerationRequest
	if err := json.Unmarshal(inputBytes, &req); err != nil {
		return fmt.Errorf("unmarshal request: %w", err)
	}

	onProgress(10)

	// Find model
	model, prov, err := s.findModel(req.Model, provider.CapabilityImage)
	if err != nil {
		return err
	}

	onProgress(20)

	// Get adapter
	adapter, err := s.adapterRegistry.GetForProvider(prov)
	if err != nil {
		return fmt.Errorf("get adapter: %w", err)
	}

	onProgress(30)

	// Execute
	adapterReq := &ImageRequest{
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
		Model:          model.ID,
	}

	resp, err := adapter.GenerateImage(ctx, adapterReq, model, prov)
	if err != nil {
		return fmt.Errorf("generate image: %w", err)
	}

	onProgress(90)

	// Store output as map
	outputBytes, _ := json.Marshal(resp)
	var outputMap map[string]any
	json.Unmarshal(outputBytes, &outputMap)
	t.Output = outputMap

	return nil
}

// executeVideoTask executes a video generation task.
func (s *Service) executeVideoTask(ctx context.Context, t *task.Task, onProgress func(int)) error {
	// Parse request from map
	inputBytes, err := json.Marshal(t.Input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}

	var req VideoGenerationRequest
	if err := json.Unmarshal(inputBytes, &req); err != nil {
		return fmt.Errorf("unmarshal request: %w", err)
	}

	onProgress(10)

	// Find model
	model, prov, err := s.findModel(req.Model, provider.CapabilityVideo)
	if err != nil {
		return err
	}

	onProgress(20)

	// Get adapter
	adapter, err := s.adapterRegistry.GetForProvider(prov)
	if err != nil {
		return fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &VideoRequest{
		Prompt:      req.Prompt,
		InputImage:  req.InputImage,
		InputVideo:  req.InputVideo,
		Duration:    req.Duration,
		AspectRatio: req.AspectRatio,
		Resolution:  req.Resolution,
		FPS:         req.FPS,
		Model:       model.ID,
	}

	// Submit to provider
	resp, err := adapter.GenerateVideo(ctx, adapterReq, model, prov)
	if err != nil {
		return fmt.Errorf("generate video: %w", err)
	}

	onProgress(30)

	// Poll for completion
	providerTaskID := resp.TaskID
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := adapter.GetVideoStatus(ctx, providerTaskID, prov)
		if err != nil {
			return fmt.Errorf("get video status: %w", err)
		}

		// Update progress
		if status.Progress > 0 {
			// Map 0-100 to 30-90
			mappedProgress := 30 + (status.Progress * 60 / 100)
			onProgress(mappedProgress)
		}

		switch status.Status {
		case VideoStateCompleted:
			// Store output as map
			outputBytes, _ := json.Marshal(status.Video)
			var outputMap map[string]any
			json.Unmarshal(outputBytes, &outputMap)
			t.Output = outputMap
			return nil
		case VideoStateFailed:
			return fmt.Errorf("video generation failed: %s", status.Error)
		}

		// Wait before polling again
		time.Sleep(5 * time.Second)
	}
}

// findModel finds a model with the given capability.
func (s *Service) findModel(modelID string, capability provider.Capability) (*provider.Model, *provider.Provider, error) {
	// If model specified, use it directly
	if modelID != "" && modelID != "auto" {
		model, prov, ok := s.registry.GetModelWithProvider(modelID)
		if !ok {
			return nil, nil, fmt.Errorf("model not found: %s", modelID)
		}

		// Check capability
		if !model.HasCapability(capability) {
			return nil, nil, fmt.Errorf("model %s does not support %s", modelID, capability)
		}

		// Check health
		if !s.healthMonitor.IsHealthy(prov.ID) {
			return nil, nil, fmt.Errorf("provider %s is unhealthy", prov.Name)
		}

		return model, prov, nil
	}

	// Auto-select model
	models := s.registry.GetModelsByCapability(capability)
	if len(models) == 0 {
		return nil, nil, fmt.Errorf("no model available for capability: %s", capability)
	}

	// Filter by health
	for _, model := range models {
		prov, ok := s.registry.GetProvider(model.ProviderID)
		if !ok {
			continue
		}

		if s.healthMonitor.IsHealthy(prov.ID) {
			return model, prov, nil
		}
	}

	return nil, nil, fmt.Errorf("no healthy provider available for capability: %s", capability)
}

// taskStatusToVideoState converts task status to video state.
func taskStatusToVideoState(status task.Status) VideoState {
	switch status {
	case task.StatusPending:
		return VideoStatePending
	case task.StatusRunning:
		return VideoStateProcessing
	case task.StatusCompleted:
		return VideoStateCompleted
	case task.StatusFailed, task.StatusCancelled:
		return VideoStateFailed
	default:
		return VideoStatePending
	}
}
