package model

import (
	"time"

	"github.com/google/uuid"
)

// CreateTeamInput represents a request to create a team.
type CreateTeamInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Visibility  TeamVisibility `json:"visibility"`
}

// UpdateTeamInput represents a request to update a team.
type UpdateTeamInput struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Visibility  *TeamVisibility `json:"visibility"`
}

// TeamOutput represents a team in API responses.
type TeamOutput struct {
	ID          uuid.UUID      `json:"id"`
	OwnerID     uuid.UUID      `json:"owner_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description,omitempty"`
	Visibility  TeamVisibility `json:"visibility"`
	MemberCount int            `json:"member_count"`
	MemberLimit int            `json:"member_limit"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	MyRole      *TeamRole      `json:"my_role,omitempty"`
}

// MemberOutput represents a team member in API responses.
type MemberOutput struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     TeamRole  `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// InviteInput represents a request to invite a user.
type InviteInput struct {
	Email string   `json:"email"`
	Role  TeamRole `json:"role"`
}

// InvitationOutput represents an invitation in API responses.
type InvitationOutput struct {
	ID           uuid.UUID        `json:"id"`
	TeamID       uuid.UUID        `json:"team_id"`
	TeamName     string           `json:"team_name,omitempty"`
	InviterID    uuid.UUID        `json:"inviter_id"`
	InviterName  string           `json:"inviter_name,omitempty"`
	InviteeEmail string           `json:"invitee_email"`
	Role         TeamRole         `json:"role"`
	Status       InvitationStatus `json:"status"`
	ExpiresAt    time.Time        `json:"expires_at"`
	CreatedAt    time.Time        `json:"created_at"`
	AcceptedAt   *time.Time       `json:"accepted_at,omitempty"`
	Token        string           `json:"token,omitempty"`
	AcceptURL    string           `json:"accept_url,omitempty"`
}
