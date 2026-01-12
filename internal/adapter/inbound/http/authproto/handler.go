package authproto

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authv1 "github.com/uniedit/server/api/pb/auth"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	authDomain auth.AuthDomain
}

func NewHandler(authDomain auth.AuthDomain) *Handler {
	return &Handler{authDomain: authDomain}
}

func (h *Handler) InitiateLogin(c *gin.Context, in *authv1.InitiateLoginRequest) (*authv1.InitiateLoginResponse, error) {
	resp, err := h.authDomain.InitiateLogin(c.Request.Context(), in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return resp, nil
}

func (h *Handler) CompleteLogin(c *gin.Context, in *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	out, err := h.authDomain.CompleteLogin(c.Request.Context(), in, userAgent, ipAddress)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) RefreshToken(c *gin.Context, in *authv1.RefreshTokenRequest) (*authv1.TokenPairResponse, error) {
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	out, err := h.authDomain.RefreshToken(c.Request.Context(), in, userAgent, ipAddress)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) GetMe(c *gin.Context, _ *commonv1.Empty) (*authv1.GetMeResponse, error) {
	token := c.GetHeader("Authorization")
	if token == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "Authorization header required"}
	}
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	out, err := h.authDomain.GetMe(c.Request.Context(), token)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) Logout(c *gin.Context, _ *commonv1.Empty) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.Logout(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return out, nil
}

func (h *Handler) CreateUserAPIKey(c *gin.Context, in *authv1.CreateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	apiKey, err := h.authDomain.CreateUserAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}

	c.Status(http.StatusCreated)
	return apiKey, nil
}

func (h *Handler) ListUserAPIKeys(c *gin.Context, _ *commonv1.Empty) (*authv1.ListUserAPIKeysResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) GetUserAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.GetUserAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) DeleteUserAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.DeleteUserAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) RotateUserAPIKey(c *gin.Context, in *authv1.RotateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	key, err := h.authDomain.RotateUserAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return key, nil
}

func (h *Handler) CreateSystemAPIKey(c *gin.Context, in *authv1.CreateSystemAPIKeyRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	result, err := h.authDomain.CreateSystemAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}

	c.Status(http.StatusCreated)
	return result, nil
}

func (h *Handler) ListSystemAPIKeys(c *gin.Context, _ *commonv1.Empty) (*authv1.ListSystemAPIKeysResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.ListSystemAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) GetSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.SystemAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	key, err := h.authDomain.GetSystemAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return key, nil
}

func (h *Handler) UpdateSystemAPIKey(c *gin.Context, in *authv1.UpdateSystemAPIKeyRequest) (*authv1.SystemAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	key, err := h.authDomain.UpdateSystemAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return key, nil
}

func (h *Handler) DeleteSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.authDomain.DeleteSystemAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return out, nil
}

func (h *Handler) RotateSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	result, err := h.authDomain.RotateSystemAPIKey(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return result, nil
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidOAuthProvider):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_provider", Message: "Invalid OAuth provider", Err: err}
	case errors.Is(err, auth.ErrInvalidOAuthState):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_state", Message: "Invalid or expired OAuth state", Err: err}
	case errors.Is(err, auth.ErrInvalidOAuthCode):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_code", Message: "Invalid OAuth authorization code", Err: err}
	case errors.Is(err, auth.ErrOAuthFailed):
		return &protohttp.HTTPError{Status: http.StatusBadGateway, Code: "oauth_failed", Message: "OAuth authentication failed", Err: err}
	case errors.Is(err, auth.ErrInvalidToken):
		return &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "invalid_token", Message: "Invalid token", Err: err}
	case errors.Is(err, auth.ErrExpiredToken):
		return &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "expired_token", Message: "Token expired", Err: err}
	case errors.Is(err, auth.ErrRevokedToken):
		return &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "revoked_token", Message: "Token revoked", Err: err}
	case errors.Is(err, auth.ErrAPIKeyNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "api_key_not_found", Message: "API key not found", Err: err}
	case errors.Is(err, auth.ErrAPIKeyAlreadyExists):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "api_key_exists", Message: "API key already exists for this provider", Err: err}
	case errors.Is(err, auth.ErrSystemAPIKeyNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "system_api_key_not_found", Message: "System API key not found", Err: err}
	case errors.Is(err, auth.ErrSystemAPIKeyDisabled):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "system_api_key_disabled", Message: "System API key is disabled", Err: err}
	case errors.Is(err, auth.ErrSystemAPIKeyExpired):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "system_api_key_expired", Message: "System API key has expired", Err: err}
	case errors.Is(err, auth.ErrSystemAPIKeyLimitExceeded):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "api_key_limit_exceeded", Message: "Maximum number of API keys reached", Err: err}
	case errors.Is(err, auth.ErrForbidden):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "Access forbidden", Err: err}
	case errors.Is(err, auth.ErrEncryptionFailed):
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "encryption_error", Message: "Encryption failed", Err: err}
	case errors.Is(err, auth.ErrDecryptionFailed):
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "decryption_error", Message: "Decryption failed", Err: err}
	default:
		return err
	}
}
