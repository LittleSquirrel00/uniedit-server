package domain

import (
	"time"

	"github.com/google/uuid"
)

// UsageRecord represents a single AI usage event.
// UsageRecord is an entity that tracks individual API usage.
type UsageRecord struct {
	id           int64
	userID       uuid.UUID
	apiKeyID     *uuid.UUID
	timestamp    time.Time
	requestID    string
	taskType     string
	providerID   uuid.UUID
	modelID      string
	inputTokens  int
	outputTokens int
	totalTokens  int
	costUSD      float64
	latencyMs    int
	success      bool
	cacheHit     bool
}

// NewUsageRecord creates a new UsageRecord.
func NewUsageRecord(
	userID uuid.UUID,
	requestID string,
	taskType string,
	providerID uuid.UUID,
	modelID string,
	inputTokens, outputTokens int,
	costUSD float64,
	latencyMs int,
	success bool,
) *UsageRecord {
	return &UsageRecord{
		userID:       userID,
		timestamp:    time.Now(),
		requestID:    requestID,
		taskType:     taskType,
		providerID:   providerID,
		modelID:      modelID,
		inputTokens:  inputTokens,
		outputTokens: outputTokens,
		totalTokens:  inputTokens + outputTokens,
		costUSD:      costUSD,
		latencyMs:    latencyMs,
		success:      success,
	}
}

// RestoreUsageRecord recreates a UsageRecord from persisted data.
func RestoreUsageRecord(
	id int64,
	userID uuid.UUID,
	apiKeyID *uuid.UUID,
	timestamp time.Time,
	requestID, taskType string,
	providerID uuid.UUID,
	modelID string,
	inputTokens, outputTokens, totalTokens int,
	costUSD float64,
	latencyMs int,
	success, cacheHit bool,
) *UsageRecord {
	return &UsageRecord{
		id:           id,
		userID:       userID,
		apiKeyID:     apiKeyID,
		timestamp:    timestamp,
		requestID:    requestID,
		taskType:     taskType,
		providerID:   providerID,
		modelID:      modelID,
		inputTokens:  inputTokens,
		outputTokens: outputTokens,
		totalTokens:  totalTokens,
		costUSD:      costUSD,
		latencyMs:    latencyMs,
		success:      success,
		cacheHit:     cacheHit,
	}
}

// --- Getters ---

// ID returns the record ID.
func (r *UsageRecord) ID() int64 {
	return r.id
}

// UserID returns the user ID.
func (r *UsageRecord) UserID() uuid.UUID {
	return r.userID
}

// APIKeyID returns the API key ID (may be nil for JWT auth).
func (r *UsageRecord) APIKeyID() *uuid.UUID {
	return r.apiKeyID
}

// Timestamp returns when the request was made.
func (r *UsageRecord) Timestamp() time.Time {
	return r.timestamp
}

// RequestID returns the request ID.
func (r *UsageRecord) RequestID() string {
	return r.requestID
}

// TaskType returns the task type (chat, image, video, embedding).
func (r *UsageRecord) TaskType() string {
	return r.taskType
}

// ProviderID returns the provider ID.
func (r *UsageRecord) ProviderID() uuid.UUID {
	return r.providerID
}

// ModelID returns the model ID.
func (r *UsageRecord) ModelID() string {
	return r.modelID
}

// InputTokens returns the number of input tokens.
func (r *UsageRecord) InputTokens() int {
	return r.inputTokens
}

// OutputTokens returns the number of output tokens.
func (r *UsageRecord) OutputTokens() int {
	return r.outputTokens
}

// TotalTokens returns the total number of tokens.
func (r *UsageRecord) TotalTokens() int {
	return r.totalTokens
}

// CostUSD returns the cost in USD.
func (r *UsageRecord) CostUSD() float64 {
	return r.costUSD
}

// LatencyMs returns the latency in milliseconds.
func (r *UsageRecord) LatencyMs() int {
	return r.latencyMs
}

// Success returns whether the request was successful.
func (r *UsageRecord) Success() bool {
	return r.success
}

// CacheHit returns whether the cache was hit.
func (r *UsageRecord) CacheHit() bool {
	return r.cacheHit
}

// --- Setters ---

// SetAPIKeyID sets the API key ID.
func (r *UsageRecord) SetAPIKeyID(apiKeyID uuid.UUID) {
	r.apiKeyID = &apiKeyID
}

// SetCacheHit sets whether the cache was hit.
func (r *UsageRecord) SetCacheHit(hit bool) {
	r.cacheHit = hit
}
