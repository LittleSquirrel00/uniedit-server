package media

import "errors"

// Domain errors for media module.
var (
	// Task errors
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskNotOwned      = errors.New("task not owned by user")
	ErrTaskAlreadyDone   = errors.New("task already completed or cancelled")
	ErrTaskInProgress    = errors.New("task is already in progress")

	// Provider errors
	ErrProviderNotFound  = errors.New("provider not found")
	ErrProviderUnhealthy = errors.New("provider is unhealthy")
	ErrProviderDisabled  = errors.New("provider is disabled")

	// Model errors
	ErrModelNotFound         = errors.New("model not found")
	ErrModelNotSupported     = errors.New("model does not support requested capability")
	ErrNoModelAvailable      = errors.New("no model available for capability")
	ErrNoHealthyProvider     = errors.New("no healthy provider available")

	// Adapter errors
	ErrAdapterNotFound       = errors.New("adapter not found for provider type")
	ErrCapabilityNotSupported = errors.New("capability not supported by adapter")

	// Request errors
	ErrInvalidRequest        = errors.New("invalid request")
	ErrMissingPrompt         = errors.New("prompt is required")
	ErrMissingInput          = errors.New("prompt, input_image, or input_video is required")
)
