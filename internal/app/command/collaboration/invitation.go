package collaboration

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
)

// RejectInvitationCommand represents a command to reject an invitation.
type RejectInvitationCommand struct {
	Token     string
	UserEmail string
}

// RejectInvitationResult is the result of rejecting an invitation.
type RejectInvitationResult struct{}

// RejectInvitationHandler handles invitation rejection.
type RejectInvitationHandler struct {
	invitationRepo collaboration.InvitationRepository
}

// NewRejectInvitationHandler creates a new handler.
func NewRejectInvitationHandler(invitationRepo collaboration.InvitationRepository) *RejectInvitationHandler {
	return &RejectInvitationHandler{invitationRepo: invitationRepo}
}

// Handle executes the command.
func (h *RejectInvitationHandler) Handle(ctx context.Context, cmd RejectInvitationCommand) (*RejectInvitationResult, error) {
	invitation, err := h.invitationRepo.GetByToken(ctx, cmd.Token)
	if err != nil {
		return nil, err
	}

	// Verify invitation is for this user
	if !invitation.IsForEmail(strings.ToLower(cmd.UserEmail)) {
		return nil, collaboration.ErrInvitationNotForYou
	}

	if !invitation.IsPending() {
		return nil, collaboration.ErrInvitationAlreadyProcessed
	}

	if err := h.invitationRepo.UpdateStatus(ctx, invitation.ID(), collaboration.InvitationStatusRejected); err != nil {
		return nil, err
	}

	return &RejectInvitationResult{}, nil
}

// RevokeInvitationCommand represents a command to revoke an invitation.
type RevokeInvitationCommand struct {
	InvitationID uuid.UUID
	RequesterID  uuid.UUID
}

// RevokeInvitationResult is the result of revoking an invitation.
type RevokeInvitationResult struct{}

// RevokeInvitationHandler handles invitation revocation.
type RevokeInvitationHandler struct {
	invitationRepo collaboration.InvitationRepository
	memberRepo     collaboration.MemberRepository
}

// NewRevokeInvitationHandler creates a new handler.
func NewRevokeInvitationHandler(
	invitationRepo collaboration.InvitationRepository,
	memberRepo collaboration.MemberRepository,
) *RevokeInvitationHandler {
	return &RevokeInvitationHandler{
		invitationRepo: invitationRepo,
		memberRepo:     memberRepo,
	}
}

// Handle executes the command.
func (h *RevokeInvitationHandler) Handle(ctx context.Context, cmd RevokeInvitationCommand) (*RevokeInvitationResult, error) {
	invitation, err := h.invitationRepo.GetByID(ctx, cmd.InvitationID)
	if err != nil {
		return nil, err
	}

	// Check if requester has permission
	member, err := h.memberRepo.Get(ctx, invitation.TeamID(), cmd.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role().HasPermission(collaboration.PermInvite) {
		return nil, collaboration.ErrInsufficientPermission
	}

	// Only pending invitations can be revoked
	if !invitation.IsPending() {
		return nil, collaboration.ErrCannotRevokeProcessed
	}

	if err := h.invitationRepo.UpdateStatus(ctx, invitation.ID(), collaboration.InvitationStatusRevoked); err != nil {
		return nil, err
	}

	return &RevokeInvitationResult{}, nil
}
