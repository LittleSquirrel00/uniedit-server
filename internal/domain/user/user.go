package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle status of a user.
type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusPending   UserStatus = "pending"   // Awaiting email verification
	StatusSuspended UserStatus = "suspended" // Admin suspended
	StatusDeleted   UserStatus = "deleted"   // Soft deleted
)

// IsValid checks if the status is a valid user status.
func (s UserStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusPending, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// User is the aggregate root for user management.
type User struct {
	id            uuid.UUID
	email         string
	name          string
	avatarURL     string
	oauthProvider *string
	oauthID       *string
	passwordHash  *string
	status        UserStatus
	emailVerified bool
	isAdmin       bool
	suspendedAt   *time.Time
	suspendReason *string
	createdAt     time.Time
	updatedAt     time.Time
	deletedAt     *time.Time
}

// NewUser creates a new User with email registration.
func NewUser(email, name string) (*User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	now := time.Now()
	return &User{
		id:            uuid.New(),
		email:         email,
		name:          name,
		status:        StatusPending,
		emailVerified: false,
		isAdmin:       false,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// NewOAuthUser creates a new User from OAuth registration.
func NewOAuthUser(email, name, provider, oauthID string) (*User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if provider == "" {
		return nil, fmt.Errorf("oauth provider cannot be empty")
	}

	now := time.Now()
	return &User{
		id:            uuid.New(),
		email:         email,
		name:          name,
		oauthProvider: &provider,
		oauthID:       &oauthID,
		status:        StatusActive,
		emailVerified: true,
		isAdmin:       false,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// RestoreUser recreates a User from persisted data.
func RestoreUser(
	id uuid.UUID,
	email, name, avatarURL string,
	oauthProvider, oauthID, passwordHash *string,
	status UserStatus,
	emailVerified, isAdmin bool,
	suspendedAt *time.Time,
	suspendReason *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *User {
	return &User{
		id:            id,
		email:         email,
		name:          name,
		avatarURL:     avatarURL,
		oauthProvider: oauthProvider,
		oauthID:       oauthID,
		passwordHash:  passwordHash,
		status:        status,
		emailVerified: emailVerified,
		isAdmin:       isAdmin,
		suspendedAt:   suspendedAt,
		suspendReason: suspendReason,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deletedAt:     deletedAt,
	}
}

// --- Getters ---

func (u *User) ID() uuid.UUID           { return u.id }
func (u *User) Email() string           { return u.email }
func (u *User) Name() string            { return u.name }
func (u *User) AvatarURL() string       { return u.avatarURL }
func (u *User) OAuthProvider() *string  { return u.oauthProvider }
func (u *User) OAuthID() *string        { return u.oauthID }
func (u *User) PasswordHash() *string   { return u.passwordHash }
func (u *User) Status() UserStatus      { return u.status }
func (u *User) EmailVerified() bool     { return u.emailVerified }
func (u *User) IsAdmin() bool           { return u.isAdmin }
func (u *User) SuspendedAt() *time.Time { return u.suspendedAt }
func (u *User) SuspendReason() *string  { return u.suspendReason }
func (u *User) CreatedAt() time.Time    { return u.createdAt }
func (u *User) UpdatedAt() time.Time    { return u.updatedAt }
func (u *User) DeletedAt() *time.Time   { return u.deletedAt }

// --- Domain Methods ---

// IsEmailUser returns true if the user registered with email/password.
func (u *User) IsEmailUser() bool {
	return u.passwordHash != nil && *u.passwordHash != ""
}

// IsOAuthUser returns true if the user registered via OAuth.
func (u *User) IsOAuthUser() bool {
	return u.oauthProvider != nil && *u.oauthProvider != ""
}

// CanLogin checks if the user is allowed to login.
func (u *User) CanLogin() bool {
	return u.status == StatusActive && u.emailVerified
}

// SetPasswordHash sets the password hash.
func (u *User) SetPasswordHash(hash string) {
	u.passwordHash = &hash
	u.updatedAt = time.Now()
}

// SetName updates the user's name.
func (u *User) SetName(name string) {
	u.name = name
	u.updatedAt = time.Now()
}

// SetAvatarURL updates the user's avatar URL.
func (u *User) SetAvatarURL(url string) {
	u.avatarURL = url
	u.updatedAt = time.Now()
}

// VerifyEmail marks the email as verified.
func (u *User) VerifyEmail() error {
	if u.emailVerified {
		return ErrEmailAlreadyVerified
	}

	u.emailVerified = true
	if u.status == StatusPending {
		u.status = StatusActive
	}
	u.updatedAt = time.Now()
	return nil
}

// Suspend suspends the user account.
func (u *User) Suspend(reason string) error {
	if u.isAdmin {
		return ErrCannotSuspendAdmin
	}
	if u.status == StatusSuspended {
		return ErrAlreadySuspended
	}

	now := time.Now()
	u.status = StatusSuspended
	u.suspendedAt = &now
	u.suspendReason = &reason
	u.updatedAt = now
	return nil
}

// Reactivate reactivates a suspended user.
func (u *User) Reactivate() error {
	if u.status != StatusSuspended {
		return ErrUserNotSuspended
	}

	u.status = StatusActive
	u.suspendedAt = nil
	u.suspendReason = nil
	u.updatedAt = time.Now()
	return nil
}

// SoftDelete marks the user as deleted.
func (u *User) SoftDelete() error {
	if u.status == StatusDeleted {
		return ErrAlreadyDeleted
	}

	now := time.Now()
	u.status = StatusDeleted
	u.deletedAt = &now
	u.updatedAt = now
	return nil
}

// SetAdminStatus sets the admin status.
func (u *User) SetAdminStatus(isAdmin bool) {
	u.isAdmin = isAdmin
	u.updatedAt = time.Now()
}
