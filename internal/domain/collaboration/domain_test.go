package collaboration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// Mock implementations

type mockTeamDB struct {
	mock.Mock
}

func (m *mockTeamDB) Create(ctx context.Context, team *model.Team) error {
	args := m.Called(ctx, team)
	return args.Error(0)
}

func (m *mockTeamDB) FindByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Team), args.Error(1)
}

func (m *mockTeamDB) FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.Team, error) {
	args := m.Called(ctx, ownerID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Team), args.Error(1)
}

func (m *mockTeamDB) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Team, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Team), args.Error(1)
}

func (m *mockTeamDB) Update(ctx context.Context, team *model.Team) error {
	args := m.Called(ctx, team)
	return args.Error(0)
}

func (m *mockTeamDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockMemberDB struct {
	mock.Mock
}

func (m *mockMemberDB) Add(ctx context.Context, member *model.TeamMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *mockMemberDB) Find(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	args := m.Called(ctx, teamID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TeamMember), args.Error(1)
}

func (m *mockMemberDB) FindByTeam(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMember, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TeamMember), args.Error(1)
}

func (m *mockMemberDB) FindByTeamWithUsers(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMemberWithUser, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TeamMemberWithUser), args.Error(1)
}

func (m *mockMemberDB) UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role model.TeamRole) error {
	args := m.Called(ctx, teamID, userID, role)
	return args.Error(0)
}

func (m *mockMemberDB) Remove(ctx context.Context, teamID, userID uuid.UUID) error {
	args := m.Called(ctx, teamID, userID)
	return args.Error(0)
}

func (m *mockMemberDB) Count(ctx context.Context, teamID uuid.UUID) (int, error) {
	args := m.Called(ctx, teamID)
	return args.Int(0), args.Error(1)
}

type mockInvitationDB struct {
	mock.Mock
}

func (m *mockInvitationDB) Create(ctx context.Context, invitation *model.TeamInvitation) error {
	args := m.Called(ctx, invitation)
	return args.Error(0)
}

func (m *mockInvitationDB) FindByID(ctx context.Context, id uuid.UUID) (*model.TeamInvitation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TeamInvitation), args.Error(1)
}

func (m *mockInvitationDB) FindByToken(ctx context.Context, token string) (*model.TeamInvitation, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TeamInvitation), args.Error(1)
}

func (m *mockInvitationDB) FindPendingByEmail(ctx context.Context, teamID uuid.UUID, email string) (*model.TeamInvitation, error) {
	args := m.Called(ctx, teamID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TeamInvitation), args.Error(1)
}

func (m *mockInvitationDB) FindByTeam(ctx context.Context, teamID uuid.UUID, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error) {
	args := m.Called(ctx, teamID, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TeamInvitation), args.Error(1)
}

func (m *mockInvitationDB) FindByEmail(ctx context.Context, email string, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error) {
	args := m.Called(ctx, email, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TeamInvitation), args.Error(1)
}

func (m *mockInvitationDB) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvitationStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockInvitationDB) CancelPendingByTeam(ctx context.Context, teamID uuid.UUID) error {
	args := m.Called(ctx, teamID)
	return args.Error(0)
}

type mockUserLookup struct {
	mock.Mock
}

func (m *mockUserLookup) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserLookup) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

type mockTransaction struct {
	mock.Mock
}

func (m *mockTransaction) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Simply execute the function - in tests we don't need actual transactions
	return fn(ctx)
}

// Test helper

func setupDomain() (*Domain, *mockTeamDB, *mockMemberDB, *mockInvitationDB, *mockUserLookup, *mockTransaction) {
	teamDB := new(mockTeamDB)
	memberDB := new(mockMemberDB)
	invitationDB := new(mockInvitationDB)
	userLookup := new(mockUserLookup)
	txPort := new(mockTransaction)

	domain := NewDomain(
		teamDB,
		memberDB,
		invitationDB,
		userLookup,
		txPort,
		DefaultConfig(),
		zap.NewNop(),
	)

	return domain, teamDB, memberDB, invitationDB, userLookup, txPort
}

