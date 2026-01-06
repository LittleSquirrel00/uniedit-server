package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/module/auth/oauth"
)

// Service provides authentication operations.
type Service struct {
	userRepo         UserRepository
	tokenRepo        RefreshTokenRepository
	apiKeyRepo       APIKeyRepository
	systemAPIKeyRepo SystemAPIKeyRepository
	jwt              *JWTManager
	crypto           *CryptoManager
	oauth            *oauth.Registry
	stateStore       StateStore
	maxAPIKeysPerUser int
}

// StateStore defines the interface for OAuth state management.
type StateStore interface {
	Set(ctx context.Context, state string, data string) error
	Get(ctx context.Context, state string) (string, error)
	Delete(ctx context.Context, state string) error
}

// ServiceConfig holds service configuration.
type ServiceConfig struct {
	JWTConfig         *JWTConfig
	MasterKey         string
	MaxAPIKeysPerUser int
}

// NewService creates a new auth service.
func NewService(
	userRepo UserRepository,
	tokenRepo RefreshTokenRepository,
	apiKeyRepo APIKeyRepository,
	systemAPIKeyRepo SystemAPIKeyRepository,
	oauthRegistry *oauth.Registry,
	stateStore StateStore,
	config *ServiceConfig,
) (*Service, error) {
	crypto, err := NewCryptoManager(config.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("create crypto manager: %w", err)
	}

	maxKeys := config.MaxAPIKeysPerUser
	if maxKeys <= 0 {
		maxKeys = 10 // Default max API keys per user
	}

	return &Service{
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		apiKeyRepo:       apiKeyRepo,
		systemAPIKeyRepo: systemAPIKeyRepo,
		jwt:              NewJWTManager(config.JWTConfig),
		crypto:           crypto,
		oauth:            oauthRegistry,
		stateStore:       stateStore,
		maxAPIKeysPerUser: maxKeys,
	}, nil
}

// --- OAuth Operations ---

// InitiateLogin starts the OAuth login flow.
func (s *Service) InitiateLogin(ctx context.Context, provider OAuthProvider) (*LoginResponse, error) {
	if !provider.IsValid() {
		return nil, ErrInvalidOAuthProvider
	}

	oauthProvider, err := s.oauth.Get(provider.String())
	if err != nil {
		return nil, ErrInvalidOAuthProvider
	}

	// Generate state token
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	// Store state for verification
	if err := s.stateStore.Set(ctx, state, provider.String()); err != nil {
		return nil, fmt.Errorf("store state: %w", err)
	}

	authURL := oauthProvider.GetAuthURL(state)

	return &LoginResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// CompleteLogin completes the OAuth login flow.
func (s *Service) CompleteLogin(ctx context.Context, req *CallbackRequest, userAgent, ipAddress string) (*TokenPair, *User, error) {
	// Verify state
	storedProvider, err := s.stateStore.Get(ctx, req.State)
	if err != nil {
		return nil, nil, ErrInvalidOAuthState
	}
	defer s.stateStore.Delete(ctx, req.State)

	if storedProvider != req.Provider.String() {
		return nil, nil, ErrInvalidOAuthState
	}

	// Get OAuth provider
	oauthProvider, err := s.oauth.Get(req.Provider.String())
	if err != nil {
		return nil, nil, ErrInvalidOAuthProvider
	}

	// Exchange code for token
	token, err := oauthProvider.Exchange(ctx, req.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidOAuthCode, err)
	}

	// Get user info from provider
	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, req.Provider, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("find or create user: %w", err)
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokenPair, user, nil
}

