package collaboration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
)

// Domain implements the collaboration domain logic.
type Domain struct {
	teamDB       outbound.TeamDatabasePort
	memberDB     outbound.TeamMemberDatabasePort
	invitationDB outbound.TeamInvitationDatabasePort
	userLookup   outbound.CollaborationUserLookupPort
	txPort       outbound.CollaborationTransactionPort
	cfg          *Config
	logger       *zap.Logger
}

// NewDomain creates a new collaboration domain.
func NewDomain(
	teamDB outbound.TeamDatabasePort,
	memberDB outbound.TeamMemberDatabasePort,
	invitationDB outbound.TeamInvitationDatabasePort,
	userLookup outbound.CollaborationUserLookupPort,
	txPort outbound.CollaborationTransactionPort,
	cfg *Config,
	logger *zap.Logger,
) *Domain {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	_ = cfg.Validate()

	return &Domain{
		teamDB:       teamDB,
		memberDB:     memberDB,
		invitationDB: invitationDB,
		userLookup:   userLookup,
		txPort:       txPort,
		cfg:          cfg,
		logger:       logger,
	}
}

// ========== Team Operations ==========

// CreateTeam creates a new team.
func (d *Domain) CreateTeam(ctx context.Context, ownerID uuid.UUID, input *inbound.CreateTeamInput) (*inbound.TeamOutput, error) {
	// Generate slug from name
	slug := generateSlug(input.Name)

	// Check if slug already exists for this owner
	existing, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil && err != ErrTeamNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugAlreadyExists
	}

	// Set default visibility
	visibility := input.Visibility
	if visibility == "" {
		visibility = model.TeamVisibilityPrivate
	}

	var team *model.Team

	err = d.txPort.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Create team
		team = &model.Team{
			ID:          uuid.New(),
			OwnerID:     ownerID,
			Name:        input.Name,
			Slug:        slug,
			Description: input.Description,
			Visibility:  visibility,
			MemberLimit: d.cfg.DefaultMemberLimit,
			Status:      model.TeamStatusActive,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := d.teamDB.Create(txCtx, team); err != nil {
			return err
		}

		// Add owner as member
		member := &model.TeamMember{
			TeamID:    team.ID,
			UserID:    ownerID,
			Role:      model.TeamRoleOwner,
			JoinedAt:  time.Now(),
			UpdatedAt: time.Now(),
		}

		return d.memberDB.Add(txCtx, member)
	})

	if err != nil {
		return nil, err
	}

	d.logger.Info("team created",
		zap.String("team_id", team.ID.String()),
		zap.String("owner_id", ownerID.String()),
		zap.String("name", team.Name),
	)

	role := model.TeamRoleOwner
	return d.teamToOutput(team, 1, &role), nil
}

// GetTeam retrieves a team by owner and slug.
func (d *Domain) GetTeam(ctx context.Context, ownerID uuid.UUID, slug string, requesterID *uuid.UUID) (*inbound.TeamOutput, error) {
	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}

	// Check access
	var myRole *model.TeamRole
	if requesterID != nil {
		member, err := d.memberDB.Find(ctx, team.ID, *requesterID)
		if err == nil {
			myRole = &member.Role
		} else if err != ErrMemberNotFound {
			return nil, err
		}
	}

	// If private and not a member, return not found
	if team.Visibility == model.TeamVisibilityPrivate && myRole == nil {
		return nil, ErrTeamNotFound
	}

	memberCount, _ := d.memberDB.Count(ctx, team.ID)
	return d.teamToOutput(team, memberCount, myRole), nil
}

// GetTeamByID retrieves a team by ID.
func (d *Domain) GetTeamByID(ctx context.Context, teamID uuid.UUID) (*model.Team, error) {
	return d.teamDB.FindByID(ctx, teamID)
}

// ListMyTeams lists teams the user belongs to.
func (d *Domain) ListMyTeams(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*inbound.TeamOutput, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	teams, err := d.teamDB.FindByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*inbound.TeamOutput, len(teams))
	for i, team := range teams {
		memberCount, _ := d.memberDB.Count(ctx, team.ID)
		member, _ := d.memberDB.Find(ctx, team.ID, userID)
		var myRole *model.TeamRole
		if member != nil {
			myRole = &member.Role
		}
		outputs[i] = d.teamToOutput(team, memberCount, myRole)
	}

	return outputs, nil
}

