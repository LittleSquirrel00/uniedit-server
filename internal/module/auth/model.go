package auth

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

// User represents a registered user.
type User struct {
	ID            uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email         string        `json:"email" gorm:"uniqueIndex;not null"`
	Name          string        `json:"name" gorm:"not null"`
	AvatarURL     string        `json:"avatar_url,omitempty"`
	OAuthProvider OAuthProvider `json:"oauth_provider" gorm:"not null"`
	OAuthID       string        `json:"-" gorm:"not null"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// TableName returns the database table name.
func (User) TableName() string {
	return "users"
}

// RefreshToken represents a JWT refresh token.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash string     `json:"-" gorm:"uniqueIndex;not null"` // SHA-256 hash
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

// IsValid checks if the token is still valid (not expired and not revoked).
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
	Provider     string         `json:"provider" gorm:"not null"`      // openai, anthropic, etc.
	Name         string         `json:"name" gorm:"not null"`          // user-defined name
	EncryptedKey string         `json:"-" gorm:"not null"`             // AES-256-GCM encrypted
	KeyPrefix    string         `json:"key_prefix,omitempty"`          // first few chars (e.g., sk-abc)
	Scopes       pq.StringArray `json:"scopes" gorm:"type:text[]"`     // permissions
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
	ExpiresIn    int64     `json:"expires_in"` // seconds until access token expires
	ExpiresAt    time.Time `json:"expires_at"`
}

// LoginRequest represents OAuth login initiation request.
type LoginRequest struct {
	Provider    OAuthProvider `json:"provider" binding:"required"`
	RedirectURL string        `json:"redirect_url,omitempty"`
}

// LoginResponse contains OAuth authorization URL.
type LoginResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// CallbackRequest represents OAuth callback request.
type CallbackRequest struct {
	Provider OAuthProvider `json:"provider" binding:"required"`
	Code     string        `json:"code" binding:"required"`
	State    string        `json:"state" binding:"required"`
}

// RefreshRequest represents token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// CreateAPIKeyRequest represents a request to create a new API key.
type CreateAPIKeyRequest struct {
	Provider string   `json:"provider" binding:"required"`
	Name     string   `json:"name" binding:"required"`
	APIKey   string   `json:"api_key" binding:"required"`
	Scopes   []string `json:"scopes,omitempty"`
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

// UserResponse represents user information for API responses.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts User to UserResponse.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Provider:  u.OAuthProvider.String(),
		CreatedAt: u.CreatedAt,
	}
}

// SystemAPIKey represents a system-generated API key for API access (OpenAI-style).
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

	// Usage statistics (denormalized)
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

	// IP Whitelist (empty = allow all)
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

// IsValid checks if the API key is valid (active and not expired).
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

// CreateSystemAPIKeyRequest represents a request to create a new system API key.
type CreateSystemAPIKeyRequest struct {
	Name         string   `json:"name" binding:"required"`
	Scopes       []string `json:"scopes,omitempty"`
	RateLimitRPM *int     `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM *int     `json:"rate_limit_tpm,omitempty"`
	ExpiresInDays *int    `json:"expires_in_days,omitempty"` // NULL = never expires
}

// UpdateSystemAPIKeyRequest represents a request to update a system API key.
type UpdateSystemAPIKeyRequest struct {
	Name         *string  `json:"name,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	RateLimitRPM *int     `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM *int     `json:"rate_limit_tpm,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
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

// SystemAPIKeyCreateResponse includes the full key (only returned on creation).
type SystemAPIKeyCreateResponse struct {
	*SystemAPIKeyResponse
	Key string `json:"key"` // Full API key, only shown once
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

// UpdateIPWhitelistRequest represents a request to update IP whitelist.
type UpdateIPWhitelistRequest struct {
	AllowedIPs []string `json:"allowed_ips" binding:"required"`
}

// ScheduleRotationRequest represents a request to schedule auto-rotation.
type ScheduleRotationRequest struct {
	RotateAfterDays *int `json:"rotate_after_days"` // NULL to disable
}

// AuditLogResponse represents an audit log entry in API responses.
type AuditLogResponse struct {
	ID        uuid.UUID         `json:"id"`
	Action    APIKeyAuditAction `json:"action"`
	Details   map[string]any    `json:"details,omitempty"`
	IPAddress string            `json:"ip_address,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ToResponse converts APIKeyAuditLog to AuditLogResponse.
func (l *APIKeyAuditLog) ToResponse() *AuditLogResponse {
	return &AuditLogResponse{
		ID:        l.ID,
		Action:    l.Action,
		Details:   l.Details,
		IPAddress: l.IPAddress,
		UserAgent: l.UserAgent,
		CreatedAt: l.CreatedAt,
	}
}
