// Package collaboration contains domain entities for team collaboration.
package collaboration

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
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

// Team represents a collaboration team.
type Team struct {
	id          uuid.UUID
	ownerID     uuid.UUID
	name        string
	slug        string
	description string
	visibility  Visibility
	memberLimit int
	status      TeamStatus
	createdAt   time.Time
	updatedAt   time.Time
}

// NewTeam creates a new team.
func NewTeam(ownerID uuid.UUID, name string) *Team {
	now := time.Now()
	return &Team{
		id:          uuid.New(),
		ownerID:     ownerID,
		name:        name,
		slug:        generateSlug(name),
		visibility:  VisibilityPrivate,
		memberLimit: 5,
		status:      TeamStatusActive,
		createdAt:   now,
		updatedAt:   now,
	}
}

// ReconstructTeam reconstructs a team from persistence.
func ReconstructTeam(
	id uuid.UUID,
	ownerID uuid.UUID,
	name string,
	slug string,
	description string,
	visibility Visibility,
	memberLimit int,
	status TeamStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *Team {
	return &Team{
		id:          id,
		ownerID:     ownerID,
		name:        name,
		slug:        slug,
		description: description,
		visibility:  visibility,
		memberLimit: memberLimit,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Getters
func (t *Team) ID() uuid.UUID        { return t.id }
func (t *Team) OwnerID() uuid.UUID   { return t.ownerID }
func (t *Team) Name() string         { return t.name }
func (t *Team) Slug() string         { return t.slug }
func (t *Team) Description() string  { return t.description }
func (t *Team) Visibility() Visibility { return t.visibility }
func (t *Team) MemberLimit() int     { return t.memberLimit }
func (t *Team) Status() TeamStatus   { return t.status }
func (t *Team) CreatedAt() time.Time { return t.createdAt }
func (t *Team) UpdatedAt() time.Time { return t.updatedAt }

// IsActive returns true if the team is active.
func (t *Team) IsActive() bool {
	return t.status == TeamStatusActive
}

// IsPublic returns true if the team is public.
func (t *Team) IsPublic() bool {
	return t.visibility == VisibilityPublic
}

// IsOwnedBy checks if the team is owned by the given user.
func (t *Team) IsOwnedBy(userID uuid.UUID) bool {
	return t.ownerID == userID
}

// Update methods
func (t *Team) SetName(name string) {
	t.name = name
	t.slug = generateSlug(name)
	t.updatedAt = time.Now()
}

func (t *Team) SetDescription(description string) {
	t.description = description
	t.updatedAt = time.Now()
}

func (t *Team) SetVisibility(visibility Visibility) {
	t.visibility = visibility
	t.updatedAt = time.Now()
}

func (t *Team) SetMemberLimit(limit int) {
	t.memberLimit = limit
	t.updatedAt = time.Now()
}

func (t *Team) Delete() {
	t.status = TeamStatusDeleted
	t.updatedAt = time.Now()
}

// Helper functions

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug generates a URL-friendly slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

// GenerateSecureToken generates a cryptographically secure random token.
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length+10], nil
}
