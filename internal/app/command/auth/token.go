package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/user"
)

// CompleteOAuthCommand represents a command to complete OAuth login.
type CompleteOAuthCommand struct {
	Provider  auth.OAuthProvider
	Code      string
	State     string
	UserAgent string
	IPAddress string
}

// CompleteOAuthResult is the result of completing OAuth login.
type CompleteOAuthResult struct {
	TokenPair *auth.TokenPair
	User      *user.User
}

// JWTGenerator defines the interface for generating tokens.
type JWTGenerator interface {
	GenerateAccessToken(userID uuid.UUID, email string) (token string, expiresAt time.Time, err error)
	GenerateRefreshToken() (rawToken string, tokenHash string, expiresAt time.Time, err error)
	GetAccessTokenExpiry() time.Duration
}

// CompleteOAuthHandler handles CompleteOAuthCommand.
type CompleteOAuthHandler struct {
	oauthRegistry    OAuthRegistry
	stateStore       auth.OAuthStateStore
	userRepo         user.Repository
	refreshTokenRepo auth.RefreshTokenRepository
	jwtGenerator     JWTGenerator
}

// NewCompleteOAuthHandler creates a new handler.
func NewCompleteOAuthHandler(
	oauthRegistry OAuthRegistry,
	stateStore auth.OAuthStateStore,
	userRepo user.Repository,
	refreshTokenRepo auth.RefreshTokenRepository,
	jwtGenerator JWTGenerator,
) *CompleteOAuthHandler {
	return &CompleteOAuthHandler{
		oauthRegistry:    oauthRegistry,
		stateStore:       stateStore,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtGenerator:     jwtGenerator,
	}
}

// Handle executes the command.
func (h *CompleteOAuthHandler) Handle(ctx context.Context, cmd CompleteOAuthCommand) (*CompleteOAuthResult, error) {
	// Verify state
	storedProvider, err := h.stateStore.Get(ctx, cmd.State)
	if err != nil {
		return nil, auth.ErrInvalidOAuthState
	}
	defer h.stateStore.Delete(ctx, cmd.State)

	if storedProvider != cmd.Provider {
		return nil, auth.ErrInvalidOAuthState
	}

	// Get OAuth provider
	oauthProvider, err := h.oauthRegistry.Get(cmd.Provider.String())
	if err != nil {
		return nil, auth.ErrInvalidOAuthProvider
	}

	// Exchange code for token
	token, err := oauthProvider.Exchange(ctx, cmd.Code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidOAuthCode, err)
	}

	// Get user info from provider
	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrOAuthFailed, err)
	}

	// Find or create user
	u, err := h.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	// Generate tokens
	tokenPair, err := h.generateTokenPair(ctx, u, cmd.UserAgent, cmd.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return &CompleteOAuthResult{
		TokenPair: tokenPair,
		User:      u,
	}, nil
}

