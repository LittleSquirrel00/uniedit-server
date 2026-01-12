package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	authv1 "github.com/uniedit/server/api/pb/auth"
	commonv1 "github.com/uniedit/server/api/pb/common"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type MockUserDomain struct {
	mock.Mock
}

func (m *MockUserDomain) GetUser(ctx context.Context, id uuid.UUID) (*commonv1.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.User), args.Error(1)
}

func (m *MockUserDomain) GetUserByEmail(ctx context.Context, email string) (*commonv1.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.User), args.Error(1)
}

func (m *MockUserDomain) Register(ctx context.Context, in *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.RegisterResponse), args.Error(1)
}

func (m *MockUserDomain) VerifyEmail(ctx context.Context, in *userv1.VerifyEmailRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) GetProfile(ctx context.Context, userID uuid.UUID) (*userv1.Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.Profile), args.Error(1)
}

func (m *MockUserDomain) UpdateProfile(ctx context.Context, userID uuid.UUID, in *userv1.UpdateProfileRequest) (*commonv1.User, error) {
	args := m.Called(ctx, userID, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.User), args.Error(1)
}

func (m *MockUserDomain) GetPreferences(ctx context.Context, userID uuid.UUID) (*userv1.Preferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.Preferences), args.Error(1)
}

func (m *MockUserDomain) UpdatePreferences(ctx context.Context, userID uuid.UUID, in *userv1.UpdatePreferencesRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, userID, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) UploadAvatar(ctx context.Context, userID uuid.UUID, in *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error) {
	args := m.Called(ctx, userID, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.UploadAvatarResponse), args.Error(1)
}

func (m *MockUserDomain) ChangePassword(ctx context.Context, userID uuid.UUID, in *userv1.ChangePasswordRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, userID, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) RequestPasswordReset(ctx context.Context, in *userv1.RequestPasswordResetRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) ResetPassword(ctx context.Context, in *userv1.CompletePasswordResetRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) DeleteAccount(ctx context.Context, userID uuid.UUID, in *userv1.DeleteAccountRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, userID, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) ListUsers(ctx context.Context, in *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.ListUsersResponse), args.Error(1)
}

func (m *MockUserDomain) GetUserByID(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.User, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.User), args.Error(1)
}

func (m *MockUserDomain) SuspendUser(ctx context.Context, in *userv1.SuspendUserRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) ReactivateUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) SetAdminStatus(ctx context.Context, in *userv1.SetAdminStatusRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) AdminDeleteUser(ctx context.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

func (m *MockUserDomain) ResendVerification(ctx context.Context, in *userv1.ResendVerificationRequest) (*commonv1.MessageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*commonv1.MessageResponse), args.Error(1)
}

type MockRefreshTokenDB struct {
	mock.Mock
}

func (m *MockRefreshTokenDB) Create(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenDB) GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenDB) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenDB) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenDB) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockUserAPIKeyDB struct {
	mock.Mock
}

func (m *MockUserAPIKeyDB) Create(ctx context.Context, key *model.UserAPIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockUserAPIKeyDB) GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserAPIKey), args.Error(1)
}

func (m *MockUserAPIKeyDB) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserAPIKey), args.Error(1)
}

func (m *MockUserAPIKeyDB) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.UserAPIKey), args.Error(1)
}

func (m *MockUserAPIKeyDB) Update(ctx context.Context, key *model.UserAPIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockUserAPIKeyDB) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserAPIKeyDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockSystemAPIKeyDB struct {
	mock.Mock
}

func (m *MockSystemAPIKeyDB) Create(ctx context.Context, key *model.SystemAPIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSystemAPIKeyDB) GetByID(ctx context.Context, id uuid.UUID) (*model.SystemAPIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SystemAPIKey), args.Error(1)
}

func (m *MockSystemAPIKeyDB) GetByHash(ctx context.Context, hash string) (*model.SystemAPIKey, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SystemAPIKey), args.Error(1)
}

func (m *MockSystemAPIKeyDB) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.SystemAPIKey, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.SystemAPIKey), args.Error(1)
}

func (m *MockSystemAPIKeyDB) Update(ctx context.Context, key *model.SystemAPIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSystemAPIKeyDB) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSystemAPIKeyDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSystemAPIKeyDB) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

type MockOAuthStateStore struct {
	mock.Mock
}

func (m *MockOAuthStateStore) Set(ctx context.Context, state, provider string) error {
	args := m.Called(ctx, state, provider)
	return args.Error(0)
}

