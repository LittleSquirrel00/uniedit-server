package collaboration

import (
	"time"

	"github.com/google/uuid"
)

// CreateTeamRequest represents a request to create a team.
type CreateTeamRequest struct {
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"max=500"`
	Visibility  Visibility `json:"visibility" binding:"omitempty,oneof=public private"`
}

// UpdateTeamRequest represents a request to update a team.
type UpdateTeamRequest struct {
	Name        *string     `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string     `json:"description" binding:"omitempty,max=500"`
	Visibility  *Visibility `json:"visibility" binding:"omitempty,oneof=public private"`
}

// TeamResponse represents a team in API responses.
type TeamResponse struct {
	ID          uuid.UUID  `json:"id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	Visibility  Visibility `json:"visibility"`
	MemberCount int        `json:"member_count"`
	MemberLimit int        `json:"member_limit"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Optional: current user's role in this team
	MyRole *Role `json:"my_role,omitempty"`
}

// ToResponse converts a Team to TeamResponse.
func (t *Team) ToResponse(memberCount int, myRole *Role) *TeamResponse {
	return &TeamResponse{
		ID:          t.ID,
		OwnerID:     t.OwnerID,
		Name:        t.Name,
		Slug:        t.Slug,
		Description: t.Description,
		Visibility:  t.Visibility,
		MemberCount: memberCount,
		MemberLimit: t.MemberLimit,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		MyRole:      myRole,
	}
}

// UpdateMemberRoleRequest represents a request to update a member's role.
type UpdateMemberRoleRequest struct {
	Role Role `json:"role" binding:"required,oneof=admin member guest"`
}

// MemberResponse represents a team member in API responses.
type MemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     Role      `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// InviteRequest represents a request to invite a user.
type InviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  Role   `json:"role" binding:"required,oneof=admin member guest"`
}

// InvitationResponse represents an invitation in API responses.
type InvitationResponse struct {
	ID           uuid.UUID        `json:"id"`
	TeamID       uuid.UUID        `json:"team_id"`
	TeamName     string           `json:"team_name,omitempty"`
	InviterID    uuid.UUID        `json:"inviter_id"`
	InviterName  string           `json:"inviter_name,omitempty"`
	InviteeEmail string           `json:"invitee_email"`
	Role         Role             `json:"role"`
	Status       InvitationStatus `json:"status"`
	ExpiresAt    time.Time        `json:"expires_at"`
	CreatedAt    time.Time        `json:"created_at"`
	AcceptedAt   *time.Time       `json:"accepted_at,omitempty"`

	// Token is included only when creating the invitation
	Token     string `json:"token,omitempty"`
	AcceptURL string `json:"accept_url,omitempty"`
}

// ToResponse converts a TeamInvitation to InvitationResponse.
func (i *TeamInvitation) ToResponse(includeToken bool, baseURL string) *InvitationResponse {
	resp := &InvitationResponse{
		ID:           i.ID,
		TeamID:       i.TeamID,
		InviterID:    i.InviterID,
		InviteeEmail: i.InviteeEmail,
		Role:         i.Role,
		Status:       i.Status,
		ExpiresAt:    i.ExpiresAt,
		CreatedAt:    i.CreatedAt,
		AcceptedAt:   i.AcceptedAt,
	}

	if i.Team != nil {
		resp.TeamName = i.Team.Name
	}
	if i.Inviter != nil {
		resp.InviterName = i.Inviter.Name
	}

	if includeToken {
		resp.Token = i.Token
		if baseURL != "" {
			resp.AcceptURL = baseURL + "/invitations/" + i.Token + "/accept"
		}
	}

	return resp
}

// ListTeamsQuery represents query parameters for listing teams.
type ListTeamsQuery struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

// ListInvitationsQuery represents query parameters for listing invitations.
type ListInvitationsQuery struct {
	Status InvitationStatus `form:"status" binding:"omitempty,oneof=pending accepted rejected revoked expired"`
	Limit  int              `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int              `form:"offset" binding:"omitempty,min=0"`
}
