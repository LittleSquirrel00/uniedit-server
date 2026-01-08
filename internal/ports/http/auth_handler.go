package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authCmd "github.com/uniedit/server/internal/app/command/auth"
	authQuery "github.com/uniedit/server/internal/app/query/auth"
	"github.com/uniedit/server/internal/domain/auth"
)

// AuthHandler handles HTTP requests for authentication using CQRS pattern.
type AuthHandler struct {
	// OAuth Commands
	initiateOAuth *authCmd.InitiateOAuthHandler
	completeOAuth *authCmd.CompleteOAuthHandler

	// Token Commands
	refreshTokens *authCmd.RefreshTokensHandler
	logout        *authCmd.LogoutHandler

	// User API Key Commands
	createUserAPIKey  *authCmd.CreateUserAPIKeyHandler
	deleteUserAPIKey  *authCmd.DeleteUserAPIKeyHandler
	rotateUserAPIKey  *authCmd.RotateUserAPIKeyHandler

	// System API Key Commands
	createSystemAPIKey   *authCmd.CreateSystemAPIKeyHandler
	updateSystemAPIKey   *authCmd.UpdateSystemAPIKeyHandler
	deleteSystemAPIKey   *authCmd.DeleteSystemAPIKeyHandler
	rotateSystemAPIKey   *authCmd.RotateSystemAPIKeyHandler
	validateSystemAPIKey *authCmd.ValidateSystemAPIKeyHandler

	// Queries
	listUserAPIKeys   *authQuery.ListUserAPIKeysHandler
	listSystemAPIKeys *authQuery.ListSystemAPIKeysHandler
	getSystemAPIKey   *authQuery.GetSystemAPIKeyHandler

	// Token Validator
	tokenValidator TokenValidator

	// Error handler
	errorHandler *ErrorHandler
}

// TokenValidator defines the interface for validating access tokens.
type TokenValidator interface {
	ValidateAccessToken(token string) (*auth.Claims, error)
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(
	initiateOAuth *authCmd.InitiateOAuthHandler,
	completeOAuth *authCmd.CompleteOAuthHandler,
	refreshTokens *authCmd.RefreshTokensHandler,
	logout *authCmd.LogoutHandler,
	createUserAPIKey *authCmd.CreateUserAPIKeyHandler,
	deleteUserAPIKey *authCmd.DeleteUserAPIKeyHandler,
	rotateUserAPIKey *authCmd.RotateUserAPIKeyHandler,
	createSystemAPIKey *authCmd.CreateSystemAPIKeyHandler,
	updateSystemAPIKey *authCmd.UpdateSystemAPIKeyHandler,
	deleteSystemAPIKey *authCmd.DeleteSystemAPIKeyHandler,
	rotateSystemAPIKey *authCmd.RotateSystemAPIKeyHandler,
	validateSystemAPIKey *authCmd.ValidateSystemAPIKeyHandler,
	listUserAPIKeys *authQuery.ListUserAPIKeysHandler,
	listSystemAPIKeys *authQuery.ListSystemAPIKeysHandler,
	getSystemAPIKey *authQuery.GetSystemAPIKeyHandler,
	tokenValidator TokenValidator,
) *AuthHandler {
	return &AuthHandler{
		initiateOAuth:        initiateOAuth,
		completeOAuth:        completeOAuth,
		refreshTokens:        refreshTokens,
		logout:               logout,
		createUserAPIKey:     createUserAPIKey,
		deleteUserAPIKey:     deleteUserAPIKey,
		rotateUserAPIKey:     rotateUserAPIKey,
		createSystemAPIKey:   createSystemAPIKey,
		updateSystemAPIKey:   updateSystemAPIKey,
		deleteSystemAPIKey:   deleteSystemAPIKey,
		rotateSystemAPIKey:   rotateSystemAPIKey,
		validateSystemAPIKey: validateSystemAPIKey,
		listUserAPIKeys:      listUserAPIKeys,
		listSystemAPIKeys:    listSystemAPIKeys,
		getSystemAPIKey:      getSystemAPIKey,
		tokenValidator:       tokenValidator,
		errorHandler:         NewErrorHandler(),
	}
}

// RegisterRoutes registers public auth routes.
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", h.InitiateLogin)
		authGroup.POST("/callback", h.Callback)
		authGroup.POST("/refresh", h.RefreshToken)
	}
}

