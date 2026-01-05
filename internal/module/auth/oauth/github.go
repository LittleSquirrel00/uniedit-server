package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	githubUserAPI   = "https://api.github.com/user"
	githubEmailsAPI = "https://api.github.com/user/emails"
)

// GitHubProvider implements OAuth for GitHub.
type GitHubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider creates a new GitHub OAuth provider.
func NewGitHubProvider(cfg *Config) *GitHubProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}

	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     github.Endpoint,
		},
	}
}

// Name returns the provider name.
func (p *GitHubProvider) Name() string {
	return "github"
}

// GetAuthURL returns the OAuth authorization URL.
func (p *GitHubProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for tokens.
func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return token, nil
}

// GetUserInfo fetches user information from GitHub.
func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	// Get user info
	resp, err := client.Get(githubUserAPI)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %s", resp.Status)
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	// If email is not public, fetch from emails API
	email := user.Email
	if email == "" {
		email, err = p.getPrimaryEmail(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("get primary email: %w", err)
		}
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	return &UserInfo{
		ID:        fmt.Sprintf("%d", user.ID),
		Email:     email,
		Name:      name,
		AvatarURL: user.AvatarURL,
	}, nil
}

// getPrimaryEmail fetches the user's primary email from GitHub.
func (p *GitHubProvider) getPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	resp, err := client.Get(githubEmailsAPI)
	if err != nil {
		return "", fmt.Errorf("get emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails api error: %s", resp.Status)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode emails: %w", err)
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// Fallback to first verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}
