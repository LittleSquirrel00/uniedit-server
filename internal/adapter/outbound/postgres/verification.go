package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// verificationAdapter implements outbound.VerificationDatabasePort.
type verificationAdapter struct {
	db *gorm.DB
}

// NewVerificationAdapter creates a new verification database adapter.
func NewVerificationAdapter(db *gorm.DB) outbound.VerificationDatabasePort {
	return &verificationAdapter{db: db}
}

func (a *verificationAdapter) CreateVerification(ctx context.Context, verification *model.EmailVerification) error {
	return a.db.WithContext(ctx).Create(verification).Error
}

func (a *verificationAdapter) GetVerificationByToken(ctx context.Context, token string) (*model.EmailVerification, error) {
	var verification model.EmailVerification
	err := a.db.WithContext(ctx).
		Where("token = ?", token).
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrInvalidToken
		}
		return nil, err
	}
	return &verification, nil
}

func (a *verificationAdapter) InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose model.VerificationPurpose) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.EmailVerification{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL", userID, purpose).
		Update("used_at", now).Error
}

func (a *verificationAdapter) MarkVerificationUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.EmailVerification{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}

// Compile-time check
var _ outbound.VerificationDatabasePort = (*verificationAdapter)(nil)
