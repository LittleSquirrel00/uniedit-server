package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
)

// UserEntity is the GORM model for users table.
type UserEntity struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email         string     `gorm:"uniqueIndex;not null"`
	Name          string     `gorm:"not null"`
	AvatarURL     string     `gorm:"column:avatar_url"`
	OAuthProvider *string    `gorm:"column:oauth_provider"`
	OAuthID       *string    `gorm:"column:oauth_id"`
	PasswordHash  *string    `gorm:"column:password_hash"`
	Status        string     `gorm:"default:pending"`
	EmailVerified bool       `gorm:"column:email_verified;default:false"`
	IsAdmin       bool       `gorm:"column:is_admin;default:false"`
	SuspendedAt   *time.Time `gorm:"column:suspended_at"`
	SuspendReason *string    `gorm:"column:suspend_reason"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index"`
}

// TableName returns the database table name.
func (UserEntity) TableName() string {
	return "users"
}

// ToDomain converts entity to domain model.
func (e *UserEntity) ToDomain() *user.User {
	return user.RestoreUser(
		e.ID,
		e.Email,
		e.Name,
		e.AvatarURL,
		e.OAuthProvider,
		e.OAuthID,
		e.PasswordHash,
		user.UserStatus(e.Status),
		e.EmailVerified,
		e.IsAdmin,
		e.SuspendedAt,
		e.SuspendReason,
		e.CreatedAt,
		e.UpdatedAt,
		e.DeletedAt,
	)
}

// FromDomainUser converts domain model to entity.
func FromDomainUser(u *user.User) *UserEntity {
	return &UserEntity{
		ID:            u.ID(),
		Email:         u.Email(),
		Name:          u.Name(),
		AvatarURL:     u.AvatarURL(),
		OAuthProvider: u.OAuthProvider(),
		OAuthID:       u.OAuthID(),
		PasswordHash:  u.PasswordHash(),
		Status:        string(u.Status()),
		EmailVerified: u.EmailVerified(),
		IsAdmin:       u.IsAdmin(),
		SuspendedAt:   u.SuspendedAt(),
		SuspendReason: u.SuspendReason(),
		CreatedAt:     u.CreatedAt(),
		UpdatedAt:     u.UpdatedAt(),
		DeletedAt:     u.DeletedAt(),
	}
}

// EmailVerificationEntity is the GORM model for email_verifications table.
type EmailVerificationEntity struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Token     string     `gorm:"not null;uniqueIndex"`
	Purpose   string     `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

// TableName returns the database table name.
func (EmailVerificationEntity) TableName() string {
	return "email_verifications"
}

// ToDomain converts entity to domain model.
func (e *EmailVerificationEntity) ToDomain() *user.EmailVerification {
	return user.RestoreEmailVerification(
		e.ID,
		e.UserID,
		e.Token,
		user.VerificationPurpose(e.Purpose),
		e.ExpiresAt,
		e.UsedAt,
		e.CreatedAt,
	)
}

// FromDomainVerification converts domain model to entity.
func FromDomainVerification(v *user.EmailVerification) *EmailVerificationEntity {
	return &EmailVerificationEntity{
		ID:        v.ID(),
		UserID:    v.UserID(),
		Token:     v.Token(),
		Purpose:   string(v.Purpose()),
		ExpiresAt: v.ExpiresAt(),
		UsedAt:    v.UsedAt(),
		CreatedAt: v.CreatedAt(),
	}
}
