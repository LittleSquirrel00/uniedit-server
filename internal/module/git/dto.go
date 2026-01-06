package git

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateRepoRequest represents a request to create a repository.
type CreateRepoRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Type        string `json:"type" binding:"omitempty,oneof=code workflow project"`
	Visibility  string `json:"visibility" binding:"omitempty,oneof=public private"`
	Description string `json:"description" binding:"max=500"`
	LFSEnabled  bool   `json:"lfs_enabled"`
}

// UpdateRepoRequest represents a request to update a repository.
type UpdateRepoRequest struct {
	Name          string `json:"name" binding:"omitempty,min=1,max=100"`
	Description   string `json:"description" binding:"max=500"`
	Visibility    string `json:"visibility" binding:"omitempty,oneof=public private"`
	DefaultBranch string `json:"default_branch" binding:"omitempty,min=1,max=255"`
	LFSEnabled    *bool  `json:"lfs_enabled"`
}

// AddCollaboratorRequest represents a request to add a collaborator.
type AddCollaboratorRequest struct {
	Permission string `json:"permission" binding:"required,oneof=read write admin"`
}

// CreatePRRequest represents a request to create a pull request.
type CreatePRRequest struct {
	Title        string `json:"title" binding:"required,min=1,max=255"`
	Description  string `json:"description" binding:"max=10000"`
	SourceBranch string `json:"source_branch" binding:"required,min=1,max=255"`
	TargetBranch string `json:"target_branch" binding:"required,min=1,max=255"`
}

// UpdatePRRequest represents a request to update a pull request.
type UpdatePRRequest struct {
	Title       string `json:"title" binding:"omitempty,min=1,max=255"`
	Description string `json:"description" binding:"max=10000"`
	Status      string `json:"status" binding:"omitempty,oneof=open closed"`
}

// --- Response DTOs ---

// RepoResponse represents a repository response.
type RepoResponse struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       uuid.UUID  `json:"owner_id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	RepoType      string     `json:"repo_type"`
	Visibility    string     `json:"visibility"`
	Description   string     `json:"description,omitempty"`
	DefaultBranch string     `json:"default_branch"`
	SizeBytes     int64      `json:"size_bytes"`
	LFSEnabled    bool       `json:"lfs_enabled"`
	LFSSizeBytes  int64      `json:"lfs_size_bytes"`
	TotalSize     int64      `json:"total_size"`
	StarsCount    int        `json:"stars_count"`
	ForksCount    int        `json:"forks_count"`
	ForkedFrom    *uuid.UUID `json:"forked_from,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	PushedAt      *time.Time `json:"pushed_at,omitempty"`
	CloneURL      string     `json:"clone_url,omitempty"`
	LFSURL        string     `json:"lfs_url,omitempty"`
}

// ToResponse converts a GitRepo to RepoResponse.
func (r *GitRepo) ToResponse(baseURL string) *RepoResponse {
	resp := &RepoResponse{
		ID:            r.ID,
		OwnerID:       r.OwnerID,
		Name:          r.Name,
		Slug:          r.Slug,
		RepoType:      string(r.RepoType),
		Visibility:    string(r.Visibility),
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		SizeBytes:     r.SizeBytes,
		LFSEnabled:    r.LFSEnabled,
		LFSSizeBytes:  r.LFSSizeBytes,
		TotalSize:     r.TotalSize(),
		StarsCount:    r.StarsCount,
		ForksCount:    r.ForksCount,
		ForkedFrom:    r.ForkedFrom,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		PushedAt:      r.PushedAt,
	}

	if baseURL != "" {
		resp.CloneURL = baseURL + "/git/" + r.OwnerID.String() + "/" + r.Slug + ".git"
		if r.LFSEnabled {
			resp.LFSURL = baseURL + "/lfs/" + r.OwnerID.String() + "/" + r.Slug
		}
	}

	return resp
}

// CollaboratorResponse represents a collaborator response.
type CollaboratorResponse struct {
	UserID     uuid.UUID `json:"user_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToResponse converts a RepoCollaborator to CollaboratorResponse.
func (c *RepoCollaborator) ToResponse() *CollaboratorResponse {
	return &CollaboratorResponse{
		UserID:     c.UserID,
		Permission: string(c.Permission),
		CreatedAt:  c.CreatedAt,
	}
}

// PRResponse represents a pull request response.
type PRResponse struct {
	ID           uuid.UUID  `json:"id"`
	RepoID       uuid.UUID  `json:"repo_id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	Status       string     `json:"status"`
	AuthorID     uuid.UUID  `json:"author_id"`
	MergedBy     *uuid.UUID `json:"merged_by,omitempty"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ToResponse converts a PullRequest to PRResponse.
func (pr *PullRequest) ToResponse() *PRResponse {
	return &PRResponse{
		ID:           pr.ID,
		RepoID:       pr.RepoID,
		Number:       pr.Number,
		Title:        pr.Title,
		Description:  pr.Description,
		SourceBranch: pr.SourceBranch,
		TargetBranch: pr.TargetBranch,
		Status:       string(pr.Status),
		AuthorID:     pr.AuthorID,
		MergedBy:     pr.MergedBy,
		MergedAt:     pr.MergedAt,
		ClosedAt:     pr.ClosedAt,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
	}
}

// LockResponse represents an LFS lock response.
type LockResponse struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	LockedAt time.Time `json:"locked_at"`
	Owner    LockOwner `json:"owner"`
}

// LockOwner represents a lock owner.
type LockOwner struct {
	Name string `json:"name"`
}

// ToResponse converts an LFSLock to LockResponse.
func (l *LFSLock) ToResponse(ownerName string) *LockResponse {
	return &LockResponse{
		ID:       l.ID.String(),
		Path:     l.Path,
		LockedAt: l.LockedAt,
		Owner: LockOwner{
			Name: ownerName,
		},
	}
}

// --- List Response ---

// ListReposResponse represents a list of repositories response.
type ListReposResponse struct {
	Repos      []*RepoResponse `json:"repos"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
}

// --- Query Parameters ---

// RepoFilter represents repository filter options.
type RepoFilter struct {
	Type       *RepoType
	Visibility *Visibility
	Search     string
	Page       int
	PageSize   int
}
