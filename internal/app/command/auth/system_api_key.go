package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
)

// CreateSystemAPIKeyCommand represents a command to create a system API key.
type CreateSystemAPIKeyCommand struct {
	UserID        uuid.UUID
	Name          string
	Scopes        []string
	RateLimitRPM  *int
	RateLimitTPM  *int
	ExpiresInDays *int
}

// CreateSystemAPIKeyResult is the result of creating a system API key.
type CreateSystemAPIKeyResult struct {
	Key      *auth.SystemAPIKey
	RawKey   string // The full key (only returned on creation)
}

// CreateSystemAPIKeyHandler handles CreateSystemAPIKeyCommand.
type CreateSystemAPIKeyHandler struct {
	repo              auth.SystemAPIKeyRepository
	maxAPIKeysPerUser int
}

// NewCreateSystemAPIKeyHandler creates a new handler.
func NewCreateSystemAPIKeyHandler(
	repo auth.SystemAPIKeyRepository,
	maxAPIKeysPerUser int,
) *CreateSystemAPIKeyHandler {
	return &CreateSystemAPIKeyHandler{
		repo:              repo,
		maxAPIKeysPerUser: maxAPIKeysPerUser,
	}
}

// Handle executes the command.
func (h *CreateSystemAPIKeyHandler) Handle(ctx context.Context, cmd CreateSystemAPIKeyCommand) (*CreateSystemAPIKeyResult, error) {
	// Check limit
	count, err := h.repo.CountByUser(ctx, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("count api keys: %w", err)
	}
	if count >= int64(h.maxAPIKeysPerUser) {
		return nil, auth.ErrSystemAPIKeyLimitExceeded
	}

	// Validate scopes
	scopes := cmd.Scopes
	if len(scopes) == 0 {
		scopes = auth.DefaultScopes()
	}
	if err := auth.ValidateScopes(scopes); err != nil {
		return nil, err
	}

	// Generate API key
	rawKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	// Set defaults
	rateLimitRPM := 60
	rateLimitTPM := 100000
	if cmd.RateLimitRPM != nil && *cmd.RateLimitRPM > 0 {
		rateLimitRPM = *cmd.RateLimitRPM
	}
	if cmd.RateLimitTPM != nil && *cmd.RateLimitTPM > 0 {
		rateLimitTPM = *cmd.RateLimitTPM
	}

	// Calculate expiration
	var expiresAt *time.Time
	if cmd.ExpiresInDays != nil && *cmd.ExpiresInDays > 0 {
		t := time.Now().AddDate(0, 0, *cmd.ExpiresInDays)
		expiresAt = &t
	}

	// Create key
	key := auth.NewSystemAPIKey(
		cmd.UserID,
		cmd.Name,
		keyHash,
		keyPrefix,
		scopes,
		rateLimitRPM,
		rateLimitTPM,
		expiresAt,
	)

	if err := h.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("create system api key: %w", err)
	}

	return &CreateSystemAPIKeyResult{
		Key:    key,
		RawKey: rawKey,
	}, nil
}

// UpdateSystemAPIKeyCommand represents a command to update a system API key.
type UpdateSystemAPIKeyCommand struct {
	UserID       uuid.UUID
	KeyID        uuid.UUID
	Name         *string
	Scopes       []string
	RateLimitRPM *int
	RateLimitTPM *int
	IsActive     *bool
}

// UpdateSystemAPIKeyResult is the result of updating a system API key.
type UpdateSystemAPIKeyResult struct {
	Key *auth.SystemAPIKey
}

// UpdateSystemAPIKeyHandler handles UpdateSystemAPIKeyCommand.
type UpdateSystemAPIKeyHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewUpdateSystemAPIKeyHandler creates a new handler.
func NewUpdateSystemAPIKeyHandler(repo auth.SystemAPIKeyRepository) *UpdateSystemAPIKeyHandler {
	return &UpdateSystemAPIKeyHandler{repo: repo}
}

