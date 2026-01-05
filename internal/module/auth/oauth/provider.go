package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// UserInfo represents user information from OAuth provider.
type UserInfo struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
}

// Provider defines the interface for OAuth providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// GetAuthURL returns the OAuth authorization URL.
	GetAuthURL(state string) string

	// Exchange exchanges the authorization code for tokens.
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)

	// GetUserInfo fetches user information using the access token.
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
}

// Config holds OAuth provider configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}