func (m *MockOAuthStateStore) Get(ctx context.Context, state string) (string, error) {
	args := m.Called(ctx, state)
	return args.String(0), args.Error(1)
}

func (m *MockOAuthStateStore) Delete(ctx context.Context, state string) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

type MockOAuthRegistry struct {
	mock.Mock
}

func (m *MockOAuthRegistry) Register(name string, provider outbound.OAuthProviderPort) {
	m.Called(name, provider)
}

func (m *MockOAuthRegistry) Get(name string) (outbound.OAuthProviderPort, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.OAuthProviderPort), args.Error(1)
}

func (m *MockOAuthRegistry) List() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

type MockOAuthProvider struct {
	mock.Mock
}

func (m *MockOAuthProvider) GetAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockOAuthProvider) Exchange(ctx context.Context, code string) (string, error) {
	args := m.Called(ctx, code)
	return args.String(0), args.Error(1)
}

func (m *MockOAuthProvider) GetUserInfo(ctx context.Context, token string) (*model.OAuthUserInfo, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OAuthUserInfo), args.Error(1)
}

type MockJWT struct {
	mock.Mock
}

func (m *MockJWT) GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockJWT) ValidateAccessToken(token string) (*outbound.JWTClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.JWTClaims), args.Error(1)
}

func (m *MockJWT) GenerateRefreshToken() (rawToken, tokenHash string, expiresAt time.Time, err error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Get(2).(time.Time), args.Error(3)
}

func (m *MockJWT) HashRefreshToken(token string) string {
	args := m.Called(token)
	return args.String(0)
}

func (m *MockJWT) GetAccessTokenExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockJWT) GetRefreshTokenExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

type MockCrypto struct {
	mock.Mock
}

func (m *MockCrypto) Encrypt(plaintext string) (string, error) {
	args := m.Called(plaintext)
	return args.String(0), args.Error(1)
}

func (m *MockCrypto) Decrypt(ciphertext string) (string, error) {
	args := m.Called(ciphertext)
	return args.String(0), args.Error(1)
}

// --- Tests ---

func TestAuthDomain_InitiateLogin(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOAuthRegistry := new(MockOAuthRegistry)
		mockOAuthProvider := new(MockOAuthProvider)
		mockStateStore := new(MockOAuthStateStore)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			mockOAuthRegistry,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockOAuthRegistry.On("Get", "github").Return(mockOAuthProvider, nil)
		mockStateStore.On("Set", mock.Anything, mock.AnythingOfType("string"), "github").Return(nil)
		mockOAuthProvider.On("GetAuthURL", mock.AnythingOfType("string")).Return("https://github.com/login/oauth/authorize?state=xxx")

		resp, err := domain.InitiateLogin(context.Background(), &authv1.InitiateLoginRequest{Provider: "github"})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.GetAuthUrl())
		assert.NotEmpty(t, resp.GetState())
		mockOAuthRegistry.AssertExpectations(t)
		mockStateStore.AssertExpectations(t)
	})

	t.Run("invalid provider", func(t *testing.T) {
		domain := NewAuthDomain(nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

		resp, err := domain.InitiateLogin(context.Background(), &authv1.InitiateLoginRequest{Provider: "invalid"})

		assert.ErrorIs(t, err, ErrInvalidOAuthProvider)
		assert.Nil(t, resp)
	})
}

