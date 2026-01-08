package model

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle status of a user.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusPending   UserStatus = "pending"   // Awaiting email verification
	UserStatusSuspended UserStatus = "suspended" // Admin suspended
	UserStatusDeleted   UserStatus = "deleted"   // Soft deleted
)

// IsValid checks if the status is a valid user status.
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusPending, UserStatusSuspended, UserStatusDeleted:
		return true
	default:
		return false
	}
}

// User represents a registered user.
type User struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string     `json:"email" gorm:"uniqueIndex;not null"`
	Name      string     `json:"name" gorm:"not null"`
	AvatarURL string     `json:"avatar_url,omitempty" gorm:"column:avatar_url"`

	// Authentication
	OAuthProvider *string `json:"oauth_provider,omitempty" gorm:"column:oauth_provider"`
	OAuthID       *string `json:"-" gorm:"column:oauth_id"`
	PasswordHash  *string `json:"-" gorm:"column:password_hash"`

	// Status
	Status        UserStatus `json:"status" gorm:"default:active"`
	EmailVerified bool       `json:"email_verified" gorm:"column:email_verified;default:false"`
	IsAdmin       bool       `json:"is_admin" gorm:"column:is_admin;default:false"`

	// Suspension
	SuspendedAt   *time.Time `json:"suspended_at,omitempty" gorm:"column:suspended_at"`
	SuspendReason *string    `json:"suspend_reason,omitempty" gorm:"column:suspend_reason"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"column:deleted_at;index"`
}

// TableName returns the database table name.
func (User) TableName() string {
	return "users"
}

// IsEmailUser returns true if the user registered with email/password.
func (u *User) IsEmailUser() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}

// IsOAuthUser returns true if the user registered via OAuth.
func (u *User) IsOAuthUser() bool {
	return u.OAuthProvider != nil && *u.OAuthProvider != ""
}

// CanLogin checks if the user is allowed to login.
func (u *User) CanLogin() bool {
	return u.Status == UserStatusActive && u.EmailVerified
}

// UserInput represents user creation/update input.
type UserInput struct {
	Email     string `json:"email" binding:"required,email"`
	Name      string `json:"name" binding:"required"`
	AvatarURL string `json:"avatar_url"`
}

// UserFilter represents user query filters.
type UserFilter struct {
	IDs     []uuid.UUID `json:"ids"`
	Email   string      `json:"email"`
	Status  *UserStatus `json:"status"`
	IsAdmin *bool       `json:"is_admin"`
	Search  string      `json:"search"`
	PaginationRequest
}

// Profile represents user profile for display.
type Profile struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	AvatarURL   string    `json:"avatar_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Preferences represents user preferences.
type Preferences struct {
	UserID   uuid.UUID `json:"user_id"`
	Theme    string    `json:"theme"`
	Language string    `json:"language"`
	Timezone string    `json:"timezone"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Provider      string     `json:"provider"`
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

// VerificationPurpose represents the purpose of an email verification token.
type VerificationPurpose string

const (
	VerificationPurposeRegistration  VerificationPurpose = "registration"
	VerificationPurposePasswordReset VerificationPurpose = "password_reset"
)

// EmailVerification represents an email verification or password reset token.
type EmailVerification struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID           `gorm:"type:uuid;not null;index"`
	Token     string              `gorm:"not null;uniqueIndex"`
	Purpose   VerificationPurpose `gorm:"not null"`
	ExpiresAt time.Time           `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

// TableName returns the database table name.
func (EmailVerification) TableName() string {
	return "email_verifications"
}

// IsExpired checks if the verification token has expired.
func (v *EmailVerification) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

// IsUsed checks if the verification token has been used.
func (v *EmailVerification) IsUsed() bool {
	return v.UsedAt != nil
}

// IsValid checks if the verification token is still valid.
func (v *EmailVerification) IsValid() bool {
	return !v.IsExpired() && !v.IsUsed()
}
