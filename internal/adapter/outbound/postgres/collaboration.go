package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// Collaboration domain errors (for mapping)
var (
	errTeamNotFound       = errors.New("team not found")
	errMemberNotFound     = errors.New("member not found")
	errInvitationNotFound = errors.New("invitation not found")
)

// ========== Team Adapter ==========

// TeamAdapter implements TeamDatabasePort.
type TeamAdapter struct {
	db *gorm.DB
}

// NewTeamAdapter creates a new team adapter.
func NewTeamAdapter(db *gorm.DB) *TeamAdapter {
	return &TeamAdapter{db: db}
}

func (a *TeamAdapter) Create(ctx context.Context, team *model.Team) error {
	return a.db.WithContext(ctx).Create(team).Error
}

func (a *TeamAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	var team model.Team
	err := a.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, model.TeamStatusActive).
		First(&team).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (a *TeamAdapter) FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.Team, error) {
	var team model.Team
	err := a.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ? AND status = ?", ownerID, slug, model.TeamStatusActive).
		First(&team).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (a *TeamAdapter) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Team, error) {
	if limit <= 0 {
		limit = 20
	}

	var teams []*model.Team
	err := a.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ? AND teams.status = ?", userID, model.TeamStatusActive).
		Order("teams.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&teams).Error
	if err != nil {
		return nil, err
	}
	return teams, nil
}

func (a *TeamAdapter) Update(ctx context.Context, team *model.Team) error {
	return a.db.WithContext(ctx).Save(team).Error
}

func (a *TeamAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).
		Model(&model.Team{}).
		Where("id = ?", id).
		Update("status", model.TeamStatusDeleted).Error
}

// ========== Member Adapter ==========

// TeamMemberAdapter implements TeamMemberDatabasePort.
type TeamMemberAdapter struct {
	db *gorm.DB
}

// NewTeamMemberAdapter creates a new team member adapter.
func NewTeamMemberAdapter(db *gorm.DB) *TeamMemberAdapter {
	return &TeamMemberAdapter{db: db}
}

func (a *TeamMemberAdapter) Add(ctx context.Context, member *model.TeamMember) error {
	return a.db.WithContext(ctx).Create(member).Error
}