func TestAuthDomain_RefreshToken(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDomain := new(MockUserDomain)
		mockRefreshTokenDB := new(MockRefreshTokenDB)
		mockJWT := new(MockJWT)

		domain := NewAuthDomain(
			mockUserDomain,
			mockRefreshTokenDB,
			nil,
			nil,
			nil,
			nil,
			mockJWT,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		tokenID := uuid.New()
		refreshToken := "raw-refresh-token"
		tokenHash := "hashed-token"

		storedToken := &model.RefreshToken{
			ID:        tokenID,
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		expectedUser := &commonv1.User{
			Id:    userID.String(),
			Email: "test@example.com",
		}

		mockJWT.On("HashRefreshToken", refreshToken).Return(tokenHash)
		mockRefreshTokenDB.On("GetByHash", mock.Anything, tokenHash).Return(storedToken, nil)
		mockRefreshTokenDB.On("Revoke", mock.Anything, tokenID).Return(nil)
		mockUserDomain.On("GetUser", mock.Anything, userID).Return(expectedUser, nil)
		mockJWT.On("GenerateAccessToken", userID, "test@example.com").Return("new-access-token", time.Now().Add(time.Hour), nil)
		mockJWT.On("GenerateRefreshToken").Return("new-refresh-token", "new-hash", time.Now().Add(7*24*time.Hour), nil)
		mockJWT.On("GetAccessTokenExpiry").Return(time.Hour)
		mockRefreshTokenDB.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil)

		tokenPair, err := domain.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: refreshToken}, "Mozilla/5.0", "127.0.0.1")

		assert.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.Equal(t, "new-access-token", tokenPair.GetAccessToken())
		assert.Equal(t, "new-refresh-token", tokenPair.GetRefreshToken())
		mockJWT.AssertExpectations(t)
		mockRefreshTokenDB.AssertExpectations(t)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockRefreshTokenDB := new(MockRefreshTokenDB)
		mockJWT := new(MockJWT)

		domain := NewAuthDomain(
			nil,
			mockRefreshTokenDB,
			nil,
			nil,
			nil,
			nil,
			mockJWT,
			nil,
			nil,
			logger,
		)

		mockJWT.On("HashRefreshToken", "invalid-token").Return("invalid-hash")
		mockRefreshTokenDB.On("GetByHash", mock.Anything, "invalid-hash").Return(nil, ErrInvalidToken)

		tokenPair, err := domain.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: "invalid-token"}, "", "")

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, tokenPair)
	})

	t.Run("expired token", func(t *testing.T) {
		mockRefreshTokenDB := new(MockRefreshTokenDB)
		mockJWT := new(MockJWT)

		domain := NewAuthDomain(
			nil,
			mockRefreshTokenDB,
			nil,
			nil,
			nil,
			nil,
			mockJWT,
			nil,
			nil,
			logger,
		)

		expiredToken := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: "hash",
			ExpiresAt: time.Now().Add(-time.Hour), // Expired
		}

		mockJWT.On("HashRefreshToken", "expired-token").Return("hash")
		mockRefreshTokenDB.On("GetByHash", mock.Anything, "hash").Return(expiredToken, nil)

		tokenPair, err := domain.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: "expired-token"}, "", "")

		assert.ErrorIs(t, err, ErrExpiredToken)
		assert.Nil(t, tokenPair)
	})
}

func TestAuthDomain_Logout(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRefreshTokenDB := new(MockRefreshTokenDB)

		domain := NewAuthDomain(
			nil,
			mockRefreshTokenDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		mockRefreshTokenDB.On("RevokeAllForUser", mock.Anything, userID).Return(nil)

		resp, err := domain.Logout(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRefreshTokenDB.AssertExpectations(t)
	})
}

func TestAuthDomain_CreateUserAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)
		mockCrypto := new(MockCrypto)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			mockCrypto,
			nil,
			logger,
		)

		userID := uuid.New()
		in := &authv1.CreateUserAPIKeyRequest{
			Provider: "openai",
			Name:     "My OpenAI Key",
			ApiKey:   "sk-1234567890",
			Scopes:   []string{"chat", "embedding"},
		}

		mockUserAPIKeyDB.On("GetByUserAndProvider", mock.Anything, userID, "openai").Return(nil, ErrAPIKeyNotFound)
		mockCrypto.On("Encrypt", in.GetApiKey()).Return("encrypted-key", nil)
		mockUserAPIKeyDB.On("Create", mock.Anything, mock.AnythingOfType("*model.UserAPIKey")).Return(nil)

		apiKey, err := domain.CreateUserAPIKey(context.Background(), userID, in)

		assert.NoError(t, err)
		assert.NotNil(t, apiKey)
		assert.Equal(t, "openai", apiKey.GetProvider())
		assert.Equal(t, "My OpenAI Key", apiKey.GetName())
		assert.Equal(t, "sk-1234", apiKey.GetKeyPrefix())
		mockUserAPIKeyDB.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("key already exists", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:       uuid.New(),
			UserID:   userID,
			Provider: "openai",
		}

		mockUserAPIKeyDB.On("GetByUserAndProvider", mock.Anything, userID, "openai").Return(existingKey, nil)

		apiKey, err := domain.CreateUserAPIKey(context.Background(), userID, &authv1.CreateUserAPIKeyRequest{
			Provider: "openai",
			Name:     "Test",
			ApiKey:   "sk-123",
		})

		assert.ErrorIs(t, err, ErrAPIKeyAlreadyExists)
		assert.Nil(t, apiKey)
	})
}

