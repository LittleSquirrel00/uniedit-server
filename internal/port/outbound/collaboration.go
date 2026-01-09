package outbound

import (
	"context"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/model"
)

// TeamDatabasePort defines team persistence operations.
type TeamDatabasePort interface {
	// Create creates a new team.
	Create(ctx context.Context, team *model.Team) error

	// FindByID retrieves a team by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.Team, error)

	// FindByOwnerAndSlug retrieves a team by owner and slug.
	FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.Team, error)

	// FindByUser lists all teams a user belongs to.
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Team, error)

	// Update updates a team.
	Update(ctx context.Context, team *model.Team) error

	// Delete soft-deletes a team.
	Delete(ctx context.Context, id uuid.UUID) error
}

// TeamMemberDatabasePort defines team member persistence operations.
type TeamMemberDatabasePort interface {
	// Add adds a member to a team.
	Add(ctx context.Context, member *model.TeamMember) error

	// Find retrieves a team member.
	Find(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error)

	// FindByTeam lists all members of a team.
	FindByTeam(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMember, error)

	// FindByTeamWithUsers lists all members with user details.
	FindByTeamWithUsers(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMemberWithUser, error)

	// UpdateRole updates a member's role.
	UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role model.TeamRole) error

	// Remove removes a member from a team.
	Remove(ctx context.Context, teamID, userID uuid.UUID) error

	// Count counts the number of members in a team.
	Count(ctx context.Context, teamID uuid.UUID) (int, error)
}

// TeamInvitationDatabasePort defines team invitation persistence operations.
type TeamInvitationDatabasePort interface {
	// Create creates a new invitation.
	Create(ctx context.Context, invitation *model.TeamInvitation) error

	// FindByID retrieves an invitation by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.TeamInvitation, error)

	// FindByToken retrieves an invitation by token.
	FindByToken(ctx context.Context, token string) (*model.TeamInvitation, error)

	// FindPendingByEmail retrieves a pending invitation for a team and email.
	FindPendingByEmail(ctx context.Context, teamID uuid.UUID, email string) (*model.TeamInvitation, error)

	// FindByTeam lists invitations for a team.
	FindByTeam(ctx context.Context, teamID uuid.UUID, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error)

	// FindByEmail lists invitations for an email.
	FindByEmail(ctx context.Context, email string, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error)

	// UpdateStatus updates an invitation's status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvitationStatus) error

	// CancelPendingByTeam cancels all pending invitations for a team.
	CancelPendingByTeam(ctx context.Context, teamID uuid.UUID) error
}

// CollaborationUserLookupPort defines user lookup for collaboration.
type CollaborationUserLookupPort interface {
	// FindByEmail retrieves a user by email.
	FindByEmail(ctx context.Context, email string) (*model.User, error)

	// FindByID retrieves a user by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// CollaborationTransactionPort defines transaction support for collaboration.
type CollaborationTransactionPort interface {
	// RunInTransaction executes the given function within a transaction.
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
