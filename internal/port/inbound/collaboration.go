package inbound

import (
	"context"

	"github.com/google/uuid"
	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	commonv1 "github.com/uniedit/server/api/pb/common"
)

// CollaborationDomain defines the collaboration domain service interface.
type CollaborationDomain interface {
	// Team operations
	CreateTeam(ctx context.Context, ownerID uuid.UUID, in *collabv1.CreateTeamRequest) (*collabv1.Team, error)
	ListMyTeams(ctx context.Context, userID uuid.UUID, in *collabv1.ListMyTeamsRequest) (*collabv1.ListMyTeamsResponse, error)
	GetTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*collabv1.Team, error)
	UpdateTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.UpdateTeamRequest) (*collabv1.Team, error)
	DeleteTeam(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error)

	// Member operations
	ListMembers(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetTeamRequest) (*collabv1.ListMembersResponse, error)
	UpdateMemberRole(ctx context.Context, requesterID uuid.UUID, in *collabv1.UpdateMemberRoleRequest) (*commonv1.MessageResponse, error)
	RemoveMember(ctx context.Context, requesterID uuid.UUID, in *collabv1.RemoveMemberRequest) (*commonv1.MessageResponse, error)
	LeaveTeam(ctx context.Context, userID uuid.UUID, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error)

	// Invitation operations
	SendInvitation(ctx context.Context, inviterID uuid.UUID, in *collabv1.SendInvitationRequest) (*collabv1.Invitation, error)
	ListTeamInvitations(ctx context.Context, requesterID uuid.UUID, in *collabv1.ListTeamInvitationsRequest) (*collabv1.ListTeamInvitationsResponse, error)
	ListMyInvitations(ctx context.Context, userEmail string, in *collabv1.ListMyInvitationsRequest) (*collabv1.ListMyInvitationsResponse, error)
	AcceptInvitation(ctx context.Context, userID uuid.UUID, userEmail string, in *collabv1.InvitationTokenRequest) (*collabv1.AcceptInvitationResponse, error)
	RejectInvitation(ctx context.Context, userEmail string, in *collabv1.InvitationTokenRequest) (*commonv1.MessageResponse, error)
	RevokeInvitation(ctx context.Context, requesterID uuid.UUID, in *collabv1.GetByIDRequest) (*commonv1.MessageResponse, error)
}
