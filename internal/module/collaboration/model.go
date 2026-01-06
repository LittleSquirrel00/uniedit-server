package collaboration

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

// Visibility represents team visibility.
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Role represents a team member's role.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

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
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID     uuid.UUID  `json:"owner_id" gorm:"type:uuid;not null"`
	Name        string     `json:"name" gorm:"not null"`
	Slug        string     `json:"slug" gorm:"not null"`
	Description string     `json:"description,omitempty"`
	Visibility  Visibility `json:"visibility" gorm:"not null;default:private"`
	MemberLimit int        `json:"member_limit" gorm:"not null;default:5"`
	Status      TeamStatus `json:"status" gorm:"not null;default:active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

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
	Role      Role      `json:"role" gorm:"not null;default:member"`
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
	Role         Role             `json:"role" gorm:"not null;default:member"`
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

// User is a minimal user struct for invitation response.
type User struct {
	ID    uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

// TableName returns the database table name.
func (User) TableName() string {
	return "users"
}
