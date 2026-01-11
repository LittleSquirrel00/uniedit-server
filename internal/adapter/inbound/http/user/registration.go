package userhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// RegistrationHandler handles user registration HTTP requests.
type RegistrationHandler struct {
	domain user.UserDomain
}

// NewRegistrationHandler creates a new registration handler.
func NewRegistrationHandler(domain user.UserDomain) *RegistrationHandler {
	return &RegistrationHandler{domain: domain}
}

// RegisterRoutes registers registration routes.
func (h *RegistrationHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verification", h.ResendVerification)
		auth.POST("/password-reset", h.RequestPasswordReset)
		auth.POST("/password-reset/complete", h.CompletePasswordReset)
	}
}

// Register handles POST /auth/register.
func (h *RegistrationHandler) Register(c *gin.Context) {
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

	u, err := h.domain.Register(c.Request.Context(), &user.RegisterInput{
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

// Login handles POST /auth/login.
func (h *RegistrationHandler) Login(c *gin.Context) {
	// Note: Login is typically handled by the auth module which manages JWT tokens
	c.JSON(http.StatusNotImplemented, model.ErrorResponse{
		Code:    "not_implemented",
		Message: "Login should be handled by auth module",
	})
}

// VerifyEmail handles POST /auth/verify-email.
func (h *RegistrationHandler) VerifyEmail(c *gin.Context) {
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

	if err := h.domain.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

// ResendVerification handles POST /auth/resend-verification.
func (h *RegistrationHandler) ResendVerification(c *gin.Context) {
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

	if err := h.domain.ResendVerification(c.Request.Context(), req.Email); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification email sent if account exists"})
}

// RequestPasswordReset handles POST /auth/password-reset.
func (h *RegistrationHandler) RequestPasswordReset(c *gin.Context) {
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

	if err := h.domain.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset email sent if account exists"})
}

// CompletePasswordReset handles POST /auth/password-reset/complete.
func (h *RegistrationHandler) CompletePasswordReset(c *gin.Context) {
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

	if err := h.domain.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// Compile-time check
var _ inbound.UserAuthPort = (*RegistrationHandler)(nil)