// RegisterProtectedRoutes registers protected auth routes.
func (h *AuthHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/logout", h.Logout)
	}

	// Provider API keys (stored third-party keys)
	keys := r.Group("/keys")
	{
		keys.GET("", h.ListUserAPIKeys)
		keys.POST("", h.CreateUserAPIKey)
		keys.DELETE("/:id", h.DeleteUserAPIKey)
		keys.POST("/:id/rotate", h.RotateUserAPIKey)
	}

	// System API keys (OpenAI-style sk-xxx keys)
	apiKeys := r.Group("/api-keys")
	{
		apiKeys.GET("", h.ListSystemAPIKeys)
		apiKeys.POST("", h.CreateSystemAPIKey)
		apiKeys.GET("/:id", h.GetSystemAPIKey)
		apiKeys.PATCH("/:id", h.UpdateSystemAPIKey)
		apiKeys.DELETE("/:id", h.DeleteSystemAPIKey)
		apiKeys.POST("/:id/rotate", h.RotateSystemAPIKey)
	}
}

// --- Request/Response Types ---

type LoginRequest struct {
	Provider string `json:"provider" binding:"required"`
}

type CallbackRequest struct {
	Provider string `json:"provider" binding:"required"`
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateUserAPIKeyRequest struct {
	Provider string   `json:"provider" binding:"required"`
	Name     string   `json:"name" binding:"required"`
	APIKey   string   `json:"api_key" binding:"required"`
	Scopes   []string `json:"scopes,omitempty"`
}

type RotateUserAPIKeyRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

type CreateSystemAPIKeyRequest struct {
	Name          string   `json:"name" binding:"required"`
	Scopes        []string `json:"scopes,omitempty"`
	RateLimitRPM  *int     `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM  *int     `json:"rate_limit_tpm,omitempty"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

type UpdateSystemAPIKeyRequest struct {
	Name         *string  `json:"name,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	RateLimitRPM *int     `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM *int     `json:"rate_limit_tpm,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
}

// --- OAuth Handlers ---

func (h *AuthHandler) InitiateLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.initiateOAuth.Handle(c.Request.Context(), authCmd.InitiateOAuthCommand{
		Provider: auth.OAuthProvider(req.Provider),
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"auth_url": result.AuthURL,
		"state":    result.State,
	})
}

func (h *AuthHandler) Callback(c *gin.Context) {
	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.completeOAuth.Handle(c.Request.Context(), authCmd.CompleteOAuthCommand{
		Provider:  auth.OAuthProvider(req.Provider),
		Code:      req.Code,
		State:     req.State,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"tokens": tokenPairToResponse(result.TokenPair),
		"user":   userToResponse(result.User),
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.refreshTokens.Handle(c.Request.Context(), authCmd.RefreshTokensCommand{
		RefreshToken: req.RefreshToken,
		UserAgent:    c.Request.UserAgent(),
		IPAddress:    c.ClientIP(),
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, tokenPairToResponse(result.TokenPair))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	_, err := h.logout.Handle(c.Request.Context(), authCmd.LogoutCommand{
		UserID: userID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "logged out"})
}

// --- User API Key Handlers ---

func (h *AuthHandler) ListUserAPIKeys(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	result, err := h.listUserAPIKeys.Handle(c.Request.Context(), authQuery.ListUserAPIKeysQuery{
		UserID: userID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	responses := make([]gin.H, len(result.Keys))
	for i, key := range result.Keys {
		responses[i] = userAPIKeyToResponse(key)
	}

	respondSuccess(c, responses)
}

func (h *AuthHandler) CreateUserAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreateUserAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.createUserAPIKey.Handle(c.Request.Context(), authCmd.CreateUserAPIKeyCommand{
		UserID:   userID,
		Provider: req.Provider,
		Name:     req.Name,
		APIKey:   req.APIKey,
		Scopes:   req.Scopes,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondCreated(c, userAPIKeyToResponse(result.Key))
}

func (h *AuthHandler) DeleteUserAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	_, err = h.deleteUserAPIKey.Handle(c.Request.Context(), authCmd.DeleteUserAPIKeyCommand{
		UserID: userID,
		KeyID:  keyID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RotateUserAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	var req RotateUserAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.rotateUserAPIKey.Handle(c.Request.Context(), authCmd.RotateUserAPIKeyCommand{
		UserID:    userID,
		KeyID:     keyID,
		NewAPIKey: req.APIKey,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, userAPIKeyToResponse(result.Key))
}

// --- System API Key Handlers ---

func (h *AuthHandler) ListSystemAPIKeys(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	result, err := h.listSystemAPIKeys.Handle(c.Request.Context(), authQuery.ListSystemAPIKeysQuery{
		UserID: userID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	responses := make([]gin.H, len(result.Keys))
	for i, key := range result.Keys {
		responses[i] = systemAPIKeyToResponse(key)
	}

	respondSuccess(c, responses)
}

func (h *AuthHandler) CreateSystemAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreateSystemAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.createSystemAPIKey.Handle(c.Request.Context(), authCmd.CreateSystemAPIKeyCommand{
		UserID:        userID,
		Name:          req.Name,
		Scopes:        req.Scopes,
		RateLimitRPM:  req.RateLimitRPM,
		RateLimitTPM:  req.RateLimitTPM,
		ExpiresInDays: req.ExpiresInDays,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	resp := systemAPIKeyToResponse(result.Key)
	resp["key"] = result.RawKey // Only returned on creation
	respondCreated(c, resp)
}

func (h *AuthHandler) GetSystemAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	result, err := h.getSystemAPIKey.Handle(c.Request.Context(), authQuery.GetSystemAPIKeyQuery{
		UserID: userID,
		KeyID:  keyID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, systemAPIKeyToResponse(result.Key))
}

func (h *AuthHandler) UpdateSystemAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	var req UpdateSystemAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.updateSystemAPIKey.Handle(c.Request.Context(), authCmd.UpdateSystemAPIKeyCommand{
		UserID:       userID,
		KeyID:        keyID,
		Name:         req.Name,
		Scopes:       req.Scopes,
		RateLimitRPM: req.RateLimitRPM,
		RateLimitTPM: req.RateLimitTPM,
		IsActive:     req.IsActive,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	respondSuccess(c, systemAPIKeyToResponse(result.Key))
}

func (h *AuthHandler) DeleteSystemAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	_, err = h.deleteSystemAPIKey.Handle(c.Request.Context(), authCmd.DeleteSystemAPIKeyCommand{
		UserID: userID,
		KeyID:  keyID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RotateSystemAPIKey(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid key id")
		return
	}

	result, err := h.rotateSystemAPIKey.Handle(c.Request.Context(), authCmd.RotateSystemAPIKeyCommand{
		UserID: userID,
		KeyID:  keyID,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	resp := systemAPIKeyToResponse(result.Key)
	resp["key"] = result.RawKey
	respondSuccess(c, resp)
}

// --- Middleware ---

// AuthMiddleware returns a middleware that validates JWT tokens.
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "authorization header required")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := h.tokenValidator.ValidateAccessToken(token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
			c.Abort()
			return
		}

		// Set user info in context
		userID, _ := uuid.Parse(claims.UserID())
		c.Set("user_id", userID)
		c.Set("email", claims.Email())
		c.Set("auth_type", "jwt")
		c.Next()
	}
}

// APIKeyAuthMiddleware returns a middleware that validates API keys (sk-xxx format).
func (h *AuthHandler) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "authorization header required")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
			c.Abort()
			return
		}

		apiKey := parts[1]

		// Validate API key
		result, err := h.validateSystemAPIKey.Handle(c.Request.Context(), authCmd.ValidateSystemAPIKeyCommand{
			APIKey: apiKey,
		})
		if err != nil {
			h.handleAuthError(c, err)
			c.Abort()
			return
		}

		key := result.Key
		// Set key info in context
		c.Set("user_id", key.UserID())
		c.Set("api_key_id", key.ID())
		c.Set("api_key_scopes", key.Scopes())
		c.Set("rate_limit_rpm", key.RateLimitRPM())
		c.Set("rate_limit_tpm", key.RateLimitTPM())
		c.Set("auth_type", "api_key")
		c.Next()
	}
}

// HybridAuthMiddleware returns a middleware that supports both JWT and API key authentication.
func (h *AuthHandler) HybridAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "authorization header required")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// Check if it's an API key (sk- prefix)
		if strings.HasPrefix(token, auth.APIKeyPrefix) {
			result, err := h.validateSystemAPIKey.Handle(c.Request.Context(), authCmd.ValidateSystemAPIKeyCommand{
				APIKey: token,
			})
			if err != nil {
				h.handleAuthError(c, err)
				c.Abort()
				return
			}

			key := result.Key
			c.Set("user_id", key.UserID())
			c.Set("api_key_id", key.ID())
			c.Set("api_key_scopes", key.Scopes())
			c.Set("rate_limit_rpm", key.RateLimitRPM())
			c.Set("rate_limit_tpm", key.RateLimitTPM())
			c.Set("auth_type", "api_key")
		} else {
			claims, err := h.tokenValidator.ValidateAccessToken(token)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
				c.Abort()
				return
			}

			userID, _ := uuid.Parse(claims.UserID())
			c.Set("user_id", userID)
			c.Set("email", claims.Email())
			c.Set("auth_type", "jwt")
		}

		c.Next()
	}
}

// --- Response Helpers ---

func tokenPairToResponse(t *auth.TokenPair) gin.H {
	return gin.H{
		"access_token":  t.AccessToken(),
		"refresh_token": t.RefreshToken(),
		"token_type":    t.TokenType(),
		"expires_in":    t.ExpiresIn(),
		"expires_at":    t.ExpiresAt(),
	}
}

func userAPIKeyToResponse(k *auth.UserAPIKey) gin.H {
	resp := gin.H{
		"id":         k.ID(),
		"provider":   k.Provider(),
		"name":       k.Name(),
		"key_prefix": k.KeyPrefix(),
		"scopes":     k.Scopes(),
		"created_at": k.CreatedAt(),
	}
	if k.LastUsedAt() != nil {
		resp["last_used_at"] = k.LastUsedAt()
	}
	return resp
}

func systemAPIKeyToResponse(k *auth.SystemAPIKey) gin.H {
	resp := gin.H{
		"id":                  k.ID(),
		"name":                k.Name(),
		"key_prefix":          k.KeyPrefix(),
		"scopes":              k.Scopes(),
		"rate_limit_rpm":      k.RateLimitRPM(),
		"rate_limit_tpm":      k.RateLimitTPM(),
		"total_requests":      k.TotalRequests(),
		"total_input_tokens":  k.TotalInputTokens(),
		"total_output_tokens": k.TotalOutputTokens(),
		"total_cost_usd":      k.TotalCostUSD(),
		"cache_hits":          k.CacheHits(),
		"cache_misses":        k.CacheMisses(),
		"is_active":           k.IsActive(),
		"allowed_ips":         k.AllowedIPs(),
		"created_at":          k.CreatedAt(),
	}
	if k.LastUsedAt() != nil {
		resp["last_used_at"] = k.LastUsedAt()
	}
	if k.ExpiresAt() != nil {
		resp["expires_at"] = k.ExpiresAt()
	}
	if k.RotateAfterDays() != nil {
		resp["rotate_after_days"] = k.RotateAfterDays()
	}
	if k.LastRotatedAt() != nil {
		resp["last_rotated_at"] = k.LastRotatedAt()
	}
	return resp
}

// --- Error Handling ---

func (h *AuthHandler) handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInvalidTokenClaims):
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token")
	case errors.Is(err, auth.ErrExpiredToken):
		respondError(c, http.StatusUnauthorized, "token_expired", "Token has expired")
	case errors.Is(err, auth.ErrRevokedToken):
		respondError(c, http.StatusUnauthorized, "token_revoked", "Token has been revoked")
	case errors.Is(err, auth.ErrTokenNotFound):
		respondError(c, http.StatusUnauthorized, "token_not_found", "Token not found")
	case errors.Is(err, auth.ErrInvalidOAuthProvider):
		respondError(c, http.StatusBadRequest, "invalid_oauth_provider", "Invalid OAuth provider")
	case errors.Is(err, auth.ErrInvalidOAuthCode):
		respondError(c, http.StatusBadRequest, "invalid_oauth_code", "Invalid OAuth code")
	case errors.Is(err, auth.ErrInvalidOAuthState):
		respondError(c, http.StatusBadRequest, "invalid_oauth_state", "Invalid OAuth state")
	case errors.Is(err, auth.ErrOAuthFailed):
		respondError(c, http.StatusBadRequest, "oauth_failed", "OAuth authentication failed")
	case errors.Is(err, auth.ErrAPIKeyNotFound):
		respondError(c, http.StatusNotFound, "api_key_not_found", "API key not found")
	case errors.Is(err, auth.ErrAPIKeyAlreadyExists):
		respondError(c, http.StatusConflict, "api_key_already_exists", "API key already exists for this provider")
	case errors.Is(err, auth.ErrForbidden):
		respondError(c, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, auth.ErrSystemAPIKeyNotFound):
		respondError(c, http.StatusNotFound, "api_key_not_found", "System API key not found")
	case errors.Is(err, auth.ErrSystemAPIKeyDisabled):
		respondError(c, http.StatusForbidden, "api_key_disabled", "System API key is disabled")
	case errors.Is(err, auth.ErrSystemAPIKeyExpired):
		respondError(c, http.StatusUnauthorized, "api_key_expired", "System API key has expired")
	case errors.Is(err, auth.ErrSystemAPIKeyLimitExceeded):
		respondError(c, http.StatusConflict, "api_key_limit_exceeded", "Maximum number of API keys reached")
	case errors.Is(err, auth.ErrInvalidAPIKeyFormat):
		respondError(c, http.StatusBadRequest, "invalid_api_key_format", "Invalid API key format")
	case errors.Is(err, auth.ErrInvalidAPIKeyScope):
		respondError(c, http.StatusBadRequest, "invalid_api_key_scope", "Invalid API key scope")
	case errors.Is(err, auth.ErrRateLimitExceeded):
		respondError(c, http.StatusTooManyRequests, "rate_limit_exceeded", "Rate limit exceeded")
	case errors.Is(err, auth.ErrTPMExceeded):
		respondError(c, http.StatusTooManyRequests, "tpm_exceeded", "Tokens per minute limit exceeded")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An internal error occurred")
	}
}
