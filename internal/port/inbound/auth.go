package inbound

import "github.com/gin-gonic/gin"

// AuthHttpPort defines HTTP handler interface for authentication.
type AuthHttpPort interface {
	// InitiateLogin handles POST /auth/oauth/:provider
	InitiateLogin(c *gin.Context)

	// CompleteLogin handles POST /auth/oauth/:provider/callback
	CompleteLogin(c *gin.Context)

	// RefreshToken handles POST /auth/refresh
	RefreshToken(c *gin.Context)

	// Logout handles POST /auth/logout
	Logout(c *gin.Context)

	// GetMe handles GET /auth/me
	GetMe(c *gin.Context)
}

// APIKeyHttpPort defines HTTP handler interface for user API keys.
type APIKeyHttpPort interface {
	// CreateAPIKey handles POST /api-keys
	CreateAPIKey(c *gin.Context)

	// ListAPIKeys handles GET /api-keys
	ListAPIKeys(c *gin.Context)

	// GetAPIKey handles GET /api-keys/:id
	GetAPIKey(c *gin.Context)

	// DeleteAPIKey handles DELETE /api-keys/:id
	DeleteAPIKey(c *gin.Context)

	// RotateAPIKey handles POST /api-keys/:id/rotate
	RotateAPIKey(c *gin.Context)
}

// SystemAPIKeyHttpPort defines HTTP handler interface for system API keys.
type SystemAPIKeyHttpPort interface {
	// CreateSystemAPIKey handles POST /system-api-keys
	CreateSystemAPIKey(c *gin.Context)

	// ListSystemAPIKeys handles GET /system-api-keys
	ListSystemAPIKeys(c *gin.Context)

	// GetSystemAPIKey handles GET /system-api-keys/:id
	GetSystemAPIKey(c *gin.Context)

	// UpdateSystemAPIKey handles PATCH /system-api-keys/:id
	UpdateSystemAPIKey(c *gin.Context)

	// DeleteSystemAPIKey handles DELETE /system-api-keys/:id
	DeleteSystemAPIKey(c *gin.Context)

	// RotateSystemAPIKey handles POST /system-api-keys/:id/rotate
	RotateSystemAPIKey(c *gin.Context)
}
