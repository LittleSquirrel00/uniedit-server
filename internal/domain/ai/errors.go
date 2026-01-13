package ai

import "errors"

// Domain errors.
var (
	// Provider errors
	ErrProviderNotFound      = errors.New("provider not found")
	ErrProviderDisabled      = errors.New("provider is disabled")
	ErrProviderUnhealthy     = errors.New("provider is unhealthy")
	ErrProviderAlreadyExists = errors.New("provider already exists")

	// Model errors
	ErrModelNotFound      = errors.New("model not found")
	ErrModelDisabled      = errors.New("model is disabled")
	ErrModelNotSupported  = errors.New("model does not support required capability")
	ErrModelAlreadyExists = errors.New("model already exists")

	// Account errors
	ErrAccountNotFound    = errors.New("account not found")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrAccountUnhealthy   = errors.New("account is unhealthy")
	ErrNoAvailableAccount = errors.New("no available account")

	// Group errors
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupDisabled      = errors.New("group is disabled")
	ErrGroupAlreadyExists = errors.New("group already exists")

	// Routing errors
	ErrNoAvailableModels  = errors.New("no available models for routing")
	ErrRoutingFailed      = errors.New("routing failed")
	ErrAllFallbacksFailed = errors.New("all fallback attempts failed")

	// Request errors
	ErrInvalidRequest = errors.New("invalid request")
	ErrEmptyMessages  = errors.New("messages cannot be empty")
	ErrEmptyInput     = errors.New("input cannot be empty")

	// Rate limit errors
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrInsufficientCredits = errors.New("insufficient credits")

	// Adapter errors
	ErrAdapterNotFound     = errors.New("adapter not found")
	ErrAdapterNotSupported = errors.New("adapter does not support this operation")

	// API errors
	ErrUpstreamError = errors.New("upstream API error")
	ErrTimeout       = errors.New("request timeout")
)
