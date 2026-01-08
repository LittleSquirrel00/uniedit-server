package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// authAdapter implements inbound.AuthHttpPort.
type authAdapter struct {
	authDomain auth.AuthDomain
}

// NewAuthAdapter creates a new auth HTTP adapter.
func NewAuthAdapter(authDomain auth.AuthDomain) inbound.AuthHttpPort {
	return &authAdapter{authDomain: authDomain}
}

// RegisterRoutes registers auth routes.
func (a *authAdapter) RegisterRoutes(r *gin.RouterGroup) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/oauth/:provider", a.InitiateLogin)
		authGroup.POST("/oauth/:provider/callback", a.CompleteLogin)
		authGroup.POST("/refresh", a.RefreshToken)
		authGroup.POST("/logout", a.Logout)
		authGroup.GET("/me", a.GetMe)
	}
}

func (a *authAdapter) InitiateLogin(c *gin.Context) {
	providerStr := c.Param("provider")
	provider := model.OAuthProvider(providerStr)

	if !provider.IsValid() {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_provider",
			Message: "Invalid OAuth provider",
		})
		return
	}

	resp, err := a.authDomain.InitiateLogin(c.Request.Context(), provider)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": resp.AuthURL,
		"state":    resp.State,
	})
}

func (a *authAdapter) CompleteLogin(c *gin.Context) {
	providerStr := c.Param("provider")
	provider := model.OAuthProvider(providerStr)

	if !provider.IsValid() {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_provider",
			Message: "Invalid OAuth provider",
		})
		return
	}

	var req struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	tokenPair, user, err := a.authDomain.CompleteLogin(c.Request.Context(), provider, req.Code, req.State, userAgent, ipAddress)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_in":    tokenPair.ExpiresIn,
		"expires_at":    tokenPair.ExpiresAt,
		"user":          user.ToResponse(),
	})
}

func (a *authAdapter) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	tokenPair, err := a.authDomain.RefreshTokens(c.Request.Context(), req.RefreshToken, userAgent, ipAddress)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_in":    tokenPair.ExpiresIn,
		"expires_at":    tokenPair.ExpiresAt,
	})
}

func (a *authAdapter) Logout(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	if err := a.authDomain.Logout(c.Request.Context(), userID); err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (a *authAdapter) GetMe(c *gin.Context) {
	// This is handled by the user adapter, redirect there
	// or implement token validation here
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "Authorization header required",
		})
		return
	}

	// Extract Bearer token
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims, err := a.authDomain.ValidateAccessToken(token)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}

// handleAuthError maps auth domain errors to HTTP responses.
func handleAuthError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, auth.ErrInvalidOAuthProvider):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_provider"
		message = "Invalid OAuth provider"

	case errors.Is(err, auth.ErrInvalidOAuthState):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_state"
		message = "Invalid or expired OAuth state"

	case errors.Is(err, auth.ErrInvalidOAuthCode):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_code"
		message = "Invalid OAuth authorization code"

	case errors.Is(err, auth.ErrOAuthFailed):
		statusCode = http.StatusBadGateway
		errorCode = "oauth_failed"
		message = "OAuth authentication failed"

	case errors.Is(err, auth.ErrInvalidToken):
		statusCode = http.StatusUnauthorized
		errorCode = "invalid_token"
		message = "Invalid token"

	case errors.Is(err, auth.ErrExpiredToken):
		statusCode = http.StatusUnauthorized
		errorCode = "expired_token"
		message = "Token expired"

	case errors.Is(err, auth.ErrRevokedToken):
		statusCode = http.StatusUnauthorized
		errorCode = "revoked_token"
		message = "Token revoked"

	case errors.Is(err, auth.ErrAPIKeyNotFound):
		statusCode = http.StatusNotFound
		errorCode = "api_key_not_found"
		message = "API key not found"

	case errors.Is(err, auth.ErrAPIKeyAlreadyExists):
		statusCode = http.StatusConflict
		errorCode = "api_key_exists"
		message = "API key already exists for this provider"

	case errors.Is(err, auth.ErrSystemAPIKeyNotFound):
		statusCode = http.StatusNotFound
		errorCode = "system_api_key_not_found"
		message = "System API key not found"

	case errors.Is(err, auth.ErrSystemAPIKeyDisabled):
		statusCode = http.StatusForbidden
		errorCode = "system_api_key_disabled"
		message = "System API key is disabled"

	case errors.Is(err, auth.ErrSystemAPIKeyExpired):
		statusCode = http.StatusForbidden
		errorCode = "system_api_key_expired"
		message = "System API key has expired"

	case errors.Is(err, auth.ErrSystemAPIKeyLimitExceeded):
		statusCode = http.StatusBadRequest
		errorCode = "api_key_limit_exceeded"
		message = "Maximum number of API keys reached"

	case errors.Is(err, auth.ErrForbidden):
		statusCode = http.StatusForbidden
		errorCode = "forbidden"
		message = "Access forbidden"

	case errors.Is(err, auth.ErrEncryptionFailed):
		statusCode = http.StatusInternalServerError
		errorCode = "encryption_error"
		message = "Encryption failed"

	case errors.Is(err, auth.ErrDecryptionFailed):
		statusCode = http.StatusInternalServerError
		errorCode = "decryption_error"
		message = "Decryption failed"

	default:
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "Internal server error"
	}

	c.JSON(statusCode, model.ErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}

// Compile-time check
var _ inbound.AuthHttpPort = (*authAdapter)(nil)
