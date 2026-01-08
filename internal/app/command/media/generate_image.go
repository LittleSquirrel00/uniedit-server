package media

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/media"
)

// GenerateImageCommand represents a command to generate images.
type GenerateImageCommand struct {
	UserID         uuid.UUID
	Prompt         string
	NegativePrompt string
	N              int
	Size           string
	Quality        string
	Style          string
	Model          string
	ResponseFormat string
}

// GenerateImageResult is the result of image generation.
type GenerateImageResult struct {
	Images    []*ImageDTO
	Model     string
	Usage     *ImageUsageDTO
	CreatedAt int64
}

// ImageDTO represents a generated image.
type ImageDTO struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageUsageDTO represents image usage.
type ImageUsageDTO struct {
	TotalImages int     `json:"total_images"`
	CostUSD     float64 `json:"cost_usd,omitempty"`
}

// GenerateImageHandler handles image generation commands.
type GenerateImageHandler struct {
	providerRegistry media.ProviderRegistry
	healthChecker    media.HealthChecker
	adapterRegistry  media.AdapterRegistry
}

// NewGenerateImageHandler creates a new handler.
func NewGenerateImageHandler(
	providerRegistry media.ProviderRegistry,
	healthChecker media.HealthChecker,
	adapterRegistry media.AdapterRegistry,
) *GenerateImageHandler {
	return &GenerateImageHandler{
		providerRegistry: providerRegistry,
		healthChecker:    healthChecker,
		adapterRegistry:  adapterRegistry,
	}
}

// Handle executes the command.
func (h *GenerateImageHandler) Handle(ctx context.Context, cmd GenerateImageCommand) (*GenerateImageResult, error) {
	if cmd.Prompt == "" {
		return nil, media.ErrMissingPrompt
	}

	// Find model
	model, provider, err := h.findModel(cmd.Model, media.CapabilityImage)
	if err != nil {
		return nil, err
	}

	// Get adapter
	adapter, err := h.adapterRegistry.GetForProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build request
	req := &media.ImageGenerationRequest{
		Prompt:         cmd.Prompt,
		NegativePrompt: cmd.NegativePrompt,
		N:              cmd.N,
		Size:           cmd.Size,
		Quality:        cmd.Quality,
		Style:          cmd.Style,
		ResponseFormat: cmd.ResponseFormat,
		Model:          model.ID(),
	}

	// Execute
	result, err := adapter.GenerateImage(ctx, req, model, provider)
	if err != nil {
		return nil, fmt.Errorf("generate image: %w", err)
	}

	// Convert to DTO
	images := make([]*ImageDTO, len(result.Images()))
	for i, img := range result.Images() {
		images[i] = &ImageDTO{
			URL:           img.URL(),
			B64JSON:       img.B64JSON(),
			RevisedPrompt: img.RevisedPrompt(),
		}
	}

	var usage *ImageUsageDTO
	if result.Usage() != nil {
		usage = &ImageUsageDTO{
			TotalImages: result.Usage().TotalImages(),
			CostUSD:     result.Usage().CostUSD(),
		}
	}

	return &GenerateImageResult{
		Images:    images,
		Model:     result.Model(),
		Usage:     usage,
		CreatedAt: result.CreatedAt(),
	}, nil
}

func (h *GenerateImageHandler) findModel(modelID string, capability media.Capability) (*media.Model, *media.Provider, error) {
	// If model specified, use it directly
	if modelID != "" && modelID != "auto" {
		model, provider, ok := h.providerRegistry.GetModelWithProvider(modelID)
		if !ok {
			return nil, nil, media.ErrModelNotFound
		}

		if !model.HasCapability(capability) {
			return nil, nil, media.ErrModelNotSupported
		}

		if !h.healthChecker.IsHealthy(provider.ID()) {
			return nil, nil, media.ErrProviderUnhealthy
		}

		return model, provider, nil
	}

	// Auto-select model
	models := h.providerRegistry.GetModelsByCapability(capability)
	if len(models) == 0 {
		return nil, nil, media.ErrNoModelAvailable
	}

	// Filter by health
	for _, model := range models {
		provider, ok := h.providerRegistry.GetProvider(model.ProviderID())
		if !ok {
			continue
		}

		if h.healthChecker.IsHealthy(provider.ID()) {
			return model, provider, nil
		}
	}

	return nil, nil, media.ErrNoHealthyProvider
}