// Tests

func TestDomain_CreateTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		ownerID := uuid.New()

		teamDB.On("FindByOwnerAndSlug", ctx, ownerID, "my-team").Return(nil, ErrTeamNotFound)
		teamDB.On("Create", ctx, mock.AnythingOfType("*model.Team")).Return(nil)
		memberDB.On("Add", ctx, mock.AnythingOfType("*model.TeamMember")).Return(nil)

		input := &inbound.CreateTeamInput{
			Name:        "My Team",
			Description: "Test team",
		}

		output, err := domain.CreateTeam(ctx, ownerID, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "My Team", output.Name)
		assert.Equal(t, "my-team", output.Slug)
		assert.Equal(t, ownerID, output.OwnerID)
		assert.NotNil(t, output.MyRole)
		assert.Equal(t, model.TeamRoleOwner, *output.MyRole)

		teamDB.AssertExpectations(t)
		memberDB.AssertExpectations(t)
	})

	t.Run("slug_already_exists", func(t *testing.T) {
		domain, teamDB, _, _, _, _ := setupDomain()
		ctx := context.Background()
		ownerID := uuid.New()

		existingTeam := &model.Team{ID: uuid.New(), Name: "My Team"}
		teamDB.On("FindByOwnerAndSlug", ctx, ownerID, "my-team").Return(existingTeam, nil)

		input := &inbound.CreateTeamInput{
			Name: "My Team",
		}

		output, err := domain.CreateTeam(ctx, ownerID, input)

		assert.Nil(t, output)
		assert.Equal(t, ErrSlugAlreadyExists, err)
	})
}

func TestDomain_GetTeam(t *testing.T) {
	t.Run("success_as_member", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		ownerID := uuid.New()
		requesterID := uuid.New()
		teamID := uuid.New()

		team := &model.Team{
			ID:         teamID,
			OwnerID:    ownerID,
			Name:       "Test Team",
			Slug:       "test-team",
			Visibility: model.TeamVisibilityPrivate,
		}
		member := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleMember,
		}

		teamDB.On("FindByOwnerAndSlug", ctx, ownerID, "test-team").Return(team, nil)
		memberDB.On("Find", ctx, teamID, requesterID).Return(member, nil)
		memberDB.On("Count", ctx, teamID).Return(3, nil)

		output, err := domain.GetTeam(ctx, ownerID, "test-team", &requesterID)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "Test Team", output.Name)
		assert.Equal(t, model.TeamRoleMember, *output.MyRole)
		assert.Equal(t, 3, output.MemberCount)
	})

	t.Run("private_team_non_member", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		ownerID := uuid.New()
		requesterID := uuid.New()
		teamID := uuid.New()

		team := &model.Team{
			ID:         teamID,
			OwnerID:    ownerID,
			Visibility: model.TeamVisibilityPrivate,
		}

		teamDB.On("FindByOwnerAndSlug", ctx, ownerID, "test-team").Return(team, nil)
		memberDB.On("Find", ctx, teamID, requesterID).Return(nil, ErrMemberNotFound)

		output, err := domain.GetTeam(ctx, ownerID, "test-team", &requesterID)

		assert.Nil(t, output)
		assert.Equal(t, ErrTeamNotFound, err)
	})
}

