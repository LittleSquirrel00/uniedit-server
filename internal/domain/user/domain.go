package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserDomain defines user domain service interface.
type UserDomain interface {
	// User profile operations
	GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input *UpdateProfileInput) (*model.User, error)
	GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs *model.Preferences) error
	UploadAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error)

	// User account operations
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error

	// Registration and verification
	Register(ctx context.Context, input *RegisterInput) (*model.User, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error

	// Password reset
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error

	// Admin operations
	ListUsers(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error)
	SuspendUser(ctx context.Context, id uuid.UUID, reason string) error
	ReactivateUser(ctx context.Context, id uuid.UUID) error
	SetAdminStatus(ctx context.Context, id uuid.UUID, isAdmin bool) error
	AdminDeleteUser(ctx context.Context, id uuid.UUID) error
}

// RegisterInput represents registration input.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// UpdateProfileInput represents profile update input.
type UpdateProfileInput struct {
	Name      *string
	AvatarURL *string
}

// userDomain implements UserDomain.
type userDomain struct {
	userDB         outbound.UserDatabasePort
	verificationDB outbound.VerificationDatabasePort
	profileDB      outbound.ProfileDatabasePort
	prefsDB        outbound.PreferencesDatabasePort
	avatarStorage  outbound.AvatarStoragePort
	emailSender    outbound.EmailSenderPort
	logger         *zap.Logger
}

// Config holds domain configuration.
type Config struct {
	VerificationTokenExpiry time.Duration
	PasswordResetExpiry     time.Duration
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		VerificationTokenExpiry: 24 * time.Hour,
		PasswordResetExpiry:     1 * time.Hour,
	}
}

// NewUserDomain creates a new user domain service.
func NewUserDomain(
	userDB outbound.UserDatabasePort,
	verificationDB outbound.VerificationDatabasePort,
	profileDB outbound.ProfileDatabasePort,
	prefsDB outbound.PreferencesDatabasePort,
	avatarStorage outbound.AvatarStoragePort,
	emailSender outbound.EmailSenderPort,
	logger *zap.Logger,
) UserDomain {
	return &userDomain{
		userDB:         userDB,
		verificationDB: verificationDB,
		profileDB:      profileDB,
		prefsDB:        prefsDB,
		avatarStorage:  avatarStorage,
		emailSender:    emailSender,
		logger:         logger,
	}
}

// --- Profile Operations ---

func (d *userDomain) GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error) {
	if d.profileDB == nil {
		// Fallback to user data if profile DB is not configured
		user, err := d.userDB.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrUserNotFound
		}
		return &model.Profile{
			UserID:      user.ID,
			DisplayName: user.Name,
			AvatarURL:   user.AvatarURL,
			UpdatedAt:   user.UpdatedAt,
		}, nil
	}

	profile, err := d.profileDB.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}
	return profile, nil
}

func (d *userDomain) UpdateProfile(ctx context.Context, userID uuid.UUID, input *UpdateProfileInput) (*model.User, error) {
	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.AvatarURL != nil {
		user.AvatarURL = *input.AvatarURL
	}

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func (d *userDomain) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error) {
	if d.prefsDB == nil {
		// Return default preferences if not configured
		return &model.Preferences{
			UserID:   userID,
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		}, nil
	}
	return d.prefsDB.GetPreferences(ctx, userID)
}

func (d *userDomain) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs *model.Preferences) error {
	if d.prefsDB == nil {
		return nil
	}
	prefs.UserID = userID
	return d.prefsDB.UpdatePreferences(ctx, prefs)
}

func (d *userDomain) UploadAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error) {
	if d.avatarStorage == nil {
		return "", fmt.Errorf("avatar storage not configured")
	}
	return d.avatarStorage.Upload(ctx, userID, data, contentType)
}

// --- User Account Operations ---

