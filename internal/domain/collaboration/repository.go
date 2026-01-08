package collaboration

import (
	"context"

	"github.com/google/uuid"
)

// TeamRepository defines the interface for team persistence.
type TeamRepository interface {
	// Create creates a new team.
	Create(ctx context.Context, team *Team) error

	// GetByID retrieves a team by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Team, error)

	// GetByOwnerAndSlug retrieves a team by owner and slug.
	GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*Team, error)

	// ListByUser lists all teams a user belongs to.
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Team, error)

	// Update updates a team.
	Update(ctx context.Context, team *Team) error

	// Delete soft-deletes a team.
	Delete(ctx context.Context, id uuid.UUID) error
}

// MemberRepository defines the interface for team member persistence.
type MemberRepository interface {
	// Add adds a member to a team.
	Add(ctx context.Context, member *TeamMember) error

	// Get retrieves a team member.
	Get(ctx context.Context, teamID, userID uuid.UUID) (*TeamMember, error)

	// List lists all members of a team.
	List(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error)

	// ListWithUsers lists all members with user details.
	ListWithUsers(ctx context.Context, teamID uuid.UUID) ([]*MemberWithUser, error)

	// UpdateRole updates a member's role.
	UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role Role) error

	// Remove removes a member from a team.
	Remove(ctx context.Context, teamID, userID uuid.UUID) error

	// Count counts the number of members in a team.
	Count(ctx context.Context, teamID uuid.UUID) (int, error)
}

// InvitationRepository defines the interface for invitation persistence.
type InvitationRepository interface {
	// Create creates a new invitation.
	Create(ctx context.Context, invitation *TeamInvitation) error

	// GetByID retrieves an invitation by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*TeamInvitation, error)

	// GetByToken retrieves an invitation by token.
	GetByToken(ctx context.Context, token string) (*TeamInvitation, error)

	// GetPendingByEmail retrieves a pending invitation for a team and email.
	GetPendingByEmail(ctx context.Context, teamID uuid.UUID, email string) (*TeamInvitation, error)

	// ListByTeam lists invitations for a team.
	ListByTeam(ctx context.Context, teamID uuid.UUID, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error)

	// ListByEmail lists invitations for an email.
	ListByEmail(ctx context.Context, email string, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error)

	// UpdateStatus updates an invitation's status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status InvitationStatus) error

	// CancelPending cancels all pending invitations for a team.
	CancelPending(ctx context.Context, teamID uuid.UUID) error
}

// UnitOfWork provides transaction support.
type UnitOfWork interface {
	// Begin starts a new transaction.
	Begin(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction.
type Transaction interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error

	// TeamRepository returns a team repository within this transaction.
	TeamRepository() TeamRepository

	// MemberRepository returns a member repository within this transaction.
	MemberRepository() MemberRepository

	// InvitationRepository returns an invitation repository within this transaction.
	InvitationRepository() InvitationRepository
}

// UserLookup provides user lookup functionality.
// This is a port interface - implementation depends on user module.
type UserLookup interface {
	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (*UserInfo, error)

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*UserInfo, error)
}

// UserInfo represents minimal user information.
type UserInfo struct {
	ID    uuid.UUID
	Email string
	Name  string
}
