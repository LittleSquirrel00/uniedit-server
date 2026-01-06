package collaboration

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for collaboration data access.
type Repository interface {
	// Team operations
	CreateTeam(ctx context.Context, team *Team) error
	GetTeamByID(ctx context.Context, id uuid.UUID) (*Team, error)
	GetTeamByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*Team, error)
	ListTeamsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Team, error)
	UpdateTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error

	// Member operations
	AddMember(ctx context.Context, member *TeamMember) error
	GetMember(ctx context.Context, teamID, userID uuid.UUID) (*TeamMember, error)
	ListMembers(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error)
	ListMembersWithUsers(ctx context.Context, teamID uuid.UUID) ([]MemberWithUser, error)
	UpdateMemberRole(ctx context.Context, teamID, userID uuid.UUID, role Role) error
	RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
	CountMembers(ctx context.Context, teamID uuid.UUID) (int, error)

	// Invitation operations
	CreateInvitation(ctx context.Context, invitation *TeamInvitation) error
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*TeamInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*TeamInvitation, error)
	GetPendingInvitationByEmail(ctx context.Context, teamID uuid.UUID, email string) (*TeamInvitation, error)
	ListInvitationsByTeam(ctx context.Context, teamID uuid.UUID, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error)
	ListInvitationsByEmail(ctx context.Context, email string, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error)
	UpdateInvitationStatus(ctx context.Context, id uuid.UUID, status InvitationStatus) error
	CancelPendingInvitations(ctx context.Context, teamID uuid.UUID) error

	// Transaction support
	WithTx(tx *gorm.DB) Repository
	BeginTx(ctx context.Context) (*gorm.DB, error)
}

// MemberWithUser represents a team member with user details.
type MemberWithUser struct {
	TeamMember
	Email string
	Name  string
}

// repository implements Repository using GORM.
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new collaboration repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// WithTx returns a new repository with the given transaction.
func (r *repository) WithTx(tx *gorm.DB) Repository {
	return &repository{db: tx}
}

// BeginTx starts a new transaction.
func (r *repository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

// CreateTeam creates a new team.
func (r *repository) CreateTeam(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

// GetTeamByID retrieves a team by ID.
func (r *repository) GetTeamByID(ctx context.Context, id uuid.UUID) (*Team, error) {
	var team Team
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, TeamStatusActive).
		First(&team).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

// GetTeamByOwnerAndSlug retrieves a team by owner and slug.
func (r *repository) GetTeamByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*Team, error) {
	var team Team
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ? AND status = ?", ownerID, slug, TeamStatusActive).
		First(&team).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

// ListTeamsByUser lists all teams a user belongs to.
func (r *repository) ListTeamsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Team, error) {
	if limit <= 0 {
		limit = 20
	}

	var teams []*Team
	err := r.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ? AND teams.status = ?", userID, TeamStatusActive).
		Order("teams.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&teams).Error
	if err != nil {
		return nil, err
	}
	return teams, nil
}

// UpdateTeam updates a team.
func (r *repository) UpdateTeam(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Save(team).Error
}

// DeleteTeam soft-deletes a team.
func (r *repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&Team{}).
		Where("id = ?", id).
		Update("status", TeamStatusDeleted).Error
}

// AddMember adds a member to a team.
func (r *repository) AddMember(ctx context.Context, member *TeamMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// GetMember retrieves a team member.
func (r *repository) GetMember(ctx context.Context, teamID, userID uuid.UUID) (*TeamMember, error) {
	var member TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

// ListMembers lists all members of a team.
func (r *repository) ListMembers(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error) {
	var members []*TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// ListMembersWithUsers lists all members with user details.
func (r *repository) ListMembersWithUsers(ctx context.Context, teamID uuid.UUID) ([]MemberWithUser, error) {
	var results []MemberWithUser
	err := r.db.WithContext(ctx).
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

// UpdateMemberRole updates a member's role.
func (r *repository) UpdateMemberRole(ctx context.Context, teamID, userID uuid.UUID, role Role) error {
	result := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Updates(map[string]interface{}{"role": role, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// RemoveMember removes a member from a team.
func (r *repository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&TeamMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// CountMembers counts the number of members in a team.
func (r *repository) CountMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// CreateInvitation creates a new invitation.
func (r *repository) CreateInvitation(ctx context.Context, invitation *TeamInvitation) error {
	return r.db.WithContext(ctx).Create(invitation).Error
}

// GetInvitationByID retrieves an invitation by ID.
func (r *repository) GetInvitationByID(ctx context.Context, id uuid.UUID) (*TeamInvitation, error) {
	var invitation TeamInvitation
	err := r.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("id = ?", id).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// GetInvitationByToken retrieves an invitation by token.
func (r *repository) GetInvitationByToken(ctx context.Context, token string) (*TeamInvitation, error) {
	var invitation TeamInvitation
	err := r.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("token = ?", token).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// GetPendingInvitationByEmail retrieves a pending invitation for a team and email.
func (r *repository) GetPendingInvitationByEmail(ctx context.Context, teamID uuid.UUID, email string) (*TeamInvitation, error) {
	var invitation TeamInvitation
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND invitee_email = ? AND status = ?", teamID, email, InvitationStatusPending).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error here
		}
		return nil, err
	}
	return &invitation, nil
}

// ListInvitationsByTeam lists invitations for a team.
func (r *repository) ListInvitationsByTeam(ctx context.Context, teamID uuid.UUID, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := r.db.WithContext(ctx).
		Preload("Inviter").
		Where("team_id = ?", teamID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var invitations []*TeamInvitation
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

// ListInvitationsByEmail lists invitations for an email.
func (r *repository) ListInvitationsByEmail(ctx context.Context, email string, status *InvitationStatus, limit, offset int) ([]*TeamInvitation, error) {
	if limit <= 0 {
		limit = 20
	}

	query := r.db.WithContext(ctx).
		Preload("Team").
		Preload("Inviter").
		Where("invitee_email = ?", email)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var invitations []*TeamInvitation
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

// UpdateInvitationStatus updates an invitation's status.
func (r *repository) UpdateInvitationStatus(ctx context.Context, id uuid.UUID, status InvitationStatus) error {
	updates := map[string]interface{}{"status": status}
	if status == InvitationStatusAccepted {
		updates["accepted_at"] = gorm.Expr("NOW()")
	}

	result := r.db.WithContext(ctx).
		Model(&TeamInvitation{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

// CancelPendingInvitations cancels all pending invitations for a team.
func (r *repository) CancelPendingInvitations(ctx context.Context, teamID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&TeamInvitation{}).
		Where("team_id = ? AND status = ?", teamID, InvitationStatusPending).
		Update("status", InvitationStatusRevoked).Error
}

// UserRepository defines the interface for user lookup.
type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// userRepository implements UserRepository.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository for collaboration module.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// GetUserByEmail retrieves a user by email.
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
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

// GetUserByID retrieves a user by ID.
func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
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