func (d *userDomain) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (d *userDomain) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (d *userDomain) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.PasswordHash == nil {
		return ErrPasswordRequired
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrIncorrectPassword
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	user.PasswordHash = &hashStr

	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (d *userDomain) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// For email users, verify password
	if user.IsEmailUser() {
		if password == "" {
			return ErrPasswordRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
			return ErrIncorrectPassword
		}
	}

	// Soft delete
	if err := d.userDB.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// --- Registration and Verification ---

func (d *userDomain) Register(ctx context.Context, input *RegisterInput) (*model.User, error) {
	// Check if email already exists
	existing, err := d.userDB.FindByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Validate password
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Create user
	user := &model.User{
		ID:            uuid.New(),
		Email:         input.Email,
		Name:          input.Name,
		PasswordHash:  &hashStr,
		Status:        model.UserStatusPending,
		EmailVerified: false,
	}

	if err := d.userDB.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate and send verification email
	if err := d.sendVerificationEmail(ctx, user, model.VerificationPurposeRegistration); err != nil {
		d.logger.Error("failed to send verification email", zap.Error(err), zap.String("email", user.Email))
	}

	return user, nil
}

func (d *userDomain) VerifyEmail(ctx context.Context, token string) error {
	verification, err := d.verificationDB.GetVerificationByToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}

	if verification.IsUsed() {
		return ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return ErrTokenExpired
	}
	if verification.Purpose != model.VerificationPurposeRegistration {
		return ErrInvalidToken
	}

	// Get user
	user, err := d.userDB.FindByID(ctx, verification.UserID)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	// Mark token as used
	if err := d.verificationDB.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// Update user status
	user.EmailVerified = true
	if user.Status == model.UserStatusPending {
		user.Status = model.UserStatusActive
	}

	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (d *userDomain) ResendVerification(ctx context.Context, email string) error {
	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// Don't reveal if email exists
		return nil
	}

	if user.EmailVerified {
		return nil
	}

	// Invalidate existing tokens
	if err := d.verificationDB.InvalidateUserVerifications(ctx, user.ID, model.VerificationPurposeRegistration); err != nil {
		d.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	return d.sendVerificationEmail(ctx, user, model.VerificationPurposeRegistration)
}

// --- Password Reset ---

func (d *userDomain) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// Don't reveal if email exists
		return nil
	}

	// Only allow for email users
	if !user.IsEmailUser() {
		return nil
	}

	// Invalidate existing reset tokens
	if err := d.verificationDB.InvalidateUserVerifications(ctx, user.ID, model.VerificationPurposePasswordReset); err != nil {
		d.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	return d.sendVerificationEmail(ctx, user, model.VerificationPurposePasswordReset)
}

func (d *userDomain) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	verification, err := d.verificationDB.GetVerificationByToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}

	if verification.IsUsed() {
		return ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return ErrTokenExpired
	}
	if verification.Purpose != model.VerificationPurposePasswordReset {
		return ErrInvalidToken
	}

	// Get user
	user, err := d.userDB.FindByID(ctx, verification.UserID)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Mark token as used
	if err := d.verificationDB.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// Update password
	user.PasswordHash = &hashStr
	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// --- Admin Operations ---

func (d *userDomain) ListUsers(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error) {
	return d.userDB.FindByFilter(ctx, filter)
}

func (d *userDomain) SuspendUser(ctx context.Context, id uuid.UUID, reason string) error {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.IsAdmin {
		return ErrCannotSuspendAdmin
	}

	now := time.Now()
	user.Status = model.UserStatusSuspended
	user.SuspendedAt = &now
	user.SuspendReason = &reason

	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (d *userDomain) ReactivateUser(ctx context.Context, id uuid.UUID) error {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.Status != model.UserStatusSuspended {
		return ErrUserAlreadyActive
	}

	user.Status = model.UserStatusActive
	user.SuspendedAt = nil
	user.SuspendReason = nil

	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (d *userDomain) SetAdminStatus(ctx context.Context, id uuid.UUID, isAdmin bool) error {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.IsAdmin = isAdmin

	if err := d.userDB.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (d *userDomain) AdminDeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.IsAdmin {
		return ErrCannotSuspendAdmin
	}

	if err := d.userDB.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// --- Helpers ---

func (d *userDomain) sendVerificationEmail(ctx context.Context, user *model.User, purpose model.VerificationPurpose) error {
	// Generate token
	token, err := generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	// Set expiration based on purpose
	var expiresAt time.Time
	switch purpose {
	case model.VerificationPurposeRegistration:
		expiresAt = time.Now().Add(24 * time.Hour)
	case model.VerificationPurposePasswordReset:
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	// Store verification
	verification := &model.EmailVerification{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     token,
		Purpose:   purpose,
		ExpiresAt: expiresAt,
	}

	if err := d.verificationDB.CreateVerification(ctx, verification); err != nil {
		return fmt.Errorf("create verification: %w", err)
	}

	// Send email
	if d.emailSender == nil {
		d.logger.Warn("email sender not configured, skipping email")
		return nil
	}

	switch purpose {
	case model.VerificationPurposeRegistration:
		return d.emailSender.SendVerificationEmail(ctx, user.Email, user.Name, token)
	case model.VerificationPurposePasswordReset:
		return d.emailSender.SendPasswordResetEmail(ctx, user.Email, user.Name, token)
	}

	return nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
