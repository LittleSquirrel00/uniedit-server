package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// AuthDomain defines authentication domain service interface.
type AuthDomain interface {
	// OAuth operations
	InitiateLogin(ctx context.Context, provider model.OAuthProvider) (*LoginResponse, error)
	CompleteLogin(ctx context.Context, provider model.OAuthProvider, code, state, userAgent, ipAddress string) (*model.TokenPair, *model.User, error)

	// Token operations
	RefreshTokens(ctx context.Context, refreshToken, userAgent, ipAddress string) (*model.TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	ValidateAccessToken(token string) (*outbound.JWTClaims, error)

	// User API key operations
	CreateUserAPIKey(ctx context.Context, userID uuid.UUID, input *CreateUserAPIKeyInput) (*model.UserAPIKey, error)
	ListUserAPIKeys(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error)
	DeleteUserAPIKey(ctx context.Context, userID, keyID uuid.UUID) error
	GetDecryptedAPIKey(ctx context.Context, userID uuid.UUID, provider string) (string, error)
	RotateUserAPIKey(ctx context.Context, userID, keyID uuid.UUID, newKey string) (*model.UserAPIKey, error)

	// System API key operations
	CreateSystemAPIKey(ctx context.Context, userID uuid.UUID, input *CreateSystemAPIKeyInput) (*SystemAPIKeyCreateResult, error)
	ListSystemAPIKeys(ctx context.Context, userID uuid.UUID) ([]*model.SystemAPIKey, error)
	GetSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*model.SystemAPIKey, error)
	UpdateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID, input *UpdateSystemAPIKeyInput) (*model.SystemAPIKey, error)
	DeleteSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) error
	RotateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*SystemAPIKeyCreateResult, error)
	ValidateSystemAPIKey(ctx context.Context, apiKey string) (*model.SystemAPIKey, error)
}

// LoginResponse contains OAuth authorization URL.
type LoginResponse struct {
	AuthURL string
	State   string
}

// CreateUserAPIKeyInput represents input for creating a user API key.
type CreateUserAPIKeyInput struct {
	Provider string
	Name     string
	APIKey   string
	Scopes   []string
}

// CreateSystemAPIKeyInput represents input for creating a system API key.
type CreateSystemAPIKeyInput struct {
	Name          string
	Scopes        []string
	RateLimitRPM  *int
	RateLimitTPM  *int
	ExpiresInDays *int
}

// UpdateSystemAPIKeyInput represents input for updating a system API key.
type UpdateSystemAPIKeyInput struct {
	Name         *string
	Scopes       []string
	RateLimitRPM *int
	RateLimitTPM *int
	IsActive     *bool
}

// SystemAPIKeyCreateResult includes the full key (only on creation/rotation).
type SystemAPIKeyCreateResult struct {
	Key       *model.SystemAPIKey
	RawAPIKey string
}

// authDomain implements AuthDomain.
type authDomain struct {
	userDomain       user.UserDomain
	tokenRepo        outbound.RefreshTokenDatabasePort
	userAPIKeyRepo   outbound.UserAPIKeyDatabasePort
	systemAPIKeyRepo outbound.SystemAPIKeyDatabasePort
	oauthRegistry    outbound.OAuthRegistryPort
	stateStore       outbound.OAuthStateStorePort
	jwt              outbound.JWTPort
	crypto           outbound.CryptoPort
	maxAPIKeysPerUser int
	logger           *zap.Logger
}

// Config holds domain configuration.
type Config struct {
	MaxAPIKeysPerUser int
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxAPIKeysPerUser: 10,
	}
}

// NewAuthDomain creates a new auth domain service.
func NewAuthDomain(
	userDomain user.UserDomain,
	tokenRepo outbound.RefreshTokenDatabasePort,
	userAPIKeyRepo outbound.UserAPIKeyDatabasePort,
	systemAPIKeyRepo outbound.SystemAPIKeyDatabasePort,
	oauthRegistry outbound.OAuthRegistryPort,
	stateStore outbound.OAuthStateStorePort,
	jwt outbound.JWTPort,
	crypto outbound.CryptoPort,
	config *Config,
	logger *zap.Logger,
) AuthDomain {
	if config == nil {
		config = DefaultConfig()
	}
	return &authDomain{
		userDomain:        userDomain,
		tokenRepo:         tokenRepo,
		userAPIKeyRepo:    userAPIKeyRepo,
		systemAPIKeyRepo:  systemAPIKeyRepo,
		oauthRegistry:     oauthRegistry,
		stateStore:        stateStore,
		jwt:               jwt,
		crypto:            crypto,
		maxAPIKeysPerUser: config.MaxAPIKeysPerUser,
		logger:            logger,
	}
}

// --- OAuth Operations ---

