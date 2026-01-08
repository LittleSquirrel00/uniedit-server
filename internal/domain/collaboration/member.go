package collaboration

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a team member's role.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// IsValid checks if the role is valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleGuest:
		return true
	default:
		return false
	}
}

// Permission represents an action that can be performed.
type Permission int

const (
	PermView Permission = iota
	PermInvite
	PermRemoveMember
	PermUpdateRole
	PermUpdateTeam
	PermDeleteTeam
)

// roleLevel maps roles to their hierarchy level.
var roleLevel = map[Role]int{
	RoleOwner:  100,
	RoleAdmin:  75,
	RoleMember: 50,
	RoleGuest:  25,
}

// Level returns the hierarchy level of the role.
func (r Role) Level() int {
	if level, ok := roleLevel[r]; ok {
		return level
	}
	return 0
}

// IsAtLeast checks if this role has at least the same level as another.
func (r Role) IsAtLeast(other Role) bool {
	return r.Level() >= other.Level()
}

// CanAssign checks if a role can assign another role to members.
func (r Role) CanAssign(target Role) bool {
	if r == RoleOwner {
		return target != RoleOwner
	}
	if r == RoleAdmin {
		return target == RoleAdmin || target == RoleMember || target == RoleGuest
	}
	return false
}

// HasPermission checks if a role has a specific permission.
func (r Role) HasPermission(perm Permission) bool {
	switch perm {
	case PermView:
		return true
	case PermInvite, PermRemoveMember, PermUpdateRole, PermUpdateTeam:
		return r == RoleOwner || r == RoleAdmin
	case PermDeleteTeam:
		return r == RoleOwner
	default:
		return false
	}
}

// GitPermission represents Git repository permission level.
type GitPermission string

const (
	GitPermissionRead  GitPermission = "read"
	GitPermissionWrite GitPermission = "write"
	GitPermissionAdmin GitPermission = "admin"
)

// ToGitPermission converts a team role to Git permission.
func (r Role) ToGitPermission() GitPermission {
	switch r {
	case RoleOwner, RoleAdmin:
		return GitPermissionAdmin
	case RoleMember:
		return GitPermissionWrite
	default:
		return GitPermissionRead
	}
}

// ValidInviteRoles returns roles that can be assigned via invitation.
func ValidInviteRoles() []Role {
	return []Role{RoleAdmin, RoleMember, RoleGuest}
}

// IsValidInviteRole checks if a role can be assigned via invitation.
func IsValidInviteRole(r Role) bool {
	for _, valid := range ValidInviteRoles() {
		if r == valid {
			return true
		}
	}
	return false
}

// TeamMember represents a team member.
type TeamMember struct {
	teamID    uuid.UUID
	userID    uuid.UUID
	role      Role
	joinedAt  time.Time
	updatedAt time.Time
}

// NewTeamMember creates a new team member.
func NewTeamMember(teamID, userID uuid.UUID, role Role) *TeamMember {
	now := time.Now()
	return &TeamMember{
		teamID:    teamID,
		userID:    userID,
		role:      role,
		joinedAt:  now,
		updatedAt: now,
	}
}

// ReconstructTeamMember reconstructs a team member from persistence.
func ReconstructTeamMember(teamID, userID uuid.UUID, role Role, joinedAt, updatedAt time.Time) *TeamMember {
	return &TeamMember{
		teamID:    teamID,
		userID:    userID,
		role:      role,
		joinedAt:  joinedAt,
		updatedAt: updatedAt,
	}
}

// Getters
func (m *TeamMember) TeamID() uuid.UUID   { return m.teamID }
func (m *TeamMember) UserID() uuid.UUID   { return m.userID }
func (m *TeamMember) Role() Role          { return m.role }
func (m *TeamMember) JoinedAt() time.Time { return m.joinedAt }
func (m *TeamMember) UpdatedAt() time.Time { return m.updatedAt }

// IsOwner checks if this member is the owner.
func (m *TeamMember) IsOwner() bool {
	return m.role == RoleOwner
}

// SetRole updates the member's role.
func (m *TeamMember) SetRole(role Role) {
	m.role = role
	m.updatedAt = time.Now()
}

// MemberWithUser represents a team member with user details.
type MemberWithUser struct {
	member *TeamMember
	email  string
	name   string
}

// NewMemberWithUser creates a new member with user details.
func NewMemberWithUser(member *TeamMember, email, name string) *MemberWithUser {
	return &MemberWithUser{
		member: member,
		email:  email,
		name:   name,
	}
}

// Member returns the team member.
func (m *MemberWithUser) Member() *TeamMember { return m.member }

// Email returns the user's email.
func (m *MemberWithUser) Email() string { return m.email }

// Name returns the user's name.
func (m *MemberWithUser) Name() string { return m.name }
