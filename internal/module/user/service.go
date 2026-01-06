package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/auth"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// TokenPair is re-exported from auth for convenience.
type TokenPair = auth.TokenPair

// Service provides user management operations.
type Service struct {
	repo       Repository
	tokenRepo  auth.RefreshTokenRepository
	jwt        *auth.JWTManager
	logger     *zap.Logger
	emailer    EmailSender
}

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendVerificationEmail(ctx context.Context, email, name, token string) error
	SendPasswordResetEmail(ctx context.Context, email, name, token string) error
}

// ServiceConfig holds service configuration.
type ServiceConfig struct {
	JWTConfig *auth.JWTConfig
}

// NewService creates a new user service.
func NewService(
	repo Repository,
	tokenRepo auth.RefreshTokenRepository,
	jwt *auth.JWTManager,
	emailer EmailSender,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:      repo,
		tokenRepo: tokenRepo,
		jwt:       jwt,
		emailer:   emailer,
		logger:    logger,
	}
}

// --- Registration ---

// Register creates a new user with email and password.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	// Check if email already exists
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}
	if err != nil && err != ErrUserNotFound {
		return nil, fmt.Errorf("check email: %w", err)
	}

	// Validate password
	if len(req.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Create user
	user := &User{
		ID:            uuid.New(),
		Email:         req.Email,
		Name:          req.Name,
		PasswordHash:  &hashStr,
		Status:        UserStatusPending,
		EmailVerified: false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate and send verification email
	if err := s.sendVerificationEmail(ctx, user, VerificationPurposeRegistration); err != nil {
		s.logger.Error("failed to send verification email", zap.Error(err), zap.String("email", user.Email))
		// Don't fail registration if email fails
	}

	return user, nil
}

// VerifyEmail verifies a user's email address.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	verification, err := s.repo.GetVerificationByToken(ctx, token)
	if err != nil {
		return err
	}

	if verification.IsUsed() {
		return ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return ErrTokenExpired
	}
	if verification.Purpose != VerificationPurposeRegistration {
		return ErrInvalidToken
	}

	// Get user
	user, err := s.repo.GetByID(ctx, verification.UserID)
	if err != nil {
		return err
	}

	// Mark token as used
	if err := s.repo.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// Update user status
	user.EmailVerified = true
	if user.Status == UserStatusPending {
		user.Status = UserStatusActive
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// ResendVerification sends a new verification email.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	if user.EmailVerified {
		return nil
	}

	// Invalidate existing tokens
	if err := s.repo.InvalidateUserVerifications(ctx, user.ID, VerificationPurposeRegistration); err != nil {
		s.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	// Send new verification
	return s.sendVerificationEmail(ctx, user, VerificationPurposeRegistration)
}

// --- Login ---

// Login authenticates a user with email and password.
func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (*TokenPair, *User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Check if user can login
	if user.Status == UserStatusDeleted {
		return nil, nil, ErrAccountDeleted
	}
	if user.Status == UserStatusSuspended {
		return nil, nil, ErrAccountSuspended
	}
	if !user.EmailVerified {
		return nil, nil, ErrEmailNotVerified
	}

	// Verify password
	if user.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

// --- Password Management ---

// RequestPasswordReset initiates a password reset.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Only allow for email users
	if !user.IsEmailUser() {
		return nil
	}

	// Invalidate existing reset tokens
	if err := s.repo.InvalidateUserVerifications(ctx, user.ID, VerificationPurposePasswordReset); err != nil {
		s.logger.Error("failed to invalidate tokens", zap.Error(err))
	}

	// Send reset email
	return s.sendVerificationEmail(ctx, user, VerificationPurposePasswordReset)
}

// ResetPassword completes a password reset.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	verification, err := s.repo.GetVerificationByToken(ctx, token)
	if err != nil {
		return err
	}

	if verification.IsUsed() {
		return ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return ErrTokenExpired
	}
	if verification.Purpose != VerificationPurposePasswordReset {
		return ErrInvalidToken
	}

	// Get user
	user, err := s.repo.GetByID(ctx, verification.UserID)
	if err != nil {
		return err
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	// Mark token as used
	if err := s.repo.MarkVerificationUsed(ctx, verification.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// Update password
	user.PasswordHash = &hashStr
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Revoke all refresh tokens
	if err := s.tokenRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		s.logger.Error("failed to revoke tokens", zap.Error(err))
	}

	return nil
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
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

	// Update password
	user.PasswordHash = &hashStr
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// --- User Operations ---

// GetUser returns a user by ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateProfile updates a user's profile.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

// DeleteAccount deletes a user's account.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
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

	// Revoke all tokens
	if err := s.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		s.logger.Error("failed to revoke tokens", zap.Error(err))
	}

	// Soft delete
	if err := s.repo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// --- Admin Operations ---

// ListUsers returns a paginated list of users.
func (s *Service) ListUsers(ctx context.Context, filter *UserFilter, pagination *Pagination) ([]*User, int64, error) {
	return s.repo.List(ctx, filter, pagination)
}

// SuspendUser suspends a user account.
func (s *Service) SuspendUser(ctx context.Context, userID uuid.UUID, reason string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.IsAdmin {
		return ErrCannotSuspendAdmin
	}

	now := time.Now()
	user.Status = UserStatusSuspended
	user.SuspendedAt = &now
	user.SuspendReason = &reason

	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Revoke all tokens
	if err := s.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		s.logger.Error("failed to revoke tokens", zap.Error(err))
	}

	return nil
}

// ReactivateUser reactivates a suspended user.
func (s *Service) ReactivateUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Status != UserStatusSuspended {
		return ErrUserAlreadyActive
	}

	user.Status = UserStatusActive
	user.SuspendedAt = nil
	user.SuspendReason = nil

	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// SetAdminStatus sets a user's admin status.
func (s *Service) SetAdminStatus(ctx context.Context, userID uuid.UUID, isAdmin bool) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.IsAdmin = isAdmin

	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// --- Helpers ---

func (s *Service) generateTokenPair(ctx context.Context, user *User, userAgent, ipAddress string) (*TokenPair, error) {
	// Create auth.User for JWT generation
	authUser := &auth.User{
		ID:    user.ID,
		Email: user.Email,
	}

	accessToken, expiresAt, err := s.jwt.GenerateAccessToken(authUser)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefreshToken, tokenHash, refreshExpiresAt, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token
	refreshTokenRecord := &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: refreshExpiresAt,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := s.tokenRepo.Create(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.GetAccessTokenExpiry().Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) sendVerificationEmail(ctx context.Context, user *User, purpose VerificationPurpose) error {
	// Generate token
	token, err := generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	// Set expiration based on purpose
	var expiresAt time.Time
	switch purpose {
	case VerificationPurposeRegistration:
		expiresAt = time.Now().Add(24 * time.Hour)
	case VerificationPurposePasswordReset:
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	// Store verification
	verification := &EmailVerification{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     token,
		Purpose:   purpose,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		return fmt.Errorf("create verification: %w", err)
	}

	// Send email
	if s.emailer == nil {
		s.logger.Warn("email sender not configured, skipping email")
		return nil
	}

	switch purpose {
	case VerificationPurposeRegistration:
		return s.emailer.SendVerificationEmail(ctx, user.Email, user.Name, token)
	case VerificationPurposePasswordReset:
		return s.emailer.SendPasswordResetEmail(ctx, user.Email, user.Name, token)
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
