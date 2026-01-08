package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// userAuthAdapter implements inbound.UserAuthPort.
type userAuthAdapter struct {
	domain user.UserDomain
}

// NewUserAuthAdapter creates a new user auth HTTP adapter.
func NewUserAuthAdapter(domain user.UserDomain) inbound.UserAuthPort {
	return &userAuthAdapter{domain: domain}
}

// RegisterRoutes registers auth routes.
func (a *userAuthAdapter) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", a.Register)
		auth.POST("/verify-email", a.VerifyEmail)
		auth.POST("/resend-verification", a.ResendVerification)
		auth.POST("/password-reset", a.RequestPasswordReset)
		auth.POST("/password-reset/complete", a.CompletePasswordReset)
	}
}

func (a *userAuthAdapter) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	u, err := a.domain.Register(c.Request.Context(), &user.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "registration successful, please check your email for verification",
		"user":    u.ToResponse(),
	})
}

func (a *userAuthAdapter) Login(c *gin.Context) {
	// Note: Login is typically handled by the auth module which manages JWT tokens
	// This is a placeholder - actual login should be integrated with auth domain
	c.JSON(http.StatusNotImplemented, model.ErrorResponse{
		Code:    "not_implemented",
		Message: "Login should be handled by auth module",
	})
}

func (a *userAuthAdapter) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

func (a *userAuthAdapter) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.ResendVerification(c.Request.Context(), req.Email); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification email sent if account exists"})
}

func (a *userAuthAdapter) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset email sent if account exists"})
}

func (a *userAuthAdapter) CompletePasswordReset(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := a.domain.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// Compile-time check
var _ inbound.UserAuthPort = (*userAuthAdapter)(nil)