func TestDomain_UpdateMemberRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()
		targetID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleOwner,
		}
		target := &model.TeamMember{
			TeamID: teamID,
			UserID: targetID,
			Role:   model.TeamRoleMember,
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		memberDB.On("Find", ctx, teamID, targetID).Return(target, nil)
		memberDB.On("UpdateRole", ctx, teamID, targetID, model.TeamRoleAdmin).Return(nil)

		err := domain.UpdateMemberRole(ctx, teamID, targetID, requesterID, model.TeamRoleAdmin)

		require.NoError(t, err)
		memberDB.AssertExpectations(t)
	})

	t.Run("cannot_change_owner", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()
		targetID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}
		target := &model.TeamMember{
			TeamID: teamID,
			UserID: targetID,
			Role:   model.TeamRoleOwner,
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		memberDB.On("Find", ctx, teamID, targetID).Return(target, nil)

		err := domain.UpdateMemberRole(ctx, teamID, targetID, requesterID, model.TeamRoleMember)

		assert.Equal(t, ErrCannotChangeOwner, err)
	})

	t.Run("insufficient_permission", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleMember, // Member cannot update roles
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)

		err := domain.UpdateMemberRole(ctx, teamID, uuid.New(), requesterID, model.TeamRoleAdmin)

		assert.Equal(t, ErrInsufficientPermission, err)
	})
}

func TestDomain_RemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()
		targetID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}
		target := &model.TeamMember{
			TeamID: teamID,
			UserID: targetID,
			Role:   model.TeamRoleMember,
		}

		memberDB.On("Find", ctx, teamID, targetID).Return(target, nil)
		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		memberDB.On("Remove", ctx, teamID, targetID).Return(nil)

		err := domain.RemoveMember(ctx, teamID, targetID, requesterID)

		require.NoError(t, err)
		memberDB.AssertExpectations(t)
	})

	t.Run("cannot_remove_owner", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()
		ownerID := uuid.New()

		owner := &model.TeamMember{
			TeamID: teamID,
			UserID: ownerID,
			Role:   model.TeamRoleOwner,
		}

		memberDB.On("Find", ctx, teamID, ownerID).Return(owner, nil)

		err := domain.RemoveMember(ctx, teamID, ownerID, requesterID)

		assert.Equal(t, ErrCannotRemoveOwner, err)
	})

	t.Run("self_leave_allowed", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		userID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   model.TeamRoleMember,
		}

		memberDB.On("Find", ctx, teamID, userID).Return(member, nil)
		memberDB.On("Remove", ctx, teamID, userID).Return(nil)

		// User can remove themselves without needing admin permission
		err := domain.RemoveMember(ctx, teamID, userID, userID)

		require.NoError(t, err)
	})
}

