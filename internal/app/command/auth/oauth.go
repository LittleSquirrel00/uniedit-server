package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/uniedit/server/internal/domain/auth"
)

// InitiateOAuthCommand represents a command to start OAuth login.
type InitiateOAuthCommand struct {
	Provider auth.OAuthProvider
}

// InitiateOAuthResult is the result of initiating OAuth login.
type InitiateOAuthResult struct {
	AuthURL string
	State   string
}

// OAuthProvider defines the interface for OAuth providers.
type OAuthProvider interface {
	GetAuthURL(state string) string
	Exchange(ctx context.Context, code string) (interface{}, error)
	GetUserInfo(ctx context.Context, token interface{}) (*auth.OAuthUserInfo, error)
}

// OAuthRegistry manages OAuth providers.
type OAuthRegistry interface {
	Get(provider string) (OAuthProvider, error)
}

// InitiateOAuthHandler handles InitiateOAuthCommand.
type InitiateOAuthHandler struct {
	oauthRegistry OAuthRegistry
	stateStore    auth.OAuthStateStore
}

// NewInitiateOAuthHandler creates a new handler.
func NewInitiateOAuthHandler(
	oauthRegistry OAuthRegistry,
	stateStore auth.OAuthStateStore,
) *InitiateOAuthHandler {
	return &InitiateOAuthHandler{
		oauthRegistry: oauthRegistry,
		stateStore:    stateStore,
	}
}

// Handle executes the command.
func (h *InitiateOAuthHandler) Handle(ctx context.Context, cmd InitiateOAuthCommand) (*InitiateOAuthResult, error) {
	if !cmd.Provider.IsValid() {
		return nil, auth.ErrInvalidOAuthProvider
	}

	oauthProvider, err := h.oauthRegistry.Get(cmd.Provider.String())
	if err != nil {
		return nil, auth.ErrInvalidOAuthProvider
	}

	// Generate state token
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	// Store state for verification
	if err := h.stateStore.Set(ctx, state, cmd.Provider); err != nil {
		return nil, fmt.Errorf("store state: %w", err)
	}

	authURL := oauthProvider.GetAuthURL(state)

	return &InitiateOAuthResult{
		AuthURL: authURL,
		State:   state,
	}, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