// Handle executes the command.
func (h *UpdateSystemAPIKeyHandler) Handle(ctx context.Context, cmd UpdateSystemAPIKeyCommand) (*UpdateSystemAPIKeyResult, error) {
	key, err := h.repo.GetByID(ctx, cmd.KeyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !key.BelongsTo(cmd.UserID) {
		return nil, auth.ErrForbidden
	}

	// Apply updates
	if cmd.Name != nil {
		key.SetName(*cmd.Name)
	}
	if len(cmd.Scopes) > 0 {
		if err := auth.ValidateScopes(cmd.Scopes); err != nil {
			return nil, err
		}
		key.SetScopes(cmd.Scopes)
	}
	if cmd.RateLimitRPM != nil || cmd.RateLimitTPM != nil {
		rpm := key.RateLimitRPM()
		tpm := key.RateLimitTPM()
		if cmd.RateLimitRPM != nil {
			rpm = *cmd.RateLimitRPM
		}
		if cmd.RateLimitTPM != nil {
			tpm = *cmd.RateLimitTPM
		}
		key.SetRateLimits(rpm, tpm)
	}
	if cmd.IsActive != nil {
		if *cmd.IsActive {
			key.Activate()
		} else {
			key.Deactivate()
		}
	}

	if err := h.repo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return &UpdateSystemAPIKeyResult{Key: key}, nil
}

// DeleteSystemAPIKeyCommand represents a command to delete a system API key.
type DeleteSystemAPIKeyCommand struct {
	UserID uuid.UUID
	KeyID  uuid.UUID
}

// DeleteSystemAPIKeyResult is the result of deleting a system API key.
type DeleteSystemAPIKeyResult struct{}

// DeleteSystemAPIKeyHandler handles DeleteSystemAPIKeyCommand.
type DeleteSystemAPIKeyHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewDeleteSystemAPIKeyHandler creates a new handler.
func NewDeleteSystemAPIKeyHandler(repo auth.SystemAPIKeyRepository) *DeleteSystemAPIKeyHandler {
	return &DeleteSystemAPIKeyHandler{repo: repo}
}

// Handle executes the command.
func (h *DeleteSystemAPIKeyHandler) Handle(ctx context.Context, cmd DeleteSystemAPIKeyCommand) (*DeleteSystemAPIKeyResult, error) {
	key, err := h.repo.GetByID(ctx, cmd.KeyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !key.BelongsTo(cmd.UserID) {
		return nil, auth.ErrForbidden
	}

	if err := h.repo.Delete(ctx, cmd.KeyID); err != nil {
		return nil, err
	}

	return &DeleteSystemAPIKeyResult{}, nil
}

// RotateSystemAPIKeyCommand represents a command to rotate a system API key.
type RotateSystemAPIKeyCommand struct {
	UserID uuid.UUID
	KeyID  uuid.UUID
}

// RotateSystemAPIKeyResult is the result of rotating a system API key.
type RotateSystemAPIKeyResult struct {
	Key    *auth.SystemAPIKey
	RawKey string // The new full key
}

// RotateSystemAPIKeyHandler handles RotateSystemAPIKeyCommand.
type RotateSystemAPIKeyHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewRotateSystemAPIKeyHandler creates a new handler.
func NewRotateSystemAPIKeyHandler(repo auth.SystemAPIKeyRepository) *RotateSystemAPIKeyHandler {
	return &RotateSystemAPIKeyHandler{repo: repo}
}

// Handle executes the command.
func (h *RotateSystemAPIKeyHandler) Handle(ctx context.Context, cmd RotateSystemAPIKeyCommand) (*RotateSystemAPIKeyResult, error) {
	key, err := h.repo.GetByID(ctx, cmd.KeyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !key.BelongsTo(cmd.UserID) {
		return nil, auth.ErrForbidden
	}

	// Generate new API key
	newKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	// Update record
	key.RotateKey(keyHash, keyPrefix)

	if err := h.repo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return &RotateSystemAPIKeyResult{
		Key:    key,
		RawKey: newKey,
	}, nil
}

// ValidateSystemAPIKeyCommand represents a command to validate a system API key.
type ValidateSystemAPIKeyCommand struct {
	APIKey string
}

// ValidateSystemAPIKeyResult is the result of validating a system API key.
type ValidateSystemAPIKeyResult struct {
	Key *auth.SystemAPIKey
}

// ValidateSystemAPIKeyHandler handles ValidateSystemAPIKeyCommand.
type ValidateSystemAPIKeyHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewValidateSystemAPIKeyHandler creates a new handler.
func NewValidateSystemAPIKeyHandler(repo auth.SystemAPIKeyRepository) *ValidateSystemAPIKeyHandler {
	return &ValidateSystemAPIKeyHandler{repo: repo}
}

// Handle executes the command.
func (h *ValidateSystemAPIKeyHandler) Handle(ctx context.Context, cmd ValidateSystemAPIKeyCommand) (*ValidateSystemAPIKeyResult, error) {
	// Check format
	if !auth.IsValidAPIKeyFormat(cmd.APIKey) {
		return nil, auth.ErrInvalidAPIKeyFormat
	}

	// Hash and lookup
	keyHash := auth.HashAPIKey(cmd.APIKey)
	key, err := h.repo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, auth.ErrSystemAPIKeyNotFound
	}

	// Check if active
	if !key.IsActive() {
		return nil, auth.ErrSystemAPIKeyDisabled
	}

	// Check expiration
	if key.IsExpired() {
		return nil, auth.ErrSystemAPIKeyExpired
	}

	// Update last used (async, don't fail on error)
	go func() {
		_ = h.repo.UpdateLastUsed(context.Background(), key.ID())
	}()

	return &ValidateSystemAPIKeyResult{Key: key}, nil
}
