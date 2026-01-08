package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
)

// UpdateProfileCommand represents a command to update user profile.
type UpdateProfileCommand struct {
	UserID    uuid.UUID
	Name      *string
	AvatarURL *string
}

// UpdateProfileResult is the result of updating profile.
type UpdateProfileResult struct {
	User *user.User
}

// UpdateProfileHandler handles UpdateProfileCommand.
type UpdateProfileHandler struct {
	repo user.Repository
}

// NewUpdateProfileHandler creates a new handler.
func NewUpdateProfileHandler(repo user.Repository) *UpdateProfileHandler {
	return &UpdateProfileHandler{repo: repo}
}

// Handle executes the command.
func (h *UpdateProfileHandler) Handle(ctx context.Context, cmd UpdateProfileCommand) (*UpdateProfileResult, error) {
	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != nil {
		u.SetName(*cmd.Name)
	}
	if cmd.AvatarURL != nil {
		u.SetAvatarURL(*cmd.AvatarURL)
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &UpdateProfileResult{User: u}, nil
}

// DeleteAccountCommand represents a command to delete user account.
type DeleteAccountCommand struct {
	UserID   uuid.UUID
	Password string // Required for email users
}

// DeleteAccountResult is the result of deleting account.
type DeleteAccountResult struct{}

// DeleteAccountHandler handles DeleteAccountCommand.
type DeleteAccountHandler struct {
	repo user.Repository
}

// NewDeleteAccountHandler creates a new handler.
func NewDeleteAccountHandler(repo user.Repository) *DeleteAccountHandler {
	return &DeleteAccountHandler{repo: repo}
}

// Handle executes the command.
func (h *DeleteAccountHandler) Handle(ctx context.Context, cmd DeleteAccountCommand) (*DeleteAccountResult, error) {
	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	// For email users, verify password
	if u.IsEmailUser() {
		if cmd.Password == "" {
			return nil, user.ErrPasswordRequired
		}
		// Note: Password verification should be done at a higher level
		// with bcrypt.CompareHashAndPassword
	}

	// Soft delete
	if err := h.repo.SoftDelete(ctx, cmd.UserID); err != nil {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	return &DeleteAccountResult{}, nil
}

// SuspendUserCommand represents a command to suspend a user (admin action).
type SuspendUserCommand struct {
	UserID uuid.UUID
	Reason string
}

// SuspendUserResult is the result of suspending user.
type SuspendUserResult struct {
	User *user.User
}

// SuspendUserHandler handles SuspendUserCommand.
type SuspendUserHandler struct {
	repo user.Repository
}

// NewSuspendUserHandler creates a new handler.
func NewSuspendUserHandler(repo user.Repository) *SuspendUserHandler {
	return &SuspendUserHandler{repo: repo}
}

// Handle executes the command.
func (h *SuspendUserHandler) Handle(ctx context.Context, cmd SuspendUserCommand) (*SuspendUserResult, error) {
	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	if err := u.Suspend(cmd.Reason); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &SuspendUserResult{User: u}, nil
}

// ReactivateUserCommand represents a command to reactivate a suspended user.
type ReactivateUserCommand struct {
	UserID uuid.UUID
}

// ReactivateUserResult is the result of reactivating user.
type ReactivateUserResult struct {
	User *user.User
}

// ReactivateUserHandler handles ReactivateUserCommand.
type ReactivateUserHandler struct {
	repo user.Repository
}

// NewReactivateUserHandler creates a new handler.
func NewReactivateUserHandler(repo user.Repository) *ReactivateUserHandler {
	return &ReactivateUserHandler{repo: repo}
}

// Handle executes the command.
func (h *ReactivateUserHandler) Handle(ctx context.Context, cmd ReactivateUserCommand) (*ReactivateUserResult, error) {
	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	if err := u.Reactivate(); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &ReactivateUserResult{User: u}, nil
}

// SetAdminStatusCommand represents a command to set user admin status.
type SetAdminStatusCommand struct {
	UserID  uuid.UUID
	IsAdmin bool
}

// SetAdminStatusResult is the result of setting admin status.
type SetAdminStatusResult struct {
	User *user.User
}

// SetAdminStatusHandler handles SetAdminStatusCommand.
type SetAdminStatusHandler struct {
	repo user.Repository
}

// NewSetAdminStatusHandler creates a new handler.
func NewSetAdminStatusHandler(repo user.Repository) *SetAdminStatusHandler {
	return &SetAdminStatusHandler{repo: repo}
}

// Handle executes the command.
func (h *SetAdminStatusHandler) Handle(ctx context.Context, cmd SetAdminStatusCommand) (*SetAdminStatusResult, error) {
	u, err := h.repo.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	u.SetAdminStatus(cmd.IsAdmin)

	if err := h.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &SetAdminStatusResult{User: u}, nil
}