// findOrCreateUser finds an existing user or creates a new one.
func (s *Service) findOrCreateUser(ctx context.Context, provider OAuthProvider, info *oauth.UserInfo) (*User, error) {
	// Try to find existing user by OAuth ID
	user, err := s.userRepo.GetByOAuth(ctx, provider, info.ID)
	if err == nil {
		// Update user info if changed
		updated := false
		if user.Email != info.Email {
			user.Email = info.Email
			updated = true
		}
		if user.Name != info.Name {
			user.Name = info.Name
			updated = true
		}
		if user.AvatarURL != info.AvatarURL {
			user.AvatarURL = info.AvatarURL
			updated = true
		}
		if updated {
			if err := s.userRepo.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("update user: %w", err)
			}
		}
		return user, nil
	}

	if err != ErrUserNotFound {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Create new user
	user = &User{
		ID:            uuid.New(),
		Email:         info.Email,
		Name:          info.Name,
		AvatarURL:     info.AvatarURL,
		OAuthProvider: provider,
		OAuthID:       info.ID,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// --- Token Operations ---

// RefreshTokens refreshes the access token using a refresh token.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string, userAgent, ipAddress string) (*TokenPair, error) {
	// Hash the token to look it up
	tokenHash := s.jwt.HashRefreshToken(refreshToken)

	// Find the refresh token
	storedToken, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Validate token
	if !storedToken.IsValid() {
		if storedToken.IsExpired() {
			return nil, ErrExpiredToken
		}
		return nil, ErrRevokedToken
	}

	// Revoke old token
	if err := s.tokenRepo.Revoke(ctx, storedToken.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Generate new token pair
	return s.generateTokenPair(ctx, user, userAgent, ipAddress)
}

// Logout revokes all refresh tokens for the user.
func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke tokens: %w", err)
	}
	return nil
}

// generateTokenPair generates a new access/refresh token pair.
func (s *Service) generateTokenPair(ctx context.Context, user *User, userAgent, ipAddress string) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	rawRefreshToken, tokenHash, refreshExpiresAt, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token
	refreshTokenRecord := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: refreshExpiresAt,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := s.tokenRepo.Create(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.GetAccessTokenExpiry().Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateAccessToken validates an access token and returns the claims.
func (s *Service) ValidateAccessToken(token string) (*Claims, error) {
	return s.jwt.ValidateAccessToken(token)
}

// --- User Operations ---

// GetUser returns user by ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// --- API Key Operations ---

// CreateAPIKey creates a new API key for a user.
func (s *Service) CreateAPIKey(ctx context.Context, userID uuid.UUID, req *CreateAPIKeyRequest) (*UserAPIKey, error) {
	// Check if key already exists for this provider
	existing, err := s.apiKeyRepo.GetByUserAndProvider(ctx, userID, req.Provider)
	if err == nil && existing != nil {
		return nil, ErrAPIKeyAlreadyExists
	}

	// Encrypt the API key
	encryptedKey, err := s.crypto.Encrypt(req.APIKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Create the key record
	apiKey := &UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Provider:     req.Provider,
		Name:         req.Name,
		EncryptedKey: encryptedKey,
		KeyPrefix:    GetKeyPrefix(req.APIKey, 7),
		Scopes:       req.Scopes,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return apiKey, nil
}

// GetAPIKey returns an API key by ID.
func (s *Service) GetAPIKey(ctx context.Context, id uuid.UUID) (*UserAPIKey, error) {
	return s.apiKeyRepo.GetByID(ctx, id)
}

// ListAPIKeys returns all API keys for a user.
func (s *Service) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]*UserAPIKey, error) {
	return s.apiKeyRepo.ListByUser(ctx, userID)
}

// DeleteAPIKey deletes an API key.
func (s *Service) DeleteAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	// Verify ownership
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return ErrForbidden
	}

	return s.apiKeyRepo.Delete(ctx, keyID)
}

// GetDecryptedAPIKey returns the decrypted API key for a provider.
// This is used internally by the AI module.
func (s *Service) GetDecryptedAPIKey(ctx context.Context, userID uuid.UUID, provider string) (string, error) {
	key, err := s.apiKeyRepo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return "", err
	}

	// Update last used
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, key.ID); err != nil {
		// Log but don't fail
	}

	// Decrypt
	decrypted, err := s.crypto.Decrypt(key.EncryptedKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return decrypted, nil
}

// RotateAPIKey rotates (replaces) an API key.
func (s *Service) RotateAPIKey(ctx context.Context, userID, keyID uuid.UUID, newAPIKey string) (*UserAPIKey, error) {
	// Get existing key
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Encrypt new key
	encryptedKey, err := s.crypto.Encrypt(newAPIKey)
	if err != nil {
		return nil, ErrEncryptionFailed
	}

	// Update
	key.EncryptedKey = encryptedKey
	key.KeyPrefix = GetKeyPrefix(newAPIKey, 7)

	if err := s.apiKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	return key, nil
}

// --- Helpers ---

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// --- System API Key Operations ---

