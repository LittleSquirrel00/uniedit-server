package collaboration

import (
	"time"

	"github.com/google/uuid"
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

// String returns the string representation.
func (s InvitationStatus) String() string {
	return string(s)
}

// TeamInvitation represents a team invitation.
type TeamInvitation struct {
	id           uuid.UUID
	teamID       uuid.UUID
	inviterID    uuid.UUID
	inviteeEmail string
	inviteeID    *uuid.UUID
	role         Role
	token        string
	status       InvitationStatus
	expiresAt    time.Time
	createdAt    time.Time
	acceptedAt   *time.Time
}

// NewTeamInvitation creates a new team invitation.
func NewTeamInvitation(teamID, inviterID uuid.UUID, email string, role Role, token string, expiresAt time.Time) *TeamInvitation {
	return &TeamInvitation{
		id:           uuid.New(),
		teamID:       teamID,
		inviterID:    inviterID,
		inviteeEmail: email,
		role:         role,
		token:        token,
		status:       InvitationStatusPending,
		expiresAt:    expiresAt,
		createdAt:    time.Now(),
	}
}

// ReconstructTeamInvitation reconstructs an invitation from persistence.
func ReconstructTeamInvitation(
	id uuid.UUID,
	teamID uuid.UUID,
	inviterID uuid.UUID,
	inviteeEmail string,
	inviteeID *uuid.UUID,
	role Role,
	token string,
	status InvitationStatus,
	expiresAt time.Time,
	createdAt time.Time,
	acceptedAt *time.Time,
) *TeamInvitation {
	return &TeamInvitation{
		id:           id,
		teamID:       teamID,
		inviterID:    inviterID,
		inviteeEmail: inviteeEmail,
		inviteeID:    inviteeID,
		role:         role,
		token:        token,
		status:       status,
		expiresAt:    expiresAt,
		createdAt:    createdAt,
		acceptedAt:   acceptedAt,
	}
}

// Getters
func (i *TeamInvitation) ID() uuid.UUID           { return i.id }
func (i *TeamInvitation) TeamID() uuid.UUID       { return i.teamID }
func (i *TeamInvitation) InviterID() uuid.UUID    { return i.inviterID }
func (i *TeamInvitation) InviteeEmail() string    { return i.inviteeEmail }
func (i *TeamInvitation) InviteeID() *uuid.UUID   { return i.inviteeID }
func (i *TeamInvitation) Role() Role              { return i.role }
func (i *TeamInvitation) Token() string           { return i.token }
func (i *TeamInvitation) Status() InvitationStatus { return i.status }
func (i *TeamInvitation) ExpiresAt() time.Time    { return i.expiresAt }
func (i *TeamInvitation) CreatedAt() time.Time    { return i.createdAt }
func (i *TeamInvitation) AcceptedAt() *time.Time  { return i.acceptedAt }

// IsExpired returns true if the invitation has expired.
func (i *TeamInvitation) IsExpired() bool {
	return time.Now().After(i.expiresAt)
}

// IsPending returns true if the invitation is still pending.
func (i *TeamInvitation) IsPending() bool {
	return i.status == InvitationStatusPending
}

// IsForEmail checks if the invitation is for the given email.
func (i *TeamInvitation) IsForEmail(email string) bool {
	return i.inviteeEmail == email
}

// SetInviteeID sets the invitee user ID.
func (i *TeamInvitation) SetInviteeID(userID uuid.UUID) {
	i.inviteeID = &userID
}

// Accept accepts the invitation.
func (i *TeamInvitation) Accept() {
	now := time.Now()
	i.status = InvitationStatusAccepted
	i.acceptedAt = &now
}

// Reject rejects the invitation.
func (i *TeamInvitation) Reject() {
	i.status = InvitationStatusRejected
}

// Revoke revokes the invitation.
func (i *TeamInvitation) Revoke() {
	i.status = InvitationStatusRevoked
}

// MarkExpired marks the invitation as expired.
func (i *TeamInvitation) MarkExpired() {
	i.status = InvitationStatusExpired
}

// InvitationWithDetails represents an invitation with team and inviter details.
type InvitationWithDetails struct {
	invitation  *TeamInvitation
	teamName    string
	inviterName string
	inviterEmail string
}

// NewInvitationWithDetails creates a new invitation with details.
func NewInvitationWithDetails(inv *TeamInvitation, teamName, inviterName, inviterEmail string) *InvitationWithDetails {
	return &InvitationWithDetails{
		invitation:   inv,
		teamName:     teamName,
		inviterName:  inviterName,
		inviterEmail: inviterEmail,
	}
}

// Invitation returns the invitation.
func (d *InvitationWithDetails) Invitation() *TeamInvitation { return d.invitation }

// TeamName returns the team name.
func (d *InvitationWithDetails) TeamName() string { return d.teamName }

// InviterName returns the inviter's name.
func (d *InvitationWithDetails) InviterName() string { return d.inviterName }

// InviterEmail returns the inviter's email.
func (d *InvitationWithDetails) InviterEmail() string { return d.inviterEmail }
