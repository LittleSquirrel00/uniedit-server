package model

import (
	"time"

	"github.com/google/uuid"
)

// TeamStatus represents the status of a team.
type TeamStatus string

const (
	TeamStatusActive  TeamStatus = "active"
	TeamStatusDeleted TeamStatus = "deleted"
)

// TeamVisibility represents team visibility.
type TeamVisibility string

const (
	TeamVisibilityPublic  TeamVisibility = "public"
	TeamVisibilityPrivate TeamVisibility = "private"
)

// TeamRole represents a team member's role.
type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
	TeamRoleGuest  TeamRole = "guest"
)

// IsValid checks if the role is valid.
func (r TeamRole) IsValid() bool {
	switch r {
	case TeamRoleOwner, TeamRoleAdmin, TeamRoleMember, TeamRoleGuest:
		return true
	default:
		return false
	}
}

// InvitationStatus represents the status of an invitation.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRejected InvitationStatus = "rejected"
	InvitationStatusRevoked  InvitationStatus = "revoked"
	InvitationStatusExpired  InvitationStatus = "expired"
)

// Team represents a collaboration team.
type Team struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID     uuid.UUID      `json:"owner_id" gorm:"type:uuid;not null"`
	Name        string         `json:"name" gorm:"not null"`
	Slug        string         `json:"slug" gorm:"not null"`
	Description string         `json:"description,omitempty"`
	Visibility  TeamVisibility `json:"visibility" gorm:"not null;default:private"`
	MemberLimit int            `json:"member_limit" gorm:"not null;default:5"`
	Status      TeamStatus     `json:"status" gorm:"not null;default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Relations (not loaded by default)
	Members []TeamMember `json:"members,omitempty" gorm:"foreignKey:TeamID"`
}

// TableName returns the database table name.
func (Team) TableName() string {
	return "teams"
}

// IsActive returns true if the team is active.
func (t *Team) IsActive() bool {
	return t.Status == TeamStatusActive
}

// TeamMember represents a team member.
type TeamMember struct {
	TeamID    uuid.UUID `json:"team_id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	Role      TeamRole  `json:"role" gorm:"not null;default:member"`
	JoinedAt  time.Time `json:"joined_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the database table name.
func (TeamMember) TableName() string {
	return "team_members"
}

// TeamInvitation represents a team invitation.
type TeamInvitation struct {
	ID           uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TeamID       uuid.UUID        `json:"team_id" gorm:"type:uuid;not null"`
	InviterID    uuid.UUID        `json:"inviter_id" gorm:"type:uuid;not null"`
	InviteeEmail string           `json:"invitee_email" gorm:"not null"`
	InviteeID    *uuid.UUID       `json:"invitee_id,omitempty" gorm:"type:uuid"`
	Role         TeamRole         `json:"role" gorm:"not null;default:member"`
	Token        string           `json:"-" gorm:"not null"`
	Status       InvitationStatus `json:"status" gorm:"not null;default:pending"`
	ExpiresAt    time.Time        `json:"expires_at" gorm:"not null"`
	CreatedAt    time.Time        `json:"created_at"`
	AcceptedAt   *time.Time       `json:"accepted_at,omitempty"`

	// Relations (for response)
	Team    *Team `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Inviter *User `json:"inviter,omitempty" gorm:"foreignKey:InviterID"`
}

// TableName returns the database table name.
func (TeamInvitation) TableName() string {
	return "team_invitations"
}

// IsExpired returns true if the invitation has expired.
func (i *TeamInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsPending returns true if the invitation is still pending.
func (i *TeamInvitation) IsPending() bool {
	return i.Status == InvitationStatusPending
}

// TeamMemberWithUser represents a team member with user details.
type TeamMemberWithUser struct {
	TeamMember
	Email string `json:"email"`
	Name  string `json:"name"`
}
