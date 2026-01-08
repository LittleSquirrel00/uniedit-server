package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
)

// ListUserAPIKeysQuery represents a query to list user API keys.
type ListUserAPIKeysQuery struct {
	UserID uuid.UUID
}

// ListUserAPIKeysResult is the result of listing user API keys.
type ListUserAPIKeysResult struct {
	Keys []*auth.UserAPIKey
}

// ListUserAPIKeysHandler handles ListUserAPIKeysQuery.
type ListUserAPIKeysHandler struct {
	repo auth.UserAPIKeyRepository
}

// NewListUserAPIKeysHandler creates a new handler.
func NewListUserAPIKeysHandler(repo auth.UserAPIKeyRepository) *ListUserAPIKeysHandler {
	return &ListUserAPIKeysHandler{repo: repo}
}

// Handle executes the query.
func (h *ListUserAPIKeysHandler) Handle(ctx context.Context, query ListUserAPIKeysQuery) (*ListUserAPIKeysResult, error) {
	keys, err := h.repo.ListByUser(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	return &ListUserAPIKeysResult{Keys: keys}, nil
}

// ListSystemAPIKeysQuery represents a query to list system API keys.
type ListSystemAPIKeysQuery struct {
	UserID uuid.UUID
}

// ListSystemAPIKeysResult is the result of listing system API keys.
type ListSystemAPIKeysResult struct {
	Keys []*auth.SystemAPIKey
}

// ListSystemAPIKeysHandler handles ListSystemAPIKeysQuery.
type ListSystemAPIKeysHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewListSystemAPIKeysHandler creates a new handler.
func NewListSystemAPIKeysHandler(repo auth.SystemAPIKeyRepository) *ListSystemAPIKeysHandler {
	return &ListSystemAPIKeysHandler{repo: repo}
}

// Handle executes the query.
func (h *ListSystemAPIKeysHandler) Handle(ctx context.Context, query ListSystemAPIKeysQuery) (*ListSystemAPIKeysResult, error) {
	keys, err := h.repo.ListByUser(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	return &ListSystemAPIKeysResult{Keys: keys}, nil
}

// GetSystemAPIKeyQuery represents a query to get a system API key.
type GetSystemAPIKeyQuery struct {
	UserID uuid.UUID
	KeyID  uuid.UUID
}

// GetSystemAPIKeyResult is the result of getting a system API key.
type GetSystemAPIKeyResult struct {
	Key *auth.SystemAPIKey
}

// GetSystemAPIKeyHandler handles GetSystemAPIKeyQuery.
type GetSystemAPIKeyHandler struct {
	repo auth.SystemAPIKeyRepository
}

// NewGetSystemAPIKeyHandler creates a new handler.
func NewGetSystemAPIKeyHandler(repo auth.SystemAPIKeyRepository) *GetSystemAPIKeyHandler {
	return &GetSystemAPIKeyHandler{repo: repo}
}

// Handle executes the query.
func (h *GetSystemAPIKeyHandler) Handle(ctx context.Context, query GetSystemAPIKeyQuery) (*GetSystemAPIKeyResult, error) {
	key, err := h.repo.GetByID(ctx, query.KeyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !key.BelongsTo(query.UserID) {
		return nil, auth.ErrForbidden
	}

	return &GetSystemAPIKeyResult{Key: key}, nil
}
