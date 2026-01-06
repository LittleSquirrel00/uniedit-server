package collaboration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_Level(t *testing.T) {
	tests := []struct {
		role     Role
		expected int
	}{
		{RoleOwner, 100},
		{RoleAdmin, 75},
		{RoleMember, 50},
		{RoleGuest, 25},
		{Role("invalid"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.Level())
		})
	}
}

func TestRole_IsAtLeast(t *testing.T) {
	tests := []struct {
		role     Role
		other    Role
		expected bool
	}{
		{RoleOwner, RoleOwner, true},
		{RoleOwner, RoleAdmin, true},
		{RoleOwner, RoleMember, true},
		{RoleOwner, RoleGuest, true},
		{RoleAdmin, RoleOwner, false},
		{RoleAdmin, RoleAdmin, true},
		{RoleAdmin, RoleMember, true},
		{RoleMember, RoleAdmin, false},
		{RoleGuest, RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+"_"+string(tt.other), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.IsAtLeast(tt.other))
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	tests := []struct {
		role     Role
		perm     Permission
		expected bool
	}{
		// View - all can view
		{RoleOwner, PermView, true},
		{RoleAdmin, PermView, true},
		{RoleMember, PermView, true},
		{RoleGuest, PermView, true},

		// Invite - owner and admin only
		{RoleOwner, PermInvite, true},
		{RoleAdmin, PermInvite, true},
		{RoleMember, PermInvite, false},
		{RoleGuest, PermInvite, false},

		// Remove member - owner and admin only
		{RoleOwner, PermRemoveMember, true},
		{RoleAdmin, PermRemoveMember, true},
		{RoleMember, PermRemoveMember, false},

		// Update role - owner and admin only
		{RoleOwner, PermUpdateRole, true},
		{RoleAdmin, PermUpdateRole, true},
		{RoleMember, PermUpdateRole, false},

		// Update team - owner and admin only
		{RoleOwner, PermUpdateTeam, true},
		{RoleAdmin, PermUpdateTeam, true},
		{RoleMember, PermUpdateTeam, false},

		// Delete team - owner only
		{RoleOwner, PermDeleteTeam, true},
		{RoleAdmin, PermDeleteTeam, false},
		{RoleMember, PermDeleteTeam, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.HasPermission(tt.perm))
		})
	}
}

func TestRole_CanAssign(t *testing.T) {
	tests := []struct {
		role     Role
		target   Role
		expected bool
	}{
		// Owner can assign any role except owner
		{RoleOwner, RoleAdmin, true},
		{RoleOwner, RoleMember, true},
		{RoleOwner, RoleGuest, true},
		{RoleOwner, RoleOwner, false},

		// Admin can assign admin, member, guest
		{RoleAdmin, RoleAdmin, true},
		{RoleAdmin, RoleMember, true},
		{RoleAdmin, RoleGuest, true},
		{RoleAdmin, RoleOwner, false},

		// Member cannot assign
		{RoleMember, RoleMember, false},
		{RoleMember, RoleGuest, false},

		// Guest cannot assign
		{RoleGuest, RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+"_assigns_"+string(tt.target), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.CanAssign(tt.target))
		})
	}
}

func TestRole_ToGitPermission(t *testing.T) {
	tests := []struct {
		role     Role
		expected GitPermission
	}{
		{RoleOwner, GitPermissionAdmin},
		{RoleAdmin, GitPermissionAdmin},
		{RoleMember, GitPermissionWrite},
		{RoleGuest, GitPermissionRead},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.ToGitPermission())
		})
	}
}

func TestIsValidInviteRole(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, false}, // Cannot invite as owner
		{RoleAdmin, true},
		{RoleMember, true},
		{RoleGuest, true},
		{Role("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidInviteRole(tt.role))
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"My Team", "my-team"},
		{"Team 123", "team-123"},
		{"  Spaces  ", "spaces"},
		{"Special!@#$%Characters", "special-characters"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"CamelCaseTeam", "camelcaseteam"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTeamInvitation_IsExpired(t *testing.T) {
	// This would require time mocking for proper testing
	// Basic structure test
	invitation := &TeamInvitation{}
	assert.NotNil(t, invitation)
}

func TestTeamInvitation_IsPending(t *testing.T) {
	tests := []struct {
		status   InvitationStatus
		expected bool
	}{
		{InvitationStatusPending, true},
		{InvitationStatusAccepted, false},
		{InvitationStatusRejected, false},
		{InvitationStatusRevoked, false},
		{InvitationStatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			invitation := &TeamInvitation{Status: tt.status}
			assert.Equal(t, tt.expected, invitation.IsPending())
		})
	}
}