func TestAuthDomain_CreateSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			&Config{MaxAPIKeysPerUser: 10},
			logger,
		)

		userID := uuid.New()
		in := &authv1.CreateSystemAPIKeyRequest{
			Name:   "My API Key",
			Scopes: []string{"chat"},
		}

		mockSystemAPIKeyDB.On("CountByUser", mock.Anything, userID).Return(int64(0), nil)
		mockSystemAPIKeyDB.On("Create", mock.Anything, mock.AnythingOfType("*model.SystemAPIKey")).Return(nil)

		result, err := domain.CreateSystemAPIKey(context.Background(), userID, in)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.GetApiKey())
		assert.True(t, len(result.GetApiKey()) > 10)
		assert.Equal(t, "My API Key", result.GetKeyDetails().GetName())
		mockSystemAPIKeyDB.AssertExpectations(t)
	})

	t.Run("limit exceeded", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			&Config{MaxAPIKeysPerUser: 5},
			logger,
		)

		userID := uuid.New()
		mockSystemAPIKeyDB.On("CountByUser", mock.Anything, userID).Return(int64(5), nil)

		result, err := domain.CreateSystemAPIKey(context.Background(), userID, &authv1.CreateSystemAPIKeyRequest{
			Name: "Test Key",
		})

		assert.ErrorIs(t, err, ErrSystemAPIKeyLimitExceeded)
		assert.Nil(t, result)
	})
}

func TestAuthDomain_ValidateSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		apiKey := "sk-" + uuid.New().String()
		keyHash := hashAPIKey(apiKey)

		storedKey := &model.SystemAPIKey{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			KeyHash:  keyHash,
			IsActive: true,
		}

		mockSystemAPIKeyDB.On("GetByHash", mock.Anything, keyHash).Return(storedKey, nil)
		mockSystemAPIKeyDB.On("UpdateLastUsed", mock.Anything, storedKey.ID).Return(nil)

		key, err := domain.ValidateSystemAPIKey(context.Background(), apiKey)

		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.True(t, key.IsActive)
	})

	t.Run("key not found", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		mockSystemAPIKeyDB.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrSystemAPIKeyNotFound)

		key, err := domain.ValidateSystemAPIKey(context.Background(), "invalid-key")

		assert.ErrorIs(t, err, ErrSystemAPIKeyNotFound)
		assert.Nil(t, key)
	})

	t.Run("key disabled", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		apiKey := "sk-" + uuid.New().String()
		keyHash := hashAPIKey(apiKey)

		storedKey := &model.SystemAPIKey{
			ID:       uuid.New(),
			KeyHash:  keyHash,
			IsActive: false,
		}

		mockSystemAPIKeyDB.On("GetByHash", mock.Anything, keyHash).Return(storedKey, nil)

		key, err := domain.ValidateSystemAPIKey(context.Background(), apiKey)

		assert.ErrorIs(t, err, ErrSystemAPIKeyDisabled)
		assert.Nil(t, key)
	})

	t.Run("key expired", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		apiKey := "sk-" + uuid.New().String()
		keyHash := hashAPIKey(apiKey)

		expiredAt := time.Now().Add(-time.Hour)
		storedKey := &model.SystemAPIKey{
			ID:        uuid.New(),
			KeyHash:   keyHash,
			IsActive:  true,
			ExpiresAt: &expiredAt,
		}

		mockSystemAPIKeyDB.On("GetByHash", mock.Anything, keyHash).Return(storedKey, nil)

		key, err := domain.ValidateSystemAPIKey(context.Background(), apiKey)

		assert.ErrorIs(t, err, ErrSystemAPIKeyExpired)
		assert.Nil(t, key)
	})
}

func TestAuthDomain_ValidateAccessToken(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockJWT := new(MockJWT)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			mockJWT,
			nil,
			nil,
			logger,
		)

		expectedClaims := &outbound.JWTClaims{
			UserID: uuid.New(),
			Email:  "test@example.com",
		}

		mockJWT.On("ValidateAccessToken", "valid-token").Return(expectedClaims, nil)

		claims, err := domain.ValidateAccessToken("valid-token")

		assert.NoError(t, err)
		assert.Equal(t, expectedClaims, claims)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockJWT := new(MockJWT)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			mockJWT,
			nil,
			nil,
			logger,
		)

		mockJWT.On("ValidateAccessToken", "invalid-token").Return(nil, ErrInvalidToken)

		claims, err := domain.ValidateAccessToken("invalid-token")

		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})
}