func TestDomain_SendInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, invitationDB, userLookup, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		inviterID := uuid.New()

		team := &model.Team{
			ID:          teamID,
			MemberLimit: 10,
		}
		inviter := &model.TeamMember{
			TeamID: teamID,
			UserID: inviterID,
			Role:   model.TeamRoleAdmin,
		}

		memberDB.On("Find", ctx, teamID, inviterID).Return(inviter, nil)
		teamDB.On("FindByID", ctx, teamID).Return(team, nil)
		memberDB.On("Count", ctx, teamID).Return(3, nil)
		userLookup.On("FindByEmail", ctx, "test@example.com").Return(nil, nil)
		invitationDB.On("FindPendingByEmail", ctx, teamID, "test@example.com").Return(nil, nil)
		invitationDB.On("Create", ctx, mock.AnythingOfType("*model.TeamInvitation")).Return(nil)
		userLookup.On("FindByID", ctx, inviterID).Return(&model.User{ID: inviterID, Name: "Inviter"}, nil)

		input := &inbound.InviteInput{
			Email: "test@example.com",
			Role:  model.TeamRoleMember,
		}

		output, err := domain.SendInvitation(ctx, teamID, inviterID, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "test@example.com", output.InviteeEmail)
		assert.Equal(t, model.TeamRoleMember, output.Role)
		assert.Equal(t, model.InvitationStatusPending, output.Status)
	})

	t.Run("already_member", func(t *testing.T) {
		domain, teamDB, memberDB, _, userLookup, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		inviterID := uuid.New()
		existingUserID := uuid.New()

		team := &model.Team{
			ID:          teamID,
			MemberLimit: 10,
		}
		inviter := &model.TeamMember{
			TeamID: teamID,
			UserID: inviterID,
			Role:   model.TeamRoleAdmin,
		}
		existingUser := &model.User{
			ID:    existingUserID,
			Email: "test@example.com",
		}
		existingMember := &model.TeamMember{
			TeamID: teamID,
			UserID: existingUserID,
		}

		memberDB.On("Find", ctx, teamID, inviterID).Return(inviter, nil)
		teamDB.On("FindByID", ctx, teamID).Return(team, nil)
		memberDB.On("Count", ctx, teamID).Return(3, nil)
		userLookup.On("FindByEmail", ctx, "test@example.com").Return(existingUser, nil)
		memberDB.On("Find", ctx, teamID, existingUserID).Return(existingMember, nil)

		input := &inbound.InviteInput{
			Email: "test@example.com",
			Role:  model.TeamRoleMember,
		}

		output, err := domain.SendInvitation(ctx, teamID, inviterID, input)

		assert.Nil(t, output)
		assert.Equal(t, ErrAlreadyMember, err)
	})

	t.Run("member_limit_exceeded", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		inviterID := uuid.New()

		team := &model.Team{
			ID:          teamID,
			MemberLimit: 5,
		}
		inviter := &model.TeamMember{
			TeamID: teamID,
			UserID: inviterID,
			Role:   model.TeamRoleAdmin,
		}

		memberDB.On("Find", ctx, teamID, inviterID).Return(inviter, nil)
		teamDB.On("FindByID", ctx, teamID).Return(team, nil)
		memberDB.On("Count", ctx, teamID).Return(5, nil) // At limit

		input := &inbound.InviteInput{
			Email: "test@example.com",
			Role:  model.TeamRoleMember,
		}

		output, err := domain.SendInvitation(ctx, teamID, inviterID, input)

		assert.Nil(t, output)
		assert.Equal(t, ErrMemberLimitExceeded, err)
	})
}

func TestDomain_AcceptInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		userID := uuid.New()

		team := &model.Team{
			ID:          teamID,
			Name:        "Test Team",
			MemberLimit: 10,
		}
		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			TeamID:       teamID,
			InviteeEmail: "user@example.com",
			Role:         model.TeamRoleMember,
			Status:       model.InvitationStatusPending,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)
		teamDB.On("FindByID", ctx, teamID).Return(team, nil)
		memberDB.On("Count", ctx, teamID).Return(3, nil)
		memberDB.On("Add", ctx, mock.AnythingOfType("*model.TeamMember")).Return(nil)
		invitationDB.On("UpdateStatus", ctx, invitation.ID, model.InvitationStatusAccepted).Return(nil)

		output, err := domain.AcceptInvitation(ctx, "test-token", userID, "user@example.com")

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "Test Team", output.Name)
	})

	t.Run("invitation_expired", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		userID := uuid.New()

		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			InviteeEmail: "user@example.com",
			Status:       model.InvitationStatusPending,
			ExpiresAt:    time.Now().Add(-24 * time.Hour), // Expired
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)
		invitationDB.On("UpdateStatus", ctx, invitation.ID, model.InvitationStatusExpired).Return(nil)

		output, err := domain.AcceptInvitation(ctx, "test-token", userID, "user@example.com")

		assert.Nil(t, output)
		assert.Equal(t, ErrInvitationExpired, err)
	})

	t.Run("invitation_not_for_you", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		userID := uuid.New()

		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			InviteeEmail: "other@example.com", // Different email
			Status:       model.InvitationStatusPending,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)

		output, err := domain.AcceptInvitation(ctx, "test-token", userID, "user@example.com")

		assert.Nil(t, output)
		assert.Equal(t, ErrInvitationNotForYou, err)
	})
}

