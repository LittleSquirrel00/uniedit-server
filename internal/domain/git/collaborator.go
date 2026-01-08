package git

import (
	"time"

	"github.com/google/uuid"
)

// Collaborator represents a repository collaborator.
type Collaborator struct {
	repoID     uuid.UUID
	userID     uuid.UUID
	permission Permission
	createdAt  time.Time
}

// NewCollaborator creates a new collaborator.
func NewCollaborator(repoID, userID uuid.UUID, permission Permission) *Collaborator {
	return &Collaborator{
		repoID:     repoID,
		userID:     userID,
		permission: permission,
		createdAt:  time.Now(),
	}
}

// ReconstructCollaborator reconstructs a collaborator from persistence.
func ReconstructCollaborator(repoID, userID uuid.UUID, permission Permission, createdAt time.Time) *Collaborator {
	return &Collaborator{
		repoID:     repoID,
		userID:     userID,
		permission: permission,
		createdAt:  createdAt,
	}
}

// Getters
func (c *Collaborator) RepoID() uuid.UUID       { return c.repoID }
func (c *Collaborator) UserID() uuid.UUID       { return c.userID }
func (c *Collaborator) Permission() Permission  { return c.permission }
func (c *Collaborator) CreatedAt() time.Time    { return c.createdAt }

// SetPermission updates the permission.
func (c *Collaborator) SetPermission(p Permission) { c.permission = p }

// HasPermission checks if the collaborator has at least the given permission.
func (c *Collaborator) HasPermission(required Permission) bool {
	return c.permission.HasLevel(required)
}
