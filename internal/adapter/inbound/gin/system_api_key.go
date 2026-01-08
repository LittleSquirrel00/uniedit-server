package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// systemAPIKeyAdapter implements inbound.SystemAPIKeyHttpPort.
type systemAPIKeyAdapter struct {
	authDomain auth.AuthDomain
}

// NewSystemAPIKeyAdapter creates a new system API key HTTP adapter.
func NewSystemAPIKeyAdapter(authDomain auth.AuthDomain) inbound.SystemAPIKeyHttpPort {
	return &systemAPIKeyAdapter{authDomain: authDomain}
}

// RegisterRoutes registers system API key routes.
func (a *systemAPIKeyAdapter) RegisterRoutes(r *gin.RouterGroup) {
	keys := r.Group("/system-api-keys")
	{
		keys.POST("", a.CreateSystemAPIKey)
		keys.GET("", a.ListSystemAPIKeys)
		keys.GET("/:id", a.GetSystemAPIKey)
		keys.PATCH("/:id", a.UpdateSystemAPIKey)
		keys.DELETE("/:id", a.DeleteSystemAPIKey)
		keys.POST("/:id/rotate", a.RotateSystemAPIKey)
	}
}

func (a *systemAPIKeyAdapter) CreateSystemAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req struct {
		Name          string   `json:"name" binding:"required"`
		Scopes        []string `json:"scopes"`
		RateLimitRPM  *int     `json:"rate_limit_rpm"`
		RateLimitTPM  *int     `json:"rate_limit_tpm"`
		ExpiresInDays *int     `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	input := &auth.CreateSystemAPIKeyInput{
		Name:          req.Name,
		Scopes:        req.Scopes,
		RateLimitRPM:  req.RateLimitRPM,
		RateLimitTPM:  req.RateLimitTPM,
		ExpiresInDays: req.ExpiresInDays,
	}

	result, err := a.authDomain.CreateSystemAPIKey(c.Request.Context(), userID, input)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key":     result.RawAPIKey,
		"key_details": result.Key.ToResponse(),
	})
}

func (a *systemAPIKeyAdapter) ListSystemAPIKeys(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	keys, err := a.authDomain.ListSystemAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	response := make([]*model.SystemAPIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

func (a *systemAPIKeyAdapter) GetSystemAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid API key ID",
		})
		return
	}

	key, err := a.authDomain.GetSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

func (a *systemAPIKeyAdapter) UpdateSystemAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid API key ID",
		})
		return
	}

	var req struct {
		Name         *string  `json:"name"`
		Scopes       []string `json:"scopes"`
		RateLimitRPM *int     `json:"rate_limit_rpm"`
		RateLimitTPM *int     `json:"rate_limit_tpm"`
		IsActive     *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	input := &auth.UpdateSystemAPIKeyInput{
		Name:         req.Name,
		Scopes:       req.Scopes,
		RateLimitRPM: req.RateLimitRPM,
		RateLimitTPM: req.RateLimitTPM,
		IsActive:     req.IsActive,
	}

	key, err := a.authDomain.UpdateSystemAPIKey(c.Request.Context(), userID, keyID, input)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

func (a *systemAPIKeyAdapter) DeleteSystemAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid API key ID",
		})
		return
	}

	if err := a.authDomain.DeleteSystemAPIKey(c.Request.Context(), userID, keyID); err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

func (a *systemAPIKeyAdapter) RotateSystemAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid API key ID",
		})
		return
	}

	result, err := a.authDomain.RotateSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key":     result.RawAPIKey,
		"key_details": result.Key.ToResponse(),
	})
}

// Compile-time check
var _ inbound.SystemAPIKeyHttpPort = (*systemAPIKeyAdapter)(nil)
