package user

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

// User represents a registered user with extended fields.
type User struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email         string     `json:"email" gorm:"uniqueIndex;not null"`
	Name          string     `json:"name" gorm:"not null"`
	AvatarURL     string     `json:"avatar_url,omitempty"`

	// Authentication
	OAuthProvider *string    `json:"oauth_provider,omitempty"` // github, google, nil for email users
	OAuthID       *string    `json:"-"`                        // OAuth provider ID
	PasswordHash  *string    `json:"-"`                        // bcrypt hash for email users

	// Status
	Status        UserStatus `json:"status" gorm:"default:active"`
	EmailVerified bool       `json:"email_verified" gorm:"default:false"`
	IsAdmin       bool       `json:"is_admin" gorm:"default:false"`

	// Suspension
	SuspendedAt   *time.Time `json:"suspended_at,omitempty"`
	SuspendReason *string    `json:"suspend_reason,omitempty"`

	// Timestamps
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"-" gorm:"index"` // Soft delete
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
