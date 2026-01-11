package userhttp

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// ProfileHandler handles user profile HTTP requests.
type ProfileHandler struct {
	domain user.UserDomain
}

// NewProfileHandler creates a new user profile handler.
func NewProfileHandler(domain user.UserDomain) *ProfileHandler {
	return &ProfileHandler{domain: domain}
}

// RegisterRoutes registers user profile routes.
func (h *ProfileHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("/me", h.GetMe)
		users.GET("/me/profile", h.GetProfile)
		users.PUT("/me/profile", h.UpdateProfile)
		users.GET("/me/preferences", h.GetPreferences)
		users.PUT("/me/preferences", h.UpdatePreferences)
		users.POST("/me/avatar", h.UploadAvatar)
		users.POST("/me/password", h.ChangePassword)
		users.DELETE("/me", h.DeleteAccount)
	}
}

// GetMe handles GET /users/me.
func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	u, err := h.domain.GetUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

// GetProfile handles GET /users/me/profile.
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	profile, err := h.domain.GetProfile(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile handles PUT /users/me/profile.
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	var input user.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	u, err := h.domain.UpdateProfile(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

// GetPreferences handles GET /users/me/preferences.
func (h *ProfileHandler) GetPreferences(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	prefs, err := h.domain.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences handles PUT /users/me/preferences.
func (h *ProfileHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	var prefs model.Preferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := h.domain.UpdatePreferences(c.Request.Context(), userID, &prefs); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}

// UploadAvatar handles POST /users/me/avatar.
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_file",
			Message: "avatar file is required",
		})
		return
	}

	f, err := file.Open()
	if err != nil {
		handleError(c, err)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		handleError(c, err)
		return
	}

	url, err := h.domain.UploadAvatar(c.Request.Context(), userID, data, file.Header.Get("Content-Type"))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// ChangePassword handles POST /users/me/password.
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := h.domain.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// DeleteAccount handles DELETE /users/me.
func (h *ProfileHandler) DeleteAccount(c *gin.Context) {
	userID, ok := requireAuth(c)
	if !ok {
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.domain.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account deleted"})
}

// Compile-time check
var _ inbound.UserHttpPort = (*ProfileHandler)(nil)
