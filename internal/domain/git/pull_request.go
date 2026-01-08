package git

import (
	"time"

	"github.com/google/uuid"
)

// PRStatus represents pull request status.
type PRStatus string

const (
	PRStatusOpen   PRStatus = "open"
	PRStatusMerged PRStatus = "merged"
	PRStatusClosed PRStatus = "closed"
)

// PullRequest represents a pull request entity.
type PullRequest struct {
	id           uuid.UUID
	repoID       uuid.UUID
	number       int
	title        string
	description  string
	sourceBranch string
	targetBranch string
	status       PRStatus
	authorID     uuid.UUID
	mergedBy     *uuid.UUID
	mergedAt     *time.Time
	closedAt     *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

// NewPullRequest creates a new pull request.
func NewPullRequest(repoID uuid.UUID, number int, title, sourceBranch, targetBranch string, authorID uuid.UUID) *PullRequest {
	return &PullRequest{
		id:           uuid.New(),
		repoID:       repoID,
		number:       number,
		title:        title,
		sourceBranch: sourceBranch,
		targetBranch: targetBranch,
		status:       PRStatusOpen,
		authorID:     authorID,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}
}

// ReconstructPullRequest reconstructs a pull request from persistence.
func ReconstructPullRequest(
	id uuid.UUID,
	repoID uuid.UUID,
	number int,
	title string,
	description string,
	sourceBranch string,
	targetBranch string,
	status PRStatus,
	authorID uuid.UUID,
	mergedBy *uuid.UUID,
	mergedAt *time.Time,
	closedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *PullRequest {
	return &PullRequest{
		id:           id,
		repoID:       repoID,
		number:       number,
		title:        title,
		description:  description,
		sourceBranch: sourceBranch,
		targetBranch: targetBranch,
		status:       status,
		authorID:     authorID,
		mergedBy:     mergedBy,
		mergedAt:     mergedAt,
		closedAt:     closedAt,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// Getters
func (p *PullRequest) ID() uuid.UUID           { return p.id }
func (p *PullRequest) RepoID() uuid.UUID       { return p.repoID }
func (p *PullRequest) Number() int             { return p.number }
func (p *PullRequest) Title() string           { return p.title }
func (p *PullRequest) Description() string     { return p.description }
func (p *PullRequest) SourceBranch() string    { return p.sourceBranch }
func (p *PullRequest) TargetBranch() string    { return p.targetBranch }
func (p *PullRequest) Status() PRStatus        { return p.status }
func (p *PullRequest) AuthorID() uuid.UUID     { return p.authorID }
func (p *PullRequest) MergedBy() *uuid.UUID    { return p.mergedBy }
func (p *PullRequest) MergedAt() *time.Time    { return p.mergedAt }
func (p *PullRequest) ClosedAt() *time.Time    { return p.closedAt }
func (p *PullRequest) CreatedAt() time.Time    { return p.createdAt }
func (p *PullRequest) UpdatedAt() time.Time    { return p.updatedAt }

// IsOpen returns true if the PR is open.
func (p *PullRequest) IsOpen() bool { return p.status == PRStatusOpen }

// IsMerged returns true if the PR is merged.
func (p *PullRequest) IsMerged() bool { return p.status == PRStatusMerged }

// IsClosed returns true if the PR is closed.
func (p *PullRequest) IsClosed() bool { return p.status == PRStatusClosed }

// Setters
func (p *PullRequest) SetTitle(title string) {
	p.title = title
	p.updatedAt = time.Now()
}

func (p *PullRequest) SetDescription(desc string) {
	p.description = desc
	p.updatedAt = time.Now()
}

// Close closes the pull request.
func (p *PullRequest) Close() {
	now := time.Now()
	p.status = PRStatusClosed
	p.closedAt = &now
	p.updatedAt = now
}

// Reopen reopens a closed pull request.
func (p *PullRequest) Reopen() {
	p.status = PRStatusOpen
	p.closedAt = nil
	p.updatedAt = time.Now()
}

// Merge merges the pull request.
func (p *PullRequest) Merge(userID uuid.UUID) {
	now := time.Now()
	p.status = PRStatusMerged
	p.mergedBy = &userID
	p.mergedAt = &now
	p.updatedAt = now
}
