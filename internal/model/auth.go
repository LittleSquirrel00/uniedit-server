package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OAuthProvider represents supported OAuth providers.
type OAuthProvider string

const (
	OAuthProviderGitHub OAuthProvider = "github"
	OAuthProviderGoogle OAuthProvider = "google"
)

// String returns the string representation of the provider.
func (p OAuthProvider) String() string {
	return string(p)
}

// IsValid checks if the provider is supported.
func (p OAuthProvider) IsValid() bool {
	switch p {
	case OAuthProviderGitHub, OAuthProviderGoogle:
		return true
	default:
		return false
	}
}

// RefreshToken represents a JWT refresh token.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash string     `json:"-" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	UserAgent string     `json:"user_agent,omitempty"`
	IPAddress string     `json:"ip_address,omitempty" gorm:"type:inet"`
}

// TableName returns the database table name.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired checks if the token has expired.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked checks if the token has been revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsValid checks if the token is still valid.
func (t *RefreshToken) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked()
}

// APIKeyScope represents the scope/permission of an API key.
type APIKeyScope string

const (
	APIKeyScopeChat      APIKeyScope = "chat"
	APIKeyScopeImage     APIKeyScope = "image"
	APIKeyScopeVideo     APIKeyScope = "video"
	APIKeyScopeAudio     APIKeyScope = "audio"
	APIKeyScopeEmbedding APIKeyScope = "embedding"
)

// UserAPIKey represents a user's stored API key for AI providers.
type UserAPIKey struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Provider     string         `json:"provider" gorm:"not null"`
	Name         string         `json:"name" gorm:"not null"`
	EncryptedKey string         `json:"-" gorm:"not null"`
	KeyPrefix    string         `json:"key_prefix,omitempty"`
	Scopes       pq.StringArray `json:"scopes" gorm:"type:text[]"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// TableName returns the database table name.
func (UserAPIKey) TableName() string {
	return "user_api_keys"
}

// HasScope checks if the API key has the specified scope.
func (k *UserAPIKey) HasScope(scope APIKeyScope) bool {
	for _, s := range k.Scopes {
		if s == string(scope) {
			return true
		}
	}
	return false
}

// APIKeyResponse represents API key information (without the actual key).
type APIKeyResponse struct {
	ID         uuid.UUID  `json:"id"`
	Provider   string     `json:"provider"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ToResponse converts UserAPIKey to APIKeyResponse.
