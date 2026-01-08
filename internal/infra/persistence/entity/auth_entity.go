package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/domain/auth"
)

// RefreshTokenEntity is the GORM entity for refresh tokens.
type RefreshTokenEntity struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	CreatedAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time `gorm:"index"`
	UserAgent string     `gorm:"size:512"`
	IPAddress string     `gorm:"type:inet"`
}

// TableName returns the database table name.
func (RefreshTokenEntity) TableName() string {
	return "refresh_tokens"
}

// ToDomain converts the entity to a domain model.
func (e *RefreshTokenEntity) ToDomain() *auth.RefreshToken {
	return auth.ReconstructRefreshToken(
		e.ID,
		e.UserID,
		e.TokenHash,
		e.ExpiresAt,
		e.CreatedAt,
		e.RevokedAt,
		e.UserAgent,
		e.IPAddress,
	)
}

// FromDomainRefreshToken converts a domain model to an entity.
func FromDomainRefreshToken(t *auth.RefreshToken) *RefreshTokenEntity {
	return &RefreshTokenEntity{
		ID:        t.ID(),
		UserID:    t.UserID(),
		TokenHash: t.TokenHash(),
		ExpiresAt: t.ExpiresAt(),
		CreatedAt: t.CreatedAt(),
		RevokedAt: t.RevokedAt(),
		UserAgent: t.UserAgent(),
		IPAddress: t.IPAddress(),
	}
}

// UserAPIKeyEntity is the GORM entity for user API keys.
type UserAPIKeyEntity struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	Provider     string         `gorm:"not null"`
	Name         string         `gorm:"not null"`
	EncryptedKey string         `gorm:"not null"`
	KeyPrefix    string         `gorm:"size:20"`
	Scopes       pq.StringArray `gorm:"type:text[]"`
	LastUsedAt   *time.Time
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// TableName returns the database table name.
func (UserAPIKeyEntity) TableName() string {
	return "user_api_keys"
}

// ToDomain converts the entity to a domain model.
func (e *UserAPIKeyEntity) ToDomain() *auth.UserAPIKey {
	return auth.ReconstructUserAPIKey(
		e.ID,
		e.UserID,
		e.Provider,
		e.Name,
		e.EncryptedKey,
		e.KeyPrefix,
		[]string(e.Scopes),
		e.LastUsedAt,
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// FromDomainUserAPIKey converts a domain model to an entity.
func FromDomainUserAPIKey(k *auth.UserAPIKey) *UserAPIKeyEntity {
	return &UserAPIKeyEntity{
		ID:           k.ID(),
		UserID:       k.UserID(),
		Provider:     k.Provider(),
		Name:         k.Name(),
		EncryptedKey: k.EncryptedKey(),
		KeyPrefix:    k.KeyPrefix(),
		Scopes:       pq.StringArray(k.Scopes()),
		LastUsedAt:   k.LastUsedAt(),
		CreatedAt:    k.CreatedAt(),
		UpdatedAt:    k.UpdatedAt(),
	}
}

// SystemAPIKeyEntity is the GORM entity for system API keys.
type SystemAPIKeyEntity struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name      string         `gorm:"not null"`
	KeyHash   string         `gorm:"uniqueIndex;not null"`
	KeyPrefix string         `gorm:"not null;size:20"`
	Scopes    pq.StringArray `gorm:"type:text[]"`

	// Rate limiting
	RateLimitRPM int `gorm:"default:60"`
	RateLimitTPM int `gorm:"default:100000"`

	// Usage statistics
	TotalRequests     int64   `gorm:"default:0"`
	TotalInputTokens  int64   `gorm:"default:0"`
	TotalOutputTokens int64   `gorm:"default:0"`
	TotalCostUSD      float64 `gorm:"type:decimal(12,6);default:0"`

	// Cache statistics
	CacheHits   int64 `gorm:"default:0"`
	CacheMisses int64 `gorm:"default:0"`

	// Status
	IsActive   bool       `gorm:"default:true"`
	LastUsedAt *time.Time
	ExpiresAt  *time.Time

	// IP Whitelist
	AllowedIPs pq.StringArray `gorm:"type:text[];default:'{}'"`

	// Auto-rotation
	RotateAfterDays *int
	LastRotatedAt   *time.Time

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName returns the database table name.
func (SystemAPIKeyEntity) TableName() string {
	return "system_api_keys"
}

// ToDomain converts the entity to a domain model.
func (e *SystemAPIKeyEntity) ToDomain() *auth.SystemAPIKey {
	return auth.ReconstructSystemAPIKey(
		e.ID,
		e.UserID,
		e.Name,
		e.KeyHash,
		e.KeyPrefix,
		[]string(e.Scopes),
		e.RateLimitRPM,
		e.RateLimitTPM,
		e.TotalRequests,
		e.TotalInputTokens,
		e.TotalOutputTokens,
		e.TotalCostUSD,
		e.CacheHits,
		e.CacheMisses,
		e.IsActive,
		e.LastUsedAt,
		e.ExpiresAt,
		[]string(e.AllowedIPs),
		e.RotateAfterDays,
		e.LastRotatedAt,
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// FromDomainSystemAPIKey converts a domain model to an entity.
func FromDomainSystemAPIKey(k *auth.SystemAPIKey) *SystemAPIKeyEntity {
	return &SystemAPIKeyEntity{
		ID:                k.ID(),
		UserID:            k.UserID(),
		Name:              k.Name(),
		KeyHash:           k.KeyHash(),
		KeyPrefix:         k.KeyPrefix(),
		Scopes:            pq.StringArray(k.Scopes()),
		RateLimitRPM:      k.RateLimitRPM(),
		RateLimitTPM:      k.RateLimitTPM(),
		TotalRequests:     k.TotalRequests(),
		TotalInputTokens:  k.TotalInputTokens(),
		TotalOutputTokens: k.TotalOutputTokens(),
		TotalCostUSD:      k.TotalCostUSD(),
		CacheHits:         k.CacheHits(),
		CacheMisses:       k.CacheMisses(),
		IsActive:          k.IsActive(),
		LastUsedAt:        k.LastUsedAt(),
		ExpiresAt:         k.ExpiresAt(),
		AllowedIPs:        pq.StringArray(k.AllowedIPs()),
		RotateAfterDays:   k.RotateAfterDays(),
		LastRotatedAt:     k.LastRotatedAt(),
		CreatedAt:         k.CreatedAt(),
		UpdatedAt:         k.UpdatedAt(),
	}
}
