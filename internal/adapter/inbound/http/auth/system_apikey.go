package authhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// SystemAPIKeyHandler handles system API key HTTP requests.
type SystemAPIKeyHandler struct {
	authDomain auth.AuthDomain
}

// NewSystemAPIKeyHandler creates a new system API key handler.
func NewSystemAPIKeyHandler(authDomain auth.AuthDomain) *SystemAPIKeyHandler {
	return &SystemAPIKeyHandler{authDomain: authDomain}
}

// RegisterRoutes registers system API key routes.
func (h *SystemAPIKeyHandler) RegisterRoutes(r *gin.RouterGroup) {
	keys := r.Group("/system-api-keys")
	{
		keys.POST("", h.CreateSystemAPIKey)
		keys.GET("", h.ListSystemAPIKeys)
		keys.GET("/:id", h.GetSystemAPIKey)
		keys.PATCH("/:id", h.UpdateSystemAPIKey)
		keys.DELETE("/:id", h.DeleteSystemAPIKey)
		keys.POST("/:id/rotate", h.RotateSystemAPIKey)
	}
}

// CreateSystemAPIKey handles POST /system-api-keys.
func (h *SystemAPIKeyHandler) CreateSystemAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
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

	result, err := h.authDomain.CreateSystemAPIKey(c.Request.Context(), userID, input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key":     result.RawAPIKey,
		"key_details": result.Key.ToResponse(),
	})
}

// ListSystemAPIKeys handles GET /system-api-keys.
func (h *SystemAPIKeyHandler) ListSystemAPIKeys(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	keys, err := h.authDomain.ListSystemAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response := make([]*model.SystemAPIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

// GetSystemAPIKey handles GET /system-api-keys/:id.
func (h *SystemAPIKeyHandler) GetSystemAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
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

	key, err := h.authDomain.GetSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

// UpdateSystemAPIKey handles PATCH /system-api-keys/:id.
func (h *SystemAPIKeyHandler) UpdateSystemAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
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

	key, err := h.authDomain.UpdateSystemAPIKey(c.Request.Context(), userID, keyID, input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, key.ToResponse())
}

// DeleteSystemAPIKey handles DELETE /system-api-keys/:id.
func (h *SystemAPIKeyHandler) DeleteSystemAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
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

	if err := h.authDomain.DeleteSystemAPIKey(c.Request.Context(), userID, keyID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

// RotateSystemAPIKey handles POST /system-api-keys/:id/rotate.
func (h *SystemAPIKeyHandler) RotateSystemAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
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

	result, err := h.authDomain.RotateSystemAPIKey(c.Request.Context(), userID, keyID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key":     result.RawAPIKey,
		"key_details": result.Key.ToResponse(),
	})
}

// Compile-time check
var _ inbound.SystemAPIKeyHttpPort = (*SystemAPIKeyHandler)(nil)
