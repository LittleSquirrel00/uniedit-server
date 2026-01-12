package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/uniedit/server/api/pb/common"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserDomain defines user domain service interface.
type UserDomain interface {
	// User profile operations
	GetProfile(ctx context.Context, userID uuid.UUID) (*userv1.Profile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, in *userv1.UpdateProfileRequest) (*commonv1.User, error)
	GetPreferences(ctx context.Context, userID uuid.UUID) (*userv1.Preferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, in *userv1.UpdatePreferencesRequest) (*commonv1.MessageResponse, error)
	UploadAvatar(ctx context.Context, userID uuid.UUID, in *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error)

	// User account operations
	GetUser(ctx context.Context, id uuid.UUID) (*commonv1.User, error)
	GetUserByEmail(ctx context.Context, email string) (*commonv1.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, in *userv1.ChangePasswordRequest) (*commonv1.MessageResponse, error)
	DeleteAccount(ctx context.Context, userID uuid.UUID, in *userv1.DeleteAccountRequest) (*commonv1.MessageResponse, error)

	// Registration and verification
	Register(ctx context.Context, in *userv1.RegisterRequest) (*userv1.RegisterResponse, error)
	VerifyEmail(ctx context.Context, in *userv1.VerifyEmailRequest) (*commonv1.MessageResponse, error)
	ResendVerification(ctx context.Context, in *userv1.ResendVerificationRequest) (*commonv1.MessageResponse, error)

	// Password reset
	RequestPasswordReset(ctx context.Context, in *userv1.RequestPasswordResetRequest) (*commonv1.MessageResponse, error)
	ResetPassword(ctx context.Context, in *userv1.CompletePasswordResetRequest) (*commonv1.MessageResponse, error)

	// Admin operations
	ListUsers(ctx context.Context, in *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
	GetUserByID(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.User, error)
	SuspendUser(ctx context.Context, in *userv1.SuspendUserRequest) (*commonv1.MessageResponse, error)
	ReactivateUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error)
	SetAdminStatus(ctx context.Context, in *userv1.SetAdminStatusRequest) (*commonv1.MessageResponse, error)
	AdminDeleteUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error)
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

func (d *userDomain) GetProfile(ctx context.Context, userID uuid.UUID) (*userv1.Profile, error) {
	if d.profileDB == nil {
		// Fallback to user data if profile DB is not configured
		user, err := d.userDB.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrUserNotFound
		}
		return toProfilePB(&model.Profile{
			UserID:      user.ID,
			DisplayName: user.Name,
			AvatarURL:   user.AvatarURL,
			UpdatedAt:   user.UpdatedAt,
		}), nil
	}

	profile, err := d.profileDB.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}
	return toProfilePB(profile), nil
}

func (d *userDomain) UpdateProfile(ctx context.Context, userID uuid.UUID, in *userv1.UpdateProfileRequest) (*commonv1.User, error) {
	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if in != nil && in.GetName() != "" {
		user.Name = in.GetName()
	}
	if in != nil && in.GetAvatarUrl() != "" {
		user.AvatarURL = in.GetAvatarUrl()
	}

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return toCommonUserPB(user), nil
}

func (d *userDomain) GetPreferences(ctx context.Context, userID uuid.UUID) (*userv1.Preferences, error) {
	if d.prefsDB == nil {
		// Return default preferences if not configured
		return toPreferencesPB(&model.Preferences{
			UserID:   userID,
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		}), nil
	}
	prefs, err := d.prefsDB.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toPreferencesPB(prefs), nil
}

func (d *userDomain) UpdatePreferences(ctx context.Context, userID uuid.UUID, in *userv1.UpdatePreferencesRequest) (*commonv1.MessageResponse, error) {
	if d.prefsDB == nil {
		return &commonv1.MessageResponse{Message: "preferences updated"}, nil
	}

	prefs := &model.Preferences{UserID: userID}
	if in != nil {
		prefs.Theme = in.GetTheme()
		prefs.Language = in.GetLanguage()
		prefs.Timezone = in.GetTimezone()
	}

	if err := d.prefsDB.UpdatePreferences(ctx, prefs); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "preferences updated"}, nil
}

