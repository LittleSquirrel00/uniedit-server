package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/collaboration"
)

// TeamEntity is the GORM entity for teams.
type TeamEntity struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null"`
	Name        string         `gorm:"not null"`
	Slug        string         `gorm:"not null"`
	Description string
	Visibility  string `gorm:"not null;default:private"`
	MemberLimit int    `gorm:"not null;default:5"`
	Status      string `gorm:"not null;default:active"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName returns the table name.
func (TeamEntity) TableName() string {
	return "teams"
}

// ToDomain converts to domain entity.
func (e *TeamEntity) ToDomain() *collaboration.Team {
	return collaboration.ReconstructTeam(
		e.ID,
		e.OwnerID,
		e.Name,
		e.Slug,
		e.Description,
		collaboration.Visibility(e.Visibility),
		e.MemberLimit,
		collaboration.TeamStatus(e.Status),
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// FromDomainTeam converts from domain entity.
func FromDomainTeam(t *collaboration.Team) *TeamEntity {
	return &TeamEntity{
		ID:          t.ID(),
		OwnerID:     t.OwnerID(),
		Name:        t.Name(),
		Slug:        t.Slug(),
		Description: t.Description(),
		Visibility:  string(t.Visibility()),
		MemberLimit: t.MemberLimit(),
		Status:      string(t.Status()),
		CreatedAt:   t.CreatedAt(),
		UpdatedAt:   t.UpdatedAt(),
	}
}

// TeamMemberEntity is the GORM entity for team members.
type TeamMemberEntity struct {
	TeamID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Role      string    `gorm:"not null;default:member"`
	JoinedAt  time.Time
	UpdatedAt time.Time
}

// TableName returns the table name.
func (TeamMemberEntity) TableName() string {
	return "team_members"
}

// ToDomain converts to domain entity.
func (e *TeamMemberEntity) ToDomain() *collaboration.TeamMember {
	return collaboration.ReconstructTeamMember(
		e.TeamID,
		e.UserID,
		collaboration.Role(e.Role),
		e.JoinedAt,
		e.UpdatedAt,
	)
}

// FromDomainTeamMember converts from domain entity.
func FromDomainTeamMember(m *collaboration.TeamMember) *TeamMemberEntity {
	return &TeamMemberEntity{
		TeamID:    m.TeamID(),
		UserID:    m.UserID(),
		Role:      m.Role().String(),
		JoinedAt:  m.JoinedAt(),
		UpdatedAt: m.UpdatedAt(),
	}
}

// MemberWithUserRow represents a join result.
type MemberWithUserRow struct {
	TeamMemberEntity
	Email string
	Name  string
}

// ToDomain converts to domain entity.
func (r *MemberWithUserRow) ToDomain() *collaboration.MemberWithUser {
	member := r.TeamMemberEntity.ToDomain()
	return collaboration.NewMemberWithUser(member, r.Email, r.Name)
}

// TeamInvitationEntity is the GORM entity for team invitations.
type TeamInvitationEntity struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TeamID       uuid.UUID  `gorm:"type:uuid;not null"`
	InviterID    uuid.UUID  `gorm:"type:uuid;not null"`
	InviteeEmail string     `gorm:"not null"`
	InviteeID    *uuid.UUID `gorm:"type:uuid"`
	Role         string     `gorm:"not null;default:member"`
	Token        string     `gorm:"not null"`
	Status       string     `gorm:"not null;default:pending"`
	ExpiresAt    time.Time  `gorm:"not null"`
	CreatedAt    time.Time
	AcceptedAt   *time.Time
}

// TableName returns the table name.
func (TeamInvitationEntity) TableName() string {
	return "team_invitations"
}

// ToDomain converts to domain entity.
func (e *TeamInvitationEntity) ToDomain() *collaboration.TeamInvitation {
	return collaboration.ReconstructTeamInvitation(
		e.ID,
		e.TeamID,
		e.InviterID,
		e.InviteeEmail,
		e.InviteeID,
		collaboration.Role(e.Role),
		e.Token,
		collaboration.InvitationStatus(e.Status),
		e.ExpiresAt,
		e.CreatedAt,
		e.AcceptedAt,
	)
}

// FromDomainTeamInvitation converts from domain entity.
func FromDomainTeamInvitation(inv *collaboration.TeamInvitation) *TeamInvitationEntity {
	return &TeamInvitationEntity{
		ID:           inv.ID(),
		TeamID:       inv.TeamID(),
		InviterID:    inv.InviterID(),
		InviteeEmail: inv.InviteeEmail(),
		InviteeID:    inv.InviteeID(),
		Role:         inv.Role().String(),
		Token:        inv.Token(),
		Status:       inv.Status().String(),
		ExpiresAt:    inv.ExpiresAt(),
		CreatedAt:    inv.CreatedAt(),
		AcceptedAt:   inv.AcceptedAt(),
	}
}
