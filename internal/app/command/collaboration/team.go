package collaboration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
)

// CreateTeamCommand represents a command to create a team.
type CreateTeamCommand struct {
	OwnerID     uuid.UUID
	Name        string
	Description string
	Visibility  string
}

// CreateTeamResult is the result of creating a team.
type CreateTeamResult struct {
	Team *TeamDTO
}

// TeamDTO represents a team (shared with query package).
type TeamDTO struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility"`
	MemberLimit int    `json:"member_limit"`
	MemberCount int    `json:"member_count,omitempty"`
	MyRole      string `json:"my_role,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

// InvitationDTO represents an invitation (shared with query package).
type InvitationDTO struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	TeamName     string `json:"team_name,omitempty"`
	InviterID    string `json:"inviter_id"`
	InviterName  string `json:"inviter_name,omitempty"`
	InviteeEmail string `json:"invitee_email"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	ExpiresAt    int64  `json:"expires_at"`
	CreatedAt    int64  `json:"created_at"`
}

// CreateTeamHandler handles team creation.
type CreateTeamHandler struct {
	uow      collaboration.UnitOfWork
	teamRepo collaboration.TeamRepository
}

// NewCreateTeamHandler creates a new handler.
func NewCreateTeamHandler(
	uow collaboration.UnitOfWork,
	teamRepo collaboration.TeamRepository,
) *CreateTeamHandler {
	return &CreateTeamHandler{
		uow:      uow,
		teamRepo: teamRepo,
	}
}

// Handle executes the command.
func (h *CreateTeamHandler) Handle(ctx context.Context, cmd CreateTeamCommand) (*CreateTeamResult, error) {
	// Create team
	team := collaboration.NewTeam(cmd.OwnerID, cmd.Name)
	if cmd.Description != "" {
		team.SetDescription(cmd.Description)
	}
	if cmd.Visibility != "" {
		team.SetVisibility(collaboration.Visibility(cmd.Visibility))
	}

	// Check if slug already exists
	_, err := h.teamRepo.GetByOwnerAndSlug(ctx, cmd.OwnerID, team.Slug())
	if err == nil {
		return nil, collaboration.ErrSlugAlreadyExists
	}
	if err != collaboration.ErrTeamNotFound {
		return nil, err
	}

	// Start transaction
	tx, err := h.uow.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create team
	if err := tx.TeamRepository().Create(ctx, team); err != nil {
		return nil, err
	}

	// Add owner as member
	member := collaboration.NewTeamMember(team.ID(), cmd.OwnerID, collaboration.RoleOwner)
	if err := tx.MemberRepository().Add(ctx, member); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &CreateTeamResult{
		Team: teamToDTO(team, 1, nil),
	}, nil
}

// UpdateTeamCommand represents a command to update a team.
type UpdateTeamCommand struct {
	TeamID      uuid.UUID
	RequesterID uuid.UUID
	Name        *string
	Description *string
	Visibility  *string
}

// UpdateTeamResult is the result of updating a team.
type UpdateTeamResult struct {
	Team *TeamDTO
}

// UpdateTeamHandler handles team updates.
type UpdateTeamHandler struct {
	teamRepo   collaboration.TeamRepository
	memberRepo collaboration.MemberRepository
}

// NewUpdateTeamHandler creates a new handler.
func NewUpdateTeamHandler(
	teamRepo collaboration.TeamRepository,
	memberRepo collaboration.MemberRepository,
) *UpdateTeamHandler {
	return &UpdateTeamHandler{
		teamRepo:   teamRepo,
		memberRepo: memberRepo,
	}
}