func (h *CompleteOAuthHandler) findOrCreateUser(ctx context.Context, info *auth.OAuthUserInfo) (*user.User, error) {
	// Try to find existing user by OAuth ID
	u, err := h.userRepo.GetByOAuth(ctx, info.Provider().String(), info.ID())
	if err == nil {
		// Update user info if changed
		updated := false
		if u.Email() != info.Email() {
			// Note: email update might need special handling
			updated = true
		}
		if u.Name() != info.Name() {
			u.SetName(info.Name())
			updated = true
		}
		if u.AvatarURL() != info.AvatarURL() {
			u.SetAvatarURL(info.AvatarURL())
			updated = true
		}
		if updated {
			if err := h.userRepo.Update(ctx, u); err != nil {
				return nil, fmt.Errorf("update user: %w", err)
			}
		}
		return u, nil
	}

	if err != user.ErrUserNotFound {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Create new user
	u, err = user.NewOAuthUser(info.Email(), info.Name(), info.Provider().String(), info.ID())
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	u.SetAvatarURL(info.AvatarURL())
	u.VerifyEmail() // OAuth users are automatically verified

	if err := h.userRepo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (h *CompleteOAuthHandler) generateTokenPair(ctx context.Context, u *user.User, userAgent, ipAddress string) (*auth.TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := h.jwtGenerator.GenerateAccessToken(u.ID(), u.Email())
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	rawRefreshToken, tokenHash, refreshExpiresAt, err := h.jwtGenerator.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token
	refreshToken := auth.NewRefreshToken(u.ID(), tokenHash, refreshExpiresAt, userAgent, ipAddress)
	if err := h.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return auth.NewTokenPair(
		accessToken,
		rawRefreshToken,
		int64(h.jwtGenerator.GetAccessTokenExpiry().Seconds()),
		expiresAt,
	), nil
}

// RefreshTokensCommand represents a command to refresh tokens.
type RefreshTokensCommand struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

// RefreshTokensResult is the result of refreshing tokens.
type RefreshTokensResult struct {
	TokenPair *auth.TokenPair
}

// RefreshTokensHandler handles RefreshTokensCommand.
type RefreshTokensHandler struct {
	refreshTokenRepo auth.RefreshTokenRepository
	userRepo         user.Repository
	jwtGenerator     JWTGenerator
}

// NewRefreshTokensHandler creates a new handler.
func NewRefreshTokensHandler(
	refreshTokenRepo auth.RefreshTokenRepository,
	userRepo user.Repository,
	jwtGenerator JWTGenerator,
) *RefreshTokensHandler {
	return &RefreshTokensHandler{
		refreshTokenRepo: refreshTokenRepo,
		userRepo:         userRepo,
		jwtGenerator:     jwtGenerator,
	}
}

// Handle executes the command.
func (h *RefreshTokensHandler) Handle(ctx context.Context, cmd RefreshTokensCommand) (*RefreshTokensResult, error) {
	// Hash the token to look it up
	hash := sha256.Sum256([]byte(cmd.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Find the refresh token
	storedToken, err := h.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	// Validate token
	if !storedToken.IsValid() {
		if storedToken.IsExpired() {
			return nil, auth.ErrExpiredToken
		}
		return nil, auth.ErrRevokedToken
	}

	// Revoke old token
	if err := h.refreshTokenRepo.Revoke(ctx, storedToken.ID()); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Get user
	u, err := h.userRepo.GetByID(ctx, storedToken.UserID())
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Generate new token pair
	tokenPair, err := h.generateTokenPair(ctx, u, cmd.UserAgent, cmd.IPAddress)
	if err != nil {
		return nil, err
	}

	return &RefreshTokensResult{TokenPair: tokenPair}, nil
}

func (h *RefreshTokensHandler) generateTokenPair(ctx context.Context, u *user.User, userAgent, ipAddress string) (*auth.TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := h.jwtGenerator.GenerateAccessToken(u.ID(), u.Email())
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	rawRefreshToken, tokenHash, refreshExpiresAt, err := h.jwtGenerator.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token
	refreshToken := auth.NewRefreshToken(u.ID(), tokenHash, refreshExpiresAt, userAgent, ipAddress)
	if err := h.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return auth.NewTokenPair(
		accessToken,
		rawRefreshToken,
		int64(h.jwtGenerator.GetAccessTokenExpiry().Seconds()),
		expiresAt,
	), nil
}

// LogoutCommand represents a command to logout.
type LogoutCommand struct {
	UserID uuid.UUID
}

// LogoutResult is the result of logout.
type LogoutResult struct{}

// LogoutHandler handles LogoutCommand.
type LogoutHandler struct {
	refreshTokenRepo auth.RefreshTokenRepository
}

// NewLogoutHandler creates a new handler.
func NewLogoutHandler(refreshTokenRepo auth.RefreshTokenRepository) *LogoutHandler {
	return &LogoutHandler{refreshTokenRepo: refreshTokenRepo}
}

// Handle executes the command.
func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) (*LogoutResult, error) {
	if err := h.refreshTokenRepo.RevokeAllForUser(ctx, cmd.UserID); err != nil {
		return nil, fmt.Errorf("revoke tokens: %w", err)
	}
	return &LogoutResult{}, nil
}
