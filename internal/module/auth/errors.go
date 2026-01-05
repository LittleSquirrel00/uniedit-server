package auth

import "errors"

// Auth module errors.
var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")

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

	// API Key errors
	ErrAPIKeyNotFound      = errors.New("API key not found")
	ErrAPIKeyAlreadyExists = errors.New("API key already exists for this provider")
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrDecryptionFailed    = errors.New("failed to decrypt API key")
	ErrEncryptionFailed    = errors.New("failed to encrypt API key")

	// Authorization errors
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
