package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	userCmd "github.com/uniedit/server/internal/app/command/user"
	userQuery "github.com/uniedit/server/internal/app/query/user"
	"github.com/uniedit/server/internal/domain/user"
)

// UserHandler handles HTTP requests for user management using CQRS pattern.
type UserHandler struct {
	// Commands
	register              *userCmd.RegisterHandler
	verifyEmail           *userCmd.VerifyEmailHandler
	resendVerification    *userCmd.ResendVerificationHandler
	requestPasswordReset  *userCmd.RequestPasswordResetHandler
	resetPassword         *userCmd.ResetPasswordHandler
	changePassword        *userCmd.ChangePasswordHandler
	updateProfile         *userCmd.UpdateProfileHandler
	deleteAccount         *userCmd.DeleteAccountHandler
	suspendUser           *userCmd.SuspendUserHandler
	reactivateUser        *userCmd.ReactivateUserHandler
	setAdminStatus        *userCmd.SetAdminStatusHandler
	// Queries
	getUser         *userQuery.GetUserHandler
	getUserByEmail  *userQuery.GetUserByEmailHandler
	listUsers       *userQuery.ListUsersHandler
	// Error handler
	errorHandler *ErrorHandler
}

// NewUserHandler creates a new user handler.
func NewUserHandler(
	register *userCmd.RegisterHandler,
	verifyEmail *userCmd.VerifyEmailHandler,
	resendVerification *userCmd.ResendVerificationHandler,
	requestPasswordReset *userCmd.RequestPasswordResetHandler,
	resetPassword *userCmd.ResetPasswordHandler,
	changePassword *userCmd.ChangePasswordHandler,
	updateProfile *userCmd.UpdateProfileHandler,
	deleteAccount *userCmd.DeleteAccountHandler,
	suspendUser *userCmd.SuspendUserHandler,
	reactivateUser *userCmd.ReactivateUserHandler,
	setAdminStatus *userCmd.SetAdminStatusHandler,
	getUser *userQuery.GetUserHandler,
	getUserByEmail *userQuery.GetUserByEmailHandler,
	listUsers *userQuery.ListUsersHandler,
) *UserHandler {
	return &UserHandler{
		register:              register,
		verifyEmail:           verifyEmail,
		resendVerification:    resendVerification,
		requestPasswordReset:  requestPasswordReset,
		resetPassword:         resetPassword,
		changePassword:        changePassword,
		updateProfile:         updateProfile,
		deleteAccount:         deleteAccount,
		suspendUser:           suspendUser,
		reactivateUser:        reactivateUser,
		setAdminStatus:        setAdminStatus,
		getUser:               getUser,
		getUserByEmail:        getUserByEmail,
		listUsers:             listUsers,
		errorHandler:          NewErrorHandler(),
	}
}

// RegisterRoutes registers public user/auth routes.
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verification", h.ResendVerification)
		auth.POST("/password/reset-request", h.RequestPasswordReset)
		auth.POST("/password/reset", h.ResetPassword)
	}
}

// RegisterProtectedRoutes registers protected user routes.
func (h *UserHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("/me", h.GetCurrentUser)
		users.PUT("/me", h.UpdateProfile)
		users.PUT("/me/password", h.ChangePassword)
		users.DELETE("/me", h.DeleteAccount)
	}
}

// RegisterAdminRoutes registers admin user management routes.
func (h *UserHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.ListUsers)
		users.GET("/:id", h.GetUser)
		users.POST("/:id/suspend", h.SuspendUser)
		users.POST("/:id/reactivate", h.ReactivateUser)
		users.PUT("/:id/admin", h.SetAdminStatus)
	}
}

// --- Request/Response Types ---

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type CompletePasswordResetRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
}

type SuspendUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SetAdminStatusRequest struct {
	IsAdmin bool `json:"is_admin"`
}

type ListUsersRequest struct {
	Status   *string `form:"status"`
	Email    *string `form:"email"`
	IsAdmin  *bool   `form:"is_admin"`
	Page     int     `form:"page"`
	PageSize int     `form:"page_size"`
}

