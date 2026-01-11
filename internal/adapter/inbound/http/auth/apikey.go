package authhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// APIKeyHandler handles user API key HTTP requests.
type APIKeyHandler struct {
	authDomain auth.AuthDomain
}

// NewAPIKeyHandler creates a new API key handler.
func NewAPIKeyHandler(authDomain auth.AuthDomain) *APIKeyHandler {
	return &APIKeyHandler{authDomain: authDomain}
}

// RegisterRoutes registers API key routes.
func (h *APIKeyHandler) RegisterRoutes(r *gin.RouterGroup) {
	apiKeys := r.Group("/api-keys")
	{
		apiKeys.POST("", h.CreateAPIKey)
		apiKeys.GET("", h.ListAPIKeys)
		apiKeys.GET("/:id", h.GetAPIKey)
		apiKeys.DELETE("/:id", h.DeleteAPIKey)
		apiKeys.POST("/:id/rotate", h.RotateAPIKey)
	}
}

// CreateAPIKey handles POST /api-keys.
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	var req struct {
		Provider string   `json:"provider" binding:"required"`
		Name     string   `json:"name" binding:"required"`
		APIKey   string   `json:"api_key" binding:"required"`
		Scopes   []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	input := &auth.CreateUserAPIKeyInput{
		Provider: req.Provider,
		Name:     req.Name,
		APIKey:   req.APIKey,
		Scopes:   req.Scopes,
	}

	apiKey, err := h.authDomain.CreateUserAPIKey(c.Request.Context(), userID, input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, apiKey.ToResponse())
}

// ListAPIKeys handles GET /api-keys.
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	apiKeys, err := h.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response := make([]*model.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

// GetAPIKey handles GET /api-keys/:id.
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
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

	apiKeys, err := h.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	for _, key := range apiKeys {
		if key.ID == keyID {
			c.JSON(http.StatusOK, key.ToResponse())
			return
		}
	}

	c.JSON(http.StatusNotFound, model.ErrorResponse{
		Code:    "not_found",
		Message: "API key not found",
	})
}

// DeleteAPIKey handles DELETE /api-keys/:id.
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
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

	if err := h.authDomain.DeleteUserAPIKey(c.Request.Context(), userID, keyID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

// RotateAPIKey handles POST /api-keys/:id/rotate.
func (h *APIKeyHandler) RotateAPIKey(c *gin.Context) {
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
		NewAPIKey string `json:"new_api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := h.authDomain.RotateUserAPIKey(c.Request.Context(), userID, keyID, req.NewAPIKey)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, apiKey.ToResponse())
}

// Compile-time check
var _ inbound.APIKeyHttpPort = (*APIKeyHandler)(nil)
