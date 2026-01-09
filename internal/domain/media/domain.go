package media

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
)

// Domain implements the media domain logic.
type Domain struct {
	providerDB     outbound.MediaProviderDatabasePort
	modelDB        outbound.MediaModelDatabasePort
	taskDB         outbound.MediaTaskDatabasePort
	healthCache    outbound.MediaProviderHealthCachePort
	vendorRegistry outbound.MediaVendorRegistryPort
	crypto         outbound.MediaCryptoPort
	config         *Config
	logger         *zap.Logger
}

// NewDomain creates a new media domain.
func NewDomain(
	providerDB outbound.MediaProviderDatabasePort,
	modelDB outbound.MediaModelDatabasePort,
	taskDB outbound.MediaTaskDatabasePort,
	healthCache outbound.MediaProviderHealthCachePort,
	vendorRegistry outbound.MediaVendorRegistryPort,
	crypto outbound.MediaCryptoPort,
	config *Config,
	logger *zap.Logger,
) *Domain {
	if config == nil {
		config = DefaultConfig()
	}
	return &Domain{
		providerDB:     providerDB,
		modelDB:        modelDB,
		taskDB:         taskDB,
		healthCache:    healthCache,
		vendorRegistry: vendorRegistry,
		crypto:         crypto,
		config:         config,
		logger:         logger,
	}
}

// GenerateImage generates images synchronously.
func (d *Domain) GenerateImage(ctx context.Context, userID uuid.UUID, input *inbound.MediaImageGenerationInput) (*inbound.MediaImageGenerationOutput, error) {
	if input.Prompt == "" {
		return nil, ErrInvalidInput
	}

	// Find model with image capability
	mediaModel, provider, apiKey, err := d.findModelWithCapability(ctx, input.Model, model.MediaCapabilityImage)
	if err != nil {
		return nil, err
	}

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAdapterFound, err)
	}

	// Build request
	req := &model.ImageRequest{
		Prompt:         input.Prompt,
		NegativePrompt: input.NegativePrompt,
		N:              input.N,
		Size:           input.Size,
		Quality:        input.Quality,
		Style:          input.Style,
		ResponseFormat: input.ResponseFormat,
		Model:          mediaModel.ID,
	}

	// Execute
	resp, err := adapter.GenerateImage(ctx, req, mediaModel, provider, apiKey)
	if err != nil {
		return nil, fmt.Errorf("generate image: %w", err)
	}

	d.logger.Info("Image generated",
		zap.String("user_id", userID.String()),
		zap.String("model", mediaModel.ID),
		zap.Int("count", len(resp.Images)),
	)

	return &inbound.MediaImageGenerationOutput{
		Images:    resp.Images,
		Model:     resp.Model,
		Usage:     resp.Usage,
		CreatedAt: resp.CreatedAt,
	}, nil
}

