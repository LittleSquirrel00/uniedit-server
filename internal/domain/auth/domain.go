package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	authv1 "github.com/uniedit/server/api/pb/auth"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// AuthDomain defines authentication domain service interface.
type AuthDomain interface {
	// OAuth operations
	InitiateLogin(ctx context.Context, in *authv1.InitiateLoginRequest) (*authv1.InitiateLoginResponse, error)
	CompleteLogin(ctx context.Context, in *authv1.CompleteLoginRequest, userAgent, ipAddress string) (*authv1.CompleteLoginResponse, error)

	// Token operations
	RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest, userAgent, ipAddress string) (*authv1.TokenPairResponse, error)
	GetMe(ctx context.Context, token string) (*authv1.GetMeResponse, error)
	Logout(ctx context.Context, userID uuid.UUID) (*commonv1.MessageResponse, error)
	ValidateAccessToken(token string) (*outbound.JWTClaims, error)

	// User API key operations
	CreateUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.CreateUserAPIKeyRequest) (*authv1.UserAPIKey, error)
	ListUserAPIKeys(ctx context.Context, userID uuid.UUID) (*authv1.ListUserAPIKeysResponse, error)
	GetUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.UserAPIKey, error)
	DeleteUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error)
	GetDecryptedAPIKey(ctx context.Context, userID uuid.UUID, provider string) (string, error)
	RotateUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.RotateUserAPIKeyRequest) (*authv1.UserAPIKey, error)

	// System API key operations
	CreateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.CreateSystemAPIKeyRequest) (*authv1.CreateSystemAPIKeyResponse, error)
	ListSystemAPIKeys(ctx context.Context, userID uuid.UUID) (*authv1.ListSystemAPIKeysResponse, error)
	GetSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.SystemAPIKey, error)
	UpdateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.UpdateSystemAPIKeyRequest) (*authv1.SystemAPIKey, error)
	DeleteSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error)
	RotateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.CreateSystemAPIKeyResponse, error)
	ValidateSystemAPIKey(ctx context.Context, apiKey string) (*model.SystemAPIKey, error)
}

// authDomain implements AuthDomain.
type authDomain struct {
	userDomain        user.UserDomain
	tokenRepo         outbound.RefreshTokenDatabasePort
	userAPIKeyRepo    outbound.UserAPIKeyDatabasePort
	systemAPIKeyRepo  outbound.SystemAPIKeyDatabasePort
	oauthRegistry     outbound.OAuthRegistryPort
	stateStore        outbound.OAuthStateStorePort
	jwt               outbound.JWTPort
	crypto            outbound.CryptoPort
	maxAPIKeysPerUser int
	logger            *zap.Logger
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

func (d *authDomain) InitiateLogin(ctx context.Context, in *authv1.InitiateLoginRequest) (*authv1.InitiateLoginResponse, error) {
	if in == nil {
		return nil, ErrInvalidOAuthProvider
	}

	provider := model.OAuthProvider(in.GetProvider())
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

	return &authv1.InitiateLoginResponse{AuthUrl: authURL, State: state}, nil
}

func (d *authDomain) CompleteLogin(ctx context.Context, in *authv1.CompleteLoginRequest, userAgent, ipAddress string) (*authv1.CompleteLoginResponse, error) {
	if in == nil {
		return nil, ErrInvalidOAuthProvider
	}

	provider := model.OAuthProvider(in.GetProvider())
	if !provider.IsValid() {
		return nil, ErrInvalidOAuthProvider
	}

	code := in.GetCode()
	state := in.GetState()

	// Verify state
	storedProvider, err := d.stateStore.Get(ctx, state)
	if err != nil {
		return nil, ErrInvalidOAuthState
	}
	defer d.stateStore.Delete(ctx, state)

	if storedProvider != provider.String() {
		return nil, ErrInvalidOAuthState
	}

	// Get OAuth provider
	oauthProvider, err := d.oauthRegistry.Get(provider.String())
	if err != nil {
		return nil, ErrInvalidOAuthProvider
	}

	// Exchange code for token
	token, err := oauthProvider.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOAuthCode, err)
	}

	// Get user info from provider
	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}

	u, err := d.findExistingUserByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	userID, err := uuid.Parse(u.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	tokenPair, err := d.generateTokenPair(ctx, userID, u.GetEmail(), userAgent, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return &authv1.CompleteLoginResponse{
		Token: toTokenPairPB(tokenPair),
		User:  u,
	}, nil
}

func (d *authDomain) findExistingUserByEmail(ctx context.Context, email string) (*commonv1.User, error) {
	u, err := d.userDomain.GetUserByEmail(ctx, email)
	if err == nil {
		return u, nil
	}
	return nil, fmt.Errorf("user creation via OAuth not yet implemented in new architecture")
}

// --- Token Operations ---

func (d *authDomain) RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest, userAgent, ipAddress string) (*authv1.TokenPairResponse, error) {
	if in == nil {
		return nil, ErrInvalidToken
	}

	refreshToken := in.GetRefreshToken()

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

	tokenPair, err := d.generateTokenPair(ctx, storedToken.UserID, u.GetEmail(), userAgent, ipAddress)
	if err != nil {
		return nil, err
	}
	return toTokenPairPB(tokenPair), nil
}

