package auth

import (
	"time"
)

// TokenPair represents access and refresh token pair.
type TokenPair struct {
	accessToken  string
	refreshToken string
	tokenType    string
	expiresIn    int64     // seconds until access token expires
	expiresAt    time.Time
}

// NewTokenPair creates a new token pair.
func NewTokenPair(accessToken, refreshToken string, expiresIn int64, expiresAt time.Time) *TokenPair {
	return &TokenPair{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		tokenType:    "Bearer",
		expiresIn:    expiresIn,
		expiresAt:    expiresAt,
	}
}

// Getters

func (t *TokenPair) AccessToken() string  { return t.accessToken }
func (t *TokenPair) RefreshToken() string { return t.refreshToken }
func (t *TokenPair) TokenType() string    { return t.tokenType }
func (t *TokenPair) ExpiresIn() int64     { return t.expiresIn }
func (t *TokenPair) ExpiresAt() time.Time { return t.expiresAt }

// Claims represents JWT token claims.
type Claims struct {
	userID    string
	email     string
	issuedAt  time.Time
	expiresAt time.Time
}

// NewClaims creates new claims.
func NewClaims(userID, email string, issuedAt, expiresAt time.Time) *Claims {
	return &Claims{
		userID:    userID,
		email:     email,
		issuedAt:  issuedAt,
		expiresAt: expiresAt,
	}
}

func (c *Claims) UserID() string       { return c.userID }
func (c *Claims) Email() string        { return c.email }
func (c *Claims) IssuedAt() time.Time  { return c.issuedAt }
func (c *Claims) ExpiresAt() time.Time { return c.expiresAt }

// IsExpired checks if the claims have expired.
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.expiresAt)
}
