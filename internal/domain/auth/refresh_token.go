package auth

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a JWT refresh token.
type RefreshToken struct {
	id        uuid.UUID
	userID    uuid.UUID
	tokenHash string
	expiresAt time.Time
	createdAt time.Time
	revokedAt *time.Time
	userAgent string
	ipAddress string
}

// NewRefreshToken creates a new refresh token.
func NewRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time, userAgent, ipAddress string) *RefreshToken {
	return &RefreshToken{
		id:        uuid.New(),
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
		createdAt: time.Now(),
		userAgent: userAgent,
		ipAddress: ipAddress,
	}
}

// ReconstructRefreshToken reconstructs a refresh token from storage.
func ReconstructRefreshToken(
	id, userID uuid.UUID,
	tokenHash string,
	expiresAt, createdAt time.Time,
	revokedAt *time.Time,
	userAgent, ipAddress string,
) *RefreshToken {
	return &RefreshToken{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
		createdAt: createdAt,
		revokedAt: revokedAt,
		userAgent: userAgent,
		ipAddress: ipAddress,
	}
}

// Getters

func (t *RefreshToken) ID() uuid.UUID        { return t.id }
func (t *RefreshToken) UserID() uuid.UUID    { return t.userID }
func (t *RefreshToken) TokenHash() string    { return t.tokenHash }
func (t *RefreshToken) ExpiresAt() time.Time { return t.expiresAt }
func (t *RefreshToken) CreatedAt() time.Time { return t.createdAt }
func (t *RefreshToken) RevokedAt() *time.Time { return t.revokedAt }
func (t *RefreshToken) UserAgent() string    { return t.userAgent }
func (t *RefreshToken) IPAddress() string    { return t.ipAddress }

// Business logic

// IsExpired checks if the token has expired.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

// IsRevoked checks if the token has been revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.revokedAt != nil
}

// IsValid checks if the token is still valid (not expired and not revoked).
func (t *RefreshToken) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked()
}

// Revoke marks the token as revoked.
func (t *RefreshToken) Revoke() {
	if t.revokedAt == nil {
		now := time.Now()
		t.revokedAt = &now
	}
}