func TestAuthDomain_ListUserAPIKeys(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		expectedKeys := []*model.UserAPIKey{
			{ID: uuid.New(), UserID: userID, Provider: "openai", Name: "Key 1"},
			{ID: uuid.New(), UserID: userID, Provider: "anthropic", Name: "Key 2"},
		}

		mockUserAPIKeyDB.On("ListByUser", mock.Anything, userID).Return(expectedKeys, nil)

		resp, err := domain.ListUserAPIKeys(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, resp.GetApiKeys(), 2)
		mockUserAPIKeyDB.AssertExpectations(t)
	})
}

func TestAuthDomain_DeleteUserAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:     keyID,
			UserID: userID,
		}

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockUserAPIKeyDB.On("Delete", mock.Anything, keyID).Return(nil)

		resp, err := domain.DeleteUserAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockUserAPIKeyDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrAPIKeyNotFound)

		resp, err := domain.DeleteUserAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
		assert.Nil(t, resp)
	})

	t.Run("not owned by user", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:     keyID,
			UserID: otherUserID,
		}

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)

		resp, err := domain.DeleteUserAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrForbidden)
		assert.Nil(t, resp)
	})
}

func TestAuthDomain_GetDecryptedAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)
		mockCrypto := new(MockCrypto)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			mockCrypto,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:           keyID,
			UserID:       userID,
			Provider:     "openai",
			EncryptedKey: "encrypted-key",
		}

		mockUserAPIKeyDB.On("GetByUserAndProvider", mock.Anything, userID, "openai").Return(existingKey, nil)
		mockUserAPIKeyDB.On("UpdateLastUsed", mock.Anything, keyID).Return(nil)
		mockCrypto.On("Decrypt", "encrypted-key").Return("sk-decrypted-key", nil)

		decrypted, err := domain.GetDecryptedAPIKey(context.Background(), userID, "openai")

		assert.NoError(t, err)
		assert.Equal(t, "sk-decrypted-key", decrypted)
		mockUserAPIKeyDB.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()

		mockUserAPIKeyDB.On("GetByUserAndProvider", mock.Anything, userID, "openai").Return(nil, ErrAPIKeyNotFound)

		decrypted, err := domain.GetDecryptedAPIKey(context.Background(), userID, "openai")

		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
		assert.Empty(t, decrypted)
	})
}

func TestAuthDomain_ListSystemAPIKeys(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		expectedKeys := []*model.SystemAPIKey{
			{ID: uuid.New(), UserID: userID, Name: "Key 1"},
			{ID: uuid.New(), UserID: userID, Name: "Key 2"},
		}

		mockSystemAPIKeyDB.On("ListByUser", mock.Anything, userID).Return(expectedKeys, nil)

		resp, err := domain.ListSystemAPIKeys(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, resp.GetApiKeys(), 2)
		mockSystemAPIKeyDB.AssertExpectations(t)
	})
}

func TestAuthDomain_GetSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:     keyID,
			UserID: userID,
			Name:   "My Key",
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)

		key, err := domain.GetSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.NoError(t, err)
		assert.Equal(t, "My Key", key.GetName())
		mockSystemAPIKeyDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrSystemAPIKeyNotFound)

		key, err := domain.GetSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrSystemAPIKeyNotFound)
		assert.Nil(t, key)
	})

	t.Run("not owned by user", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:     keyID,
			UserID: otherUserID,
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)

		key, err := domain.GetSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrForbidden)
		assert.Nil(t, key)
	})
}

func TestAuthDomain_DeleteSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:     keyID,
			UserID: userID,
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockSystemAPIKeyDB.On("Delete", mock.Anything, keyID).Return(nil)

		resp, err := domain.DeleteSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockSystemAPIKeyDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrSystemAPIKeyNotFound)

		resp, err := domain.DeleteSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrSystemAPIKeyNotFound)
		assert.Nil(t, resp)
	})
}