// CreateSystemAPIKey creates a new system API key.
func (s *Service) CreateSystemAPIKey(ctx context.Context, userID uuid.UUID, req *CreateSystemAPIKeyRequest) (*SystemAPIKeyCreateResponse, error) {
	// Check limit
	count, err := s.systemAPIKeyRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count api keys: %w", err)
	}
	if count >= int64(s.maxAPIKeysPerUser) {
		return nil, ErrSystemAPIKeyLimitExceeded
	}

	// Validate scopes
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = DefaultScopes()
	}
	if err := ValidateScopes(scopes); err != nil {
		return nil, ErrInvalidAPIKeyScope
	}

	// Generate API key
	key, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	// Set defaults
	rateLimitRPM := 60
	rateLimitTPM := 100000
	if req.RateLimitRPM != nil && *req.RateLimitRPM > 0 {
		rateLimitRPM = *req.RateLimitRPM
	}
	if req.RateLimitTPM != nil && *req.RateLimitTPM > 0 {
		rateLimitTPM = *req.RateLimitTPM
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		t := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &t
	}

	// Create record
	apiKey := &SystemAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		KeyHash:      keyHash,
		KeyPrefix:    keyPrefix,
		Scopes:       pq.StringArray(scopes),
		RateLimitRPM: rateLimitRPM,
		RateLimitTPM: rateLimitTPM,
		IsActive:     true,
		ExpiresAt:    expiresAt,
	}

	if err := s.systemAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create system api key: %w", err)
	}

	return &SystemAPIKeyCreateResponse{
		SystemAPIKeyResponse: apiKey.ToResponse(),
		Key:                  key, // Only returned on creation
	}, nil
}

// GetSystemAPIKey returns a system API key by ID.
func (s *Service) GetSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*SystemAPIKey, error) {
	key, err := s.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	return key, nil
}

// ListSystemAPIKeys returns all system API keys for a user.
func (s *Service) ListSystemAPIKeys(ctx context.Context, userID uuid.UUID) ([]*SystemAPIKey, error) {
	return s.systemAPIKeyRepo.ListByUser(ctx, userID)
}

// UpdateSystemAPIKey updates a system API key.
func (s *Service) UpdateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID, req *UpdateSystemAPIKeyRequest) (*SystemAPIKey, error) {
	key, err := s.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Apply updates
	if req.Name != nil {
		key.Name = *req.Name
	}
	if len(req.Scopes) > 0 {
		if err := ValidateScopes(req.Scopes); err != nil {
			return nil, ErrInvalidAPIKeyScope
		}
		key.Scopes = pq.StringArray(req.Scopes)
	}
	if req.RateLimitRPM != nil {
		key.RateLimitRPM = *req.RateLimitRPM
	}
	if req.RateLimitTPM != nil {
		key.RateLimitTPM = *req.RateLimitTPM
	}
	if req.IsActive != nil {
		key.IsActive = *req.IsActive
	}

	if err := s.systemAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return key, nil
}

// DeleteSystemAPIKey deletes a system API key.
func (s *Service) DeleteSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	key, err := s.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}

	// Verify ownership
	if key.UserID != userID {
		return ErrForbidden
	}

	return s.systemAPIKeyRepo.Delete(ctx, keyID)
}

// RotateSystemAPIKey generates a new key for an existing system API key.
func (s *Service) RotateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*SystemAPIKeyCreateResponse, error) {
	key, err := s.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Generate new API key
	newKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	// Update record
	key.KeyHash = keyHash
	key.KeyPrefix = keyPrefix

	if err := s.systemAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return &SystemAPIKeyCreateResponse{
		SystemAPIKeyResponse: key.ToResponse(),
		Key:                  newKey,
	}, nil
}

// ValidateSystemAPIKey validates a system API key and returns the key record.
func (s *Service) ValidateSystemAPIKey(ctx context.Context, apiKey string) (*SystemAPIKey, error) {
	// Check format
	if !IsValidAPIKeyFormat(apiKey) {
		return nil, ErrInvalidAPIKeyFormat
	}

	// Hash and lookup
	keyHash := HashAPIKey(apiKey)
	key, err := s.systemAPIKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	// Check if active
	if !key.IsActive {
		return nil, ErrSystemAPIKeyDisabled
	}

	// Check expiration
	if key.IsExpired() {
		return nil, ErrSystemAPIKeyExpired
	}

	// Update last used (async, don't fail on error)
	go func() {
		_ = s.systemAPIKeyRepo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}
