package user

import (
	"time"

	"github.com/google/uuid"
)

// VerificationPurpose represents the purpose of an email verification token.
type VerificationPurpose string

const (
	PurposeRegistration  VerificationPurpose = "registration"
	PurposePasswordReset VerificationPurpose = "password_reset"
)

// EmailVerification represents an email verification or password reset token.
type EmailVerification struct {
	id        uuid.UUID
	userID    uuid.UUID
	token     string
	purpose   VerificationPurpose
	expiresAt time.Time
	usedAt    *time.Time
	createdAt time.Time
}

// NewEmailVerification creates a new email verification token.
func NewEmailVerification(userID uuid.UUID, token string, purpose VerificationPurpose, expiresAt time.Time) *EmailVerification {
	return &EmailVerification{
		id:        uuid.New(),
		userID:    userID,
		token:     token,
		purpose:   purpose,
		expiresAt: expiresAt,
		createdAt: time.Now(),
	}
}

// RestoreEmailVerification recreates an EmailVerification from persisted data.
func RestoreEmailVerification(
	id, userID uuid.UUID,
	token string,
	purpose VerificationPurpose,
	expiresAt time.Time,
	usedAt *time.Time,
	createdAt time.Time,
) *EmailVerification {
	return &EmailVerification{
		id:        id,
		userID:    userID,
		token:     token,
		purpose:   purpose,
		expiresAt: expiresAt,
		usedAt:    usedAt,
		createdAt: createdAt,
	}
}

// --- Getters ---

func (v *EmailVerification) ID() uuid.UUID              { return v.id }
func (v *EmailVerification) UserID() uuid.UUID          { return v.userID }
func (v *EmailVerification) Token() string              { return v.token }
func (v *EmailVerification) Purpose() VerificationPurpose { return v.purpose }
func (v *EmailVerification) ExpiresAt() time.Time       { return v.expiresAt }
func (v *EmailVerification) UsedAt() *time.Time         { return v.usedAt }
func (v *EmailVerification) CreatedAt() time.Time       { return v.createdAt }

// --- Domain Methods ---

// IsExpired checks if the verification token has expired.
func (v *EmailVerification) IsExpired() bool {
	return time.Now().After(v.expiresAt)
}

// IsUsed checks if the verification token has been used.
func (v *EmailVerification) IsUsed() bool {
	return v.usedAt != nil
}

// IsValid checks if the verification token is still valid.
func (v *EmailVerification) IsValid() bool {
	return !v.IsExpired() && !v.IsUsed()
}

// MarkAsUsed marks the token as used.
func (v *EmailVerification) MarkAsUsed() error {
	if v.IsUsed() {
		return ErrTokenAlreadyUsed
	}
	if v.IsExpired() {
		return ErrTokenExpired
	}

	now := time.Now()
	v.usedAt = &now
	return nil
}