func (d *authDomain) GetMe(ctx context.Context, token string) (*authv1.GetMeResponse, error) {
	claims, err := d.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}
	return &authv1.GetMeResponse{
		UserId: claims.UserID.String(),
		Email:  claims.Email,
	}, nil
}

func (d *authDomain) Logout(ctx context.Context, userID uuid.UUID) (*commonv1.MessageResponse, error) {
	if err := d.tokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("revoke tokens: %w", err)
	}
	return &commonv1.MessageResponse{Message: "logged out"}, nil
}

func (d *authDomain) ValidateAccessToken(token string) (*outbound.JWTClaims, error) {
	return d.jwt.ValidateAccessToken(token)
}

func (d *authDomain) generateTokenPair(ctx context.Context, userID uuid.UUID, email, userAgent, ipAddress string) (*model.TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := d.jwt.GenerateAccessToken(userID, email)
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
		UserID:    userID,
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

func (d *authDomain) CreateUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.CreateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	if in == nil {
		return nil, ErrInvalidOAuthProvider
	}

	// Check if key already exists for this provider
	existing, err := d.userAPIKeyRepo.GetByUserAndProvider(ctx, userID, in.GetProvider())
	if err == nil && existing != nil {
		return nil, ErrAPIKeyAlreadyExists
	}

	// Encrypt the API key
	encryptedKey, err := d.crypto.Encrypt(in.GetApiKey())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Create the key record
	apiKey := &model.UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Provider:     in.GetProvider(),
		Name:         in.GetName(),
		EncryptedKey: encryptedKey,
		KeyPrefix:    getKeyPrefix(in.GetApiKey(), 7),
		Scopes:       pq.StringArray(in.GetScopes()),
	}

	if err := d.userAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return toUserAPIKeyPB(apiKey), nil
}

func (d *authDomain) ListUserAPIKeys(ctx context.Context, userID uuid.UUID) (*authv1.ListUserAPIKeysResponse, error) {
	keys, err := d.userAPIKeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]*authv1.UserAPIKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, toUserAPIKeyPB(k))
	}
	return &authv1.ListUserAPIKeysResponse{ApiKeys: out}, nil
}

func (d *authDomain) GetUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.UserAPIKey, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrAPIKeyNotFound
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	key, err := d.userAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	return toUserAPIKeyPB(key), nil
}

func (d *authDomain) DeleteUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrAPIKeyNotFound
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	key, err := d.userAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	if err := d.userAPIKeyRepo.Delete(ctx, keyID); err != nil {
		return nil, err
	}

	return &commonv1.MessageResponse{Message: "API key deleted"}, nil
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

