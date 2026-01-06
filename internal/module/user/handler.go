package user

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for user management.
type Handler struct {
	service *Service
}

// NewHandler creates a new user handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the user routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verification", h.ResendVerification)
		auth.POST("/login/password", h.Login)
		auth.POST("/password/reset-request", h.RequestPasswordReset)
		auth.POST("/password/reset", h.ResetPassword)
	}
}

// RegisterProtectedRoutes registers routes that require authentication.
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("/me", h.GetCurrentUser)
		users.PUT("/me", h.UpdateProfile)
		users.PUT("/me/password", h.ChangePassword)
		users.DELETE("/me", h.DeleteAccount)
	}
}

// Register handles user registration.
//
//	@Summary		Register new user
//	@Description	Create a new user account with email and password
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration request"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("[DEBUG] Register error: %v\n", err)
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":    user.ToResponse(),
		"message": "Verification email sent. Please check your inbox.",
	})
}

// VerifyEmail handles email verification.
//
//	@Summary		Verify email
//	@Description	Verify user email with the verification token
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		VerifyEmailRequest	true	"Verification request"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Email verified successfully"})
}

// ResendVerification handles resending verification email.
//
//	@Summary		Resend verification email
//	@Description	Resend the email verification link
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ResendVerificationRequest	true	"Resend request"
//	@Success		200		{object}	MessageResponse
//	@Router			/auth/resend-verification [post]
func (h *Handler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return success to not reveal email existence
	_ = h.service.ResendVerification(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, MessageResponse{Message: "If the email exists, a verification email has been sent"})
}

// Login handles email/password login.
//
//	@Summary		Login with password
//	@Description	Authenticate with email and password
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/auth/login/password [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, userAgent, ipAddress)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"user":   user.ToResponse(),
	})
}

// RequestPasswordReset handles password reset request.
//
//	@Summary		Request password reset
//	@Description	Request a password reset email
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PasswordResetRequest	true	"Password reset request"
//	@Success		200		{object}	MessageResponse
//	@Router			/auth/password/reset-request [post]
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return success to not reveal email existence
	_ = h.service.RequestPasswordReset(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, MessageResponse{Message: "If the email exists, a password reset link has been sent"})
}

// ResetPassword handles password reset completion.
//
//	@Summary		Reset password
//	@Description	Complete password reset with token and new password
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CompletePasswordResetRequest	true	"Reset request"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/auth/password/reset [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req CompletePasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Password reset successfully"})
}

// GetCurrentUser returns the current authenticated user.
//
//	@Summary		Get current user profile
//	@Description	Get the profile of the currently authenticated user
//	@Tags			User
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/users/me [get]
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateProfile handles profile updates.
//
//	@Summary		Update user profile
//	@Description	Update the current user's profile information
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateProfileRequest	true	"Update request"
//	@Success		200		{object}	UserResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/users/me [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ChangePassword handles password change.
//
//	@Summary		Change password
//	@Description	Change the current user's password
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ChangePasswordRequest	true	"Change password request"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/users/me/password [put]
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Password changed successfully"})
}

// DeleteAccount handles account deletion.
//
//	@Summary		Delete account
//	@Description	Permanently delete the current user's account
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		DeleteAccountRequest	true	"Delete account request"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/users/me [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Account deleted successfully"})
}

// --- Helpers ---

func getUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found", "message": "User not found"})
	case errors.Is(err, ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "email_already_registered", "message": "Email already registered"})
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials", "message": "Invalid email or password"})
	case errors.Is(err, ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, gin.H{"error": "email_not_verified", "message": "Please verify your email before logging in"})
	case errors.Is(err, ErrAccountSuspended):
		c.JSON(http.StatusForbidden, gin.H{"error": "account_suspended", "message": "Your account has been suspended"})
	case errors.Is(err, ErrAccountDeleted):
		c.JSON(http.StatusForbidden, gin.H{"error": "account_deleted", "message": "This account has been deleted"})
	case errors.Is(err, ErrIncorrectPassword):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect_password", "message": "Current password is incorrect"})
	case errors.Is(err, ErrPasswordTooShort):
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_too_short", "message": "Password must be at least 8 characters"})
	case errors.Is(err, ErrPasswordRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_required", "message": "Password is required"})
	case errors.Is(err, ErrInvalidToken):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token", "message": "Invalid verification token"})
	case errors.Is(err, ErrTokenExpired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_expired", "message": "Verification token has expired"})
	case errors.Is(err, ErrTokenAlreadyUsed):
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_already_used", "message": "Verification token has already been used"})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Access denied"})
	case errors.Is(err, ErrCannotSuspendAdmin):
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot_suspend_admin", "message": "Cannot suspend admin users"})
	case errors.Is(err, ErrUserAlreadyActive):
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_already_active", "message": "User is already active"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "An internal error occurred"})
	}
}
