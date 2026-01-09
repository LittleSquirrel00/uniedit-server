package media

import "errors"

var (
	// ErrProviderNotFound is returned when a provider is not found.
	ErrProviderNotFound = errors.New("media provider not found")

	// ErrModelNotFound is returned when a model is not found.
	ErrModelNotFound = errors.New("media model not found")

	// ErrTaskNotFound is returned when a task is not found.
	ErrTaskNotFound = errors.New("media task not found")

	// ErrTaskNotOwned is returned when the user doesn't own the task.
	ErrTaskNotOwned = errors.New("task not owned by user")

	// ErrNoAdapterFound is returned when no adapter is found for a provider.
	ErrNoAdapterFound = errors.New("no adapter found for provider")

	// ErrProviderUnhealthy is returned when a provider is unhealthy.
	ErrProviderUnhealthy = errors.New("provider is unhealthy")

	// ErrNoHealthyProvider is returned when no healthy provider is available.
	ErrNoHealthyProvider = errors.New("no healthy provider available")

	// ErrCapabilityNotSupported is returned when a model doesn't support a capability.
	ErrCapabilityNotSupported = errors.New("model does not support this capability")

	// ErrInvalidInput is returned when input is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrTaskAlreadyCompleted is returned when trying to cancel a completed task.
	ErrTaskAlreadyCompleted = errors.New("task already completed")

	// ErrTaskAlreadyCancelled is returned when trying to cancel a cancelled task.
	ErrTaskAlreadyCancelled = errors.New("task already cancelled")
)