func TestDomain_DeleteTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		ownerID := uuid.New()

		owner := &model.TeamMember{
			TeamID: teamID,
			UserID: ownerID,
			Role:   model.TeamRoleOwner,
		}

		memberDB.On("Find", ctx, teamID, ownerID).Return(owner, nil)
		invitationDB.On("CancelPendingByTeam", ctx, teamID).Return(nil)
		teamDB.On("Delete", ctx, teamID).Return(nil)

		err := domain.DeleteTeam(ctx, teamID, ownerID)

		require.NoError(t, err)
		teamDB.AssertExpectations(t)
		invitationDB.AssertExpectations(t)
	})

	t.Run("only_owner_can_delete", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		adminID := uuid.New()

		admin := &model.TeamMember{
			TeamID: teamID,
			UserID: adminID,
			Role:   model.TeamRoleAdmin, // Not owner
		}

		memberDB.On("Find", ctx, teamID, adminID).Return(admin, nil)

		err := domain.DeleteTeam(ctx, teamID, adminID)

		assert.Equal(t, ErrOnlyOwnerCanDelete, err)
	})
}

func TestRolePermissions(t *testing.T) {
	t.Run("owner_has_all_permissions", func(t *testing.T) {
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermView))
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermInvite))
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermRemoveMember))
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermUpdateRole))
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermUpdateTeam))
		assert.True(t, RoleHasPermission(model.TeamRoleOwner, PermDeleteTeam))
	})

	t.Run("admin_permissions", func(t *testing.T) {
		assert.True(t, RoleHasPermission(model.TeamRoleAdmin, PermView))
		assert.True(t, RoleHasPermission(model.TeamRoleAdmin, PermInvite))
		assert.True(t, RoleHasPermission(model.TeamRoleAdmin, PermRemoveMember))
		assert.True(t, RoleHasPermission(model.TeamRoleAdmin, PermUpdateRole))
		assert.True(t, RoleHasPermission(model.TeamRoleAdmin, PermUpdateTeam))
		assert.False(t, RoleHasPermission(model.TeamRoleAdmin, PermDeleteTeam)) // Only owner can delete
	})

	t.Run("member_permissions", func(t *testing.T) {
		assert.True(t, RoleHasPermission(model.TeamRoleMember, PermView))
		assert.False(t, RoleHasPermission(model.TeamRoleMember, PermInvite))
		assert.False(t, RoleHasPermission(model.TeamRoleMember, PermRemoveMember))
		assert.False(t, RoleHasPermission(model.TeamRoleMember, PermUpdateRole))
		assert.False(t, RoleHasPermission(model.TeamRoleMember, PermUpdateTeam))
		assert.False(t, RoleHasPermission(model.TeamRoleMember, PermDeleteTeam))
	})

	t.Run("guest_permissions", func(t *testing.T) {
		assert.True(t, RoleHasPermission(model.TeamRoleGuest, PermView))
		assert.False(t, RoleHasPermission(model.TeamRoleGuest, PermInvite))
		assert.False(t, RoleHasPermission(model.TeamRoleGuest, PermRemoveMember))
	})
}

func TestRoleCanAssign(t *testing.T) {
	t.Run("owner_can_assign_all_except_owner", func(t *testing.T) {
		assert.False(t, RoleCanAssign(model.TeamRoleOwner, model.TeamRoleOwner))
		assert.True(t, RoleCanAssign(model.TeamRoleOwner, model.TeamRoleAdmin))
		assert.True(t, RoleCanAssign(model.TeamRoleOwner, model.TeamRoleMember))
		assert.True(t, RoleCanAssign(model.TeamRoleOwner, model.TeamRoleGuest))
	})

	t.Run("admin_can_assign_admin_member_guest", func(t *testing.T) {
		assert.False(t, RoleCanAssign(model.TeamRoleAdmin, model.TeamRoleOwner))
		assert.True(t, RoleCanAssign(model.TeamRoleAdmin, model.TeamRoleAdmin))
		assert.True(t, RoleCanAssign(model.TeamRoleAdmin, model.TeamRoleMember))
		assert.True(t, RoleCanAssign(model.TeamRoleAdmin, model.TeamRoleGuest))
	})

	t.Run("member_cannot_assign", func(t *testing.T) {
		assert.False(t, RoleCanAssign(model.TeamRoleMember, model.TeamRoleOwner))
		assert.False(t, RoleCanAssign(model.TeamRoleMember, model.TeamRoleAdmin))
		assert.False(t, RoleCanAssign(model.TeamRoleMember, model.TeamRoleMember))
		assert.False(t, RoleCanAssign(model.TeamRoleMember, model.TeamRoleGuest))
	})
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"My Team", "my-team"},
		{"Test Team 123", "test-team-123"},
		{"  Spaces  ", "spaces"},
		{"Special@#$Characters", "special-characters"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := generateSlug(tt.name)
			assert.Equal(t, tt.expected, slug)
		})
	}
}

