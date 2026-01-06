package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers auth routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.InitiateLogin)
		auth.POST("/callback", h.Callback)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
	}

	users := r.Group("/users")
	users.Use(h.AuthMiddleware())
	{
		users.GET("/me", h.GetCurrentUser)
	}

	// Provider API keys (stored third-party keys)
	keys := r.Group("/keys")
	keys.Use(h.AuthMiddleware())
	{
		keys.GET("", h.ListAPIKeys)
		keys.POST("", h.CreateAPIKey)
		keys.DELETE("/:id", h.DeleteAPIKey)
		keys.POST("/:id/rotate", h.RotateAPIKey)
	}

	// System API keys (OpenAI-style sk-xxx keys)
	apiKeys := r.Group("/api-keys")
	apiKeys.Use(h.AuthMiddleware())
	{
		apiKeys.GET("", h.ListSystemAPIKeys)
		apiKeys.POST("", h.CreateSystemAPIKey)
		apiKeys.GET("/:id", h.GetSystemAPIKey)
		apiKeys.PATCH("/:id", h.UpdateSystemAPIKey)
		apiKeys.DELETE("/:id", h.DeleteSystemAPIKey)
		apiKeys.POST("/:id/rotate", h.RotateSystemAPIKey)
	}
}

// --- Auth Endpoints ---

// InitiateLogin starts the OAuth login flow.
// POST /auth/login
func (h *Handler) InitiateLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.InitiateLogin(c.Request.Context(), req.Provider)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Callback handles the OAuth callback.
// POST /auth/callback
func (h *Handler) Callback(c *gin.Context) {
	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	tokens, user, err := h.service.CompleteLogin(c.Request.Context(), &req, userAgent, ipAddress)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"user":   user.ToResponse(),
	})
}

// RefreshToken refreshes the access token.
// POST /auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	tokens, err := h.service.RefreshTokens(c.Request.Context(), req.RefreshToken, userAgent, ipAddress)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Logout revokes all tokens for the user.
// POST /auth/logout
func (h *Handler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.Logout(c.Request.Context(), userID.(uuid.UUID)); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// --- User Endpoints ---

// GetCurrentUser returns the current user's profile.
// GET /users/me
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// --- API Key Endpoints ---

// ListAPIKeys returns all API keys for the current user.
// GET /keys
func (h *Handler) ListAPIKeys(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keys, err := h.service.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]*APIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// CreateAPIKey creates a new API key.
// POST /keys
func (h *Handler) CreateAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.service.CreateAPIKey(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, key.ToResponse())
}