func TestAuthDomain_UpdateSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:       keyID,
			UserID:   userID,
			Name:     "Old Name",
			IsActive: true,
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockSystemAPIKeyDB.On("Update", mock.Anything, mock.AnythingOfType("*model.SystemAPIKey")).Return(nil)

		in := &authv1.UpdateSystemAPIKeyRequest{
			Id:       keyID.String(),
			Name:     &commonv1.StringValue{Value: "New Name"},
			IsActive: &commonv1.BoolValue{Value: false},
		}

		key, err := domain.UpdateSystemAPIKey(context.Background(), userID, in)

		assert.NoError(t, err)
		assert.Equal(t, "New Name", key.GetName())
		assert.False(t, key.GetIsActive())
		mockSystemAPIKeyDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrSystemAPIKeyNotFound)

		key, err := domain.UpdateSystemAPIKey(context.Background(), userID, &authv1.UpdateSystemAPIKeyRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrSystemAPIKeyNotFound)
		assert.Nil(t, key)
	})
}

func TestAuthDomain_CompleteLogin(t *testing.T) {
	logger := zap.NewNop()

	t.Run("invalid state", func(t *testing.T) {
		mockStateStore := new(MockOAuthStateStore)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			nil,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockStateStore.On("Get", mock.Anything, "invalid-state").Return("", ErrInvalidOAuthState)

		resp, err := domain.CompleteLogin(context.Background(), &authv1.CompleteLoginRequest{
			Provider: "github",
			Code:     "code",
			State:    "invalid-state",
		}, "Mozilla/5.0", "127.0.0.1")

		assert.ErrorIs(t, err, ErrInvalidOAuthState)
		assert.Nil(t, resp)
	})

	t.Run("provider mismatch", func(t *testing.T) {
		mockStateStore := new(MockOAuthStateStore)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			nil,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockStateStore.On("Get", mock.Anything, "valid-state").Return("google", nil) // stored provider is google
		mockStateStore.On("Delete", mock.Anything, "valid-state").Return(nil)

		resp, err := domain.CompleteLogin(context.Background(), &authv1.CompleteLoginRequest{
			Provider: "github",
			Code:     "code",
			State:    "valid-state",
		}, "Mozilla/5.0", "127.0.0.1")

		assert.ErrorIs(t, err, ErrInvalidOAuthState)
		assert.Nil(t, resp)
	})

	t.Run("invalid oauth provider", func(t *testing.T) {
		mockStateStore := new(MockOAuthStateStore)
		mockOAuthRegistry := new(MockOAuthRegistry)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			mockOAuthRegistry,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockStateStore.On("Get", mock.Anything, "valid-state").Return("github", nil)
		mockStateStore.On("Delete", mock.Anything, "valid-state").Return(nil)
		mockOAuthRegistry.On("Get", "github").Return(nil, ErrInvalidOAuthProvider)

		resp, err := domain.CompleteLogin(context.Background(), &authv1.CompleteLoginRequest{
			Provider: "github",
			Code:     "code",
			State:    "valid-state",
		}, "Mozilla/5.0", "127.0.0.1")

		assert.ErrorIs(t, err, ErrInvalidOAuthProvider)
		assert.Nil(t, resp)
	})

	t.Run("invalid oauth code", func(t *testing.T) {
		mockStateStore := new(MockOAuthStateStore)
		mockOAuthRegistry := new(MockOAuthRegistry)
		mockOAuthProvider := new(MockOAuthProvider)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			mockOAuthRegistry,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockStateStore.On("Get", mock.Anything, "valid-state").Return("github", nil)
		mockStateStore.On("Delete", mock.Anything, "valid-state").Return(nil)
		mockOAuthRegistry.On("Get", "github").Return(mockOAuthProvider, nil)
		mockOAuthProvider.On("Exchange", mock.Anything, "invalid-code").Return("", ErrInvalidOAuthCode)

		resp, err := domain.CompleteLogin(context.Background(), &authv1.CompleteLoginRequest{
			Provider: "github",
			Code:     "invalid-code",
			State:    "valid-state",
		}, "Mozilla/5.0", "127.0.0.1")

		assert.ErrorIs(t, err, ErrInvalidOAuthCode)
		assert.Nil(t, resp)
	})

	t.Run("get user info failed", func(t *testing.T) {
		mockStateStore := new(MockOAuthStateStore)
		mockOAuthRegistry := new(MockOAuthRegistry)
		mockOAuthProvider := new(MockOAuthProvider)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			nil,
			mockOAuthRegistry,
			mockStateStore,
			nil,
			nil,
			nil,
			logger,
		)

		mockStateStore.On("Get", mock.Anything, "valid-state").Return("github", nil)
		mockStateStore.On("Delete", mock.Anything, "valid-state").Return(nil)
		mockOAuthRegistry.On("Get", "github").Return(mockOAuthProvider, nil)
		mockOAuthProvider.On("Exchange", mock.Anything, "valid-code").Return("access-token", nil)
		mockOAuthProvider.On("GetUserInfo", mock.Anything, "access-token").Return(nil, ErrOAuthFailed)

		resp, err := domain.CompleteLogin(context.Background(), &authv1.CompleteLoginRequest{
			Provider: "github",
			Code:     "valid-code",
			State:    "valid-state",
		}, "Mozilla/5.0", "127.0.0.1")

		assert.ErrorIs(t, err, ErrOAuthFailed)
		assert.Nil(t, resp)
	})
}

