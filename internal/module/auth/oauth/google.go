package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoAPI = "https://www.googleapis.com/oauth2/v2/userinfo"

// GoogleProvider implements OAuth for Google.
type GoogleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider creates a new Google OAuth provider.
func NewGoogleProvider(cfg *Config) *GoogleProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	}

	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
	}
}

// Name returns the provider name.
func (p *GoogleProvider) Name() string {
	return "google"
}

// GetAuthURL returns the OAuth authorization URL.
func (p *GoogleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for tokens.
func (p *GoogleProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return token, nil
}

// GetUserInfo fetches user information from Google.
func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get(googleUserInfoAPI)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api error: %s", resp.Status)
	}

	var user struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	if !user.VerifiedEmail {
		return nil, fmt.Errorf("email not verified")
	}

	return &UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.Picture,
	}, nil
}