func (d *authDomain) InitiateLogin(ctx context.Context, provider model.OAuthProvider) (*LoginResponse, error) {
	if !provider.IsValid() {
		return nil, ErrInvalidOAuthProvider
	}

	oauthProvider, err := d.oauthRegistry.Get(provider.String())
	if err != nil {
		return nil, ErrInvalidOAuthProvider
	}

	// Generate state token
	state := uuid.New().String()

	// Store state for verification
	if err := d.stateStore.Set(ctx, state, provider.String()); err != nil {
		return nil, fmt.Errorf("store state: %w", err)
	}

	authURL := oauthProvider.GetAuthURL(state)

	return &LoginResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

func (d *authDomain) CompleteLogin(ctx context.Context, provider model.OAuthProvider, code, state, userAgent, ipAddress string) (*model.TokenPair, *model.User, error) {
	// Verify state
	storedProvider, err := d.stateStore.Get(ctx, state)
	if err != nil {
		return nil, nil, ErrInvalidOAuthState
	}
	defer d.stateStore.Delete(ctx, state)

	if storedProvider != provider.String() {
		return nil, nil, ErrInvalidOAuthState
	}

	// Get OAuth provider
	oauthProvider, err := d.oauthRegistry.Get(provider.String())
	if err != nil {
		return nil, nil, ErrInvalidOAuthProvider
	}

	// Exchange code for token
	token, err := oauthProvider.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidOAuthCode, err)
	}

	// Get user info from provider
	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}

	// Find or create user
	u, err := d.findOrCreateUser(ctx, provider, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("find or create user: %w", err)
	}

	// Generate tokens
	tokenPair, err := d.generateTokenPair(ctx, u, userAgent, ipAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokenPair, u, nil
}

func (d *authDomain) findOrCreateUser(ctx context.Context, provider model.OAuthProvider, info *model.OAuthUserInfo) (*model.User, error) {
	// Try to find existing user by email
	u, err := d.userDomain.GetUserByEmail(ctx, info.Email)
	if err == nil {
		return u, nil
	}

	// Create new user via user domain - this is handled by registration
	// For OAuth users, we create them directly
	return nil, fmt.Errorf("user creation via OAuth not yet implemented in new architecture")
}

// --- Token Operations ---

func (d *authDomain) RefreshTokens(ctx context.Context, refreshToken, userAgent, ipAddress string) (*model.TokenPair, error) {
	// Hash the token to look it up
	tokenHash := d.jwt.HashRefreshToken(refreshToken)

	// Find the refresh token
	storedToken, err := d.tokenRepo.GetByHash(ctx, tokenHash)
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
	if err := d.tokenRepo.Revoke(ctx, storedToken.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Get user
	u, err := d.userDomain.GetUser(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Generate new token pair
	return d.generateTokenPair(ctx, u, userAgent, ipAddress)
}

func (d *authDomain) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := d.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke tokens: %w", err)
	}
	return nil
}

func (d *authDomain) ValidateAccessToken(token string) (*outbound.JWTClaims, error) {
	return d.jwt.ValidateAccessToken(token)
}

