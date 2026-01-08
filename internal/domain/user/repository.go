package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user data access (Port).
type Repository interface {
	// User operations
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByOAuth(ctx context.Context, provider, oauthID string) (*User, error)
	Update(ctx context.Context, user *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *UserFilter, pagination *Pagination) ([]*User, int64, error)

	// Email verification operations
	CreateVerification(ctx context.Context, verification *EmailVerification) error
	GetVerificationByToken(ctx context.Context, token string) (*EmailVerification, error)
	InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose VerificationPurpose) error
	MarkVerificationUsed(ctx context.Context, id uuid.UUID) error
}

// UserFilter represents filters for listing users.
type UserFilter struct {
	Status  *UserStatus
	Email   *string
	IsAdmin *bool
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// NewPagination creates pagination with defaults.
func NewPagination(page, pageSize int) *Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset returns the offset for database queries.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