// DeleteAPIKey deletes an API key.
// DELETE /keys/:id
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	if err := h.service.DeleteAPIKey(c.Request.Context(), userID, keyID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RotateAPIKey rotates an API key.
// POST /keys/:id/rotate
func (h *Handler) RotateAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.service.RotateAPIKey(c.Request.Context(), userID, keyID, req.APIKey)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

// --- Middleware ---

// AuthMiddleware returns a middleware that validates JWT tokens.
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := h.service.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// --- Error Handling ---

func (h *Handler) handleError(c *gin.Context, err error) {
	switch err {
	case ErrUserNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	case ErrInvalidToken, ErrInvalidTokenClaims:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	case ErrExpiredToken:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
	case ErrRevokedToken:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
	case ErrTokenNotFound:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token not found"})
	case ErrInvalidOAuthProvider:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth provider"})
	case ErrInvalidOAuthCode:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth code"})
	case ErrInvalidOAuthState:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
	case ErrOAuthFailed:
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth authentication failed"})
	case ErrAPIKeyNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
	case ErrAPIKeyAlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": "api key already exists for this provider"})
	case ErrForbidden:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case ErrSystemAPIKeyNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
	case ErrSystemAPIKeyDisabled:
		c.JSON(http.StatusForbidden, gin.H{"error": "api key is disabled"})
	case ErrSystemAPIKeyExpired:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "api key has expired"})
	case ErrSystemAPIKeyLimitExceeded:
		c.JSON(http.StatusConflict, gin.H{"error": "maximum number of api keys reached"})
	case ErrInvalidAPIKeyFormat:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid api key format"})
	case ErrInvalidAPIKeyScope:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid api key scope"})
	case ErrRateLimitExceeded:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
	case ErrTPMExceeded:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "tokens per minute limit exceeded"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// --- System API Key Endpoints ---

// ListSystemAPIKeys returns all system API keys for the current user.
// GET /api-keys
func (h *Handler) ListSystemAPIKeys(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keys, err := h.service.ListSystemAPIKeys(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]*SystemAPIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// CreateSystemAPIKey creates a new system API key.
// POST /api-keys
func (h *Handler) CreateSystemAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req CreateSystemAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateSystemAPIKey(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetSystemAPIKey returns a system API key by ID.
// GET /api-keys/:id
func (h *Handler) GetSystemAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	key, err := h.service.GetSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

// UpdateSystemAPIKey updates a system API key.
// PATCH /api-keys/:id
func (h *Handler) UpdateSystemAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var req UpdateSystemAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.service.UpdateSystemAPIKey(c.Request.Context(), userID, keyID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

// DeleteSystemAPIKey deletes a system API key.
// DELETE /api-keys/:id
func (h *Handler) DeleteSystemAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	if err := h.service.DeleteSystemAPIKey(c.Request.Context(), userID, keyID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RotateSystemAPIKey generates a new key for an existing system API key.
// POST /api-keys/:id/rotate
func (h *Handler) RotateSystemAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	resp, err := h.service.RotateSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// --- API Key Authentication Middleware ---

// APIKeyAuthMiddleware returns a middleware that validates API keys (sk-xxx format).
// This is used for API access instead of JWT tokens.
func (h *Handler) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		apiKey := parts[1]

		// Validate API key
		key, err := h.service.ValidateSystemAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			h.handleError(c, err)
			c.Abort()
			return
		}

		// Set key info in context
		c.Set("user_id", key.UserID)
		c.Set("api_key_id", key.ID)
		c.Set("api_key_scopes", []string(key.Scopes))
		c.Set("rate_limit_rpm", key.RateLimitRPM)
		c.Set("rate_limit_tpm", key.RateLimitTPM)
		c.Next()
	}
}

// RateLimitMiddleware returns a middleware that enforces rate limits for API key authentication.
// This should be used after HybridAuthMiddleware or APIKeyAuthMiddleware.
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply rate limiting for API key auth
		authType, exists := c.Get("auth_type")
		if !exists || authType != "api_key" {
			c.Next()
			return
		}

		keyID, exists := c.Get("api_key_id")
		if !exists {
			c.Next()
			return
		}

		rpmLimit, _ := c.Get("rate_limit_rpm")
		rpm, ok := rpmLimit.(int)
		if !ok {
			rpm = 60 // Default
		}

		// Check RPM limit
		result, err := limiter.CheckRPM(c.Request.Context(), keyID.(uuid.UUID), rpm)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit check failed"})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit-Requests", fmt.Sprintf("%d", result.Limit))
		c.Header("X-RateLimit-Remaining-Requests", fmt.Sprintf("%d", result.Remaining))
		c.Header("X-RateLimit-Reset-Requests", fmt.Sprintf("%d", result.ResetAt))

		if !result.Allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"type":  "requests",
				"limit": result.Limit,
				"reset": result.ResetAt,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HybridAuthMiddleware returns a middleware that supports both JWT and API key authentication.
// It checks for sk- prefix to determine if it's an API key, otherwise treats it as JWT.
func (h *Handler) HybridAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Check if it's an API key (sk- prefix)
		if strings.HasPrefix(token, APIKeyPrefix) {
			// Validate API key
			key, err := h.service.ValidateSystemAPIKey(c.Request.Context(), token)
			if err != nil {
				h.handleError(c, err)
				c.Abort()
				return
			}

			// Set key info in context
			c.Set("user_id", key.UserID)
			c.Set("api_key_id", key.ID)
			c.Set("api_key_scopes", []string(key.Scopes))
			c.Set("rate_limit_rpm", key.RateLimitRPM)
			c.Set("rate_limit_tpm", key.RateLimitTPM)
			c.Set("auth_type", "api_key")
		} else {
			// Validate JWT
			claims, err := h.service.ValidateAccessToken(token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				c.Abort()
				return
			}

			// Set user info in context
			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)
			c.Set("auth_type", "jwt")
		}

		c.Next()
	}
}