func TestDomain_GetTeamByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, _, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()

		team := &model.Team{
			ID:   teamID,
			Name: "Test Team",
		}

		teamDB.On("FindByID", ctx, teamID).Return(team, nil)

		result, err := domain.GetTeamByID(ctx, teamID)

		require.NoError(t, err)
		assert.Equal(t, team, result)
	})
}

func TestDomain_ListMyTeams(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		userID := uuid.New()

		teams := []*model.Team{
			{ID: uuid.New(), Name: "Team 1"},
			{ID: uuid.New(), Name: "Team 2"},
		}
		member := &model.TeamMember{
			Role: model.TeamRoleMember,
		}

		teamDB.On("FindByUser", ctx, userID, 20, 0).Return(teams, nil)
		memberDB.On("Count", ctx, teams[0].ID).Return(3, nil)
		memberDB.On("Count", ctx, teams[1].ID).Return(5, nil)
		memberDB.On("Find", ctx, teams[0].ID, userID).Return(member, nil)
		memberDB.On("Find", ctx, teams[1].ID, userID).Return(member, nil)

		result, err := domain.ListMyTeams(ctx, userID, 20, 0)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		teamDB.AssertExpectations(t)
	})

	t.Run("limit_capped", func(t *testing.T) {
		domain, teamDB, _, _, _, _ := setupDomain()
		ctx := context.Background()
		userID := uuid.New()

		teamDB.On("FindByUser", ctx, userID, 100, 0).Return([]*model.Team{}, nil)

		_, err := domain.ListMyTeams(ctx, userID, 200, 0) // Should be capped at 100

		require.NoError(t, err)
		teamDB.AssertExpectations(t)
	})
}

func TestDomain_UpdateTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, teamDB, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}
		team := &model.Team{
			ID:          teamID,
			Name:        "Old Name",
			Description: "Old Description",
		}

		newName := "New Name"
		newDesc := "New Description"

		memberDB.On("Find", ctx, teamID, requesterID).Return(member, nil)
		teamDB.On("FindByID", ctx, teamID).Return(team, nil)
		teamDB.On("Update", ctx, mock.AnythingOfType("*model.Team")).Return(nil)
		memberDB.On("Count", ctx, teamID).Return(3, nil)

		input := &inbound.UpdateTeamInput{
			Name:        &newName,
			Description: &newDesc,
		}

		output, err := domain.UpdateTeam(ctx, teamID, requesterID, input)

		require.NoError(t, err)
		assert.Equal(t, newName, output.Name)
	})

	t.Run("insufficient_permission", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleMember, // Member cannot update team
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(member, nil)

		input := &inbound.UpdateTeamInput{}
		output, err := domain.UpdateTeam(ctx, teamID, requesterID, input)

		assert.ErrorIs(t, err, ErrInsufficientPermission)
		assert.Nil(t, output)
	})
}