// GenerateVideo generates videos asynchronously.
func (d *Domain) GenerateVideo(ctx context.Context, userID uuid.UUID, input *inbound.MediaVideoGenerationInput) (*inbound.MediaVideoGenerationOutput, error) {
	// Validate input
	if input.Prompt == "" && input.InputImage == "" && input.InputVideo == "" {
		return nil, ErrInvalidInput
	}

	// Serialize input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	// Create task
	now := time.Now()
	task := &model.MediaTask{
		ID:        uuid.New(),
		OwnerID:   userID,
		Type:      "video_generation",
		Status:    model.MediaTaskStatusPending,
		Progress:  0,
		Input:     string(inputBytes),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.taskDB.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	d.logger.Info("Video generation task created",
		zap.String("task_id", task.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return &inbound.MediaVideoGenerationOutput{
		TaskID:    task.ID.String(),
		Status:    model.VideoStatePending,
		Progress:  0,
		CreatedAt: task.CreatedAt.Unix(),
	}, nil
}

// GetVideoStatus returns the status of a video generation task.
func (d *Domain) GetVideoStatus(ctx context.Context, userID uuid.UUID, taskID string) (*inbound.MediaVideoGenerationOutput, error) {
	id, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task id: %w", err)
	}

	task, err := d.taskDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	// Check ownership
	if task.OwnerID != userID {
		return nil, ErrTaskNotOwned
	}

	resp := &inbound.MediaVideoGenerationOutput{
		TaskID:    task.ID.String(),
		Status:    taskStatusToVideoState(task.Status),
		Progress:  task.Progress,
		CreatedAt: task.CreatedAt.Unix(),
	}

	// Parse output if completed
	if task.Status == model.MediaTaskStatusCompleted && task.Output != "" {
		var video model.GeneratedVideo
		if err := json.Unmarshal([]byte(task.Output), &video); err == nil {
			resp.Video = &video
		}
	}

	// Include error if failed
	if task.Error != "" {
		resp.Error = task.Error
	}

	return resp, nil
}

// GetTask returns a task by ID.
func (d *Domain) GetTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*inbound.MediaTaskOutput, error) {
	task, err := d.taskDB.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	// Check ownership
	if task.OwnerID != userID {
		return nil, ErrTaskNotOwned
	}

	return &inbound.MediaTaskOutput{
		ID:        task.ID,
		OwnerID:   task.OwnerID,
		Type:      task.Type,
		Status:    task.Status,
		Progress:  task.Progress,
		Error:     task.Error,
		CreatedAt: task.CreatedAt.Unix(),
		UpdatedAt: task.UpdatedAt.Unix(),
	}, nil
}

// ListTasks lists tasks for a user.
func (d *Domain) ListTasks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*inbound.MediaTaskOutput, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	tasks, err := d.taskDB.FindByOwner(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*inbound.MediaTaskOutput, len(tasks))
	for i, task := range tasks {
		result[i] = &inbound.MediaTaskOutput{
			ID:        task.ID,
			OwnerID:   task.OwnerID,
			Type:      task.Type,
			Status:    task.Status,
			Progress:  task.Progress,
			Error:     task.Error,
			CreatedAt: task.CreatedAt.Unix(),
			UpdatedAt: task.UpdatedAt.Unix(),
		}
	}

	return result, nil
}

