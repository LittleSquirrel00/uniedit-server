package authproto

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authv1 "github.com/uniedit/server/api/pb/auth"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
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
	provider := model.OAuthProvider(in.GetProvider())
	if !provider.IsValid() {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_provider", Message: "Invalid OAuth provider"}
	}

	resp, err := h.authDomain.InitiateLogin(c.Request.Context(), provider)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &authv1.InitiateLoginResponse{
		AuthUrl: resp.AuthURL,
		State:   resp.State,
	}, nil
}

func (h *Handler) CompleteLogin(c *gin.Context, in *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	provider := model.OAuthProvider(in.GetProvider())
	if !provider.IsValid() {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_provider", Message: "Invalid OAuth provider"}
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	tokenPair, user, err := h.authDomain.CompleteLogin(
		c.Request.Context(),
		provider,
		in.GetCode(),
		in.GetState(),
		userAgent,
		ipAddress,
	)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &authv1.CompleteLoginResponse{
		Token: &authv1.TokenPairResponse{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			TokenType:    tokenPair.TokenType,
			ExpiresIn:    tokenPair.ExpiresIn,
			ExpiresAt:    tokenPair.ExpiresAt.UTC().Format(time.RFC3339Nano),
		},
		User: toCommonUser(user),
	}, nil
}

func (h *Handler) RefreshToken(c *gin.Context, in *authv1.RefreshTokenRequest) (*authv1.TokenPairResponse, error) {
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	tokenPair, err := h.authDomain.RefreshTokens(c.Request.Context(), in.GetRefreshToken(), userAgent, ipAddress)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &authv1.TokenPairResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		ExpiresAt:    tokenPair.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (h *Handler) GetMe(c *gin.Context, _ *commonv1.Empty) (*authv1.GetMeResponse, error) {
	token := c.GetHeader("Authorization")
	if token == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "Authorization header required"}
	}
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims, err := h.authDomain.ValidateAccessToken(token)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &authv1.GetMeResponse{
		UserId: claims.UserID.String(),
		Email:  claims.Email,
	}, nil
}

func (h *Handler) Logout(c *gin.Context, _ *commonv1.Empty) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	if err := h.authDomain.Logout(c.Request.Context(), userID); err != nil {
		return nil, mapAuthError(err)
	}

	return &commonv1.MessageResponse{Message: "logged out"}, nil
}

func (h *Handler) CreateUserAPIKey(c *gin.Context, in *authv1.CreateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	apiKey, err := h.authDomain.CreateUserAPIKey(c.Request.Context(), userID, &auth.CreateUserAPIKeyInput{
		Provider: in.GetProvider(),
		Name:     in.GetName(),
		APIKey:   in.GetApiKey(),
		Scopes:   in.GetScopes(),
	})
	if err != nil {
		return nil, mapAuthError(err)
	}

	c.Status(http.StatusCreated)
	return toUserAPIKey(apiKey), nil
}

func (h *Handler) ListUserAPIKeys(c *gin.Context, _ *commonv1.Empty) (*authv1.ListUserAPIKeysResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keys, err := h.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := make([]*authv1.UserAPIKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, toUserAPIKey(k))
	}
	return &authv1.ListUserAPIKeysResponse{ApiKeys: out}, nil
}

func (h *Handler) GetUserAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	keys, err := h.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}

	for _, k := range keys {
		if k.ID == keyID {
			return toUserAPIKey(k), nil
		}
	}
	return nil, &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: "API key not found"}
}

func (h *Handler) DeleteUserAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	if err := h.authDomain.DeleteUserAPIKey(c.Request.Context(), userID, keyID); err != nil {
		return nil, mapAuthError(err)
	}

	return &commonv1.MessageResponse{Message: "API key deleted"}, nil
}

func (h *Handler) RotateUserAPIKey(c *gin.Context, in *authv1.RotateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	key, err := h.authDomain.RotateUserAPIKey(c.Request.Context(), userID, keyID, in.GetNewApiKey())
	if err != nil {
		return nil, mapAuthError(err)
	}
	return toUserAPIKey(key), nil
}

func (h *Handler) CreateSystemAPIKey(c *gin.Context, in *authv1.CreateSystemAPIKeyRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	var rateLimitRPM *int
	if in.GetRateLimitRpm() != 0 {
		v := int(in.GetRateLimitRpm())
		rateLimitRPM = &v
	}
	var rateLimitTPM *int
	if in.GetRateLimitTpm() != 0 {
		v := int(in.GetRateLimitTpm())
		rateLimitTPM = &v
	}
	var expiresInDays *int
	if in.GetExpiresInDays() != 0 {
		v := int(in.GetExpiresInDays())
		expiresInDays = &v
	}

	result, err := h.authDomain.CreateSystemAPIKey(c.Request.Context(), userID, &auth.CreateSystemAPIKeyInput{
		Name:          in.GetName(),
		Scopes:        in.GetScopes(),
		RateLimitRPM:  rateLimitRPM,
		RateLimitTPM:  rateLimitTPM,
		ExpiresInDays: expiresInDays,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}

	c.Status(http.StatusCreated)
	return &authv1.CreateSystemAPIKeyResponse{
		ApiKey:     result.RawAPIKey,
		KeyDetails: toSystemAPIKey(result.Key),
	}, nil
}

func (h *Handler) ListSystemAPIKeys(c *gin.Context, _ *commonv1.Empty) (*authv1.ListSystemAPIKeysResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keys, err := h.authDomain.ListSystemAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := make([]*authv1.SystemAPIKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, toSystemAPIKey(k))
	}
	return &authv1.ListSystemAPIKeysResponse{ApiKeys: out}, nil
}