// UpdateTeam updates a team.
func (d *Domain) UpdateTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID, input *inbound.UpdateTeamInput) (*inbound.TeamOutput, error) {
	// Check permission
	member, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(member.Role, PermUpdateTeam) {
		return nil, ErrInsufficientPermission
	}

	// Get team
	team, err := d.teamDB.FindByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if input.Name != nil {
		team.Name = *input.Name
		team.Slug = generateSlug(*input.Name)
	}
	if input.Description != nil {
		team.Description = *input.Description
	}
	if input.Visibility != nil {
		team.Visibility = *input.Visibility
	}
	team.UpdatedAt = time.Now()

	if err := d.teamDB.Update(ctx, team); err != nil {
		return nil, err
	}

	memberCount, _ := d.memberDB.Count(ctx, team.ID)
	return d.teamToOutput(team, memberCount, &member.Role), nil
}

// DeleteTeam soft-deletes a team.
func (d *Domain) DeleteTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) error {
	// Check permission (only owner can delete)
	member, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !RoleHasPermission(member.Role, PermDeleteTeam) {
		return ErrOnlyOwnerCanDelete
	}

	err = d.txPort.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Cancel pending invitations
		if err := d.invitationDB.CancelPendingByTeam(txCtx, teamID); err != nil {
			return err
		}

		// Soft delete team
		return d.teamDB.Delete(txCtx, teamID)
	})

	if err != nil {
		return err
	}

	d.logger.Info("team deleted",
		zap.String("team_id", teamID.String()),
		zap.String("deleted_by", requesterID.String()),
	)

	return nil
}

// ========== Member Operations ==========

// ListMembers lists team members with user details.
func (d *Domain) ListMembers(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) ([]*inbound.MemberOutput, error) {
	// Check if requester is a member
	_, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	members, err := d.memberDB.FindByTeamWithUsers(ctx, teamID)
	if err != nil {
		return nil, err
	}

	outputs := make([]*inbound.MemberOutput, len(members))
	for i, m := range members {
		outputs[i] = &inbound.MemberOutput{
			UserID:   m.UserID,
			Email:    m.Email,
			Name:     m.Name,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		}
	}

	return outputs, nil
}

// UpdateMemberRole updates a member's role.
func (d *Domain) UpdateMemberRole(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID, newRole model.TeamRole) error {
	// Check requester permission
	requester, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !RoleHasPermission(requester.Role, PermUpdateRole) {
		return ErrInsufficientPermission
	}

	// Get target member
	target, err := d.memberDB.Find(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}

	// Cannot change owner's role
	if target.Role == model.TeamRoleOwner {
		return ErrCannotChangeOwner
	}

	// Only owner can promote to admin
	if newRole == model.TeamRoleOwner {
		return ErrOnlyOwnerCanTransfer
	}

	// Check if requester can assign this role
	if !RoleCanAssign(requester.Role, newRole) {
		return ErrInsufficientPermission
	}

	return d.memberDB.UpdateRole(ctx, teamID, targetUserID, newRole)
}

// RemoveMember removes a member from a team.
func (d *Domain) RemoveMember(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID) error {
	// Get target member first
	target, err := d.memberDB.Find(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}

	// Cannot remove owner
	if target.Role == model.TeamRoleOwner {
		return ErrCannotRemoveOwner
	}

	// Check requester permission (unless removing self)
	if targetUserID != requesterID {
		requester, err := d.memberDB.Find(ctx, teamID, requesterID)
		if err != nil {
			if err == ErrMemberNotFound {
				return ErrInsufficientPermission
			}
			return err
		}

		if !RoleHasPermission(requester.Role, PermRemoveMember) {
			return ErrInsufficientPermission
		}
	}

	return d.memberDB.Remove(ctx, teamID, targetUserID)
}

