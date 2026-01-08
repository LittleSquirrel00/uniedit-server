package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// RegisterCommand represents a command to register a new user.
type RegisterCommand struct {
	Email    string
	Password string
	Name     string
}

// RegisterResult is the result of user registration.
type RegisterResult struct {
	User              *user.User
	VerificationToken string
}

// RegisterHandler handles RegisterCommand.
type RegisterHandler struct {
	repo user.Repository
}

// NewRegisterHandler creates a new handler.
func NewRegisterHandler(repo user.Repository) *RegisterHandler {
	return &RegisterHandler{repo: repo}
}

// Handle executes the command.
func (h *RegisterHandler) Handle(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
	// Check if email already exists
	existing, err := h.repo.GetByEmail(ctx, cmd.Email)
	if err == nil && existing != nil {
		return nil, user.ErrEmailAlreadyExists
	}
	if err != nil && err != user.ErrUserNotFound {
		return nil, fmt.Errorf("check email: %w", err)
	}

	// Validate password
	if len(cmd.Password) < 8 {
		return nil, user.ErrPasswordTooShort
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	u, err := user.NewUser(cmd.Email, cmd.Name)
	if err != nil {
		return nil, err
	}
	u.SetPasswordHash(string(hash))

	if err := h.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate verification token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create verification record
	verification := user.NewEmailVerification(
		u.ID(),
		token,
		user.PurposeRegistration,
		time.Now().Add(24*time.Hour),
	)

	if err := h.repo.CreateVerification(ctx, verification); err != nil {
		return nil, fmt.Errorf("create verification: %w", err)
	}

	return &RegisterResult{
		User:              u,
		VerificationToken: token,
	}, nil
}

// VerifyEmailCommand represents a command to verify email.
type VerifyEmailCommand struct {
	Token string
}

// VerifyEmailResult is the result of email verification.
type VerifyEmailResult struct {
	User *user.User
}

// VerifyEmailHandler handles VerifyEmailCommand.
type VerifyEmailHandler struct {
	repo user.Repository
}

// NewVerifyEmailHandler creates a new handler.
func NewVerifyEmailHandler(repo user.Repository) *VerifyEmailHandler {
	return &VerifyEmailHandler{repo: repo}
}

// Handle executes the command.
func (h *VerifyEmailHandler) Handle(ctx context.Context, cmd VerifyEmailCommand) (*VerifyEmailResult, error) {
	verification, err := h.repo.GetVerificationByToken(ctx, cmd.Token)
	if err != nil {
		return nil, err
	}

	if verification.IsUsed() {
		return nil, user.ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return nil, user.ErrTokenExpired
	}
	if verification.Purpose() != user.PurposeRegistration {
		return nil, user.ErrInvalidToken
	}

	// Get user
	u, err := h.repo.GetByID(ctx, verification.UserID())
	if err != nil {
		return nil, err
	}

	// Mark token as used
	if err := h.repo.MarkVerificationUsed(ctx, verification.ID()); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Verify email
	if err := u.VerifyEmail(); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &VerifyEmailResult{User: u}, nil
}

// ResendVerificationCommand represents a command to resend verification email.
type ResendVerificationCommand struct {
	Email string
}

// ResendVerificationResult is the result of resending verification.
type ResendVerificationResult struct {
	Token string // Empty if user not found or already verified
}

// ResendVerificationHandler handles ResendVerificationCommand.
type ResendVerificationHandler struct {
	repo user.Repository
}

// NewResendVerificationHandler creates a new handler.
func NewResendVerificationHandler(repo user.Repository) *ResendVerificationHandler {
	return &ResendVerificationHandler{repo: repo}
}

// Handle executes the command.
func (h *ResendVerificationHandler) Handle(ctx context.Context, cmd ResendVerificationCommand) (*ResendVerificationResult, error) {
	u, err := h.repo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		// Don't reveal if email exists
		return &ResendVerificationResult{}, nil
	}

	if u.EmailVerified() {
		return &ResendVerificationResult{}, nil
	}

	// Invalidate existing tokens
	_ = h.repo.InvalidateUserVerifications(ctx, u.ID(), user.PurposeRegistration)

	// Generate new token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create verification record
	verification := user.NewEmailVerification(
		u.ID(),
		token,
		user.PurposeRegistration,
		time.Now().Add(24*time.Hour),
	)

	if err := h.repo.CreateVerification(ctx, verification); err != nil {
		return nil, fmt.Errorf("create verification: %w", err)
	}

	return &ResendVerificationResult{Token: token}, nil
}

