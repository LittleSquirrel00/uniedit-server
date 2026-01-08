package auth

import (
	"time"

	"github.com/google/uuid"
)

// SystemAPIKey represents a system-generated API key for API access (OpenAI-style).
// Format: sk-xxxx... (similar to OpenAI API keys)
type SystemAPIKey struct {
	id        uuid.UUID
	userID    uuid.UUID
	name      string
	keyHash   string
	keyPrefix string
	scopes    []string

	// Rate limiting
	rateLimitRPM int
	rateLimitTPM int

	// Usage statistics
	totalRequests     int64
	totalInputTokens  int64
	totalOutputTokens int64
	totalCostUSD      float64

	// Cache statistics
	cacheHits   int64
	cacheMisses int64

	// Status
	isActive   bool
	lastUsedAt *time.Time
	expiresAt  *time.Time

	// IP Whitelist
	allowedIPs []string

	// Auto-rotation
	rotateAfterDays *int
	lastRotatedAt   *time.Time

	createdAt time.Time
	updatedAt time.Time
}

// NewSystemAPIKey creates a new system API key.
func NewSystemAPIKey(
	userID uuid.UUID,
	name, keyHash, keyPrefix string,
	scopes []string,
	rateLimitRPM, rateLimitTPM int,
	expiresAt *time.Time,
) *SystemAPIKey {
	now := time.Now()
	return &SystemAPIKey{
		id:           uuid.New(),
		userID:       userID,
		name:         name,
		keyHash:      keyHash,
		keyPrefix:    keyPrefix,
		scopes:       scopes,
		rateLimitRPM: rateLimitRPM,
		rateLimitTPM: rateLimitTPM,
		isActive:     true,
		allowedIPs:   []string{},
		expiresAt:    expiresAt,
		createdAt:    now,
		updatedAt:    now,
	}
}

