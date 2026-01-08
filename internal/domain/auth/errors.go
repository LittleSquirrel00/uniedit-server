package auth

import "errors"

// Domain errors.
var (
	// Token errors
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrRevokedToken       = errors.New("token has been revoked")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrInvalidTokenClaims = errors.New("invalid token claims")

	// OAuth errors
	ErrInvalidOAuthProvider = errors.New("invalid OAuth provider")
	ErrInvalidOAuthCode     = errors.New("invalid OAuth code")
	ErrInvalidOAuthState    = errors.New("invalid OAuth state")
	ErrOAuthFailed          = errors.New("OAuth authentication failed")

	// User API Key errors
	ErrAPIKeyNotFound      = errors.New("API key not found")
	ErrAPIKeyAlreadyExists = errors.New("API key already exists for this provider")
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrDecryptionFailed    = errors.New("failed to decrypt API key")
	ErrEncryptionFailed    = errors.New("failed to encrypt API key")

	// System API Key errors
	ErrSystemAPIKeyNotFound      = errors.New("system API key not found")
	ErrSystemAPIKeyDisabled      = errors.New("system API key is disabled")
	ErrSystemAPIKeyExpired       = errors.New("system API key has expired")
	ErrSystemAPIKeyLimitExceeded = errors.New("maximum number of API keys reached")
	ErrInvalidAPIKeyFormat       = errors.New("invalid API key format")
	ErrInvalidAPIKeyScope        = errors.New("invalid API key scope")
	ErrRateLimitExceeded         = errors.New("rate limit exceeded")
	ErrTPMExceeded               = errors.New("tokens per minute limit exceeded")

	// Authorization errors
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
