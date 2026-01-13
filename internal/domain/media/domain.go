package media

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
	"github.com/uniedit/server/internal/utils/requestctx"
)

// Domain implements the media domain logic.
type Domain struct {
	providerDB     outbound.MediaProviderDatabasePort
	modelDB        outbound.MediaModelDatabasePort
	taskDB         outbound.MediaTaskDatabasePort
	usageDB        outbound.UsageRecordDatabasePort
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
	usageDB outbound.UsageRecordDatabasePort,
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
		usageDB:        usageDB,
		healthCache:    healthCache,
		vendorRegistry: vendorRegistry,
		crypto:         crypto,
		config:         config,
		logger:         logger,
	}
}

// GenerateImage generates images synchronously.
func (d *Domain) GenerateImage(ctx context.Context, userID uuid.UUID, in *mediav1.GenerateImageRequest) (*mediav1.GenerateImageResponse, error) {
	if in.GetPrompt() == "" {
		return nil, ErrInvalidInput
	}

	startTime := time.Now()

	// Find model with image capability
	mediaModel, provider, apiKey, err := d.findModelWithCapability(ctx, in.GetModel(), model.MediaCapabilityImage)
	if err != nil {
		return nil, err
	}

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAdapterFound, err)
	}

	// Build request from proto directly
	req := &model.ImageRequest{
		Prompt:         in.GetPrompt(),
		NegativePrompt: in.GetNegativePrompt(),
		N:              int(in.GetN()),
		Size:           in.GetSize(),
		Quality:        in.GetQuality(),
		Style:          in.GetStyle(),
		ResponseFormat: in.GetResponseFormat(),
		Model:          mediaModel.ID,
	}

	// Execute
	resp, err := adapter.GenerateImage(ctx, req, mediaModel, provider, apiKey)
	if err != nil {
		d.recordMediaUsage(ctx, userID, "", startTime, time.Since(startTime).Milliseconds(), false, string(model.AITaskTypeImage), provider.ID, mediaModel.ID, apiKey, 0)
		return nil, fmt.Errorf("generate image: %w", err)
	}

	latencyMs := time.Since(startTime).Milliseconds()
	costUSD := float64(0)
	if resp != nil && resp.Usage != nil {
		costUSD = resp.Usage.CostUSD
	}
	d.recordMediaUsage(ctx, userID, "", startTime, latencyMs, true, string(model.AITaskTypeImage), provider.ID, mediaModel.ID, apiKey, costUSD)

	d.logger.Info("Image generated",
		zap.String("user_id", userID.String()),
		zap.String("model", mediaModel.ID),
		zap.Int("count", len(resp.Images)),
	)

	return toGenerateImageResponseFromResult(resp), nil
}

// GenerateVideo generates videos asynchronously.
func (d *Domain) GenerateVideo(ctx context.Context, userID uuid.UUID, in *mediav1.GenerateVideoRequest) (*mediav1.VideoGenerationStatus, error) {
	// Validate input from proto directly
	if in.GetPrompt() == "" && in.GetInputImage() == "" && in.GetInputVideo() == "" {
		return nil, ErrInvalidInput
	}

	// Serialize input for task storage
	inputData := &videoGenerationInput{
		Prompt:      in.GetPrompt(),
		InputImage:  in.GetInputImage(),
		InputVideo:  in.GetInputVideo(),
		Duration:    int(in.GetDuration()),
		AspectRatio: in.GetAspectRatio(),
		Resolution:  in.GetResolution(),
		FPS:         int(in.GetFps()),
		Model:       in.GetModel(),
	}

	inputBytes, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	// Create task
	now := time.Now()
	inputStr := string(inputBytes)
	task := &model.MediaTask{
		ID:        uuid.New(),
		OwnerID:   userID,
		Type:      "video_generation",
		Status:    model.MediaTaskStatusPending,
		Progress:  0,
		Input:     &inputStr,
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

	return toVideoStatusPending(task), nil
}

// GetVideoStatus returns the status of a video generation task.
func (d *Domain) GetVideoStatus(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*mediav1.VideoGenerationStatus, error) {
	id, err := uuid.Parse(in.GetTaskId())
	if err != nil {
		return nil, ErrInvalidInput
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

	// Parse output if completed
	var video *model.GeneratedVideo
	if task.Status == model.MediaTaskStatusCompleted && task.Output != nil && *task.Output != "" {
		var v model.GeneratedVideo
		if err := json.Unmarshal([]byte(*task.Output), &v); err == nil {
			video = &v
		}
	}

	return toVideoStatusFromTask(task, video), nil
}

// GetTask returns a task by ID.
func (d *Domain) GetTask(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*mediav1.MediaTask, error) {
	taskID, err := uuid.Parse(in.GetTaskId())
	if err != nil {
		return nil, ErrInvalidInput
	}

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

	return toMediaTaskPB(task), nil
}

// ListTasks lists tasks for a user.
func (d *Domain) ListTasks(ctx context.Context, userID uuid.UUID, in *mediav1.ListTasksRequest) (*mediav1.ListTasksResponse, error) {
	limit := int(in.GetLimit())
	offset := int(in.GetOffset())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	tasks, err := d.taskDB.FindByOwner(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	out := make([]*mediav1.MediaTask, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, toMediaTaskPB(task))
	}

	return &mediav1.ListTasksResponse{Tasks: out}, nil
}

// CancelTask cancels a task.
func (d *Domain) CancelTask(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*commonv1.Empty, error) {
	taskID, err := uuid.Parse(in.GetTaskId())
	if err != nil {
		return nil, ErrInvalidInput
	}

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

	// Check status
	switch task.Status {
	case model.MediaTaskStatusCompleted:
		return nil, ErrTaskAlreadyCompleted
	case model.MediaTaskStatusCancelled:
		return nil, ErrTaskAlreadyCancelled
	case model.MediaTaskStatusFailed:
		return nil, ErrTaskAlreadyCompleted
	}

	// Update status
	if err := d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusCancelled, task.Progress, "", "cancelled by user"); err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}

	d.logger.Info("Task cancelled",
		zap.String("task_id", taskID.String()),
		zap.String("user_id", userID.String()),
	)

	return empty(), nil
}