func (k *UserAPIKey) ToResponse() *APIKeyResponse {
	scopes := make([]string, len(k.Scopes))
	copy(scopes, k.Scopes)
	return &APIKeyResponse{
		ID:         k.ID,
		Provider:   k.Provider,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     scopes,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
}

// SystemAPIKey represents a system-generated API key for API access.
type SystemAPIKey struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Name      string         `json:"name" gorm:"not null"`
	KeyHash   string         `json:"-" gorm:"uniqueIndex;not null"`
	KeyPrefix string         `json:"key_prefix" gorm:"not null"`
	Scopes    pq.StringArray `json:"scopes" gorm:"type:text[]"`

	// Rate limiting
	RateLimitRPM int `json:"rate_limit_rpm" gorm:"default:60"`
	RateLimitTPM int `json:"rate_limit_tpm" gorm:"default:100000"`

	// Usage statistics
	TotalRequests     int64   `json:"total_requests" gorm:"default:0"`
	TotalInputTokens  int64   `json:"total_input_tokens" gorm:"default:0"`
	TotalOutputTokens int64   `json:"total_output_tokens" gorm:"default:0"`
	TotalCostUSD      float64 `json:"total_cost_usd" gorm:"type:decimal(12,6);default:0"`

	// Cache statistics
	CacheHits   int64 `json:"cache_hits" gorm:"default:0"`
	CacheMisses int64 `json:"cache_misses" gorm:"default:0"`

	// Status
	IsActive   bool       `json:"is_active" gorm:"default:true"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`

	// IP Whitelist
	AllowedIPs pq.StringArray `json:"allowed_ips" gorm:"type:text[];default:'{}'"`

	// Auto-rotation
	RotateAfterDays *int       `json:"rotate_after_days,omitempty"`
	LastRotatedAt   *time.Time `json:"last_rotated_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the database table name.
func (SystemAPIKey) TableName() string {
	return "system_api_keys"
}

// IsExpired checks if the API key has expired.
func (k *SystemAPIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if the API key is valid.
func (k *SystemAPIKey) IsValid() bool {
	return k.IsActive && !k.IsExpired()
}

// HasScope checks if the API key has the specified scope.
func (k *SystemAPIKey) HasScope(scope APIKeyScope) bool {
	for _, s := range k.Scopes {
		if s == string(scope) {
			return true
		}
	}
	return false
}

// SystemAPIKeyResponse represents system API key information for API responses.
type SystemAPIKeyResponse struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	KeyPrefix         string     `json:"key_prefix"`
	Scopes            []string   `json:"scopes"`
	RateLimitRPM      int        `json:"rate_limit_rpm"`
	RateLimitTPM      int        `json:"rate_limit_tpm"`
	TotalRequests     int64      `json:"total_requests"`
	TotalInputTokens  int64      `json:"total_input_tokens"`
	TotalOutputTokens int64      `json:"total_output_tokens"`
	TotalCostUSD      float64    `json:"total_cost_usd"`
	CacheHits         int64      `json:"cache_hits"`
	CacheMisses       int64      `json:"cache_misses"`
	IsActive          bool       `json:"is_active"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	AllowedIPs        []string   `json:"allowed_ips"`
	RotateAfterDays   *int       `json:"rotate_after_days,omitempty"`
	LastRotatedAt     *time.Time `json:"last_rotated_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ToResponse converts SystemAPIKey to SystemAPIKeyResponse.
func (k *SystemAPIKey) ToResponse() *SystemAPIKeyResponse {
	scopes := make([]string, len(k.Scopes))
	copy(scopes, k.Scopes)
	return &SystemAPIKeyResponse{
		ID:                k.ID,
		Name:              k.Name,
		KeyPrefix:         k.KeyPrefix,
		Scopes:            scopes,
		RateLimitRPM:      k.RateLimitRPM,
		RateLimitTPM:      k.RateLimitTPM,
		TotalRequests:     k.TotalRequests,
		TotalInputTokens:  k.TotalInputTokens,
		TotalOutputTokens: k.TotalOutputTokens,
		TotalCostUSD:      k.TotalCostUSD,
		CacheHits:         k.CacheHits,
		CacheMisses:       k.CacheMisses,
		IsActive:          k.IsActive,
		LastUsedAt:        k.LastUsedAt,
		ExpiresAt:         k.ExpiresAt,
		AllowedIPs:        []string(k.AllowedIPs),
		RotateAfterDays:   k.RotateAfterDays,
		LastRotatedAt:     k.LastRotatedAt,
		CreatedAt:         k.CreatedAt,
	}
}

// OAuthUserInfo represents user information from OAuth provider.
type OAuthUserInfo struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
	Provider  OAuthProvider
}

// TokenPair represents access and refresh token pair.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// APIKeyAuditAction represents the type of audit action.
type APIKeyAuditAction string

const (
	AuditActionCreated  APIKeyAuditAction = "created"
	AuditActionUsed     APIKeyAuditAction = "used"
	AuditActionRotated  APIKeyAuditAction = "rotated"
	AuditActionDisabled APIKeyAuditAction = "disabled"
	AuditActionDeleted  APIKeyAuditAction = "deleted"
	AuditActionUpdated  APIKeyAuditAction = "updated"
)

// APIKeyAuditLog tracks API key operations.
type APIKeyAuditLog struct {
	ID        uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	APIKeyID  uuid.UUID         `json:"api_key_id" gorm:"type:uuid;not null;index"`
	Action    APIKeyAuditAction `json:"action" gorm:"not null"`
	Details   map[string]any    `json:"details,omitempty" gorm:"type:jsonb;serializer:json"`
	IPAddress string            `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent string            `json:"user_agent,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// TableName returns the database table name.
func (APIKeyAuditLog) TableName() string {
	return "api_key_audit_logs"
}
