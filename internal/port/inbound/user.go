package inbound

import "github.com/gin-gonic/gin"

// UserHttpPort defines HTTP handler interface for user operations.
type UserHttpPort interface {
	// GetProfile handles GET /users/me/profile
	GetProfile(c *gin.Context)

	// UpdateProfile handles PUT /users/me/profile
	UpdateProfile(c *gin.Context)

	// GetPreferences handles GET /users/me/preferences
	GetPreferences(c *gin.Context)

	// UpdatePreferences handles PUT /users/me/preferences
	UpdatePreferences(c *gin.Context)

	// UploadAvatar handles POST /users/me/avatar
	UploadAvatar(c *gin.Context)

	// GetMe handles GET /users/me
	GetMe(c *gin.Context)

	// ChangePassword handles POST /users/me/password
	ChangePassword(c *gin.Context)

	// DeleteAccount handles DELETE /users/me
	DeleteAccount(c *gin.Context)
}

// UserAdminPort defines admin interface for user management.
type UserAdminPort interface {
	// ListUsers handles GET /admin/users
	ListUsers(c *gin.Context)

	// GetUser handles GET /admin/users/:id
	GetUser(c *gin.Context)

	// SuspendUser handles POST /admin/users/:id/suspend
	SuspendUser(c *gin.Context)

	// ReactivateUser handles POST /admin/users/:id/reactivate
	ReactivateUser(c *gin.Context)

	// SetAdminStatus handles PUT /admin/users/:id/admin
	SetAdminStatus(c *gin.Context)

	// DeleteUser handles DELETE /admin/users/:id
	DeleteUser(c *gin.Context)
}

// UserAuthPort defines HTTP handler interface for user authentication.
type UserAuthPort interface {
	// Register handles POST /auth/register
	Register(c *gin.Context)

	// Login handles POST /auth/login
	Login(c *gin.Context)

	// VerifyEmail handles POST /auth/verify-email
	VerifyEmail(c *gin.Context)

	// ResendVerification handles POST /auth/resend-verification
	ResendVerification(c *gin.Context)

	// RequestPasswordReset handles POST /auth/password-reset
	RequestPasswordReset(c *gin.Context)

	// CompletePasswordReset handles POST /auth/password-reset/complete
	CompletePasswordReset(c *gin.Context)
}
