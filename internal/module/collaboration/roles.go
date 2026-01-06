package collaboration

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

// roleLevel maps roles to their hierarchy level (higher = more permissions).
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

// IsAtLeast checks if this role has at least the same level as another role.
func (r Role) IsAtLeast(other Role) bool {
	return r.Level() >= other.Level()
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

// CanAssign checks if a role can assign another role to members.
func (r Role) CanAssign(target Role) bool {
	// Owner can assign any role except owner
	if r == RoleOwner {
		return target != RoleOwner
	}
	// Admin can assign admin, member, or guest
	if r == RoleAdmin {
		return target == RoleAdmin || target == RoleMember || target == RoleGuest
	}
	return false
}

// HasPermission checks if a role has a specific permission.
func (r Role) HasPermission(perm Permission) bool {
	switch perm {
	case PermView:
		return true // All roles can view

	case PermInvite:
		return r == RoleOwner || r == RoleAdmin

	case PermRemoveMember:
		return r == RoleOwner || r == RoleAdmin

	case PermUpdateRole:
		return r == RoleOwner || r == RoleAdmin

	case PermUpdateTeam:
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
	case RoleGuest:
		return GitPermissionRead
	default:
		return GitPermissionRead
	}
}

// ValidInviteRoles returns the roles that can be assigned via invitation.
// Owner role cannot be assigned via invitation.
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