func (h *Handler) GetSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.SystemAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	key, err := h.authDomain.GetSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return toSystemAPIKey(key), nil
}

func (h *Handler) UpdateSystemAPIKey(c *gin.Context, in *authv1.UpdateSystemAPIKeyRequest) (*authv1.SystemAPIKey, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	var name *string
	if in.GetName() != nil {
		v := in.GetName().GetValue()
		name = &v
	}
	var scopes []string
	if in.GetScopes() != nil {
		scopes = in.GetScopes().GetValues()
	}
	var rateLimitRPM *int
	if in.GetRateLimitRpm() != nil {
		v := int(in.GetRateLimitRpm().GetValue())
		rateLimitRPM = &v
	}
	var rateLimitTPM *int
	if in.GetRateLimitTpm() != nil {
		v := int(in.GetRateLimitTpm().GetValue())
		rateLimitTPM = &v
	}
	var isActive *bool
	if in.GetIsActive() != nil {
		v := in.GetIsActive().GetValue()
		isActive = &v
	}

	key, err := h.authDomain.UpdateSystemAPIKey(c.Request.Context(), userID, keyID, &auth.UpdateSystemAPIKeyInput{
		Name:         name,
		Scopes:       scopes,
		RateLimitRPM: rateLimitRPM,
		RateLimitTPM: rateLimitTPM,
		IsActive:     isActive,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}
	return toSystemAPIKey(key), nil
}

func (h *Handler) DeleteSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	if err := h.authDomain.DeleteSystemAPIKey(c.Request.Context(), userID, keyID); err != nil {
		return nil, mapAuthError(err)
	}
	return &commonv1.MessageResponse{Message: "API key deleted"}, nil
}

func (h *Handler) RotateSystemAPIKey(c *gin.Context, in *authv1.GetByIDRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid API key ID", Err: err}
	}

	result, err := h.authDomain.RotateSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		return nil, mapAuthError(err)
	}

	return &authv1.CreateSystemAPIKeyResponse{
		ApiKey:     result.RawAPIKey,
		KeyDetails: toSystemAPIKey(result.Key),
	}, nil
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

func toCommonUser(u *model.User) *commonv1.User {
	if u == nil {
		return nil
	}
	provider := "email"
	if u.OAuthProvider != nil && *u.OAuthProvider != "" {
		provider = *u.OAuthProvider
	}
	suspendedAt := ""
	if u.SuspendedAt != nil {
		suspendedAt = u.SuspendedAt.UTC().Format(time.RFC3339Nano)
	}
	suspendReason := ""
	if u.SuspendReason != nil {
		suspendReason = *u.SuspendReason
	}
	return &commonv1.User{
		Id:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		AvatarUrl:     u.AvatarURL,
		Provider:      provider,
		Status:        string(u.Status),
		EmailVerified: u.EmailVerified,
		IsAdmin:       u.IsAdmin,
		SuspendedAt:   suspendedAt,
		SuspendReason: suspendReason,
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toUserAPIKey(k *model.UserAPIKey) *authv1.UserAPIKey {
	if k == nil {
		return nil
	}
	lastUsedAt := ""
	if k.LastUsedAt != nil {
		lastUsedAt = k.LastUsedAt.UTC().Format(time.RFC3339Nano)
	}
	return &authv1.UserAPIKey{
		Id:         k.ID.String(),
		Provider:   k.Provider,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     []string(k.Scopes),
		LastUsedAt: lastUsedAt,
		CreatedAt:  k.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toSystemAPIKey(k *model.SystemAPIKey) *authv1.SystemAPIKey {
	if k == nil {
		return nil
	}
	lastUsedAt := ""
	if k.LastUsedAt != nil {
		lastUsedAt = k.LastUsedAt.UTC().Format(time.RFC3339Nano)
	}
	expiresAt := ""
	if k.ExpiresAt != nil {
		expiresAt = k.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	lastRotatedAt := ""
	if k.LastRotatedAt != nil {
		lastRotatedAt = k.LastRotatedAt.UTC().Format(time.RFC3339Nano)
	}
	rotateAfterDays := int32(0)
	if k.RotateAfterDays != nil {
		rotateAfterDays = int32(*k.RotateAfterDays)
	}
	return &authv1.SystemAPIKey{
		Id:                k.ID.String(),
		Name:              k.Name,
		KeyPrefix:         k.KeyPrefix,
		Scopes:            []string(k.Scopes),
		RateLimitRpm:      int32(k.RateLimitRPM),
		RateLimitTpm:      int32(k.RateLimitTPM),
		TotalRequests:     k.TotalRequests,
		TotalInputTokens:  k.TotalInputTokens,
		TotalOutputTokens: k.TotalOutputTokens,
		TotalCostUsd:      k.TotalCostUSD,
		CacheHits:         k.CacheHits,
		CacheMisses:       k.CacheMisses,
		IsActive:          k.IsActive,
		LastUsedAt:        lastUsedAt,
		ExpiresAt:         expiresAt,
		AllowedIps:        []string(k.AllowedIPs),
		RotateAfterDays:   rotateAfterDays,
		LastRotatedAt:     lastRotatedAt,
		CreatedAt:         k.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