func TestDomain_ListMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleMember,
		}
		members := []*model.TeamMemberWithUser{
			{
				TeamMember: model.TeamMember{
					TeamID: teamID,
					UserID: uuid.New(),
					Role:   model.TeamRoleOwner,
				},
				Email: "owner@example.com",
				Name:  "Owner",
			},
			{
				TeamMember: model.TeamMember{
					TeamID: teamID,
					UserID: requesterID,
					Role:   model.TeamRoleMember,
				},
				Email: "member@example.com",
				Name:  "Member",
			},
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		memberDB.On("FindByTeamWithUsers", ctx, teamID).Return(members, nil)

		result, err := domain.ListMembers(ctx, teamID, requesterID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("not_member", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		memberDB.On("Find", ctx, teamID, requesterID).Return(nil, ErrMemberNotFound)

		result, err := domain.ListMembers(ctx, teamID, requesterID)

		assert.ErrorIs(t, err, ErrInsufficientPermission)
		assert.Nil(t, result)
	})
}

func TestDomain_LeaveTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		userID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   model.TeamRoleMember,
		}

		memberDB.On("Find", ctx, teamID, userID).Return(member, nil)
		memberDB.On("Remove", ctx, teamID, userID).Return(nil)

		err := domain.LeaveTeam(ctx, teamID, userID)

		require.NoError(t, err)
	})

	t.Run("owner_cannot_leave", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		ownerID := uuid.New()

		owner := &model.TeamMember{
			TeamID: teamID,
			UserID: ownerID,
			Role:   model.TeamRoleOwner,
		}

		memberDB.On("Find", ctx, teamID, ownerID).Return(owner, nil)

		err := domain.LeaveTeam(ctx, teamID, ownerID)

		assert.ErrorIs(t, err, ErrCannotRemoveOwner)
	})
}

func TestDomain_GetMemberCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()

		memberDB.On("Count", ctx, teamID).Return(5, nil)

		count, err := domain.GetMemberCount(ctx, teamID)

		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})
}

func TestDomain_GetMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		userID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   model.TeamRoleMember,
		}

		memberDB.On("Find", ctx, teamID, userID).Return(member, nil)

		result, err := domain.GetMember(ctx, teamID, userID)

		require.NoError(t, err)
		assert.Equal(t, member, result)
	})
}

func TestDomain_ListTeamInvitations(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}
		invitations := []*model.TeamInvitation{
			{
				ID:           uuid.New(),
				TeamID:       teamID,
				InviteeEmail: "test@example.com",
				Status:       model.InvitationStatusPending,
			},
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		invitationDB.On("FindByTeam", ctx, teamID, (*model.InvitationStatus)(nil), 20, 0).Return(invitations, nil)

		result, err := domain.ListTeamInvitations(ctx, teamID, requesterID, nil, 20, 0)

		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("insufficient_permission", func(t *testing.T) {
		domain, _, memberDB, _, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		requesterID := uuid.New()

		member := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleMember, // Member cannot list invitations
		}

		memberDB.On("Find", ctx, teamID, requesterID).Return(member, nil)

		result, err := domain.ListTeamInvitations(ctx, teamID, requesterID, nil, 20, 0)

		assert.ErrorIs(t, err, ErrInsufficientPermission)
		assert.Nil(t, result)
	})
}