// ReconstructSystemAPIKey reconstructs a system API key from storage.
func ReconstructSystemAPIKey(
	id, userID uuid.UUID,
	name, keyHash, keyPrefix string,
	scopes []string,
	rateLimitRPM, rateLimitTPM int,
	totalRequests, totalInputTokens, totalOutputTokens int64,
	totalCostUSD float64,
	cacheHits, cacheMisses int64,
	isActive bool,
	lastUsedAt, expiresAt *time.Time,
	allowedIPs []string,
	rotateAfterDays *int,
	lastRotatedAt *time.Time,
	createdAt, updatedAt time.Time,
) *SystemAPIKey {
	return &SystemAPIKey{
		id:                id,
		userID:            userID,
		name:              name,
		keyHash:           keyHash,
		keyPrefix:         keyPrefix,
		scopes:            scopes,
		rateLimitRPM:      rateLimitRPM,
		rateLimitTPM:      rateLimitTPM,
		totalRequests:     totalRequests,
		totalInputTokens:  totalInputTokens,
		totalOutputTokens: totalOutputTokens,
		totalCostUSD:      totalCostUSD,
		cacheHits:         cacheHits,
		cacheMisses:       cacheMisses,
		isActive:          isActive,
		lastUsedAt:        lastUsedAt,
		expiresAt:         expiresAt,
		allowedIPs:        allowedIPs,
		rotateAfterDays:   rotateAfterDays,
		lastRotatedAt:     lastRotatedAt,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

// Getters

func (k *SystemAPIKey) ID() uuid.UUID        { return k.id }
func (k *SystemAPIKey) UserID() uuid.UUID    { return k.userID }
func (k *SystemAPIKey) Name() string         { return k.name }
func (k *SystemAPIKey) KeyHash() string      { return k.keyHash }
func (k *SystemAPIKey) KeyPrefix() string    { return k.keyPrefix }
func (k *SystemAPIKey) Scopes() []string     { return k.scopes }
func (k *SystemAPIKey) RateLimitRPM() int    { return k.rateLimitRPM }
func (k *SystemAPIKey) RateLimitTPM() int    { return k.rateLimitTPM }
func (k *SystemAPIKey) TotalRequests() int64 { return k.totalRequests }
func (k *SystemAPIKey) TotalInputTokens() int64  { return k.totalInputTokens }
func (k *SystemAPIKey) TotalOutputTokens() int64 { return k.totalOutputTokens }
func (k *SystemAPIKey) TotalCostUSD() float64    { return k.totalCostUSD }
func (k *SystemAPIKey) CacheHits() int64     { return k.cacheHits }
func (k *SystemAPIKey) CacheMisses() int64   { return k.cacheMisses }
func (k *SystemAPIKey) IsActive() bool       { return k.isActive }
func (k *SystemAPIKey) LastUsedAt() *time.Time { return k.lastUsedAt }
func (k *SystemAPIKey) ExpiresAt() *time.Time  { return k.expiresAt }
func (k *SystemAPIKey) AllowedIPs() []string { return k.allowedIPs }
func (k *SystemAPIKey) RotateAfterDays() *int { return k.rotateAfterDays }
func (k *SystemAPIKey) LastRotatedAt() *time.Time { return k.lastRotatedAt }
func (k *SystemAPIKey) CreatedAt() time.Time { return k.createdAt }
func (k *SystemAPIKey) UpdatedAt() time.Time { return k.updatedAt }

// Business logic

// IsExpired checks if the API key has expired.
func (k *SystemAPIKey) IsExpired() bool {
	if k.expiresAt == nil {
		return false
	}
	return time.Now().After(*k.expiresAt)
}

// IsValid checks if the API key is valid (active and not expired).
func (k *SystemAPIKey) IsValid() bool {
	return k.isActive && !k.IsExpired()
}

// HasScope checks if the API key has the specified scope.
func (k *SystemAPIKey) HasScope(scope APIKeyScope) bool {
	for _, s := range k.scopes {
		if s == string(scope) {
			return true
		}
	}
	return false
}

// BelongsTo checks if the key belongs to the specified user.
func (k *SystemAPIKey) BelongsTo(userID uuid.UUID) bool {
	return k.userID == userID
}

// Update methods

// SetName updates the name.
func (k *SystemAPIKey) SetName(name string) {
	k.name = name
	k.updatedAt = time.Now()
}

// SetScopes updates the scopes.
func (k *SystemAPIKey) SetScopes(scopes []string) {
	k.scopes = scopes
	k.updatedAt = time.Now()
}

// SetRateLimits updates the rate limits.
func (k *SystemAPIKey) SetRateLimits(rpm, tpm int) {
	k.rateLimitRPM = rpm
	k.rateLimitTPM = tpm
	k.updatedAt = time.Now()
}

// Activate activates the key.
func (k *SystemAPIKey) Activate() {
	k.isActive = true
	k.updatedAt = time.Now()
}

// Deactivate deactivates the key.
func (k *SystemAPIKey) Deactivate() {
	k.isActive = false
	k.updatedAt = time.Now()
}

// MarkUsed marks the key as used.
func (k *SystemAPIKey) MarkUsed() {
	now := time.Now()
	k.lastUsedAt = &now
}

// RotateKey updates the key hash and prefix after rotation.
func (k *SystemAPIKey) RotateKey(newHash, newPrefix string) {
	k.keyHash = newHash
	k.keyPrefix = newPrefix
	now := time.Now()
	k.lastRotatedAt = &now
	k.updatedAt = now
}

// RecordUsage records usage statistics.
func (k *SystemAPIKey) RecordUsage(inputTokens, outputTokens int64, costUSD float64) {
	k.totalRequests++
	k.totalInputTokens += inputTokens
	k.totalOutputTokens += outputTokens
	k.totalCostUSD += costUSD
	k.updatedAt = time.Now()
}

// RecordCacheHit records a cache hit.
func (k *SystemAPIKey) RecordCacheHit() {
	k.cacheHits++
}

// RecordCacheMiss records a cache miss.
func (k *SystemAPIKey) RecordCacheMiss() {
	k.cacheMisses++
}

// SetAllowedIPs updates the IP whitelist.
func (k *SystemAPIKey) SetAllowedIPs(ips []string) {
	k.allowedIPs = ips
	k.updatedAt = time.Now()
}

// SetRotateAfterDays sets auto-rotation schedule.
func (k *SystemAPIKey) SetRotateAfterDays(days *int) {
	k.rotateAfterDays = days
	k.updatedAt = time.Now()
}
