package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/infra/persistence/entity"
)

// CollaborationUnitOfWork implements collaboration.UnitOfWork.
type CollaborationUnitOfWork struct {
	db *gorm.DB
}

// NewCollaborationUnitOfWork creates a new unit of work.
func NewCollaborationUnitOfWork(db *gorm.DB) *CollaborationUnitOfWork {
	return &CollaborationUnitOfWork{db: db}
}

// Begin starts a new transaction.
func (u *CollaborationUnitOfWork) Begin(ctx context.Context) (collaboration.Transaction, error) {
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &collaborationTransaction{tx: tx}, nil
}

// collaborationTransaction implements collaboration.Transaction.
type collaborationTransaction struct {
	tx *gorm.DB
}

func (t *collaborationTransaction) Commit() error {
	return t.tx.Commit().Error
}

func (t *collaborationTransaction) Rollback() error {
	return t.tx.Rollback().Error
}

func (t *collaborationTransaction) TeamRepository() collaboration.TeamRepository {
	return &TeamRepository{db: t.tx}
}

func (t *collaborationTransaction) MemberRepository() collaboration.MemberRepository {
	return &MemberRepository{db: t.tx}
}

func (t *collaborationTransaction) InvitationRepository() collaboration.InvitationRepository {
	return &InvitationRepository{db: t.tx}
}

// TeamRepository implements collaboration.TeamRepository.
type TeamRepository struct {
	db *gorm.DB
}

// NewTeamRepository creates a new team repository.
func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

var _ collaboration.TeamRepository = (*TeamRepository)(nil)

func (r *TeamRepository) Create(ctx context.Context, team *collaboration.Team) error {
	e := entity.FromDomainTeam(team)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create team: %w", err)
	}
	return nil
}

func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*collaboration.Team, error) {
	var e entity.TeamEntity
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, collaboration.TeamStatusActive).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collaboration.ErrTeamNotFound
		}
		return nil, fmt.Errorf("get team by ID: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *TeamRepository) GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*collaboration.Team, error) {
	var e entity.TeamEntity
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ? AND status = ?", ownerID, slug, collaboration.TeamStatusActive).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collaboration.ErrTeamNotFound
		}
		return nil, fmt.Errorf("get team by owner and slug: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *TeamRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*collaboration.Team, error) {
	if limit <= 0 {
		limit = 20
	}

	var entities []entity.TeamEntity
	err := r.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ? AND teams.status = ?", userID, collaboration.TeamStatusActive).
		Order("teams.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list teams by user: %w", err)
	}

	teams := make([]*collaboration.Team, len(entities))
	for i, e := range entities {
		teams[i] = e.ToDomain()
	}
	return teams, nil
}

func (r *TeamRepository) Update(ctx context.Context, team *collaboration.Team) error {
	e := entity.FromDomainTeam(team)
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return fmt.Errorf("update team: %w", err)
	}
	return nil
}

func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&entity.TeamEntity{}).
		Where("id = ?", id).
		Update("status", collaboration.TeamStatusDeleted)
	if result.Error != nil {
		return fmt.Errorf("delete team: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return collaboration.ErrTeamNotFound
	}
	return nil
}

// MemberRepository implements collaboration.MemberRepository.
type MemberRepository struct {
	db *gorm.DB
}

// NewMemberRepository creates a new member repository.
func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

var _ collaboration.MemberRepository = (*MemberRepository)(nil)

func (r *MemberRepository) Add(ctx context.Context, member *collaboration.TeamMember) error {
	e := entity.FromDomainTeamMember(member)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *MemberRepository) Get(ctx context.Context, teamID, userID uuid.UUID) (*collaboration.TeamMember, error) {
	var e entity.TeamMemberEntity
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collaboration.ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *MemberRepository) List(ctx context.Context, teamID uuid.UUID) ([]*collaboration.TeamMember, error) {
	var entities []entity.TeamMemberEntity
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("joined_at ASC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	members := make([]*collaboration.TeamMember, len(entities))
	for i, e := range entities {
		members[i] = e.ToDomain()
	}
	return members, nil
}

func (r *MemberRepository) ListWithUsers(ctx context.Context, teamID uuid.UUID) ([]*collaboration.MemberWithUser, error) {
	var rows []entity.MemberWithUserRow
	err := r.db.WithContext(ctx).
		Table("team_members").
		Select("team_members.*, users.email, users.name").
		Joins("JOIN users ON users.id = team_members.user_id").
		Where("team_members.team_id = ?", teamID).
		Order("team_members.joined_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list members with users: %w", err)
	}

	members := make([]*collaboration.MemberWithUser, len(rows))
	for i, row := range rows {
		members[i] = row.ToDomain()
	}
	return members, nil
}