// CancelTask cancels a task.
func (d *Domain) CancelTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	task, err := d.taskDB.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	// Check ownership
	if task.OwnerID != userID {
		return ErrTaskNotOwned
	}

	// Check status
	switch task.Status {
	case model.MediaTaskStatusCompleted:
		return ErrTaskAlreadyCompleted
	case model.MediaTaskStatusCancelled:
		return ErrTaskAlreadyCancelled
	case model.MediaTaskStatusFailed:
		return ErrTaskAlreadyCompleted
	}

	// Update status
	if err := d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusCancelled, task.Progress, "", "cancelled by user"); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	d.logger.Info("Task cancelled",
		zap.String("task_id", taskID.String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// GetProvider returns a provider by ID.
func (d *Domain) GetProvider(ctx context.Context, id uuid.UUID) (*model.MediaProvider, error) {
	provider, err := d.providerDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

// ListProviders lists all providers.
func (d *Domain) ListProviders(ctx context.Context) ([]*model.MediaProvider, error) {
	return d.providerDB.FindAll(ctx)
}

// GetModel returns a model by ID.
func (d *Domain) GetModel(ctx context.Context, id string) (*model.MediaModel, error) {
	m, err := d.modelDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrModelNotFound
	}
	return m, nil
}

// ListModelsByCapability lists models with a capability.
func (d *Domain) ListModelsByCapability(ctx context.Context, cap model.MediaCapability) ([]*model.MediaModel, error) {
	return d.modelDB.FindByCapability(ctx, cap)
}

// findModelWithCapability finds a model with the given capability.
func (d *Domain) findModelWithCapability(ctx context.Context, modelID string, capability model.MediaCapability) (*model.MediaModel, *model.MediaProvider, string, error) {
	// If model specified, use it directly
	if modelID != "" && modelID != "auto" {
		mediaModel, err := d.modelDB.FindByID(ctx, modelID)
		if err != nil {
			return nil, nil, "", err
		}
		if mediaModel == nil {
			return nil, nil, "", ErrModelNotFound
		}

		// Check capability
		if !mediaModel.HasCapability(capability) {
			return nil, nil, "", ErrCapabilityNotSupported
		}

		// Get provider
		provider, err := d.providerDB.FindByID(ctx, mediaModel.ProviderID)
		if err != nil {
			return nil, nil, "", err
		}
		if provider == nil {
			return nil, nil, "", ErrProviderNotFound
		}

		// Check health
		healthy, err := d.healthCache.GetHealth(ctx, provider.ID)
		if err != nil {
			d.logger.Warn("Failed to get health status", zap.Error(err))
		} else if !healthy {
			return nil, nil, "", ErrProviderUnhealthy
		}

		// Decrypt API key
		apiKey, err := d.crypto.Decrypt(provider.EncryptedKey)
		if err != nil {
			return nil, nil, "", fmt.Errorf("decrypt api key: %w", err)
		}

		return mediaModel, provider, apiKey, nil
	}

	// Auto-select model
	models, err := d.modelDB.FindByCapability(ctx, capability)
	if err != nil {
		return nil, nil, "", err
	}
	if len(models) == 0 {
		return nil, nil, "", ErrModelNotFound
	}

	// Filter by health
	for _, mediaModel := range models {
		if !mediaModel.Enabled {
			continue
		}

		provider, err := d.providerDB.FindByID(ctx, mediaModel.ProviderID)
		if err != nil || provider == nil || !provider.Enabled {
			continue
		}

		healthy, err := d.healthCache.GetHealth(ctx, provider.ID)
		if err != nil {
			d.logger.Warn("Failed to get health status", zap.Error(err))
			continue
		}
		if !healthy {
			continue
		}

		// Decrypt API key
		apiKey, err := d.crypto.Decrypt(provider.EncryptedKey)
		if err != nil {
			d.logger.Warn("Failed to decrypt API key", zap.Error(err))
			continue
		}

		return mediaModel, provider, apiKey, nil
	}

	return nil, nil, "", ErrNoHealthyProvider
}

// ExecuteVideoTask executes a video generation task.
// This is called by a task worker.
func (d *Domain) ExecuteVideoTask(ctx context.Context, taskID uuid.UUID) error {
	task, err := d.taskDB.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	// Update to running
	if err := d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusRunning, 10, "", ""); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	// Parse input
	var input inbound.MediaVideoGenerationInput
	if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
		d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", "invalid input")
		return fmt.Errorf("unmarshal input: %w", err)
	}

	// Find model
	mediaModel, provider, apiKey, err := d.findModelWithCapability(ctx, input.Model, model.MediaCapabilityVideo)
	if err != nil {
		d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", err.Error())
		return err
	}

	d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusRunning, 20, "", "")

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(provider)
	if err != nil {
		d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", err.Error())
		return fmt.Errorf("%w: %v", ErrNoAdapterFound, err)
	}

	// Build request
	req := &model.VideoRequest{
		Prompt:      input.Prompt,
		InputImage:  input.InputImage,
		InputVideo:  input.InputVideo,
		Duration:    input.Duration,
		AspectRatio: input.AspectRatio,
		Resolution:  input.Resolution,
		FPS:         input.FPS,
		Model:       mediaModel.ID,
	}

	// Submit to provider
	resp, err := adapter.GenerateVideo(ctx, req, mediaModel, provider, apiKey)
	if err != nil {
		d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", err.Error())
		return fmt.Errorf("generate video: %w", err)
	}

	d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusRunning, 30, "", "")

	// Poll for completion
	providerTaskID := resp.TaskID
	for {
		select {
		case <-ctx.Done():
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", "context cancelled")
			return ctx.Err()
		default:
		}

		status, err := adapter.GetVideoStatus(ctx, providerTaskID, provider, apiKey)
		if err != nil {
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", err.Error())
			return fmt.Errorf("get video status: %w", err)
		}

		// Update progress (map 0-100 to 30-90)
		if status.Progress > 0 {
			mappedProgress := 30 + (status.Progress * 60 / 100)
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusRunning, mappedProgress, "", "")
		}

		switch status.Status {
		case model.VideoStateCompleted:
			outputBytes, _ := json.Marshal(status.Video)
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusCompleted, 100, string(outputBytes), "")
			d.logger.Info("Video generation completed",
				zap.String("task_id", taskID.String()),
			)
			return nil
		case model.VideoStateFailed:
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", status.Error)
			return fmt.Errorf("video generation failed: %s", status.Error)
		}

		// Wait before polling again
		time.Sleep(d.config.VideoPollInterval)
	}
}

// taskStatusToVideoState converts task status to video state.
func taskStatusToVideoState(status model.MediaTaskStatus) model.VideoState {
	switch status {
	case model.MediaTaskStatusPending:
		return model.VideoStatePending
	case model.MediaTaskStatusRunning:
		return model.VideoStateProcessing
	case model.MediaTaskStatusCompleted:
		return model.VideoStateCompleted
	case model.MediaTaskStatusFailed, model.MediaTaskStatusCancelled:
		return model.VideoStateFailed
	default:
		return model.VideoStatePending
	}
}

// Compile-time interface check
var _ inbound.MediaDomain = (*Domain)(nil)
