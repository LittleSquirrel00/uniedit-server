package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// apiKeyAdapter implements inbound.APIKeyHttpPort.
type apiKeyAdapter struct {
	authDomain auth.AuthDomain
}

// NewAPIKeyAdapter creates a new API key HTTP adapter.
func NewAPIKeyAdapter(authDomain auth.AuthDomain) inbound.APIKeyHttpPort {
	return &apiKeyAdapter{authDomain: authDomain}
}

// RegisterRoutes registers API key routes.
func (a *apiKeyAdapter) RegisterRoutes(r *gin.RouterGroup) {
	apiKeys := r.Group("/api-keys")
	{
		apiKeys.POST("", a.CreateAPIKey)
		apiKeys.GET("", a.ListAPIKeys)
		apiKeys.GET("/:id", a.GetAPIKey)
		apiKeys.DELETE("/:id", a.DeleteAPIKey)
		apiKeys.POST("/:id/rotate", a.RotateAPIKey)
	}
}

func (a *apiKeyAdapter) CreateAPIKey(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
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

	apiKey, err := a.authDomain.CreateUserAPIKey(c.Request.Context(), userID, input)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, apiKey.ToResponse())
}

func (a *apiKeyAdapter) ListAPIKeys(c *gin.Context) {
	userID := MustGetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	apiKeys, err := a.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	response := make([]*model.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = key.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

func (a *apiKeyAdapter) GetAPIKey(c *gin.Context) {
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

	// Get decrypted key for the specified provider
	// Note: This assumes the ID param is actually a provider name for decrypted access
	// For actual key retrieval by ID, we'd need a different method
	apiKeys, err := a.authDomain.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleAuthError(c, err)
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

func (a *apiKeyAdapter) DeleteAPIKey(c *gin.Context) {
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

	if err := a.authDomain.DeleteUserAPIKey(c.Request.Context(), userID, keyID); err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

func (a *apiKeyAdapter) RotateAPIKey(c *gin.Context) {
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
		NewAPIKey string `json:"new_api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	apiKey, err := a.authDomain.RotateUserAPIKey(c.Request.Context(), userID, keyID, req.NewAPIKey)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, apiKey.ToResponse())
}

// Compile-time check
var _ inbound.APIKeyHttpPort = (*apiKeyAdapter)(nil)