// RequestPasswordResetCommand represents a command to request password reset.
type RequestPasswordResetCommand struct {
	Email string
}

// RequestPasswordResetResult is the result of requesting password reset.
type RequestPasswordResetResult struct {
	Token string // Empty if user not found
}

// RequestPasswordResetHandler handles RequestPasswordResetCommand.
type RequestPasswordResetHandler struct {
	repo user.Repository
}

// NewRequestPasswordResetHandler creates a new handler.
func NewRequestPasswordResetHandler(repo user.Repository) *RequestPasswordResetHandler {
	return &RequestPasswordResetHandler{repo: repo}
}

// Handle executes the command.
func (h *RequestPasswordResetHandler) Handle(ctx context.Context, cmd RequestPasswordResetCommand) (*RequestPasswordResetResult, error) {
	u, err := h.repo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		// Don't reveal if email exists
		return &RequestPasswordResetResult{}, nil
	}

	// Only allow for email users
	if !u.IsEmailUser() {
		return &RequestPasswordResetResult{}, nil
	}

	// Invalidate existing reset tokens
	_ = h.repo.InvalidateUserVerifications(ctx, u.ID(), user.PurposePasswordReset)

	// Generate new token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create verification record (1 hour expiry for password reset)
	verification := user.NewEmailVerification(
		u.ID(),
		token,
		user.PurposePasswordReset,
		time.Now().Add(1*time.Hour),
	)

	if err := h.repo.CreateVerification(ctx, verification); err != nil {
		return nil, fmt.Errorf("create verification: %w", err)
	}

	return &RequestPasswordResetResult{Token: token}, nil
}

// ResetPasswordCommand represents a command to reset password.
type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

// ResetPasswordResult is the result of resetting password.
type ResetPasswordResult struct {
	User *user.User
}

// ResetPasswordHandler handles ResetPasswordCommand.
type ResetPasswordHandler struct {
	repo user.Repository
}

// NewResetPasswordHandler creates a new handler.
func NewResetPasswordHandler(repo user.Repository) *ResetPasswordHandler {
	return &ResetPasswordHandler{repo: repo}
}

// Handle executes the command.
func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd ResetPasswordCommand) (*ResetPasswordResult, error) {
	if len(cmd.NewPassword) < 8 {
		return nil, user.ErrPasswordTooShort
	}

	verification, err := h.repo.GetVerificationByToken(ctx, cmd.Token)
	if err != nil {
		return nil, err
	}

	if verification.IsUsed() {
		return nil, user.ErrTokenAlreadyUsed
	}
	if verification.IsExpired() {
		return nil, user.ErrTokenExpired
	}
	if verification.Purpose() != user.PurposePasswordReset {
		return nil, user.ErrInvalidToken
	}

	// Get user
	u, err := h.repo.GetByID(ctx, verification.UserID())
	if err != nil {
		return nil, err
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Mark token as used
	if err := h.repo.MarkVerificationUsed(ctx, verification.ID()); err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}

	// Update password
	u.SetPasswordHash(string(hash))
	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &ResetPasswordResult{User: u}, nil
}

// ChangePasswordCommand represents a command to change password.
type ChangePasswordCommand struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ChangePasswordResult is the result of changing password.
type ChangePasswordResult struct {
	User *user.User
}

// ChangePasswordHandler handles ChangePasswordCommand.
type ChangePasswordHandler struct {
	repo user.Repository
}

// NewChangePasswordHandler creates a new handler.
func NewChangePasswordHandler(repo user.Repository) *ChangePasswordHandler {
	return &ChangePasswordHandler{repo: repo}
}

// Handle executes the command.
func (h *ChangePasswordHandler) Handle(ctx context.Context, cmd ChangePasswordCommand) (*ChangePasswordResult, error) {
	if len(cmd.NewPassword) < 8 {
		return nil, user.ErrPasswordTooShort
	}

	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	if u.PasswordHash() == nil {
		return nil, user.ErrPasswordRequired
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash()), []byte(cmd.CurrentPassword)); err != nil {
		return nil, user.ErrIncorrectPassword
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Update password
	u.SetPasswordHash(string(hash))
	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &ChangePasswordResult{User: u}, nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
