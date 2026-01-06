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
)

// Service provides collaboration business logic.
type Service struct {
	repo     Repository
	userRepo UserRepository
	logger   *zap.Logger
}

// NewService creates a new collaboration service.
func NewService(repo Repository, userRepo UserRepository, logger *zap.Logger) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// ========== Team Operations ==========

// CreateTeam creates a new team.
func (s *Service) CreateTeam(ctx context.Context, ownerID uuid.UUID, req *CreateTeamRequest) (*Team, error) {
	// Generate slug from name
	slug := generateSlug(req.Name)

	// Check if slug already exists for this owner
	existing, err := s.repo.GetTeamByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil && err != ErrTeamNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugAlreadyExists
	}

	// Set default visibility
	visibility := req.Visibility
	if visibility == "" {
		visibility = VisibilityPrivate
	}

	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txRepo := s.repo.WithTx(tx)

	// Create team
	team := &Team{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Visibility:  visibility,
		MemberLimit: 5, // Default limit
		Status:      TeamStatusActive,
	}

	if err := txRepo.CreateTeam(ctx, team); err != nil {
		return nil, err
	}

	// Add owner as member
	member := &TeamMember{
		TeamID:   team.ID,
		UserID:   ownerID,
		Role:     RoleOwner,
		JoinedAt: time.Now(),
	}

	if err := txRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	s.logger.Info("team created",
		zap.String("team_id", team.ID.String()),
		zap.String("owner_id", ownerID.String()),
		zap.String("name", team.Name),
	)

	return team, nil
}

// GetTeam retrieves a team by owner and slug.
func (s *Service) GetTeam(ctx context.Context, ownerID uuid.UUID, slug string, requesterID *uuid.UUID) (*Team, *Role, error) {
	team, err := s.repo.GetTeamByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, nil, err
	}

	// Check access
	var myRole *Role
	if requesterID != nil {
		member, err := s.repo.GetMember(ctx, team.ID, *requesterID)
		if err == nil {
			myRole = &member.Role
		} else if err != ErrMemberNotFound {
			return nil, nil, err
		}
	}

	// If private and not a member, return not found
	if team.Visibility == VisibilityPrivate && myRole == nil {
		return nil, nil, ErrTeamNotFound
	}

	return team, myRole, nil
}

// GetTeamByID retrieves a team by ID.
func (s *Service) GetTeamByID(ctx context.Context, teamID uuid.UUID) (*Team, error) {
	return s.repo.GetTeamByID(ctx, teamID)
}

// ListMyTeams lists teams the user belongs to.
func (s *Service) ListMyTeams(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Team, error) {
	return s.repo.ListTeamsByUser(ctx, userID, limit, offset)
}

// UpdateTeam updates a team.
func (s *Service) UpdateTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID, req *UpdateTeamRequest) (*Team, error) {
	// Check permission
	member, err := s.repo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role.HasPermission(PermUpdateTeam) {
		return nil, ErrInsufficientPermission
	}

	// Get team
	team, err := s.repo.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		team.Name = *req.Name
		team.Slug = generateSlug(*req.Name)
	}
	if req.Description != nil {
		team.Description = *req.Description
	}
	if req.Visibility != nil {
		team.Visibility = *req.Visibility
	}

	if err := s.repo.UpdateTeam(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// DeleteTeam soft-deletes a team.
func (s *Service) DeleteTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) error {
	// Check permission (only owner can delete)
	member, err := s.repo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !member.Role.HasPermission(PermDeleteTeam) {
		return ErrOnlyOwnerCanDelete
	}

	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := s.repo.WithTx(tx)

	// Cancel pending invitations
	if err := txRepo.CancelPendingInvitations(ctx, teamID); err != nil {
		return err
	}

	// Soft delete team
	if err := txRepo.DeleteTeam(ctx, teamID); err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	s.logger.Info("team deleted",
		zap.String("team_id", teamID.String()),
		zap.String("deleted_by", requesterID.String()),
	)

	return nil
}

// ========== Member Operations ==========

// ListMembers lists team members with user details.
func (s *Service) ListMembers(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) ([]MemberWithUser, error) {
	// Check if requester is a member
	_, err := s.repo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	return s.repo.ListMembersWithUsers(ctx, teamID)
}

// UpdateMemberRole updates a member's role.
func (s *Service) UpdateMemberRole(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID, newRole Role) error {
	// Check requester permission
	requester, err := s.repo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !requester.Role.HasPermission(PermUpdateRole) {
		return ErrInsufficientPermission
	}

	// Get target member
	target, err := s.repo.GetMember(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}

	// Cannot change owner's role
	if target.Role == RoleOwner {
		return ErrCannotChangeOwner
	}

	// Only owner can promote to admin
	if newRole == RoleOwner {
		return ErrOnlyOwnerCanTransfer
	}

	// Check if requester can assign this role
	if !requester.Role.CanAssign(newRole) {
		return ErrInsufficientPermission
	}

	return s.repo.UpdateMemberRole(ctx, teamID, targetUserID, newRole)
}

// RemoveMember removes a member from a team.
func (s *Service) RemoveMember(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID) error {
	// Get target member first
	target, err := s.repo.GetMember(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}

	// Cannot remove owner
	if target.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	// Check requester permission (unless removing self)
	if targetUserID != requesterID {
		requester, err := s.repo.GetMember(ctx, teamID, requesterID)
		if err != nil {
			if err == ErrMemberNotFound {
				return ErrInsufficientPermission
			}
			return err
		}

		if !requester.Role.HasPermission(PermRemoveMember) {
			return ErrInsufficientPermission
		}
	}

	return s.repo.RemoveMember(ctx, teamID, targetUserID)
}

