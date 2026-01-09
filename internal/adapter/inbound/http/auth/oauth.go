package authhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// OAuthHandler handles OAuth authentication HTTP requests.
type OAuthHandler struct {
	authDomain auth.AuthDomain
}

// NewOAuthHandler creates a new OAuth handler.
func NewOAuthHandler(authDomain auth.AuthDomain) *OAuthHandler {
	return &OAuthHandler{authDomain: authDomain}
}

// RegisterRoutes registers OAuth routes.
func (h *OAuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/oauth/:provider", h.InitiateLogin)
		auth.POST("/oauth/:provider/callback", h.CompleteLogin)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", h.GetMe)
	}
}

// InitiateLogin handles POST /auth/oauth/:provider.
func (h *OAuthHandler) InitiateLogin(c *gin.Context) {
	providerStr := c.Param("provider")
	provider := model.OAuthProvider(providerStr)

	if !provider.IsValid() {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_provider",
			Message: "Invalid OAuth provider",
		})
		return
	}

	resp, err := h.authDomain.InitiateLogin(c.Request.Context(), provider)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": resp.AuthURL,
		"state":    resp.State,
	})
}

// CompleteLogin handles POST /auth/oauth/:provider/callback.
func (h *OAuthHandler) CompleteLogin(c *gin.Context) {
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

	tokenPair, user, err := h.authDomain.CompleteLogin(c.Request.Context(), provider, req.Code, req.State, userAgent, ipAddress)
	if err != nil {
		handleError(c, err)
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

// RefreshToken handles POST /auth/refresh.
func (h *OAuthHandler) RefreshToken(c *gin.Context) {
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

	tokenPair, err := h.authDomain.RefreshTokens(c.Request.Context(), req.RefreshToken, userAgent, ipAddress)
	if err != nil {
		handleError(c, err)
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

// Logout handles POST /auth/logout.
func (h *OAuthHandler) Logout(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	if err := h.authDomain.Logout(c.Request.Context(), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// GetMe handles GET /auth/me.
func (h *OAuthHandler) GetMe(c *gin.Context) {
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

	claims, err := h.authDomain.ValidateAccessToken(token)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}

// Compile-time check
var _ inbound.AuthHttpPort = (*OAuthHandler)(nil)
