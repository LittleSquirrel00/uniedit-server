package collaboration

import (
	"context"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
)

// UpdateMemberRoleCommand represents a command to update a member's role.
type UpdateMemberRoleCommand struct {
	TeamID       uuid.UUID
	TargetUserID uuid.UUID
	RequesterID  uuid.UUID
	NewRole      string
}

// UpdateMemberRoleResult is the result of updating a member's role.
type UpdateMemberRoleResult struct{}

// UpdateMemberRoleHandler handles member role updates.
type UpdateMemberRoleHandler struct {
	memberRepo collaboration.MemberRepository
}

// NewUpdateMemberRoleHandler creates a new handler.
func NewUpdateMemberRoleHandler(memberRepo collaboration.MemberRepository) *UpdateMemberRoleHandler {
	return &UpdateMemberRoleHandler{memberRepo: memberRepo}
}

// Handle executes the command.
func (h *UpdateMemberRoleHandler) Handle(ctx context.Context, cmd UpdateMemberRoleCommand) (*UpdateMemberRoleResult, error) {
	// Check requester permission
	requester, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !requester.Role().HasPermission(collaboration.PermUpdateRole) {
		return nil, collaboration.ErrInsufficientPermission
	}

	// Get target member
	target, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.TargetUserID)
	if err != nil {
		return nil, err
	}

	// Cannot change owner's role
	if target.Role() == collaboration.RoleOwner {
		return nil, collaboration.ErrCannotChangeOwner
	}

	newRole := collaboration.Role(cmd.NewRole)
	if newRole == collaboration.RoleOwner {
		return nil, collaboration.ErrOnlyOwnerCanTransfer
	}

	// Check if requester can assign this role
	if !requester.Role().CanAssign(newRole) {
		return nil, collaboration.ErrInsufficientPermission
	}

	if err := h.memberRepo.UpdateRole(ctx, cmd.TeamID, cmd.TargetUserID, newRole); err != nil {
		return nil, err
	}

	return &UpdateMemberRoleResult{}, nil
}

// RemoveMemberCommand represents a command to remove a member.
type RemoveMemberCommand struct {
	TeamID       uuid.UUID
	TargetUserID uuid.UUID
	RequesterID  uuid.UUID
}

// RemoveMemberResult is the result of removing a member.
type RemoveMemberResult struct{}

// RemoveMemberHandler handles member removal.
type RemoveMemberHandler struct {
	memberRepo collaboration.MemberRepository
}

// NewRemoveMemberHandler creates a new handler.
func NewRemoveMemberHandler(memberRepo collaboration.MemberRepository) *RemoveMemberHandler {
	return &RemoveMemberHandler{memberRepo: memberRepo}
}

// Handle executes the command.
func (h *RemoveMemberHandler) Handle(ctx context.Context, cmd RemoveMemberCommand) (*RemoveMemberResult, error) {
	// Get target member
	target, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.TargetUserID)
	if err != nil {
		return nil, err
	}

	// Cannot remove owner
	if target.Role() == collaboration.RoleOwner {
		return nil, collaboration.ErrCannotRemoveOwner
	}

	// Check permission (unless removing self)
	if cmd.TargetUserID != cmd.RequesterID {
		requester, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.RequesterID)
		if err != nil {
			if err == collaboration.ErrMemberNotFound {
				return nil, collaboration.ErrInsufficientPermission
			}
			return nil, err
		}

		if !requester.Role().HasPermission(collaboration.PermRemoveMember) {
			return nil, collaboration.ErrInsufficientPermission
		}
	}

	if err := h.memberRepo.Remove(ctx, cmd.TeamID, cmd.TargetUserID); err != nil {
		return nil, err
	}

	return &RemoveMemberResult{}, nil
}

// LeaveTeamCommand represents a command to leave a team.
type LeaveTeamCommand struct {
	TeamID uuid.UUID
	UserID uuid.UUID
}

// LeaveTeamResult is the result of leaving a team.
type LeaveTeamResult struct{}

// LeaveTeamHandler handles leaving a team.
type LeaveTeamHandler struct {
	memberRepo collaboration.MemberRepository
}

// NewLeaveTeamHandler creates a new handler.
func NewLeaveTeamHandler(memberRepo collaboration.MemberRepository) *LeaveTeamHandler {
	return &LeaveTeamHandler{memberRepo: memberRepo}
}

// Handle executes the command.
func (h *LeaveTeamHandler) Handle(ctx context.Context, cmd LeaveTeamCommand) (*LeaveTeamResult, error) {
	member, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.UserID)
	if err != nil {
		return nil, err
	}

	// Owner cannot leave
	if member.Role() == collaboration.RoleOwner {
		return nil, collaboration.ErrCannotRemoveOwner
	}

	if err := h.memberRepo.Remove(ctx, cmd.TeamID, cmd.UserID); err != nil {
		return nil, err
	}

	return &LeaveTeamResult{}, nil
}
