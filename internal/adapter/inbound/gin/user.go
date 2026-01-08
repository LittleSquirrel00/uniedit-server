package gin

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// userAdapter implements inbound.UserHttpPort.
type userAdapter struct {
	domain user.UserDomain
}

// NewUserAdapter creates a new user HTTP adapter.
func NewUserAdapter(domain user.UserDomain) inbound.UserHttpPort {
	return &userAdapter{domain: domain}
}

// RegisterRoutes registers user routes.
func (a *userAdapter) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("/me", a.GetMe)
		users.GET("/me/profile", a.GetProfile)
		users.PUT("/me/profile", a.UpdateProfile)
		users.GET("/me/preferences", a.GetPreferences)
		users.PUT("/me/preferences", a.UpdatePreferences)
		users.POST("/me/avatar", a.UploadAvatar)
		users.POST("/me/password", a.ChangePassword)
		users.DELETE("/me", a.DeleteAccount)
	}
}

func (a *userAdapter) GetMe(c *gin.Context) {
	userID := MustGetUserID(c)

	u, err := a.domain.GetUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

func (a *userAdapter) GetProfile(c *gin.Context) {
	userID := MustGetUserID(c)

	profile, err := a.domain.GetProfile(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (a *userAdapter) UpdateProfile(c *gin.Context) {
	userID := MustGetUserID(c)

	var input user.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	u, err := a.domain.UpdateProfile(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

func (a *userAdapter) GetPreferences(c *gin.Context) {
	userID := MustGetUserID(c)

	prefs, err := a.domain.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (a *userAdapter) UpdatePreferences(c *gin.Context) {
	userID := MustGetUserID(c)

	var prefs model.Preferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.UpdatePreferences(c.Request.Context(), userID, &prefs); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}

func (a *userAdapter) UploadAvatar(c *gin.Context) {
	userID := MustGetUserID(c)

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

	url, err := a.domain.UploadAvatar(c.Request.Context(), userID, data, file.Header.Get("Content-Type"))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (a *userAdapter) ChangePassword(c *gin.Context) {
	userID := MustGetUserID(c)

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

	if err := a.domain.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func (a *userAdapter) DeleteAccount(c *gin.Context) {
	userID := MustGetUserID(c)

	var req struct {
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := a.domain.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account deleted"})
}

// Compile-time check
var _ inbound.UserHttpPort = (*userAdapter)(nil)
