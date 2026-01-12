package collaboration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	commonv1 "github.com/uniedit/server/api/pb/common"
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
func (d *Domain) CreateTeam(ctx context.Context, ownerID uuid.UUID, in *collabv1.CreateTeamRequest) (*collabv1.Team, error) {
	if in == nil || strings.TrimSpace(in.GetName()) == "" {
		return nil, ErrInvalidRequest
	}

	// Generate slug from name
	slug := generateSlug(in.GetName())

	// Check if slug already exists for this owner
	existing, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil && err != ErrTeamNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugAlreadyExists
	}

	// Set default visibility
	visibility := model.TeamVisibilityPrivate
	if v := in.GetVisibility(); v != "" {
		visibility = model.TeamVisibility(v)
	}

	var team *model.Team

	err = d.txPort.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Create team
		team = &model.Team{
			ID:          uuid.New(),
			OwnerID:     ownerID,
			Name:        in.GetName(),
			Slug:        slug,
			Description: in.GetDescription(),
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
	return toTeamPB(team, 1, &role), nil
}

func (d *Domain) GetTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*collabv1.Team, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	ownerID, err := resolveOwnerID(requesterID, in.GetOwnerId())
	if err != nil {
		return nil, err
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, in.GetSlug())
	if err != nil {
		return nil, err
	}

	// Check access
	var myRole *model.TeamRole
	member, err := d.memberDB.Find(ctx, team.ID, requesterID)
	if err == nil {
		myRole = &member.Role
	} else if err != ErrMemberNotFound {
		return nil, err
	}

	// If private and not a member, return not found
	if team.Visibility == model.TeamVisibilityPrivate && myRole == nil {
		return nil, ErrTeamNotFound
	}

	memberCount, _ := d.memberDB.Count(ctx, team.ID)
	return toTeamPB(team, memberCount, myRole), nil
}

// GetTeamByID retrieves a team by ID.
func (d *Domain) GetTeamByID(ctx context.Context, teamID uuid.UUID) (*model.Team, error) {
	return d.teamDB.FindByID(ctx, teamID)
}

// ListMyTeams lists teams the user belongs to.
func (d *Domain) ListMyTeams(ctx context.Context, userID uuid.UUID, in *collabv1.ListMyTeamsRequest) (*collabv1.ListMyTeamsResponse, error) {
	limit := 20
	offset := 0
	if in != nil {
		if v := int(in.GetLimit()); v > 0 {
			limit = v
		}
		if v := int(in.GetOffset()); v > 0 {
			offset = v
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	teams, err := d.teamDB.FindByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*collabv1.Team, 0, len(teams))
	for _, team := range teams {
		memberCount, _ := d.memberDB.Count(ctx, team.ID)
		member, _ := d.memberDB.Find(ctx, team.ID, userID)
		var myRole *model.TeamRole
		if member != nil {
			myRole = &member.Role
		}
		outputs = append(outputs, toTeamPB(team, memberCount, myRole))
	}

	return &collabv1.ListMyTeamsResponse{Teams: outputs}, nil
}

// UpdateTeam updates a team.
func (d *Domain) UpdateTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.UpdateTeamRequest) (*collabv1.Team, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, requesterID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

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

	// Update fields
	if v := in.GetName(); v != nil && strings.TrimSpace(v.GetValue()) != "" {
		team.Name = v.GetValue()
		team.Slug = generateSlug(v.GetValue())
	}
	if v := in.GetDescription(); v != nil {
		team.Description = v.GetValue()
	}
	if v := in.GetVisibility(); v != nil && v.GetValue() != "" {
		team.Visibility = model.TeamVisibility(v.GetValue())
	}
	team.UpdatedAt = time.Now()

	if err := d.teamDB.Update(ctx, team); err != nil {
		return nil, err
	}

	memberCount, _ := d.memberDB.Count(ctx, team.ID)
	return toTeamPB(team, memberCount, &member.Role), nil
}

// DeleteTeam soft-deletes a team.
func (d *Domain) DeleteTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	ownerID, err := resolveOwnerID(requesterID, in.GetOwnerId())
	if err != nil {
		return nil, err
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

	// Check permission (only owner can delete)
	member, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(member.Role, PermDeleteTeam) {
		return nil, ErrOnlyOwnerCanDelete
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
		return nil, err
	}

	d.logger.Info("team deleted",
		zap.String("team_id", teamID.String()),
		zap.String("deleted_by", requesterID.String()),
	)

	return &commonv1.MessageResponse{Message: "Team deleted successfully"}, nil
}

