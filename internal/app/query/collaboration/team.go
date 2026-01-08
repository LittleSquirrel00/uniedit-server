package collaboration

import (
	"context"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
)

// GetTeamQuery represents a query to get a team.
type GetTeamQuery struct {
	OwnerID     uuid.UUID
	Slug        string
	RequesterID *uuid.UUID
}

// GetTeamResult is the result of getting a team.
type GetTeamResult struct {
	Team   *TeamDTO
	MyRole *string
}

// GetTeamHandler handles team retrieval.
type GetTeamHandler struct {
	teamRepo   collaboration.TeamRepository
	memberRepo collaboration.MemberRepository
}

// NewGetTeamHandler creates a new handler.
func NewGetTeamHandler(
	teamRepo collaboration.TeamRepository,
	memberRepo collaboration.MemberRepository,
) *GetTeamHandler {
	return &GetTeamHandler{
		teamRepo:   teamRepo,
		memberRepo: memberRepo,
	}
}

// Handle executes the query.
func (h *GetTeamHandler) Handle(ctx context.Context, q GetTeamQuery) (*GetTeamResult, error) {
	team, err := h.teamRepo.GetByOwnerAndSlug(ctx, q.OwnerID, q.Slug)
	if err != nil {
		return nil, err
	}

	// Check access
	var myRole *collaboration.Role
	if q.RequesterID != nil {
		member, err := h.memberRepo.Get(ctx, team.ID(), *q.RequesterID)
		if err == nil {
			r := member.Role()
			myRole = &r
		} else if err != collaboration.ErrMemberNotFound {
			return nil, err
		}
	}

	// If private and not a member, return not found
	if team.Visibility() == collaboration.VisibilityPrivate && myRole == nil {
		return nil, collaboration.ErrTeamNotFound
	}

	memberCount, _ := h.memberRepo.Count(ctx, team.ID())

	dto := teamToDTO(team, memberCount, myRole)
	var roleStr *string
	if myRole != nil {
		r := myRole.String()
		roleStr = &r
	}

	return &GetTeamResult{
		Team:   dto,
		MyRole: roleStr,
	}, nil
}

// ListMyTeamsQuery represents a query to list user's teams.
type ListMyTeamsQuery struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

// ListMyTeamsHandler handles team listing.
type ListMyTeamsHandler struct {
	teamRepo collaboration.TeamRepository
}

// NewListMyTeamsHandler creates a new handler.
func NewListMyTeamsHandler(teamRepo collaboration.TeamRepository) *ListMyTeamsHandler {
	return &ListMyTeamsHandler{teamRepo: teamRepo}
}

// Handle executes the query.
func (h *ListMyTeamsHandler) Handle(ctx context.Context, q ListMyTeamsQuery) ([]*TeamDTO, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	teams, err := h.teamRepo.ListByUser(ctx, q.UserID, limit, q.Offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*TeamDTO, len(teams))
	for i, team := range teams {
		dtos[i] = teamToDTO(team, 0, nil)
	}

	return dtos, nil
}

// ListMembersQuery represents a query to list team members.
type ListMembersQuery struct {
	TeamID      uuid.UUID
	RequesterID uuid.UUID
}

// ListMembersHandler handles member listing.
type ListMembersHandler struct {
	memberRepo collaboration.MemberRepository
}

// NewListMembersHandler creates a new handler.
func NewListMembersHandler(memberRepo collaboration.MemberRepository) *ListMembersHandler {
	return &ListMembersHandler{memberRepo: memberRepo}
}

// Handle executes the query.
func (h *ListMembersHandler) Handle(ctx context.Context, q ListMembersQuery) ([]*MemberDTO, error) {
	// Check if requester is a member
	_, err := h.memberRepo.Get(ctx, q.TeamID, q.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	members, err := h.memberRepo.ListWithUsers(ctx, q.TeamID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*MemberDTO, len(members))
	for i, m := range members {
		dtos[i] = &MemberDTO{
			UserID:   m.Member().UserID().String(),
			Email:    m.Email(),
			Name:     m.Name(),
			Role:     m.Member().Role().String(),
			JoinedAt: m.Member().JoinedAt().Unix(),
		}
	}

	return dtos, nil
}

// ListInvitationsQuery represents a query to list invitations.
type ListInvitationsQuery struct {
	TeamID      uuid.UUID
	RequesterID uuid.UUID
	Status      *string
	Limit       int
	Offset      int
}

// ListInvitationsHandler handles invitation listing.
type ListInvitationsHandler struct {
	memberRepo     collaboration.MemberRepository
	invitationRepo collaboration.InvitationRepository
}

// NewListInvitationsHandler creates a new handler.
func NewListInvitationsHandler(
	memberRepo collaboration.MemberRepository,
	invitationRepo collaboration.InvitationRepository,
) *ListInvitationsHandler {
	return &ListInvitationsHandler{
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
	}
}

// Handle executes the query.
func (h *ListInvitationsHandler) Handle(ctx context.Context, q ListInvitationsQuery) ([]*InvitationDTO, error) {
	// Check permission
	member, err := h.memberRepo.Get(ctx, q.TeamID, q.RequesterID)
	if err != nil {
		if err == collaboration.ErrMemberNotFound {
			return nil, collaboration.ErrInsufficientPermission
		}
		return nil, err
	}

	if !member.Role().HasPermission(collaboration.PermInvite) {
		return nil, collaboration.ErrInsufficientPermission
	}

	var status *collaboration.InvitationStatus
	if q.Status != nil {
		s := collaboration.InvitationStatus(*q.Status)
		status = &s
	}

	invitations, err := h.invitationRepo.ListByTeam(ctx, q.TeamID, status, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*InvitationDTO, len(invitations))
	for i, inv := range invitations {
		dtos[i] = invitationToDTO(inv, "", "")
	}

	return dtos, nil
}

// ListMyInvitationsQuery represents a query to list user's invitations.
type ListMyInvitationsQuery struct {
	Email  string
	Limit  int
	Offset int
}

// ListMyInvitationsHandler handles user invitation listing.
type ListMyInvitationsHandler struct {
	invitationRepo collaboration.InvitationRepository
}

// NewListMyInvitationsHandler creates a new handler.
func NewListMyInvitationsHandler(invitationRepo collaboration.InvitationRepository) *ListMyInvitationsHandler {
	return &ListMyInvitationsHandler{invitationRepo: invitationRepo}
}

// Handle executes the query.
func (h *ListMyInvitationsHandler) Handle(ctx context.Context, q ListMyInvitationsQuery) ([]*InvitationDTO, error) {
	status := collaboration.InvitationStatusPending
	invitations, err := h.invitationRepo.ListByEmail(ctx, q.Email, &status, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*InvitationDTO, len(invitations))
	for i, inv := range invitations {
		dtos[i] = invitationToDTO(inv, "", "")
	}

	return dtos, nil
}

// Helper
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