// LeaveTeam allows a member to leave a team.
func (d *Domain) LeaveTeam(ctx context.Context, teamID, userID uuid.UUID) error {
	member, err := d.memberDB.Find(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Owner cannot leave without transferring ownership
	if member.Role == model.TeamRoleOwner {
		return ErrCannotRemoveOwner
	}

	return d.memberDB.Remove(ctx, teamID, userID)
}

// GetMemberCount returns the member count for a team.
func (d *Domain) GetMemberCount(ctx context.Context, teamID uuid.UUID) (int, error) {
	return d.memberDB.Count(ctx, teamID)
}

// GetMember returns a member by team and user ID.
func (d *Domain) GetMember(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	return d.memberDB.Find(ctx, teamID, userID)
}

// ========== Invitation Operations ==========

// SendInvitation sends an invitation to join a team.
func (d *Domain) SendInvitation(ctx context.Context, teamID, inviterID uuid.UUID, input *inbound.InviteInput) (*inbound.InvitationOutput, error) {
	// Check inviter permission
	inviter, err := d.memberDB.Find(ctx, teamID, inviterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(inviter.Role, PermInvite) {
		return nil, ErrInsufficientPermission
	}

	// Validate invite role
	if !IsValidInviteRole(input.Role) {
		return nil, ErrInvalidRole
	}

	// Check if requester can assign this role
	if !RoleCanAssign(inviter.Role, input.Role) {
		return nil, ErrInsufficientPermission
	}

	// Get team to check limits
	team, err := d.teamDB.FindByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Check member limit
	memberCount, err := d.memberDB.Count(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if memberCount >= team.MemberLimit {
		return nil, ErrMemberLimitExceeded
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(input.Email))

	// Check if user is already a member
	user, err := d.userLookup.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if user != nil {
		_, err := d.memberDB.Find(ctx, teamID, user.ID)
		if err == nil {
			return nil, ErrAlreadyMember
		} else if err != ErrMemberNotFound {
			return nil, err
		}
	}

	// Check for existing pending invitation
	existing, err := d.invitationDB.FindPendingByEmail(ctx, teamID, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrInvitationAlreadyPending
	}

	// Generate secure token
	token, err := generateSecureToken(d.cfg.InvitationTokenLength)
	if err != nil {
		return nil, err
	}

	// Create invitation
	invitation := &model.TeamInvitation{
		ID:           uuid.New(),
		TeamID:       teamID,
		InviterID:    inviterID,
		InviteeEmail: email,
		Role:         input.Role,
		Token:        token,
		Status:       model.InvitationStatusPending,
		ExpiresAt:    time.Now().Add(d.cfg.InvitationExpiry),
		CreatedAt:    time.Now(),
	}

	if user != nil {
		invitation.InviteeID = &user.ID
	}

	if err := d.invitationDB.Create(ctx, invitation); err != nil {
		return nil, err
	}

	// Load relations for response
	invitation.Team = team
	if inviterUser, _ := d.userLookup.FindByID(ctx, inviterID); inviterUser != nil {
		invitation.Inviter = inviterUser
	}

	d.logger.Info("invitation sent",
		zap.String("invitation_id", invitation.ID.String()),
		zap.String("team_id", teamID.String()),
		zap.String("invitee_email", email),
	)

	return d.invitationToOutput(invitation, true), nil
}

// ListTeamInvitations lists invitations for a team.
func (d *Domain) ListTeamInvitations(ctx context.Context, teamID, requesterID uuid.UUID, status *model.InvitationStatus, limit, offset int) ([]*inbound.InvitationOutput, error) {
	// Check permission
	member, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(member.Role, PermInvite) {
		return nil, ErrInsufficientPermission
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	invitations, err := d.invitationDB.FindByTeam(ctx, teamID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*inbound.InvitationOutput, len(invitations))
	for i, inv := range invitations {
		outputs[i] = d.invitationToOutput(inv, false)
	}

	return outputs, nil
}

// ListMyInvitations lists pending invitations for a user.
func (d *Domain) ListMyInvitations(ctx context.Context, email string, limit, offset int) ([]*inbound.InvitationOutput, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	status := model.InvitationStatusPending
	invitations, err := d.invitationDB.FindByEmail(ctx, email, &status, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*inbound.InvitationOutput, len(invitations))
	for i, inv := range invitations {
		outputs[i] = d.invitationToOutput(inv, true)
	}

	return outputs, nil
}

// AcceptInvitation accepts an invitation.
func (d *Domain) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID, userEmail string) (*inbound.TeamOutput, error) {
	invitation, err := d.invitationDB.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Verify invitation is for this user
	if strings.ToLower(invitation.InviteeEmail) != strings.ToLower(userEmail) {
		return nil, ErrInvitationNotForYou
	}

	// Check status
	if !invitation.IsPending() {
		return nil, ErrInvitationAlreadyProcessed
	}

	// Check expiration
	if invitation.IsExpired() {
		// Update status to expired
		_ = d.invitationDB.UpdateStatus(ctx, invitation.ID, model.InvitationStatusExpired)
		return nil, ErrInvitationExpired
	}

	// Get team to check limits
	team, err := d.teamDB.FindByID(ctx, invitation.TeamID)
	if err != nil {
		return nil, err
	}

	var role model.TeamRole
	err = d.txPort.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Check member limit again within transaction
		memberCount, err := d.memberDB.Count(txCtx, team.ID)
		if err != nil {
			return err
		}

		if memberCount >= team.MemberLimit {
			return ErrMemberLimitExceeded
		}

		// Add member
		member := &model.TeamMember{
			TeamID:    team.ID,
			UserID:    userID,
			Role:      invitation.Role,
			JoinedAt:  time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := d.memberDB.Add(txCtx, member); err != nil {
			return err
		}

		role = invitation.Role

		// Update invitation status
		return d.invitationDB.UpdateStatus(txCtx, invitation.ID, model.InvitationStatusAccepted)
	})

	if err != nil {
		return nil, err
	}

	d.logger.Info("invitation accepted",
		zap.String("invitation_id", invitation.ID.String()),
		zap.String("team_id", team.ID.String()),
		zap.String("user_id", userID.String()),
	)

	memberCount, _ := d.memberDB.Count(ctx, team.ID)
	return d.teamToOutput(team, memberCount, &role), nil
}

// RejectInvitation rejects an invitation.
func (d *Domain) RejectInvitation(ctx context.Context, token string, userEmail string) error {
	invitation, err := d.invitationDB.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	// Verify invitation is for this user
	if strings.ToLower(invitation.InviteeEmail) != strings.ToLower(userEmail) {
		return ErrInvitationNotForYou
	}

	// Check status
	if !invitation.IsPending() {
		return ErrInvitationAlreadyProcessed
	}

	return d.invitationDB.UpdateStatus(ctx, invitation.ID, model.InvitationStatusRejected)
}

// RevokeInvitation revokes an invitation.
func (d *Domain) RevokeInvitation(ctx context.Context, invitationID, requesterID uuid.UUID) error {
	invitation, err := d.invitationDB.FindByID(ctx, invitationID)
	if err != nil {
		return err
	}

	// Check permission
	member, err := d.memberDB.Find(ctx, invitation.TeamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !RoleHasPermission(member.Role, PermInvite) {
		return ErrInsufficientPermission
	}

	// Check status
	if !invitation.IsPending() {
		return ErrCannotRevokeProcessed
	}

	return d.invitationDB.UpdateStatus(ctx, invitationID, model.InvitationStatusRevoked)
}

// ========== Helper Methods ==========

func (d *Domain) teamToOutput(team *model.Team, memberCount int, myRole *model.TeamRole) *inbound.TeamOutput {
	return &inbound.TeamOutput{
		ID:          team.ID,
		OwnerID:     team.OwnerID,
		Name:        team.Name,
		Slug:        team.Slug,
		Description: team.Description,
		Visibility:  team.Visibility,
		MemberCount: memberCount,
		MemberLimit: team.MemberLimit,
		CreatedAt:   team.CreatedAt,
		UpdatedAt:   team.UpdatedAt,
		MyRole:      myRole,
	}
}

func (d *Domain) invitationToOutput(inv *model.TeamInvitation, includeToken bool) *inbound.InvitationOutput {
	output := &inbound.InvitationOutput{
		ID:           inv.ID,
		TeamID:       inv.TeamID,
		InviterID:    inv.InviterID,
		InviteeEmail: inv.InviteeEmail,
		Role:         inv.Role,
		Status:       inv.Status,
		ExpiresAt:    inv.ExpiresAt,
		CreatedAt:    inv.CreatedAt,
		AcceptedAt:   inv.AcceptedAt,
	}

	if inv.Team != nil {
		output.TeamName = inv.Team.Name
	}
	if inv.Inviter != nil {
		output.InviterName = inv.Inviter.Name
	}

	if includeToken {
		output.Token = inv.Token
		if d.cfg.BaseURL != "" {
			output.AcceptURL = d.cfg.BaseURL + "/invitations/" + inv.Token + "/accept"
		}
	}

	return output
}

// ========== Utility Functions ==========

// generateSlug generates a URL-friendly slug from a name.
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length+10], nil
}

// Compile-time interface check
var _ inbound.CollaborationDomain = (*Domain)(nil)
