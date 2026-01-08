package auth

// OAuthProvider represents supported OAuth providers.
type OAuthProvider string

const (
	OAuthProviderGitHub OAuthProvider = "github"
	OAuthProviderGoogle OAuthProvider = "google"
)

// String returns the string representation of the provider.
func (p OAuthProvider) String() string {
	return string(p)
}

// IsValid checks if the provider is supported.
func (p OAuthProvider) IsValid() bool {
	switch p {
	case OAuthProviderGitHub, OAuthProviderGoogle:
		return true
	default:
		return false
	}
}

// OAuthUserInfo represents user information from OAuth provider.
type OAuthUserInfo struct {
	id        string
	email     string
	name      string
	avatarURL string
	provider  OAuthProvider
}

// NewOAuthUserInfo creates a new OAuthUserInfo.
func NewOAuthUserInfo(id, email, name, avatarURL string, provider OAuthProvider) *OAuthUserInfo {
	return &OAuthUserInfo{
		id:        id,
		email:     email,
		name:      name,
		avatarURL: avatarURL,
		provider:  provider,
	}
}

// Getters

func (u *OAuthUserInfo) ID() string            { return u.id }
func (u *OAuthUserInfo) Email() string         { return u.email }
func (u *OAuthUserInfo) Name() string          { return u.name }
func (u *OAuthUserInfo) AvatarURL() string     { return u.avatarURL }
func (u *OAuthUserInfo) Provider() OAuthProvider { return u.provider }

// OAuthState represents an OAuth state token for CSRF protection.
type OAuthState struct {
	state    string
	provider OAuthProvider
}

// NewOAuthState creates a new OAuthState.
func NewOAuthState(state string, provider OAuthProvider) *OAuthState {
	return &OAuthState{
		state:    state,
		provider: provider,
	}
}

func (s *OAuthState) State() string           { return s.state }
func (s *OAuthState) Provider() OAuthProvider { return s.provider }
