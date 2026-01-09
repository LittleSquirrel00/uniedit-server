package collaboration

import "github.com/uniedit/server/internal/model"

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
var roleLevel = map[model.TeamRole]int{
	model.TeamRoleOwner:  100,
	model.TeamRoleAdmin:  75,
	model.TeamRoleMember: 50,
	model.TeamRoleGuest:  25,
}

// RoleLevel returns the hierarchy level of the role.
func RoleLevel(r model.TeamRole) int {
	if level, ok := roleLevel[r]; ok {
		return level
	}
	return 0
}

// RoleIsAtLeast checks if a role has at least the same level as another role.
func RoleIsAtLeast(r, other model.TeamRole) bool {
	return RoleLevel(r) >= RoleLevel(other)
}

// RoleCanAssign checks if a role can assign another role to members.
func RoleCanAssign(r, target model.TeamRole) bool {
	// Owner can assign any role except owner
	if r == model.TeamRoleOwner {
		return target != model.TeamRoleOwner
	}
	// Admin can assign admin, member, or guest
	if r == model.TeamRoleAdmin {
		return target == model.TeamRoleAdmin || target == model.TeamRoleMember || target == model.TeamRoleGuest
	}
	return false
}

// RoleHasPermission checks if a role has a specific permission.
func RoleHasPermission(r model.TeamRole, perm Permission) bool {
	switch perm {
	case PermView:
		return true // All roles can view

	case PermInvite:
		return r == model.TeamRoleOwner || r == model.TeamRoleAdmin

	case PermRemoveMember:
		return r == model.TeamRoleOwner || r == model.TeamRoleAdmin

	case PermUpdateRole:
		return r == model.TeamRoleOwner || r == model.TeamRoleAdmin

	case PermUpdateTeam:
		return r == model.TeamRoleOwner || r == model.TeamRoleAdmin

	case PermDeleteTeam:
		return r == model.TeamRoleOwner

	default:
		return false
	}
}

// ValidInviteRoles returns the roles that can be assigned via invitation.
// Owner role cannot be assigned via invitation.
func ValidInviteRoles() []model.TeamRole {
	return []model.TeamRole{model.TeamRoleAdmin, model.TeamRoleMember, model.TeamRoleGuest}
}

// IsValidInviteRole checks if a role can be assigned via invitation.
func IsValidInviteRole(r model.TeamRole) bool {
	for _, valid := range ValidInviteRoles() {
		if r == valid {
			return true
		}
	}
	return false
}

// ToGitPermission converts a team role to Git permission.
func ToGitPermission(r model.TeamRole) model.GitPermission {
	switch r {
	case model.TeamRoleOwner, model.TeamRoleAdmin:
		return model.GitPermissionAdmin
	case model.TeamRoleMember:
		return model.GitPermissionWrite
	case model.TeamRoleGuest:
		return model.GitPermissionRead
	default:
		return model.GitPermissionRead
	}
}