// --- Handlers ---

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.register.Handle(c.Request.Context(), userCmd.RegisterCommand{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondCreated(c, gin.H{
		"user":    userToResponse(result.User),
		"message": "Verification email sent. Please check your inbox.",
	})
}

func (h *UserHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_, err := h.verifyEmail.Handle(c.Request.Context(), userCmd.VerifyEmailCommand{
		Token: req.Token,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Email verified successfully"})
}

func (h *UserHandler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Always return success to not reveal email existence
	_, _ = h.resendVerification.Handle(c.Request.Context(), userCmd.ResendVerificationCommand{
		Email: req.Email,
	})

	respondSuccess(c, gin.H{"message": "If the email exists, a verification email has been sent"})
}

func (h *UserHandler) RequestPasswordReset(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Always return success to not reveal email existence
	_, _ = h.requestPasswordReset.Handle(c.Request.Context(), userCmd.RequestPasswordResetCommand{
		Email: req.Email,
	})

	respondSuccess(c, gin.H{"message": "If the email exists, a password reset link has been sent"})
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req CompletePasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_, err := h.resetPassword.Handle(c.Request.Context(), userCmd.ResetPasswordCommand{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Password reset successfully"})
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	result, err := h.getUser.Handle(c.Request.Context(), userQuery.GetUserQuery{
		UserID: userID,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, userToResponse(result.User))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.updateProfile.Handle(c.Request.Context(), userCmd.UpdateProfileCommand{
		UserID:    userID,
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, userToResponse(result.User))
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_, err := h.changePassword.Handle(c.Request.Context(), userCmd.ChangePasswordCommand{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Password changed successfully"})
}

func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_, err := h.deleteAccount.Handle(c.Request.Context(), userCmd.DeleteAccountCommand{
		UserID:   userID,
		Password: req.Password,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Account deleted successfully"})
}

// --- Admin Handlers ---

func (h *UserHandler) ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var status *user.UserStatus
	if req.Status != nil {
		s := user.UserStatus(*req.Status)
		status = &s
	}

	result, err := h.listUsers.Handle(c.Request.Context(), userQuery.ListUsersQuery{
		Status:   status,
		Email:    req.Email,
		IsAdmin:  req.IsAdmin,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	users := make([]gin.H, len(result.Users))
	for i, u := range result.Users {
		users[i] = userToResponse(u)
	}

	respondPaginated(c, users, result.Total, page, pageSize)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid user ID format")
		return
	}

	result, err := h.getUser.Handle(c.Request.Context(), userQuery.GetUserQuery{
		UserID: userID,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"user": userToResponse(result.User)})
}

func (h *UserHandler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid user ID format")
		return
	}

	var req SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.suspendUser.Handle(c.Request.Context(), userCmd.SuspendUserCommand{
		UserID: userID,
		Reason: req.Reason,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"user": userToResponse(result.User)})
}

func (h *UserHandler) ReactivateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid user ID format")
		return
	}

	result, err := h.reactivateUser.Handle(c.Request.Context(), userCmd.ReactivateUserCommand{
		UserID: userID,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"user": userToResponse(result.User)})
}

func (h *UserHandler) SetAdminStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_id", "invalid user ID format")
		return
	}

	var req SetAdminStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.setAdminStatus.Handle(c.Request.Context(), userCmd.SetAdminStatusCommand{
		UserID:  userID,
		IsAdmin: req.IsAdmin,
	})
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	respondSuccess(c, gin.H{"user": userToResponse(result.User)})
}

// --- Response helpers ---

func userToResponse(u *user.User) gin.H {
	provider := "email"
	if u.OAuthProvider() != nil {
		provider = *u.OAuthProvider()
	}
	resp := gin.H{
		"id":             u.ID(),
		"email":          u.Email(),
		"name":           u.Name(),
		"provider":       provider,
		"status":         u.Status(),
		"email_verified": u.EmailVerified(),
		"is_admin":       u.IsAdmin(),
		"created_at":     u.CreatedAt(),
	}
	if u.AvatarURL() != "" {
		resp["avatar_url"] = u.AvatarURL()
	}
	if u.SuspendedAt() != nil {
		resp["suspended_at"] = u.SuspendedAt()
	}
	if u.SuspendReason() != nil {
		resp["suspend_reason"] = *u.SuspendReason()
	}
	return resp
}

func (h *UserHandler) handleUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		respondError(c, http.StatusNotFound, "user_not_found", "User not found")
	case errors.Is(err, user.ErrEmailAlreadyExists):
		respondError(c, http.StatusConflict, "email_already_registered", "Email already registered")
	case errors.Is(err, user.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
	case errors.Is(err, user.ErrEmailNotVerified):
		respondError(c, http.StatusForbidden, "email_not_verified", "Please verify your email before logging in")
	case errors.Is(err, user.ErrAccountSuspended):
		respondError(c, http.StatusForbidden, "account_suspended", "Your account has been suspended")
	case errors.Is(err, user.ErrAccountDeleted):
		respondError(c, http.StatusForbidden, "account_deleted", "This account has been deleted")
	case errors.Is(err, user.ErrIncorrectPassword):
		respondError(c, http.StatusUnauthorized, "incorrect_password", "Current password is incorrect")
	case errors.Is(err, user.ErrPasswordTooShort):
		respondError(c, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters")
	case errors.Is(err, user.ErrPasswordRequired):
		respondError(c, http.StatusBadRequest, "password_required", "Password is required")
	case errors.Is(err, user.ErrInvalidToken):
		respondError(c, http.StatusBadRequest, "invalid_token", "Invalid verification token")
	case errors.Is(err, user.ErrTokenExpired):
		respondError(c, http.StatusBadRequest, "token_expired", "Verification token has expired")
	case errors.Is(err, user.ErrTokenAlreadyUsed):
		respondError(c, http.StatusBadRequest, "token_already_used", "Verification token has already been used")
	case errors.Is(err, user.ErrCannotSuspendAdmin):
		respondError(c, http.StatusBadRequest, "cannot_suspend_admin", "Cannot suspend admin users")
	case errors.Is(err, user.ErrAlreadySuspended):
		respondError(c, http.StatusBadRequest, "already_suspended", "User is already suspended")
	case errors.Is(err, user.ErrUserNotSuspended):
		respondError(c, http.StatusBadRequest, "user_not_suspended", "User is not suspended")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An internal error occurred")
	}
}