func (d *userDomain) UploadAvatar(ctx context.Context, userID uuid.UUID, in *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error) {
	if d.avatarStorage == nil {
		return nil, fmt.Errorf("avatar storage not configured")
	}
	if in == nil {
		return nil, ErrInvalidRequest
	}
	url, err := d.avatarStorage.Upload(ctx, userID, in.GetData(), in.GetContentType())
	if err != nil {
		return nil, err
	}
	return &userv1.UploadAvatarResponse{Url: url}, nil
}

// --- User Account Operations ---

func (d *userDomain) GetUser(ctx context.Context, id uuid.UUID) (*commonv1.User, error) {
	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return toCommonUserPB(user), nil
}

func (d *userDomain) GetUserByEmail(ctx context.Context, email string) (*commonv1.User, error) {
	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return toCommonUserPB(user), nil
}

func (d *userDomain) ChangePassword(ctx context.Context, userID uuid.UUID, in *userv1.ChangePasswordRequest) (*commonv1.MessageResponse, error) {
	currentPassword := ""
	newPassword := ""
	if in != nil {
		currentPassword = in.GetCurrentPassword()
		newPassword = in.GetNewPassword()
	}

	if len(newPassword) < 8 {
		return nil, ErrPasswordTooShort
	}

	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if user.PasswordHash == nil {
		return nil, ErrPasswordRequired
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword)); err != nil {
		return nil, ErrIncorrectPassword
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	user.PasswordHash = &hashStr

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "password changed"}, nil
}

func (d *userDomain) DeleteAccount(ctx context.Context, userID uuid.UUID, in *userv1.DeleteAccountRequest) (*commonv1.MessageResponse, error) {
	password := ""
	if in != nil {
		password = in.GetPassword()
	}

	user, err := d.userDB.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// For email users, verify password
	if user.IsEmailUser() {
		if password == "" {
			return nil, ErrPasswordRequired
		}
		if user.PasswordHash == nil {
			return nil, ErrPasswordRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
			return nil, ErrIncorrectPassword
		}
	}

	// Soft delete
	if err := d.userDB.SoftDelete(ctx, userID); err != nil {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "account deleted"}, nil
}

// --- Registration and Verification ---

func (d *userDomain) Register(ctx context.Context, in *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	// Check if email already exists
	existing, err := d.userDB.FindByEmail(ctx, in.GetEmail())
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Validate password
	if len(in.GetPassword()) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(in.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Create user
	user := &model.User{
		ID:            uuid.New(),
		Email:         in.GetEmail(),
		Name:          in.GetName(),
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

	return &userv1.RegisterResponse{
		Message: "registration successful, please check your email for verification",
		User:    toCommonUserPB(user),
	}, nil
}

func (d *userDomain) VerifyEmail(ctx context.Context, in *userv1.VerifyEmailRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetToken() == "" {
		return nil, ErrInvalidRequest
	}

	verification, err := d.verificationDB.GetVerificationByToken(ctx, in.GetToken())
	if err != nil {
		return nil, ErrInvalidToken
	}

	if verification.IsUsed() {
		return nil, ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return nil, ErrTokenExpired
	}
	if verification.Purpose != model.VerificationPurposeRegistration {
		return nil, ErrInvalidToken
	}

	// Get user
	user, err := d.userDB.FindByID(ctx, verification.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// Mark token as used
	if err := d.verificationDB.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Update user status
	user.EmailVerified = true
	if user.Status == model.UserStatusPending {
		user.Status = model.UserStatusActive
	}

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "email verified successfully"}, nil
}

func (d *userDomain) ResendVerification(ctx context.Context, in *userv1.ResendVerificationRequest) (*commonv1.MessageResponse, error) {
	email := ""
	if in != nil {
		email = in.GetEmail()
	}

	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// Don't reveal if email exists
		return &commonv1.MessageResponse{Message: "verification email sent if account exists"}, nil
	}

	if user.EmailVerified {
		return &commonv1.MessageResponse{Message: "verification email sent if account exists"}, nil
	}

	// Invalidate existing tokens
	if err := d.verificationDB.InvalidateUserVerifications(ctx, user.ID, model.VerificationPurposeRegistration); err != nil {
		d.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	if err := d.sendVerificationEmail(ctx, user, model.VerificationPurposeRegistration); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "verification email sent if account exists"}, nil
}

// --- Password Reset ---

func (d *userDomain) RequestPasswordReset(ctx context.Context, in *userv1.RequestPasswordResetRequest) (*commonv1.MessageResponse, error) {
	email := ""
	if in != nil {
		email = in.GetEmail()
	}

	user, err := d.userDB.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// Don't reveal if email exists
		return &commonv1.MessageResponse{Message: "password reset email sent if account exists"}, nil
	}

	// Only allow for email users
	if !user.IsEmailUser() {
		return &commonv1.MessageResponse{Message: "password reset email sent if account exists"}, nil
	}

	// Invalidate existing reset tokens
	if err := d.verificationDB.InvalidateUserVerifications(ctx, user.ID, model.VerificationPurposePasswordReset); err != nil {
		d.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	if err := d.sendVerificationEmail(ctx, user, model.VerificationPurposePasswordReset); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "password reset email sent if account exists"}, nil
}

