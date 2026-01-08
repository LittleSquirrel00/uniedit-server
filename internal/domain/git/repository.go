package git

import (
	"time"

	"github.com/google/uuid"
)

// RepoType represents the type of repository.
type RepoType string

const (
	RepoTypeCode     RepoType = "code"
	RepoTypeWorkflow RepoType = "workflow"
	RepoTypeProject  RepoType = "project"
)

// Visibility represents repository visibility.
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Permission represents collaboration permission level.
type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
	PermissionAdmin Permission = "admin"
)

// HasLevel checks if this permission has at least the given level.
func (p Permission) HasLevel(required Permission) bool {
	levels := map[Permission]int{
		PermissionRead:  1,
		PermissionWrite: 2,
		PermissionAdmin: 3,
	}
	return levels[p] >= levels[required]
}

// Repository represents a Git repository entity.
type Repository struct {
	id            uuid.UUID
	ownerID       uuid.UUID
	name          string
	slug          string
	repoType      RepoType
	visibility    Visibility
	description   string
	defaultBranch string
	sizeBytes     int64
	lfsEnabled    bool
	lfsSizeBytes  int64
	storagePath   string
	starsCount    int
	forksCount    int
	forkedFrom    *uuid.UUID
	createdAt     time.Time
	updatedAt     time.Time
	pushedAt      *time.Time
}

// NewRepository creates a new repository.
func NewRepository(ownerID uuid.UUID, name, slug, storagePath string) *Repository {
	return &Repository{
		id:            uuid.New(),
		ownerID:       ownerID,
		name:          name,
		slug:          slug,
		repoType:      RepoTypeCode,
		visibility:    VisibilityPrivate,
		defaultBranch: "main",
		storagePath:   storagePath,
		createdAt:     time.Now(),
		updatedAt:     time.Now(),
	}
}

// ReconstructRepository reconstructs a repository from persistence.
func ReconstructRepository(
	id uuid.UUID,
	ownerID uuid.UUID,
	name string,
	slug string,
	repoType RepoType,
	visibility Visibility,
	description string,
	defaultBranch string,
	sizeBytes int64,
	lfsEnabled bool,
	lfsSizeBytes int64,
	storagePath string,
	starsCount int,
	forksCount int,
	forkedFrom *uuid.UUID,
	createdAt time.Time,
	updatedAt time.Time,
	pushedAt *time.Time,
) *Repository {
	return &Repository{
		id:            id,
		ownerID:       ownerID,
		name:          name,
		slug:          slug,
		repoType:      repoType,
		visibility:    visibility,
		description:   description,
		defaultBranch: defaultBranch,
		sizeBytes:     sizeBytes,
		lfsEnabled:    lfsEnabled,
		lfsSizeBytes:  lfsSizeBytes,
		storagePath:   storagePath,
		starsCount:    starsCount,
		forksCount:    forksCount,
		forkedFrom:    forkedFrom,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		pushedAt:      pushedAt,
	}
}

// Getters
func (r *Repository) ID() uuid.UUID            { return r.id }
func (r *Repository) OwnerID() uuid.UUID       { return r.ownerID }
func (r *Repository) Name() string             { return r.name }
func (r *Repository) Slug() string             { return r.slug }
func (r *Repository) RepoType() RepoType       { return r.repoType }
func (r *Repository) Visibility() Visibility   { return r.visibility }
func (r *Repository) Description() string      { return r.description }
func (r *Repository) DefaultBranch() string    { return r.defaultBranch }
func (r *Repository) SizeBytes() int64         { return r.sizeBytes }
func (r *Repository) LFSEnabled() bool         { return r.lfsEnabled }
func (r *Repository) LFSSizeBytes() int64      { return r.lfsSizeBytes }
func (r *Repository) StoragePath() string      { return r.storagePath }
func (r *Repository) StarsCount() int          { return r.starsCount }
func (r *Repository) ForksCount() int          { return r.forksCount }
func (r *Repository) ForkedFrom() *uuid.UUID   { return r.forkedFrom }
func (r *Repository) CreatedAt() time.Time     { return r.createdAt }
func (r *Repository) UpdatedAt() time.Time     { return r.updatedAt }
func (r *Repository) PushedAt() *time.Time     { return r.pushedAt }

// TotalSize returns total storage (Git + LFS).
func (r *Repository) TotalSize() int64 {
	return r.sizeBytes + r.lfsSizeBytes
}

// IsPublic returns true if the repository is public.
func (r *Repository) IsPublic() bool {
	return r.visibility == VisibilityPublic
}

// IsOwnedBy returns true if the repository is owned by the given user.
func (r *Repository) IsOwnedBy(userID uuid.UUID) bool {
	return r.ownerID == userID
}

// Setters
func (r *Repository) SetName(name string)                { r.name = name; r.updatedAt = time.Now() }
func (r *Repository) SetSlug(slug string)                { r.slug = slug; r.updatedAt = time.Now() }
func (r *Repository) SetRepoType(t RepoType)             { r.repoType = t; r.updatedAt = time.Now() }
func (r *Repository) SetVisibility(v Visibility)         { r.visibility = v; r.updatedAt = time.Now() }
func (r *Repository) SetDescription(desc string)         { r.description = desc; r.updatedAt = time.Now() }
func (r *Repository) SetDefaultBranch(branch string)     { r.defaultBranch = branch; r.updatedAt = time.Now() }
func (r *Repository) SetLFSEnabled(enabled bool)         { r.lfsEnabled = enabled; r.updatedAt = time.Now() }
func (r *Repository) SetSizeBytes(size int64)            { r.sizeBytes = size; r.updatedAt = time.Now() }
func (r *Repository) SetLFSSizeBytes(size int64)         { r.lfsSizeBytes = size; r.updatedAt = time.Now() }
func (r *Repository) SetForkedFrom(id *uuid.UUID)        { r.forkedFrom = id }
func (r *Repository) SetPushedAt(t *time.Time)           { r.pushedAt = t; r.updatedAt = time.Now() }
