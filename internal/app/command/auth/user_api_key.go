package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
)

// CryptoManager defines the interface for encryption/decryption.
type CryptoManager interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// CreateUserAPIKeyCommand represents a command to create a user API key.
type CreateUserAPIKeyCommand struct {
	UserID   uuid.UUID
	Provider string
	Name     string
	APIKey   string
	Scopes   []string
}

// CreateUserAPIKeyResult is the result of creating a user API key.
type CreateUserAPIKeyResult struct {
	Key *auth.UserAPIKey
}

// CreateUserAPIKeyHandler handles CreateUserAPIKeyCommand.
type CreateUserAPIKeyHandler struct {
	repo   auth.UserAPIKeyRepository
	crypto CryptoManager
}

// NewCreateUserAPIKeyHandler creates a new handler.
func NewCreateUserAPIKeyHandler(
	repo auth.UserAPIKeyRepository,
	crypto CryptoManager,
) *CreateUserAPIKeyHandler {
	return &CreateUserAPIKeyHandler{
		repo:   repo,
		crypto: crypto,
	}
}

// Handle executes the command.
func (h *CreateUserAPIKeyHandler) Handle(ctx context.Context, cmd CreateUserAPIKeyCommand) (*CreateUserAPIKeyResult, error) {
	// Check if key already exists for this provider
	existing, err := h.repo.GetByUserAndProvider(ctx, cmd.UserID, cmd.Provider)
	if err == nil && existing != nil {
		return nil, auth.ErrAPIKeyAlreadyExists
	}

	// Encrypt the API key
	encryptedKey, err := h.crypto.Encrypt(cmd.APIKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrEncryptionFailed, err)
	}

	// Create the key record
	keyPrefix := auth.GetKeyPrefix(cmd.APIKey, 7)
	key := auth.NewUserAPIKey(cmd.UserID, cmd.Provider, cmd.Name, encryptedKey, keyPrefix, cmd.Scopes)

	if err := h.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &CreateUserAPIKeyResult{Key: key}, nil
}

// DeleteUserAPIKeyCommand represents a command to delete a user API key.
type DeleteUserAPIKeyCommand struct {
	UserID uuid.UUID
	KeyID  uuid.UUID
}

// DeleteUserAPIKeyResult is the result of deleting a user API key.
type DeleteUserAPIKeyResult struct{}

// DeleteUserAPIKeyHandler handles DeleteUserAPIKeyCommand.
type DeleteUserAPIKeyHandler struct {
	repo auth.UserAPIKeyRepository
}

// NewDeleteUserAPIKeyHandler creates a new handler.
func NewDeleteUserAPIKeyHandler(repo auth.UserAPIKeyRepository) *DeleteUserAPIKeyHandler {
	return &DeleteUserAPIKeyHandler{repo: repo}
}

// Handle executes the command.
func (h *DeleteUserAPIKeyHandler) Handle(ctx context.Context, cmd DeleteUserAPIKeyCommand) (*DeleteUserAPIKeyResult, error) {
	// Verify ownership
	key, err := h.repo.GetByID(ctx, cmd.KeyID)
	if err != nil {
		return nil, err
	}
	if !key.BelongsTo(cmd.UserID) {
		return nil, auth.ErrForbidden
	}

	if err := h.repo.Delete(ctx, cmd.KeyID); err != nil {
		return nil, err
	}

	return &DeleteUserAPIKeyResult{}, nil
}

// RotateUserAPIKeyCommand represents a command to rotate a user API key.
type RotateUserAPIKeyCommand struct {
	UserID    uuid.UUID
	KeyID     uuid.UUID
	NewAPIKey string
}

// RotateUserAPIKeyResult is the result of rotating a user API key.
type RotateUserAPIKeyResult struct {
	Key *auth.UserAPIKey
}

// RotateUserAPIKeyHandler handles RotateUserAPIKeyCommand.
type RotateUserAPIKeyHandler struct {
	repo   auth.UserAPIKeyRepository
	crypto CryptoManager
}

// NewRotateUserAPIKeyHandler creates a new handler.
func NewRotateUserAPIKeyHandler(
	repo auth.UserAPIKeyRepository,
	crypto CryptoManager,
) *RotateUserAPIKeyHandler {
	return &RotateUserAPIKeyHandler{
		repo:   repo,
		crypto: crypto,
	}
}

// Handle executes the command.
func (h *RotateUserAPIKeyHandler) Handle(ctx context.Context, cmd RotateUserAPIKeyCommand) (*RotateUserAPIKeyResult, error) {
	// Get existing key
	key, err := h.repo.GetByID(ctx, cmd.KeyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !key.BelongsTo(cmd.UserID) {
		return nil, auth.ErrForbidden
	}

	// Encrypt new key
	encryptedKey, err := h.crypto.Encrypt(cmd.NewAPIKey)
	if err != nil {
		return nil, auth.ErrEncryptionFailed
	}

	// Update
	keyPrefix := auth.GetKeyPrefix(cmd.NewAPIKey, 7)
	key.UpdateKey(encryptedKey, keyPrefix)

	if err := h.repo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	return &RotateUserAPIKeyResult{Key: key}, nil
}

// GetDecryptedAPIKeyCommand represents a command to get a decrypted API key.
type GetDecryptedAPIKeyCommand struct {
	UserID   uuid.UUID
	Provider string
}

// GetDecryptedAPIKeyResult is the result of getting a decrypted API key.
type GetDecryptedAPIKeyResult struct {
	APIKey string
}

// GetDecryptedAPIKeyHandler handles GetDecryptedAPIKeyCommand.
type GetDecryptedAPIKeyHandler struct {
	repo   auth.UserAPIKeyRepository
	crypto CryptoManager
}

// NewGetDecryptedAPIKeyHandler creates a new handler.
func NewGetDecryptedAPIKeyHandler(
	repo auth.UserAPIKeyRepository,
	crypto CryptoManager,
) *GetDecryptedAPIKeyHandler {
	return &GetDecryptedAPIKeyHandler{
		repo:   repo,
		crypto: crypto,
	}
}

// Handle executes the command.
func (h *GetDecryptedAPIKeyHandler) Handle(ctx context.Context, cmd GetDecryptedAPIKeyCommand) (*GetDecryptedAPIKeyResult, error) {
	key, err := h.repo.GetByUserAndProvider(ctx, cmd.UserID, cmd.Provider)
	if err != nil {
		return nil, err
	}

	// Update last used (ignore error)
	_ = h.repo.UpdateLastUsed(ctx, key.ID())

	// Decrypt
	decrypted, err := h.crypto.Decrypt(key.EncryptedKey())
	if err != nil {
		return nil, auth.ErrDecryptionFailed
	}

	return &GetDecryptedAPIKeyResult{APIKey: decrypted}, nil
}
