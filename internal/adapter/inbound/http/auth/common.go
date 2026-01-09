package authhttp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/utils/middleware"
)

// getUserID returns the user ID from context.
func getUserID(c *gin.Context) uuid.UUID {
	return middleware.GetUserID(c)
}

// requireAuth checks if the user is authenticated and returns an error response if not.
func requireAuth(c *gin.Context) (uuid.UUID, bool) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return uuid.Nil, false
	}
	return userID, true
}

// handleError maps auth domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
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
