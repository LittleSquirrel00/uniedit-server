package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// Config holds OAuth provider configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// --- GitHub Provider ---

type githubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider creates a new GitHub OAuth provider.
func NewGitHubProvider(cfg *Config) outbound.OAuthProviderPort {
	return &githubProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     github.Endpoint,
		},
	}
}

func (p *githubProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *githubProvider) Exchange(ctx context.Context, code string) (string, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (p *githubProvider) GetUserInfo(ctx context.Context, accessToken string) (*model.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	// GitHub may not return email in user endpoint
	email := data.Email
	if email == "" {
		email, _ = p.fetchPrimaryEmail(ctx, accessToken)
	}

	name := data.Name
	if name == "" {
		name = data.Login
	}

	return &model.OAuthUserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Email:     email,
		Name:      name,
		AvatarURL: data.AvatarURL,
		Provider:  model.OAuthProviderGitHub,
	}, nil
}

func (p *githubProvider) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no primary email found")
}

// Compile-time check
var _ outbound.OAuthProviderPort = (*githubProvider)(nil)

// --- Google Provider ---

type googleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider creates a new Google OAuth provider.
func NewGoogleProvider(cfg *Config) outbound.OAuthProviderPort {
	return &googleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
	}
}

func (p *googleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *googleProvider) Exchange(ctx context.Context, code string) (string, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (p *googleProvider) GetUserInfo(ctx context.Context, accessToken string) (*model.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api error: %d", resp.StatusCode)
	}

	var data struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &model.OAuthUserInfo{
		ID:        data.ID,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.Picture,
		Provider:  model.OAuthProviderGoogle,
	}, nil
}

// Compile-time check
var _ outbound.OAuthProviderPort = (*googleProvider)(nil)