func (a *TeamMemberAdapter) Find(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	var member model.TeamMember
	err := a.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (a *TeamMemberAdapter) FindByTeam(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMember, error) {
	var members []*model.TeamMember
	err := a.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (a *TeamMemberAdapter) FindByTeamWithUsers(ctx context.Context, teamID uuid.UUID) ([]*model.TeamMemberWithUser, error) {
	var results []*model.TeamMemberWithUser
	err := a.db.WithContext(ctx).
		Table("team_members").
		Select("team_members.*, users.email, users.name").
		Joins("JOIN users ON users.id = team_members.user_id").
		Where("team_members.team_id = ?", teamID).
		Order("team_members.joined_at ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (a *TeamMemberAdapter) UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role model.TeamRole) error {
	result := a.db.WithContext(ctx).
		Model(&model.TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Updates(map[string]interface{}{"role": role, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errMemberNotFound
	}
	return nil
}

func (a *TeamMemberAdapter) Remove(ctx context.Context, teamID, userID uuid.UUID) error {
	result := a.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&model.TeamMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errMemberNotFound
	}
	return nil
}

func (a *TeamMemberAdapter) Count(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Model(&model.TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ========== Invitation Adapter ==========

// TeamInvitationAdapter implements TeamInvitationDatabasePort.
type TeamInvitationAdapter struct {
	db *gorm.DB
}

// NewTeamInvitationAdapter creates a new team invitation adapter.
func NewTeamInvitationAdapter(db *gorm.DB) *TeamInvitationAdapter {
	return &TeamInvitationAdapter{db: db}
}

func (a *TeamInvitationAdapter) Create(ctx context.Context, invitation *model.TeamInvitation) error {
	return a.db.WithContext(ctx).Create(invitation).Error
}

func (a *TeamInvitationAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.TeamInvitation, error) {
	var invitation model.TeamInvitation
	err := a.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("id = ?", id).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

func (a *TeamInvitationAdapter) FindByToken(ctx context.Context, token string) (*model.TeamInvitation, error) {
	var invitation model.TeamInvitation
	err := a.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("token = ?", token).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

func (a *TeamInvitationAdapter) FindPendingByEmail(ctx context.Context, teamID uuid.UUID, email string) (*model.TeamInvitation, error) {
	var invitation model.TeamInvitation
	err := a.db.WithContext(ctx).
		Where("team_id = ? AND invitee_email = ? AND status = ?", teamID, email, model.InvitationStatusPending).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error here
		}
		return nil, err
	}
	return &invitation, nil
}

func (a *TeamInvitationAdapter) FindByTeam(ctx context.Context, teamID uuid.UUID, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := a.db.WithContext(ctx).
		Preload("Inviter").
		Where("team_id = ?", teamID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var invitations []*model.TeamInvitation
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&invitations).Error
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

func (a *TeamInvitationAdapter) FindByEmail(ctx context.Context, email string, status *model.InvitationStatus, limit, offset int) ([]*model.TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := a.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("invitee_email = ?", email)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var invitations []*model.TeamInvitation
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&invitations).Error
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

func (a *TeamInvitationAdapter) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvitationStatus) error {
	updates := map[string]interface{}{"status": status}
	if status == model.InvitationStatusAccepted {
		updates["accepted_at"] = gorm.Expr("NOW()")
	}

	result := a.db.WithContext(ctx).
		Model(&model.TeamInvitation{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errInvitationNotFound
	}
	return nil
}

func (a *TeamInvitationAdapter) CancelPendingByTeam(ctx context.Context, teamID uuid.UUID) error {
	return a.db.WithContext(ctx).
		Model(&model.TeamInvitation{}).
		Where("team_id = ? AND status = ?", teamID, model.InvitationStatusPending).
		Update("status", model.InvitationStatusRevoked).Error
}

// ========== User Lookup Adapter ==========

// CollaborationUserLookupAdapter implements CollaborationUserLookupPort.
type CollaborationUserLookupAdapter struct {
	db *gorm.DB
}

// NewCollaborationUserLookupAdapter creates a new user lookup adapter.
func NewCollaborationUserLookupAdapter(db *gorm.DB) *CollaborationUserLookupAdapter {
	return &CollaborationUserLookupAdapter{db: db}
}

func (a *CollaborationUserLookupAdapter) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := a.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &user, nil
}

func (a *CollaborationUserLookupAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := a.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ========== Transaction Adapter ==========

// CollaborationTransactionAdapter implements CollaborationTransactionPort.
type CollaborationTransactionAdapter struct {
	db *gorm.DB
}

// NewCollaborationTransactionAdapter creates a new transaction adapter.
func NewCollaborationTransactionAdapter(db *gorm.DB) *CollaborationTransactionAdapter {
	return &CollaborationTransactionAdapter{db: db}
}

func (a *CollaborationTransactionAdapter) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Store tx in context for nested operations
		txCtx := context.WithValue(ctx, txContextKey, tx)
		return fn(txCtx)
	})
}

// txContextKey is used to store transaction in context.
type txContextKeyType struct{}

var txContextKey = txContextKeyType{}

// Compile-time interface checks
var (
	_ outbound.TeamDatabasePort             = (*TeamAdapter)(nil)
	_ outbound.TeamMemberDatabasePort       = (*TeamMemberAdapter)(nil)
	_ outbound.TeamInvitationDatabasePort   = (*TeamInvitationAdapter)(nil)
	_ outbound.CollaborationUserLookupPort  = (*CollaborationUserLookupAdapter)(nil)
	_ outbound.CollaborationTransactionPort = (*CollaborationTransactionAdapter)(nil)
)
