package ai

import "errors"

// Provider errors.
var (
	ErrProviderNotFound    = errors.New("provider not found")
	ErrProviderDisabled    = errors.New("provider is disabled")
	ErrProviderUnhealthy   = errors.New("provider is unhealthy")
	ErrProviderRateLimited = errors.New("provider rate limited")
)

// Model errors.
var (
	ErrModelNotFound    = errors.New("model not found")
	ErrModelDisabled    = errors.New("model is disabled")
	ErrModelUnavailable = errors.New("model is unavailable")
)

// Group errors.
var (
	ErrGroupNotFound  = errors.New("group not found")
	ErrGroupDisabled  = errors.New("group is disabled")
	ErrNoModelsInGroup = errors.New("no models in group")
)

// Routing errors.
var (
	ErrNoModelAvailable       = errors.New("no model available")
	ErrNoProviderAvailable    = errors.New("no provider available")
	ErrCapabilityNotSupported = errors.New("capability not supported")
	ErrAllFallbacksFailed     = errors.New("all fallback attempts failed")
)

// Request errors.
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrContextCancelled   = errors.New("context cancelled")
	ErrRequestTimeout     = errors.New("request timeout")
	ErrInsufficientQuota  = errors.New("insufficient quota")
)

// Response errors.
var (
	ErrInvalidResponse    = errors.New("invalid response from provider")
	ErrResponseTruncated  = errors.New("response was truncated")
	ErrContentFiltered    = errors.New("content was filtered")
)