func (d *authDomain) generateTokenPair(ctx context.Context, u *model.User, userAgent, ipAddress string) (*model.TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := d.jwt.GenerateAccessToken(u.ID, u.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	rawRefreshToken, tokenHash, refreshExpiresAt, err := d.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token
	refreshTokenRecord := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: refreshExpiresAt,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := d.tokenRepo.Create(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(d.jwt.GetAccessTokenExpiry().Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// --- User API Key Operations ---

func (d *authDomain) CreateUserAPIKey(ctx context.Context, userID uuid.UUID, input *CreateUserAPIKeyInput) (*model.UserAPIKey, error) {
	// Check if key already exists for this provider
	existing, err := d.userAPIKeyRepo.GetByUserAndProvider(ctx, userID, input.Provider)
	if err == nil && existing != nil {
		return nil, ErrAPIKeyAlreadyExists
	}

	// Encrypt the API key
	encryptedKey, err := d.crypto.Encrypt(input.APIKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Create the key record
	apiKey := &model.UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Provider:     input.Provider,
		Name:         input.Name,
		EncryptedKey: encryptedKey,
		KeyPrefix:    getKeyPrefix(input.APIKey, 7),
		Scopes:       input.Scopes,
	}

	if err := d.userAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return apiKey, nil
}

func (d *authDomain) ListUserAPIKeys(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	return d.userAPIKeyRepo.ListByUser(ctx, userID)
}

func (d *authDomain) DeleteUserAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	// Verify ownership
	key, err := d.userAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return ErrForbidden
	}

	return d.userAPIKeyRepo.Delete(ctx, keyID)
}

func (d *authDomain) GetDecryptedAPIKey(ctx context.Context, userID uuid.UUID, provider string) (string, error) {
	key, err := d.userAPIKeyRepo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return "", err
	}

	// Update last used
	_ = d.userAPIKeyRepo.UpdateLastUsed(ctx, key.ID)

	// Decrypt
	decrypted, err := d.crypto.Decrypt(key.EncryptedKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return decrypted, nil
}

func (d *authDomain) RotateUserAPIKey(ctx context.Context, userID, keyID uuid.UUID, newKey string) (*model.UserAPIKey, error) {
	// Get existing key
	key, err := d.userAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Encrypt new key
	encryptedKey, err := d.crypto.Encrypt(newKey)
	if err != nil {
		return nil, ErrEncryptionFailed
	}

	// Update
	key.EncryptedKey = encryptedKey
	key.KeyPrefix = getKeyPrefix(newKey, 7)

	if err := d.userAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	return key, nil
}

// --- System API Key Operations ---

func (d *authDomain) CreateSystemAPIKey(ctx context.Context, userID uuid.UUID, input *CreateSystemAPIKeyInput) (*SystemAPIKeyCreateResult, error) {
	// Check limit
	count, err := d.systemAPIKeyRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count api keys: %w", err)
	}
	if count >= int64(d.maxAPIKeysPerUser) {
		return nil, ErrSystemAPIKeyLimitExceeded
	}

	// Generate API key
	rawKey, keyHash, keyPrefix := generateAPIKey()

	// Set defaults
	rateLimitRPM := 60
	rateLimitTPM := 100000
	if input.RateLimitRPM != nil && *input.RateLimitRPM > 0 {
		rateLimitRPM = *input.RateLimitRPM
	}
	if input.RateLimitTPM != nil && *input.RateLimitTPM > 0 {
		rateLimitTPM = *input.RateLimitTPM
	}

	// Create record
	apiKey := &model.SystemAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         input.Name,
		KeyHash:      keyHash,
		KeyPrefix:    keyPrefix,
		Scopes:       pq.StringArray(input.Scopes),
		RateLimitRPM: rateLimitRPM,
		RateLimitTPM: rateLimitTPM,
		IsActive:     true,
	}

	if err := d.systemAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create system api key: %w", err)
	}

	return &SystemAPIKeyCreateResult{
		Key:       apiKey,
		RawAPIKey: rawKey,
	}, nil
}

func (d *authDomain) ListSystemAPIKeys(ctx context.Context, userID uuid.UUID) ([]*model.SystemAPIKey, error) {
	return d.systemAPIKeyRepo.ListByUser(ctx, userID)
}

func (d *authDomain) GetSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*model.SystemAPIKey, error) {
	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	return key, nil
}

func (d *authDomain) UpdateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID, input *UpdateSystemAPIKeyInput) (*model.SystemAPIKey, error) {
	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Apply updates
	if input.Name != nil {
		key.Name = *input.Name
	}
	if len(input.Scopes) > 0 {
		key.Scopes = pq.StringArray(input.Scopes)
	}
	if input.RateLimitRPM != nil {
		key.RateLimitRPM = *input.RateLimitRPM
	}
	if input.RateLimitTPM != nil {
		key.RateLimitTPM = *input.RateLimitTPM
	}
	if input.IsActive != nil {
		key.IsActive = *input.IsActive
	}

	if err := d.systemAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return key, nil
}

func (d *authDomain) DeleteSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key.UserID != userID {
		return ErrForbidden
	}

	return d.systemAPIKeyRepo.Delete(ctx, keyID)
}

func (d *authDomain) RotateSystemAPIKey(ctx context.Context, userID, keyID uuid.UUID) (*SystemAPIKeyCreateResult, error) {
	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Generate new API key
	rawKey, keyHash, keyPrefix := generateAPIKey()

	// Update record
	key.KeyHash = keyHash
	key.KeyPrefix = keyPrefix

	if err := d.systemAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return &SystemAPIKeyCreateResult{
		Key:       key,
		RawAPIKey: rawKey,
	}, nil
}

func (d *authDomain) ValidateSystemAPIKey(ctx context.Context, apiKey string) (*model.SystemAPIKey, error) {
	// Hash and lookup
	keyHash := hashAPIKey(apiKey)
	key, err := d.systemAPIKeyRepo.GetByHash(ctx, keyHash)
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
		_ = d.systemAPIKeyRepo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

// --- Helpers ---

func getKeyPrefix(key string, length int) string {
	if len(key) <= length {
		return key
	}
	return key[:length]
}

func generateAPIKey() (rawKey, keyHash, keyPrefix string) {
	// Generate a random key
	id := uuid.New().String()
	rawKey = "sk-" + id

	// Hash for storage
	keyHash = hashAPIKey(rawKey)

	// Prefix for display
	keyPrefix = rawKey[:10]

	return
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