func (r *MemberRepository) UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role collaboration.Role) error {
	result := r.db.WithContext(ctx).
		Model(&entity.TeamMemberEntity{}).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Updates(map[string]interface{}{"role": role.String(), "updated_at": time.Now()})
	if result.Error != nil {
		return fmt.Errorf("update role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return collaboration.ErrMemberNotFound
	}
	return nil
}

func (r *MemberRepository) Remove(ctx context.Context, teamID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&entity.TeamMemberEntity{})
	if result.Error != nil {
		return fmt.Errorf("remove member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return collaboration.ErrMemberNotFound
	}
	return nil
}

func (r *MemberRepository) Count(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.TeamMemberEntity{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}
	return int(count), nil
}

// InvitationRepository implements collaboration.InvitationRepository.
type InvitationRepository struct {
	db *gorm.DB
}

// NewInvitationRepository creates a new invitation repository.
func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

var _ collaboration.InvitationRepository = (*InvitationRepository)(nil)

func (r *InvitationRepository) Create(ctx context.Context, invitation *collaboration.TeamInvitation) error {
	e := entity.FromDomainTeamInvitation(invitation)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

func (r *InvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (*collaboration.TeamInvitation, error) {
	var e entity.TeamInvitationEntity
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collaboration.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by ID: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *InvitationRepository) GetByToken(ctx context.Context, token string) (*collaboration.TeamInvitation, error) {
	var e entity.TeamInvitationEntity
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collaboration.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("get invitation by token: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *InvitationRepository) GetPendingByEmail(ctx context.Context, teamID uuid.UUID, email string) (*collaboration.TeamInvitation, error) {
	var e entity.TeamInvitationEntity
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND invitee_email = ? AND status = ?", teamID, email, collaboration.InvitationStatusPending).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("get pending invitation: %w", err)
	}
	return e.ToDomain(), nil
}

func (r *InvitationRepository) ListByTeam(ctx context.Context, teamID uuid.UUID, status *collaboration.InvitationStatus, limit, offset int) ([]*collaboration.TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := r.db.WithContext(ctx).Where("team_id = ?", teamID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var entities []entity.TeamInvitationEntity
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list invitations by team: %w", err)
	}

	invitations := make([]*collaboration.TeamInvitation, len(entities))
	for i, e := range entities {
		invitations[i] = e.ToDomain()
	}
	return invitations, nil
}

func (r *InvitationRepository) ListByEmail(ctx context.Context, email string, status *collaboration.InvitationStatus, limit, offset int) ([]*collaboration.TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := r.db.WithContext(ctx).Where("invitee_email = ?", email)
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var entities []entity.TeamInvitationEntity
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list invitations by email: %w", err)
	}

	invitations := make([]*collaboration.TeamInvitation, len(entities))
	for i, e := range entities {
		invitations[i] = e.ToDomain()
	}
	return invitations, nil
}

func (r *InvitationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status collaboration.InvitationStatus) error {
	updates := map[string]interface{}{"status": status.String()}
	if status == collaboration.InvitationStatusAccepted {
		now := time.Now()
		updates["accepted_at"] = &now
	}

	result := r.db.WithContext(ctx).
		Model(&entity.TeamInvitationEntity{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return collaboration.ErrInvitationNotFound
	}
	return nil
}

func (r *InvitationRepository) CancelPending(ctx context.Context, teamID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&entity.TeamInvitationEntity{}).
		Where("team_id = ? AND status = ?", teamID, collaboration.InvitationStatusPending).
		Update("status", collaboration.InvitationStatusRevoked).Error
	if err != nil {
		return fmt.Errorf("cancel pending invitations: %w", err)
	}
	return nil
}