func (d *userDomain) ResetPassword(ctx context.Context, in *userv1.CompletePasswordResetRequest) (*commonv1.MessageResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}
	token := in.GetToken()
	newPassword := in.GetNewPassword()

	if len(newPassword) < 8 {
		return nil, ErrPasswordTooShort
	}

	verification, err := d.verificationDB.GetVerificationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if verification.IsUsed() {
		return nil, ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return nil, ErrTokenExpired
	}
	if verification.Purpose != model.VerificationPurposePasswordReset {
		return nil, ErrInvalidToken
	}

	// Get user
	user, err := d.userDB.FindByID(ctx, verification.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Mark token as used
	if err := d.verificationDB.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Update password
	user.PasswordHash = &hashStr
	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "password reset successfully"}, nil
}

// --- Admin Operations ---

func (d *userDomain) ListUsers(ctx context.Context, in *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	filter := model.UserFilter{}

	if in != nil {
		if len(in.GetIds()) > 0 {
			filter.IDs = make([]uuid.UUID, 0, len(in.GetIds()))
			for _, idStr := range in.GetIds() {
				if idStr == "" {
					continue
				}
				id, err := uuid.Parse(idStr)
				if err != nil {
					return nil, ErrInvalidRequest
				}
				filter.IDs = append(filter.IDs, id)
			}
		}
		if in.GetEmail() != "" {
			filter.Email = in.GetEmail()
		}
		if in.GetStatus() != "" {
			status := model.UserStatus(in.GetStatus())
			if !status.IsValid() {
				return nil, ErrInvalidRequest
			}
			filter.Status = &status
		}
		if in.GetSearch() != "" {
			filter.Search = in.GetSearch()
		}
		if in.GetPage() > 0 {
			filter.Page = int(in.GetPage())
		}
		if in.GetPageSize() > 0 {
			filter.PageSize = int(in.GetPageSize())
		}
		filter.DefaultPagination()

		if in.GetIsAdmin() != nil {
			v := in.GetIsAdmin().GetValue()
			filter.IsAdmin = &v
		}
	} else {
		filter.DefaultPagination()
	}

	users, total, err := d.userDB.FindByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make([]*commonv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, toCommonUserPB(u))
	}

	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = int32((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	return &userv1.ListUsersResponse{
		Data:       out,
		Total:      total,
		Page:       int32(filter.Page),
		PageSize:   int32(filter.PageSize),
		TotalPages: totalPages,
	}, nil
}

func (d *userDomain) GetUserByID(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.User, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}
	return d.GetUser(ctx, id)
}

func (d *userDomain) SuspendUser(ctx context.Context, in *userv1.SuspendUserRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if user.IsAdmin {
		return nil, ErrCannotSuspendAdmin
	}

	now := time.Now()
	user.Status = model.UserStatusSuspended
	user.SuspendedAt = &now
	reason := in.GetReason()
	user.SuspendReason = &reason

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "user suspended"}, nil
}

func (d *userDomain) ReactivateUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if user.Status != model.UserStatusSuspended {
		return nil, ErrUserAlreadyActive
	}

	user.Status = model.UserStatusActive
	user.SuspendedAt = nil
	user.SuspendReason = nil

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "user reactivated"}, nil
}

func (d *userDomain) SetAdminStatus(ctx context.Context, in *userv1.SetAdminStatusRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.IsAdmin = in.GetIsAdmin()

	if err := d.userDB.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "admin status updated"}, nil
}

func (d *userDomain) AdminDeleteUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	user, err := d.userDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if user.IsAdmin {
		return nil, ErrCannotSuspendAdmin
	}

	if err := d.userDB.SoftDelete(ctx, id); err != nil {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	return &commonv1.MessageResponse{Message: "user deleted"}, nil
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