// LeaveTeam allows a member to leave a team.
func (s *Service) LeaveTeam(ctx context.Context, teamID, userID uuid.UUID) error {
	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Owner cannot leave without transferring ownership
	if member.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	return s.repo.RemoveMember(ctx, teamID, userID)
}

// GetMemberCount returns the member count for a team.
func (s *Service) GetMemberCount(ctx context.Context, teamID uuid.UUID) (int, error) {
	return s.repo.CountMembers(ctx, teamID)
}

// ========== Invitation Operations ==========

// SendInvitation sends an invitation to join a team.
func (s *Service) SendInvitation(ctx context.Context, teamID, inviterID uuid.UUID, req *InviteRequest) (*TeamInvitation, error) {
	// Check inviter permission
	inviter, err := s.repo.GetMember(ctx, teamID, inviterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !inviter.Role.HasPermission(PermInvite) {
		return nil, ErrInsufficientPermission
	}

	// Validate invite role
	if !IsValidInviteRole(req.Role) {
		return nil, ErrInvalidRole
	}

	// Check if requester can assign this role
	if !inviter.Role.CanAssign(req.Role) {
		return nil, ErrInsufficientPermission
	}

	// Get team to check limits
	team, err := s.repo.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Check member limit
	memberCount, err := s.repo.CountMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if memberCount >= team.MemberLimit {
		return nil, ErrMemberLimitExceeded
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user is already a member
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if user != nil {
		_, err := s.repo.GetMember(ctx, teamID, user.ID)
		if err == nil {
			return nil, ErrAlreadyMember
		} else if err != ErrMemberNotFound {
			return nil, err
		}
	}

	// Check for existing pending invitation
	existing, err := s.repo.GetPendingInvitationByEmail(ctx, teamID, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrInvitationAlreadyPending
	}

	// Generate secure token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	// Create invitation
	invitation := &TeamInvitation{
		ID:           uuid.New(),
		TeamID:       teamID,
		InviterID:    inviterID,
		InviteeEmail: email,
		Role:         req.Role,
		Token:        token,
		Status:       InvitationStatusPending,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if user != nil {
		invitation.InviteeID = &user.ID
	}

	if err := s.repo.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}

	// Load relations for response
	invitation.Team = team
	if inviterUser, _ := s.userRepo.GetUserByID(ctx, inviterID); inviterUser != nil {
		invitation.Inviter = inviterUser
	}

	s.logger.Info("invitation sent",
		zap.String("invitation_id", invitation.ID.String()),
		zap.String("team_id", teamID.String()),
		zap.String("invitee_email", email),
	)

	return invitation, nil
}

// ListTeamInvitations lists invitations for a team.
func (s *Service) ListTeamInvitations(ctx context.Context, teamID, requesterID uuid.UUID, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error) {
	// Check permission
	member, err := s.repo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return nil, ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role.HasPermission(PermInvite) {
		return nil, ErrInsufficientPermission
	}

	return s.repo.ListInvitationsByTeam(ctx, teamID, status, limit, offset)
}

// ListMyInvitations lists pending invitations for a user.
func (s *Service) ListMyInvitations(ctx context.Context, email string, limit, offset int) ([]*TeamInvitation, error) {
	status := InvitationStatusPending
	return s.repo.ListInvitationsByEmail(ctx, email, &status, limit, offset)
}

// AcceptInvitation accepts an invitation.
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID, userEmail string) (*Team, error) {
	invitation, err := s.repo.GetInvitationByToken(ctx, token)
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
		_ = s.repo.UpdateInvitationStatus(ctx, invitation.ID, InvitationStatusExpired)
		return nil, ErrInvitationExpired
	}

	// Get team to check limits
	team, err := s.repo.GetTeamByID(ctx, invitation.TeamID)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txRepo := s.repo.WithTx(tx)

	// Check member limit again within transaction
	memberCount, err := txRepo.CountMembers(ctx, team.ID)
	if err != nil {
		return nil, err
	}

	if memberCount >= team.MemberLimit {
		return nil, ErrMemberLimitExceeded
	}

	// Add member
	member := &TeamMember{
		TeamID:   team.ID,
		UserID:   userID,
		Role:     invitation.Role,
		JoinedAt: time.Now(),
	}

	if err := txRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	// Update invitation status
	if err := txRepo.UpdateInvitationStatus(ctx, invitation.ID, InvitationStatusAccepted); err != nil {
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	s.logger.Info("invitation accepted",
		zap.String("invitation_id", invitation.ID.String()),
		zap.String("team_id", team.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return team, nil
}

// RejectInvitation rejects an invitation.
func (s *Service) RejectInvitation(ctx context.Context, token string, userEmail string) error {
	invitation, err := s.repo.GetInvitationByToken(ctx, token)
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

	return s.repo.UpdateInvitationStatus(ctx, invitation.ID, InvitationStatusRejected)
}

// RevokeInvitation revokes an invitation.
func (s *Service) RevokeInvitation(ctx context.Context, invitationID, requesterID uuid.UUID) error {
	invitation, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}

	// Check permission
	member, err := s.repo.GetMember(ctx, invitation.TeamID, requesterID)
	if err != nil {
		if err == ErrMemberNotFound {
			return ErrInsufficientPermission
		}
		return err
	}

	if !member.Role.HasPermission(PermInvite) {
		return ErrInsufficientPermission
	}

	// Check status
	if !invitation.IsPending() {
		return ErrCannotRevokeProcessed
	}

	return s.repo.UpdateInvitationStatus(ctx, invitationID, InvitationStatusRevoked)
}

// ========== Helper Functions ==========

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