// Handle executes the command.
func (h *UpdateTeamHandler) Handle(ctx context.Context, cmd UpdateTeamCommand) (*UpdateTeamResult, error) {
	// Check permission
	member, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role().HasPermission(collaboration.PermUpdateTeam) {
		return nil, collaboration.ErrInsufficientPermission
	}

	// Get team
	team, err := h.teamRepo.GetByID(ctx, cmd.TeamID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if cmd.Name != nil {
		team.SetName(*cmd.Name)
	}
	if cmd.Description != nil {
		team.SetDescription(*cmd.Description)
	}
	if cmd.Visibility != nil {
		team.SetVisibility(collaboration.Visibility(*cmd.Visibility))
	}

	if err := h.teamRepo.Update(ctx, team); err != nil {
		return nil, err
	}

	return &UpdateTeamResult{
		Team: teamToDTO(team, 0, nil),
	}, nil
}

// DeleteTeamCommand represents a command to delete a team.
type DeleteTeamCommand struct {
	TeamID      uuid.UUID
	RequesterID uuid.UUID
}

// DeleteTeamResult is the result of deleting a team.
type DeleteTeamResult struct{}

// DeleteTeamHandler handles team deletion.
type DeleteTeamHandler struct {
	uow        collaboration.UnitOfWork
	memberRepo collaboration.MemberRepository
}

// NewDeleteTeamHandler creates a new handler.
func NewDeleteTeamHandler(
	uow collaboration.UnitOfWork,
	memberRepo collaboration.MemberRepository,
) *DeleteTeamHandler {
	return &DeleteTeamHandler{
		uow:        uow,
		memberRepo: memberRepo,
	}
}

// Handle executes the command.
func (h *DeleteTeamHandler) Handle(ctx context.Context, cmd DeleteTeamCommand) (*DeleteTeamResult, error) {
	// Check permission
	member, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role().HasPermission(collaboration.PermDeleteTeam) {
		return nil, collaboration.ErrOnlyOwnerCanDelete
	}

	// Start transaction
	tx, err := h.uow.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Cancel pending invitations
	if err := tx.InvitationRepository().CancelPending(ctx, cmd.TeamID); err != nil {
		return nil, err
	}

	// Delete team
	if err := tx.TeamRepository().Delete(ctx, cmd.TeamID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &DeleteTeamResult{}, nil
}

// SendInvitationCommand represents a command to send an invitation.
type SendInvitationCommand struct {
	TeamID    uuid.UUID
	InviterID uuid.UUID
	Email     string
	Role      string
}

// SendInvitationResult is the result of sending an invitation.
type SendInvitationResult struct {
	Invitation *InvitationDTO
}

// SendInvitationHandler handles invitation sending.
type SendInvitationHandler struct {
	teamRepo       collaboration.TeamRepository
	memberRepo     collaboration.MemberRepository
	invitationRepo collaboration.InvitationRepository
	userLookup     collaboration.UserLookup
}

// NewSendInvitationHandler creates a new handler.
func NewSendInvitationHandler(
	teamRepo collaboration.TeamRepository,
	memberRepo collaboration.MemberRepository,
	invitationRepo collaboration.InvitationRepository,
	userLookup collaboration.UserLookup,
) *SendInvitationHandler {
	return &SendInvitationHandler{
		teamRepo:       teamRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		userLookup:     userLookup,
	}
}

// Handle executes the command.
func (h *SendInvitationHandler) Handle(ctx context.Context, cmd SendInvitationCommand) (*SendInvitationResult, error) {
	// Check inviter permission
	inviter, err := h.memberRepo.Get(ctx, cmd.TeamID, cmd.InviterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !inviter.Role().HasPermission(collaboration.PermInvite) {
		return nil, collaboration.ErrInsufficientPermission
	}

	// Validate role
	role := collaboration.Role(cmd.Role)
	if !collaboration.IsValidInviteRole(role) {
		return nil, collaboration.ErrInvalidRole
	}

	if !inviter.Role().CanAssign(role) {
		return nil, collaboration.ErrInsufficientPermission
	}

	// Get team to check limits
	team, err := h.teamRepo.GetByID(ctx, cmd.TeamID)
	if err != nil {
		return nil, err
	}

	// Check member limit
	memberCount, err := h.memberRepo.Count(ctx, cmd.TeamID)
	if err != nil {
		return nil, err
	}
	if memberCount >= team.MemberLimit() {
		return nil, collaboration.ErrMemberLimitExceeded
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(cmd.Email))

	// Check if user is already a member
	user, _ := h.userLookup.GetByEmail(ctx, email)
	if user != nil {
		_, err := h.memberRepo.Get(ctx, cmd.TeamID, user.ID)
		if err == nil {
			return nil, collaboration.ErrAlreadyMember
		}
		if err != collaboration.ErrMemberNotFound {
			return nil, err
		}
	}

	// Check for existing pending invitation
	existing, err := h.invitationRepo.GetPendingByEmail(ctx, cmd.TeamID, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, collaboration.ErrInvitationAlreadyPending
	}

	// Generate token
	token, err := collaboration.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create invitation
	invitation := collaboration.NewTeamInvitation(
		cmd.TeamID,
		cmd.InviterID,
		email,
		role,
		token,
		time.Now().Add(7*24*time.Hour),
	)

	if user != nil {
		invitation.SetInviteeID(user.ID)
	}

	if err := h.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, err
	}

	return &SendInvitationResult{
		Invitation: invitationToDTO(invitation, team.Name(), ""),
	}, nil
}

// AcceptInvitationCommand represents a command to accept an invitation.
type AcceptInvitationCommand struct {
	Token     string
	UserID    uuid.UUID
	UserEmail string
}

// AcceptInvitationResult is the result of accepting an invitation.
type AcceptInvitationResult struct {
	Team *TeamDTO
}

// AcceptInvitationHandler handles invitation acceptance.
type AcceptInvitationHandler struct {
	uow            collaboration.UnitOfWork
	invitationRepo collaboration.InvitationRepository
	teamRepo       collaboration.TeamRepository
}

// NewAcceptInvitationHandler creates a new handler.
func NewAcceptInvitationHandler(
	uow collaboration.UnitOfWork,
	invitationRepo collaboration.InvitationRepository,
	teamRepo collaboration.TeamRepository,
) *AcceptInvitationHandler {
	return &AcceptInvitationHandler{
		uow:            uow,
		invitationRepo: invitationRepo,
		teamRepo:       teamRepo,
	}
}

// Handle executes the command.
func (h *AcceptInvitationHandler) Handle(ctx context.Context, cmd AcceptInvitationCommand) (*AcceptInvitationResult, error) {
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

	if invitation.IsExpired() {
		_ = h.invitationRepo.UpdateStatus(ctx, invitation.ID(), collaboration.InvitationStatusExpired)
		return nil, collaboration.ErrInvitationExpired
	}

	// Get team
	team, err := h.teamRepo.GetByID(ctx, invitation.TeamID())
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := h.uow.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check member limit
	memberCount, err := tx.MemberRepository().Count(ctx, team.ID())
	if err != nil {
		return nil, err
	}
	if memberCount >= team.MemberLimit() {
		return nil, collaboration.ErrMemberLimitExceeded
	}

	// Add member
	member := collaboration.NewTeamMember(team.ID(), cmd.UserID, invitation.Role())
	if err := tx.MemberRepository().Add(ctx, member); err != nil {
		return nil, err
	}

	// Update invitation status
	if err := tx.InvitationRepository().UpdateStatus(ctx, invitation.ID(), collaboration.InvitationStatusAccepted); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &AcceptInvitationResult{
		Team: teamToDTO(team, 0, nil),
	}, nil
}

// Helper functions

func teamToDTO(t *collaboration.Team, memberCount int, myRole *collaboration.Role) *TeamDTO {
	dto := &TeamDTO{
		ID:          t.ID().String(),
		OwnerID:     t.OwnerID().String(),
		Name:        t.Name(),
		Slug:        t.Slug(),
		Description: t.Description(),
		Visibility:  string(t.Visibility()),
		MemberLimit: t.MemberLimit(),
		MemberCount: memberCount,
		CreatedAt:   t.CreatedAt().Unix(),
	}
	if myRole != nil {
		dto.MyRole = myRole.String()
	}
	return dto
}

func invitationToDTO(inv *collaboration.TeamInvitation, teamName, inviterName string) *InvitationDTO {
	return &InvitationDTO{
		ID:           inv.ID().String(),
		TeamID:       inv.TeamID().String(),
		TeamName:     teamName,
		InviterID:    inv.InviterID().String(),
		InviterName:  inviterName,
		InviteeEmail: inv.InviteeEmail(),
		Role:         inv.Role().String(),
		Status:       inv.Status().String(),
		ExpiresAt:    inv.ExpiresAt().Unix(),
		CreatedAt:    inv.CreatedAt().Unix(),
	}
}
