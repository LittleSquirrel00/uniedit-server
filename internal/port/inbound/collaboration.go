package inbound

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/model"
)

// --- Request/Response Types ---

// CreateTeamInput represents a request to create a team.
type CreateTeamInput struct {
	Name        string             `json:"name" binding:"required,min=1,max=100"`
	Description string             `json:"description" binding:"max=500"`
	Visibility  model.TeamVisibility `json:"visibility" binding:"omitempty,oneof=public private"`
}

// UpdateTeamInput represents a request to update a team.
type UpdateTeamInput struct {
	Name        *string              `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string              `json:"description" binding:"omitempty,max=500"`
	Visibility  *model.TeamVisibility `json:"visibility" binding:"omitempty,oneof=public private"`
}

// TeamOutput represents a team in API responses.
type TeamOutput struct {
	ID          uuid.UUID            `json:"id"`
	OwnerID     uuid.UUID            `json:"owner_id"`
	Name        string               `json:"name"`
	Slug        string               `json:"slug"`
	Description string               `json:"description,omitempty"`
	Visibility  model.TeamVisibility `json:"visibility"`
	MemberCount int                  `json:"member_count"`
	MemberLimit int                  `json:"member_limit"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	MyRole      *model.TeamRole      `json:"my_role,omitempty"`
}

// UpdateMemberRoleInput represents a request to update a member's role.
type UpdateMemberRoleInput struct {
	Role model.TeamRole `json:"role" binding:"required,oneof=admin member guest"`
}

// MemberOutput represents a team member in API responses.
type MemberOutput struct {
	UserID   uuid.UUID      `json:"user_id"`
	Email    string         `json:"email"`
	Name     string         `json:"name"`
	Role     model.TeamRole `json:"role"`
	JoinedAt time.Time      `json:"joined_at"`
}

// InviteInput represents a request to invite a user.
type InviteInput struct {
	Email string         `json:"email" binding:"required,email"`
	Role  model.TeamRole `json:"role" binding:"required,oneof=admin member guest"`
}

// InvitationOutput represents an invitation in API responses.
type InvitationOutput struct {
	ID           uuid.UUID              `json:"id"`
	TeamID       uuid.UUID              `json:"team_id"`
	TeamName     string                 `json:"team_name,omitempty"`
	InviterID    uuid.UUID              `json:"inviter_id"`
	InviterName  string                 `json:"inviter_name,omitempty"`
	InviteeEmail string                 `json:"invitee_email"`
	Role         model.TeamRole         `json:"role"`
	Status       model.InvitationStatus `json:"status"`
	ExpiresAt    time.Time              `json:"expires_at"`
	CreatedAt    time.Time              `json:"created_at"`
	AcceptedAt   *time.Time             `json:"accepted_at,omitempty"`
	Token        string                 `json:"token,omitempty"`
	AcceptURL    string                 `json:"accept_url,omitempty"`
}

// --- Domain Interface ---

// CollaborationDomain defines the collaboration domain service interface.
type CollaborationDomain interface {
	// Team operations
	CreateTeam(ctx context.Context, ownerID uuid.UUID, input *CreateTeamInput) (*TeamOutput, error)
	GetTeam(ctx context.Context, ownerID uuid.UUID, slug string, requesterID *uuid.UUID) (*TeamOutput, error)
	GetTeamByID(ctx context.Context, teamID uuid.UUID) (*model.Team, error)
	ListMyTeams(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TeamOutput, error)
	UpdateTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID, input *UpdateTeamInput) (*TeamOutput, error)
	DeleteTeam(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) error

	// Member operations
	ListMembers(ctx context.Context, teamID uuid.UUID, requesterID uuid.UUID) ([]*MemberOutput, error)
	UpdateMemberRole(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID, newRole model.TeamRole) error
	RemoveMember(ctx context.Context, teamID, targetUserID, requesterID uuid.UUID) error
	LeaveTeam(ctx context.Context, teamID, userID uuid.UUID) error
	GetMemberCount(ctx context.Context, teamID uuid.UUID) (int, error)
	GetMember(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error)

	// Invitation operations
	SendInvitation(ctx context.Context, teamID, inviterID uuid.UUID, input *InviteInput) (*InvitationOutput, error)
	ListTeamInvitations(ctx context.Context, teamID, requesterID uuid.UUID, status *model.InvitationStatus, limit, offset int) ([]*InvitationOutput, error)
	ListMyInvitations(ctx context.Context, email string, limit, offset int) ([]*InvitationOutput, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID, userEmail string) (*TeamOutput, error)
	RejectInvitation(ctx context.Context, token string, userEmail string) error
	RevokeInvitation(ctx context.Context, invitationID, requesterID uuid.UUID) error
}

// --- HTTP Port Interfaces ---

// TeamHttpPort defines team HTTP handlers.
type TeamHttpPort interface {
	// CreateTeam handles team creation.
	CreateTeam(c *gin.Context)

	// GetTeam handles getting a team.
	GetTeam(c *gin.Context)

	// ListMyTeams handles listing user's teams.
	ListMyTeams(c *gin.Context)

	// UpdateTeam handles updating a team.
	UpdateTeam(c *gin.Context)

	// DeleteTeam handles deleting a team.
	DeleteTeam(c *gin.Context)
}

// TeamMemberHttpPort defines team member HTTP handlers.
type TeamMemberHttpPort interface {
	// ListMembers handles listing team members.
	ListMembers(c *gin.Context)

	// UpdateMemberRole handles updating a member's role.
	UpdateMemberRole(c *gin.Context)

	// RemoveMember handles removing a member.
	RemoveMember(c *gin.Context)

	// LeaveTeam handles leaving a team.
	LeaveTeam(c *gin.Context)
}

// InvitationHttpPort defines invitation HTTP handlers.
type InvitationHttpPort interface {
	// SendInvitation handles sending an invitation.
	SendInvitation(c *gin.Context)

	// ListTeamInvitations handles listing team invitations.
	ListTeamInvitations(c *gin.Context)

	// ListMyInvitations handles listing user's invitations.
	ListMyInvitations(c *gin.Context)

	// AcceptInvitation handles accepting an invitation.
	AcceptInvitation(c *gin.Context)

	// RejectInvitation handles rejecting an invitation.
	RejectInvitation(c *gin.Context)

	// RevokeInvitation handles revoking an invitation.
	RevokeInvitation(c *gin.Context)
}

// CollaborationHttpPort combines all collaboration HTTP handlers.
type CollaborationHttpPort interface {
	TeamHttpPort
	TeamMemberHttpPort
	InvitationHttpPort
}