func TestDomain_ListMyInvitations(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		email := "user@example.com"

		team := &model.Team{
			ID:   uuid.New(),
			Name: "Test Team",
		}
		inviter := &model.User{
			ID:   uuid.New(),
			Name: "Inviter",
		}
		invitations := []*model.TeamInvitation{
			{
				ID:           uuid.New(),
				TeamID:       team.ID,
				InviteeEmail: email,
				Status:       model.InvitationStatusPending,
				Team:         team,
				Inviter:      inviter,
			},
		}

		status := model.InvitationStatusPending
		invitationDB.On("FindByEmail", ctx, email, &status, 20, 0).Return(invitations, nil)

		result, err := domain.ListMyInvitations(ctx, email, 20, 0)

		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestDomain_RejectInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		userEmail := "user@example.com"

		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			TeamID:       uuid.New(),
			InviteeEmail: userEmail,
			Status:       model.InvitationStatusPending,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)
		invitationDB.On("UpdateStatus", ctx, invitation.ID, model.InvitationStatusRejected).Return(nil)

		err := domain.RejectInvitation(ctx, "test-token", userEmail)

		require.NoError(t, err)
	})

	t.Run("not_for_you", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()

		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			InviteeEmail: "other@example.com",
			Status:       model.InvitationStatusPending,
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)

		err := domain.RejectInvitation(ctx, "test-token", "user@example.com")

		assert.ErrorIs(t, err, ErrInvitationNotForYou)
	})

	t.Run("already_processed", func(t *testing.T) {
		domain, _, _, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		userEmail := "user@example.com"

		invitation := &model.TeamInvitation{
			ID:           uuid.New(),
			InviteeEmail: userEmail,
			Status:       model.InvitationStatusAccepted, // Already accepted
		}

		invitationDB.On("FindByToken", ctx, "test-token").Return(invitation, nil)

		err := domain.RejectInvitation(ctx, "test-token", userEmail)

		assert.ErrorIs(t, err, ErrInvitationAlreadyProcessed)
	})
}

func TestDomain_RevokeInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domain, _, memberDB, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		invitationID := uuid.New()
		requesterID := uuid.New()

		invitation := &model.TeamInvitation{
			ID:     invitationID,
			TeamID: teamID,
			Status: model.InvitationStatusPending,
		}
		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}

		invitationDB.On("FindByID", ctx, invitationID).Return(invitation, nil)
		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)
		invitationDB.On("UpdateStatus", ctx, invitationID, model.InvitationStatusRevoked).Return(nil)

		err := domain.RevokeInvitation(ctx, invitationID, requesterID)

		require.NoError(t, err)
	})

	t.Run("cannot_revoke_processed", func(t *testing.T) {
		domain, _, memberDB, invitationDB, _, _ := setupDomain()
		ctx := context.Background()
		teamID := uuid.New()
		invitationID := uuid.New()
		requesterID := uuid.New()

		invitation := &model.TeamInvitation{
			ID:     invitationID,
			TeamID: teamID,
			Status: model.InvitationStatusAccepted, // Already accepted
		}
		requester := &model.TeamMember{
			TeamID: teamID,
			UserID: requesterID,
			Role:   model.TeamRoleAdmin,
		}

		invitationDB.On("FindByID", ctx, invitationID).Return(invitation, nil)
		memberDB.On("Find", ctx, teamID, requesterID).Return(requester, nil)

		err := domain.RevokeInvitation(ctx, invitationID, requesterID)

		assert.ErrorIs(t, err, ErrCannotRevokeProcessed)
	})
}

func TestRoleLevel(t *testing.T) {
	assert.Equal(t, 100, RoleLevel(model.TeamRoleOwner))
	assert.Equal(t, 75, RoleLevel(model.TeamRoleAdmin))
	assert.Equal(t, 50, RoleLevel(model.TeamRoleMember))
	assert.Equal(t, 25, RoleLevel(model.TeamRoleGuest))
	assert.Equal(t, 0, RoleLevel("unknown"))
}

func TestRoleIsAtLeast(t *testing.T) {
	assert.True(t, RoleIsAtLeast(model.TeamRoleOwner, model.TeamRoleAdmin))
	assert.True(t, RoleIsAtLeast(model.TeamRoleOwner, model.TeamRoleOwner))
	assert.True(t, RoleIsAtLeast(model.TeamRoleAdmin, model.TeamRoleMember))
	assert.False(t, RoleIsAtLeast(model.TeamRoleMember, model.TeamRoleAdmin))
	assert.False(t, RoleIsAtLeast(model.TeamRoleGuest, model.TeamRoleMember))
}
