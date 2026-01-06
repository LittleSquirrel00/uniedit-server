package user

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents an email/password login request.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// VerifyEmailRequest represents an email verification request.
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerificationRequest represents a request to resend verification email.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetRequest represents a password reset request.
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// CompletePasswordResetRequest represents completing a password reset.
type CompletePasswordResetRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePasswordRequest represents a password change request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdateProfileRequest represents a profile update request.
type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// DeleteAccountRequest represents an account deletion request.
type DeleteAccountRequest struct {
	Password string `json:"password"` // Required for email users
}

// SuspendUserRequest represents an admin request to suspend a user.
type SuspendUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// SetAdminStatusRequest represents a request to change admin status.
type SetAdminStatusRequest struct {
	IsAdmin bool `json:"is_admin"`
}

// UserFilter represents filters for listing users.
type UserFilter struct {
	Status *UserStatus `form:"status"`
	Email  *string     `form:"email"`
	IsAdmin *bool      `form:"is_admin"`
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

// NewPagination creates pagination with defaults.
func NewPagination() *Pagination {
	return &Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Offset returns the offset for database queries.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Provider      string     `json:"provider"` // oauth provider or "email"
	Status        UserStatus `json:"status"`
	EmailVerified bool       `json:"email_verified"`
	IsAdmin       bool       `json:"is_admin"`
	SuspendedAt   *time.Time `json:"suspended_at,omitempty"`
	SuspendReason *string    `json:"suspend_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ToResponse converts a User to UserResponse.
func (u *User) ToResponse() *UserResponse {
	provider := "email"
	if u.OAuthProvider != nil {
		provider = *u.OAuthProvider
	}
	return &UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		Provider:      provider,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
		IsAdmin:       u.IsAdmin,
		SuspendedAt:   u.SuspendedAt,
		SuspendReason: u.SuspendReason,
		CreatedAt:     u.CreatedAt,
	}
}

// UserListResponse represents a paginated list of users.
type UserListResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}