func TestAuthDomain_RotateUserAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)
		mockCrypto := new(MockCrypto)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			mockCrypto,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:           keyID,
			UserID:       userID,
			Provider:     "openai",
			EncryptedKey: "old-encrypted-key",
		}

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockCrypto.On("Encrypt", "new-api-key").Return("new-encrypted-key", nil)
		mockUserAPIKeyDB.On("Update", mock.Anything, mock.AnythingOfType("*model.UserAPIKey")).Return(nil)

		key, err := domain.RotateUserAPIKey(context.Background(), userID, &authv1.RotateUserAPIKeyRequest{
			Id:        keyID.String(),
			NewApiKey: "new-api-key",
		})

		assert.NoError(t, err)
		assert.NotNil(t, key)
		mockUserAPIKeyDB.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrAPIKeyNotFound)

		key, err := domain.RotateUserAPIKey(context.Background(), userID, &authv1.RotateUserAPIKeyRequest{
			Id:        keyID.String(),
			NewApiKey: "new-api-key",
		})

		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
		assert.Nil(t, key)
	})

	t.Run("not owned by user", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:     keyID,
			UserID: otherUserID,
		}

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)

		key, err := domain.RotateUserAPIKey(context.Background(), userID, &authv1.RotateUserAPIKeyRequest{
			Id:        keyID.String(),
			NewApiKey: "new-api-key",
		})

		assert.ErrorIs(t, err, ErrForbidden)
		assert.Nil(t, key)
	})

	t.Run("encryption failed", func(t *testing.T) {
		mockUserAPIKeyDB := new(MockUserAPIKeyDB)
		mockCrypto := new(MockCrypto)

		domain := NewAuthDomain(
			nil,
			nil,
			mockUserAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			mockCrypto,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.UserAPIKey{
			ID:     keyID,
			UserID: userID,
		}

		mockUserAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockCrypto.On("Encrypt", "new-api-key").Return("", ErrEncryptionFailed)

		key, err := domain.RotateUserAPIKey(context.Background(), userID, &authv1.RotateUserAPIKeyRequest{
			Id:        keyID.String(),
			NewApiKey: "new-api-key",
		})

		assert.ErrorIs(t, err, ErrEncryptionFailed)
		assert.Nil(t, key)
	})
}

func TestAuthDomain_RotateSystemAPIKey(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:       keyID,
			UserID:   userID,
			Name:     "My Key",
			KeyHash:  "old-hash",
			IsActive: true,
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)
		mockSystemAPIKeyDB.On("Update", mock.Anything, mock.AnythingOfType("*model.SystemAPIKey")).Return(nil)

		result, err := domain.RotateSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.GetApiKey())
		assert.Equal(t, "My Key", result.GetKeyDetails().GetName())
		mockSystemAPIKeyDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		keyID := uuid.New()

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(nil, ErrSystemAPIKeyNotFound)

		result, err := domain.RotateSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrSystemAPIKeyNotFound)
		assert.Nil(t, result)
	})

	t.Run("not owned by user", func(t *testing.T) {
		mockSystemAPIKeyDB := new(MockSystemAPIKeyDB)

		domain := NewAuthDomain(
			nil,
			nil,
			nil,
			mockSystemAPIKeyDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		keyID := uuid.New()
		existingKey := &model.SystemAPIKey{
			ID:     keyID,
			UserID: otherUserID,
		}

		mockSystemAPIKeyDB.On("GetByID", mock.Anything, keyID).Return(existingKey, nil)

		result, err := domain.RotateSystemAPIKey(context.Background(), userID, &authv1.GetByIDRequest{Id: keyID.String()})

		assert.ErrorIs(t, err, ErrForbidden)
		assert.Nil(t, result)
	})
}