// ========== Member Operations ==========

// ListMembers lists team members with user details.
func (d *Domain) ListMembers(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*collabv1.ListMembersResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	ownerID, err := resolveOwnerID(requesterID, in.GetOwnerId())
	if err != nil {
		return nil, err
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

	// Check if requester is a member
	_, err = d.memberDB.Find(ctx, teamID, requesterID)
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

	out := make([]*collabv1.Member, 0, len(members))
	for _, m := range members {
		out = append(out, toMemberPB(m))
	}
	return &collabv1.ListMembersResponse{Members: out}, nil
}

// UpdateMemberRole updates a member's role.
func (d *Domain) UpdateMemberRole(ctx context.Context, requesterID uuid.UUID, in *collabv1.UpdateMemberRoleRequest) (*commonv1.MessageResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" || in.GetUserId() == "" || in.GetRole() == "" {
		return nil, ErrInvalidRequest
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, requesterID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

	targetUserID, err := uuid.Parse(in.GetUserId())
	if err != nil {
		return nil, ErrInvalidRequest
	}
	newRole := model.TeamRole(in.GetRole())

	// Check requester permission
	requester, err := d.memberDB.Find(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(requester.Role, PermUpdateRole) {
		return nil, ErrInsufficientPermission
	}

	// Get target member
	target, err := d.memberDB.Find(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}

	// Cannot change owner's role
	if target.Role == model.TeamRoleOwner {
		return nil, ErrCannotChangeOwner
	}

	// Only owner can promote to admin
	if newRole == model.TeamRoleOwner {
		return nil, ErrOnlyOwnerCanTransfer
	}

	// Check if requester can assign this role
	if !RoleCanAssign(requester.Role, newRole) {
		return nil, ErrInsufficientPermission
	}

	if err := d.memberDB.UpdateRole(ctx, teamID, targetUserID, newRole); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "Member role updated"}, nil
}

// RemoveMember removes a member from a team.
func (d *Domain) RemoveMember(ctx context.Context, requesterID uuid.UUID, in *collabv1.RemoveMemberRequest) (*commonv1.MessageResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" || in.GetUserId() == "" {
		return nil, ErrInvalidRequest
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, requesterID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

	targetUserID, err := uuid.Parse(in.GetUserId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	// Get target member first
	target, err := d.memberDB.Find(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}

	// Cannot remove owner
	if target.Role == model.TeamRoleOwner {
		return nil, ErrCannotRemoveOwner
	}

	// Check requester permission (unless removing self)
	if targetUserID != requesterID {
		requester, err := d.memberDB.Find(ctx, teamID, requesterID)
		if err != nil {
			if err == ErrMemberNotFound {
				return nil, ErrInsufficientPermission
			}
			return nil, err
		}

		if !RoleHasPermission(requester.Role, PermRemoveMember) {
			return nil, ErrInsufficientPermission
		}
	}

	if err := d.memberDB.Remove(ctx, teamID, targetUserID); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "Member removed"}, nil
}

// LeaveTeam allows a member to leave a team.
func (d *Domain) LeaveTeam(ctx context.Context, userID uuid.UUID, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	ownerID, err := resolveOwnerID(userID, in.GetOwnerId())
	if err != nil {
		return nil, err
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

	member, err := d.memberDB.Find(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}

	// Owner cannot leave without transferring ownership
	if member.Role == model.TeamRoleOwner {
		return nil, ErrCannotRemoveOwner
	}

	if err := d.memberDB.Remove(ctx, teamID, userID); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "Left team"}, nil
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
func (d *Domain) SendInvitation(ctx context.Context, inviterID uuid.UUID, in *collabv1.SendInvitationRequest) (*collabv1.Invitation, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" || strings.TrimSpace(in.GetEmail()) == "" || in.GetRole() == "" {
		return nil, ErrInvalidRequest
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, inviterID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

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
	role := model.TeamRole(in.GetRole())
	if !IsValidInviteRole(role) {
		return nil, ErrInvalidRole
	}

	// Check if requester can assign this role
	if !RoleCanAssign(inviter.Role, role) {
		return nil, ErrInsufficientPermission
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
	email := strings.ToLower(strings.TrimSpace(in.GetEmail()))

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
		Role:         role,
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

	return toInvitationPB(invitation, true, d.cfg.BaseURL), nil
}

// ListTeamInvitations lists invitations for a team.
func (d *Domain) ListTeamInvitations(ctx context.Context, requesterID uuid.UUID, in *collabv1.ListTeamInvitationsRequest) (*collabv1.ListTeamInvitationsResponse, error) {
	if in == nil || strings.TrimSpace(in.GetSlug()) == "" {
		return nil, ErrInvalidRequest
	}

	ownerID, err := resolveOwnerID(requesterID, in.GetOwnerId())
	if err != nil {
		return nil, err
	}

	team, err := d.teamDB.FindByOwnerAndSlug(ctx, ownerID, in.GetSlug())
	if err != nil {
		return nil, err
	}
	teamID := team.ID

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

	var status *model.InvitationStatus
	if s := in.GetStatus(); s != "" {
		v := model.InvitationStatus(s)
		status = &v
	}

	limit := int(in.GetLimit())
	offset := int(in.GetOffset())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	invitations, err := d.invitationDB.FindByTeam(ctx, teamID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*collabv1.Invitation, 0, len(invitations))
	for _, inv := range invitations {
		outputs = append(outputs, toInvitationPB(inv, false, d.cfg.BaseURL))
	}

	return &collabv1.ListTeamInvitationsResponse{Invitations: outputs}, nil
}

// ListMyInvitations lists pending invitations for a user.
func (d *Domain) ListMyInvitations(ctx context.Context, email string, in *collabv1.ListMyInvitationsRequest) (*collabv1.ListMyInvitationsResponse, error) {
	limit := 20
	offset := 0
	if in != nil {
		if v := int(in.GetLimit()); v > 0 {
			limit = v
		}
		if v := int(in.GetOffset()); v > 0 {
			offset = v
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	status := model.InvitationStatusPending
	invitations, err := d.invitationDB.FindByEmail(ctx, email, &status, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]*collabv1.Invitation, 0, len(invitations))
	for _, inv := range invitations {
		outputs = append(outputs, toInvitationPB(inv, true, d.cfg.BaseURL))
	}

	return &collabv1.ListMyInvitationsResponse{Invitations: outputs}, nil
}

// AcceptInvitation accepts an invitation.
func (d *Domain) AcceptInvitation(ctx context.Context, userID uuid.UUID, userEmail string, in *collabv1.InvitationTokenRequest) (*collabv1.AcceptInvitationResponse, error) {
	if in == nil || strings.TrimSpace(in.GetToken()) == "" {
		return nil, ErrInvalidRequest
	}
	token := in.GetToken()

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
	return &collabv1.AcceptInvitationResponse{
		Message: "Invitation accepted",
		Team:    toTeamPB(team, memberCount, &role),
	}, nil
}

// RejectInvitation rejects an invitation.
func (d *Domain) RejectInvitation(ctx context.Context, userEmail string, in *collabv1.InvitationTokenRequest) (*commonv1.MessageResponse, error) {
	if in == nil || strings.TrimSpace(in.GetToken()) == "" {
		return nil, ErrInvalidRequest
	}
	token := in.GetToken()

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

	if err := d.invitationDB.UpdateStatus(ctx, invitation.ID, model.InvitationStatusRejected); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "Invitation rejected"}, nil
}

// RevokeInvitation revokes an invitation.
func (d *Domain) RevokeInvitation(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrInvalidRequest
	}
	invitationID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	invitation, err := d.invitationDB.FindByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}

	// Check permission
	member, err := d.memberDB.Find(ctx, invitation.TeamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !RoleHasPermission(member.Role, PermInvite) {
		return nil, ErrInsufficientPermission
	}

	// Check status
	if !invitation.IsPending() {
		return nil, ErrCannotRevokeProcessed
	}

	if err := d.invitationDB.UpdateStatus(ctx, invitationID, model.InvitationStatusRevoked); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "Invitation revoked"}, nil
}

func resolveOwnerID(defaultID uuid.UUID, ownerIDStr string) (uuid.UUID, error) {
	if ownerIDStr == "" {
		return defaultID, nil
	}
	id, err := uuid.Parse(ownerIDStr)
	if err != nil {
		return uuid.Nil, ErrInvalidRequest
	}
	return id, nil
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
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length+10], nil
}

// Compile-time interface check
var _ inbound.CollaborationDomain = (*Domain)(nil)