func (d *authDomain) RotateUserAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.RotateUserAPIKeyRequest) (*authv1.UserAPIKey, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrAPIKeyNotFound
	}

	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	key, err := d.userAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}

	// Verify ownership
	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Encrypt new key
	encryptedKey, err := d.crypto.Encrypt(in.GetNewApiKey())
	if err != nil {
		return nil, ErrEncryptionFailed
	}

	// Update
	key.EncryptedKey = encryptedKey
	key.KeyPrefix = getKeyPrefix(in.GetNewApiKey(), 7)

	if err := d.userAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	return toUserAPIKeyPB(key), nil
}

// --- System API Key Operations ---

func (d *authDomain) CreateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.CreateSystemAPIKeyRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
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
	if in != nil && in.GetRateLimitRpm() > 0 {
		rateLimitRPM = int(in.GetRateLimitRpm())
	}
	if in != nil && in.GetRateLimitTpm() > 0 {
		rateLimitTPM = int(in.GetRateLimitTpm())
	}

	// Create record
	apiKey := &model.SystemAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "",
		KeyHash:      keyHash,
		KeyPrefix:    keyPrefix,
		Scopes:       nil,
		RateLimitRPM: rateLimitRPM,
		RateLimitTPM: rateLimitTPM,
		IsActive:     true,
	}
	if in != nil {
		apiKey.Name = in.GetName()
		apiKey.Scopes = pq.StringArray(in.GetScopes())
		if in.GetExpiresInDays() > 0 {
			t := time.Now().Add(time.Duration(in.GetExpiresInDays()) * 24 * time.Hour)
			apiKey.ExpiresAt = &t
		}
	}

	if err := d.systemAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create system api key: %w", err)
	}

	return &authv1.CreateSystemAPIKeyResponse{
		ApiKey:      rawKey,
		KeyDetails:  toSystemAPIKeyPB(apiKey),
	}, nil
}

func (d *authDomain) ListSystemAPIKeys(ctx context.Context, userID uuid.UUID) (*authv1.ListSystemAPIKeysResponse, error) {
	keys, err := d.systemAPIKeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]*authv1.SystemAPIKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, toSystemAPIKeyPB(k))
	}
	return &authv1.ListSystemAPIKeysResponse{ApiKeys: out}, nil
}

func (d *authDomain) GetSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.SystemAPIKey, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrSystemAPIKeyNotFound
	}
	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	return toSystemAPIKeyPB(key), nil
}

func (d *authDomain) UpdateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.UpdateSystemAPIKeyRequest) (*authv1.SystemAPIKey, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrSystemAPIKeyNotFound
	}
	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	// Apply updates
	if in.GetName() != nil {
		key.Name = in.GetName().GetValue()
	}
	if in.GetScopes() != nil {
		key.Scopes = pq.StringArray(in.GetScopes().GetValues())
	}
	if in.GetRateLimitRpm() != nil {
		key.RateLimitRPM = int(in.GetRateLimitRpm().GetValue())
	}
	if in.GetRateLimitTpm() != nil {
		key.RateLimitTPM = int(in.GetRateLimitTpm().GetValue())
	}
	if in.GetIsActive() != nil {
		key.IsActive = in.GetIsActive().GetValue()
	}

	if err := d.systemAPIKeyRepo.Update(ctx, key); err != nil {
		return nil, fmt.Errorf("update system api key: %w", err)
	}

	return toSystemAPIKeyPB(key), nil
}

func (d *authDomain) DeleteSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrSystemAPIKeyNotFound
	}
	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	if key.UserID != userID {
		return nil, ErrForbidden
	}

	if err := d.systemAPIKeyRepo.Delete(ctx, keyID); err != nil {
		return nil, err
	}
	return &commonv1.MessageResponse{Message: "API key deleted"}, nil
}

func (d *authDomain) RotateSystemAPIKey(ctx context.Context, userID uuid.UUID, in *authv1.GetByIDRequest) (*authv1.CreateSystemAPIKeyResponse, error) {
	if in == nil || in.GetId() == "" {
		return nil, ErrSystemAPIKeyNotFound
	}
	keyID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, ErrSystemAPIKeyNotFound
	}

	key, err := d.systemAPIKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrSystemAPIKeyNotFound
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

	return &authv1.CreateSystemAPIKeyResponse{
		ApiKey:     rawKey,
		KeyDetails: toSystemAPIKeyPB(key),
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
