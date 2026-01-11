package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// UserDatabasePort defines user persistence operations.
type UserDatabasePort interface {
	// Create creates a new user.
	Create(ctx context.Context, user *model.User) error

	// FindByID finds a user by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	// FindByEmail finds a user by email.
	FindByEmail(ctx context.Context, email string) (*model.User, error)

	// FindByFilter finds users by filter.
	FindByFilter(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error)

	// Update updates a user.
	Update(ctx context.Context, user *model.User) error

	// SoftDelete soft deletes a user.
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// VerificationDatabasePort defines email verification persistence operations.
type VerificationDatabasePort interface {
	// CreateVerification creates a new email verification.
	CreateVerification(ctx context.Context, verification *model.EmailVerification) error

	// GetVerificationByToken gets a verification by token.
	GetVerificationByToken(ctx context.Context, token string) (*model.EmailVerification, error)

	// InvalidateUserVerifications invalidates all pending verifications for a user.
	InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose model.VerificationPurpose) error

	// MarkVerificationUsed marks a verification as used.
	MarkVerificationUsed(ctx context.Context, id uuid.UUID) error
}

// ProfileDatabasePort defines profile persistence operations.
type ProfileDatabasePort interface {
	// GetProfile gets user profile.
	GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error)

	// UpdateProfile updates user profile.
	UpdateProfile(ctx context.Context, profile *model.Profile) error
}

// PreferencesDatabasePort defines preferences persistence operations.
type PreferencesDatabasePort interface {
	// GetPreferences gets user preferences.
	GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error)

	// UpdatePreferences updates user preferences.
	UpdatePreferences(ctx context.Context, prefs *model.Preferences) error
}

// AvatarStoragePort defines avatar storage operations.
type AvatarStoragePort interface {
	// Upload uploads user avatar.
	Upload(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error)

	// Delete deletes user avatar.
	Delete(ctx context.Context, userID uuid.UUID) error

	// GetURL gets avatar URL.
	GetURL(ctx context.Context, userID uuid.UUID) (string, error)
}

// EmailSenderPort defines email sending operations.
type EmailSenderPort interface {
	// SendVerificationEmail sends a verification email.
	SendVerificationEmail(ctx context.Context, email, name, token string) error

	// SendPasswordResetEmail sends a password reset email.
	SendPasswordResetEmail(ctx context.Context, email, name, token string) error
}