// GetProvider returns a provider by ID.
func (d *Domain) GetProvider(ctx context.Context, in *mediav1.GetByIDRequest) (*mediav1.MediaProvider, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidInput
	}

	provider, err := d.providerDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	return toProvider(provider), nil
}

// ListProviders lists all providers.
func (d *Domain) ListProviders(ctx context.Context, _ *commonv1.Empty) (*mediav1.ListProvidersResponse, error) {
	providers, err := d.providerDB.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*mediav1.MediaProvider, 0, len(providers))
	for _, p := range providers {
		out = append(out, toProvider(p))
	}
	return &mediav1.ListProvidersResponse{Providers: out}, nil
}

// ListModels lists models by capability.
func (d *Domain) ListModels(ctx context.Context, in *mediav1.ListModelsRequest) (*mediav1.ListModelsResponse, error) {
	capability := in.GetCapability()
	if capability == "" {
		return &mediav1.ListModelsResponse{Models: []*mediav1.MediaModel{}}, nil
	}

	models, err := d.modelDB.FindByCapability(ctx, model.MediaCapability(capability))
	if err != nil {
		return nil, err
	}

	out := make([]*mediav1.MediaModel, 0, len(models))
	for _, m := range models {
		out = append(out, toModel(m))
	}
	return &mediav1.ListModelsResponse{Models: out}, nil
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
	var input videoGenerationInput
	if task.Input == nil {
		d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", "missing input")
		return fmt.Errorf("task input is nil")
	}
	if err := json.Unmarshal([]byte(*task.Input), &input); err != nil {
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
		d.recordMediaUsage(ctx, task.OwnerID, task.ID.String(), task.CreatedAt, time.Since(task.CreatedAt).Milliseconds(), false, string(model.AITaskTypeVideo), provider.ID, mediaModel.ID, apiKey, 0)
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
			d.recordMediaUsage(ctx, task.OwnerID, task.ID.String(), task.CreatedAt, time.Since(task.CreatedAt).Milliseconds(), true, string(model.AITaskTypeVideo), provider.ID, mediaModel.ID, apiKey, 0)
			d.logger.Info("Video generation completed",
				zap.String("task_id", taskID.String()),
			)
			return nil
		case model.VideoStateFailed:
			d.recordMediaUsage(ctx, task.OwnerID, task.ID.String(), task.CreatedAt, time.Since(task.CreatedAt).Milliseconds(), false, string(model.AITaskTypeVideo), provider.ID, mediaModel.ID, apiKey, 0)
			d.taskDB.UpdateStatus(ctx, taskID, model.MediaTaskStatusFailed, 0, "", status.Error)
			return fmt.Errorf("video generation failed: %s", status.Error)
		}

		// Wait before polling again
		time.Sleep(d.config.VideoPollInterval)
	}
}

func (d *Domain) recordMediaUsage(
	ctx context.Context,
	userID uuid.UUID,
	requestID string,
	startTime time.Time,
	latencyMs int64,
	success bool,
	taskType string,
	providerID uuid.UUID,
	modelID string,
	apiKey string,
	costUSD float64,
) {
	if d.usageDB == nil {
		return
	}

	if requestID == "" {
		requestID = requestctx.RequestID(ctx)
	}
	if requestID == "" {
		requestID = uuid.New().String()
	}

	record := &model.UsageRecord{
		UserID:         userID,
		Timestamp:      startTime.UTC(),
		RequestID:      requestID,
		TaskType:       taskType,
		ProviderID:     providerID,
		ModelID:        modelID,
		LatencyMs:      int(latencyMs),
		Success:        success,
		CostUSD:        costUSD,
		CostMultiplier: 1,
	}

	if prefix := keyPrefix(apiKey, 10); prefix != "" {
		record.APIKeyPrefix = &prefix
	}

	_ = d.usageDB.Create(ctx, record)
}

func keyPrefix(key string, length int) string {
	if key == "" {
		return ""
	}
	if len(key) <= length {
		return key
	}
	return key[:length]
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

// videoGenerationInput is an internal struct for task storage.
// This replaces the external model.MediaVideoGenerationInput.
type videoGenerationInput struct {
	Prompt      string `json:"prompt,omitempty"`
	InputImage  string `json:"input_image,omitempty"`
	InputVideo  string `json:"input_video,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	FPS         int    `json:"fps,omitempty"`
	Model       string `json:"model,omitempty"`
}

// Compile-time interface check
var _ inbound.MediaDomain = (*Domain)(nil)
