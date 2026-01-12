package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock implementations ---

type MockUserDatabasePort struct {
	mock.Mock
}

func (m *MockUserDatabasePort) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserDatabasePort) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserDatabasePort) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserDatabasePort) FindByFilter(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*model.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserDatabasePort) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserDatabasePort) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockVerificationDatabasePort struct {
	mock.Mock
}

func (m *MockVerificationDatabasePort) CreateVerification(ctx context.Context, verification *model.EmailVerification) error {
	args := m.Called(ctx, verification)
	return args.Error(0)
}

func (m *MockVerificationDatabasePort) GetVerificationByToken(ctx context.Context, token string) (*model.EmailVerification, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.EmailVerification), args.Error(1)
}

func (m *MockVerificationDatabasePort) InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose model.VerificationPurpose) error {
	args := m.Called(ctx, userID, purpose)
	return args.Error(0)
}

func (m *MockVerificationDatabasePort) MarkVerificationUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockEmailSenderPort struct {
	mock.Mock
}

func (m *MockEmailSenderPort) SendVerificationEmail(ctx context.Context, email, name, token string) error {
	args := m.Called(ctx, email, name, token)
	return args.Error(0)
}

func (m *MockEmailSenderPort) SendPasswordResetEmail(ctx context.Context, email, name, token string) error {
	args := m.Called(ctx, email, name, token)
	return args.Error(0)
}

type MockProfileDatabasePort struct {
	mock.Mock
}

func (m *MockProfileDatabasePort) GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Profile), args.Error(1)
}

func (m *MockProfileDatabasePort) UpdateProfile(ctx context.Context, profile *model.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

type MockPreferencesDatabasePort struct {
	mock.Mock
}

func (m *MockPreferencesDatabasePort) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.Preferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Preferences), args.Error(1)
}

func (m *MockPreferencesDatabasePort) UpdatePreferences(ctx context.Context, prefs *model.Preferences) error {
	args := m.Called(ctx, prefs)
	return args.Error(0)
}

type MockAvatarStoragePort struct {
	mock.Mock
}

func (m *MockAvatarStoragePort) Upload(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, userID, data, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockAvatarStoragePort) Delete(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAvatarStoragePort) GetURL(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

// --- Tests ---

func TestUserDomain_GetUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		expectedUser := &model.User{
			ID:    userID,
			Email: "test@example.com",
			Name:  "Test User",
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(expectedUser, nil)

		user, err := domain.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser.Email, user.GetEmail())
		mockUserDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		user, err := domain.GetUser(context.Background(), userID)

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
		mockUserDB.AssertExpectations(t)
	})
}

func TestUserDomain_Register(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)
		mockEmailSender := new(MockEmailSenderPort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, mockEmailSender, logger)

		in := &userv1.RegisterRequest{Email: "test@example.com", Password: "password123", Name: "Test User"}

		mockUserDB.On("FindByEmail", mock.Anything, in.GetEmail()).Return(nil, ErrUserNotFound)
		mockUserDB.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
		mockVerificationDB.On("CreateVerification", mock.Anything, mock.AnythingOfType("*model.EmailVerification")).Return(nil)
		mockEmailSender.On("SendVerificationEmail", mock.Anything, in.GetEmail(), in.GetName(), mock.AnythingOfType("string")).Return(nil)

		out, err := domain.Register(context.Background(), in)

		assert.NoError(t, err)
		assert.NotNil(t, out)
		assert.NotNil(t, out.GetUser())
		assert.Equal(t, in.GetEmail(), out.GetUser().GetEmail())
		assert.Equal(t, in.GetName(), out.GetUser().GetName())
		assert.Equal(t, string(model.UserStatusPending), out.GetUser().GetStatus())
		assert.False(t, out.GetUser().GetEmailVerified())
		mockUserDB.AssertExpectations(t)
		mockVerificationDB.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		in := &userv1.RegisterRequest{Email: "test@example.com", Password: "password123", Name: "Test User"}

		existingUser := &model.User{ID: uuid.New(), Email: in.GetEmail()}
		mockUserDB.On("FindByEmail", mock.Anything, in.GetEmail()).Return(existingUser, nil)

		out, err := domain.Register(context.Background(), in)

		assert.ErrorIs(t, err, ErrEmailAlreadyExists)
		assert.Nil(t, out)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("password too short", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		in := &userv1.RegisterRequest{Email: "test@example.com", Password: "short", Name: "Test User"}

		mockUserDB.On("FindByEmail", mock.Anything, in.GetEmail()).Return(nil, ErrUserNotFound)

		out, err := domain.Register(context.Background(), in)

		assert.ErrorIs(t, err, ErrPasswordTooShort)
		assert.Nil(t, out)
	})
}

func TestUserDomain_VerifyEmail(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		verificationID := uuid.New()
		token := "valid-token"

		verification := &model.EmailVerification{
			ID:        verificationID,
			UserID:    userID,
			Token:     token,
			Purpose:   model.VerificationPurposeRegistration,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		user := &model.User{
			ID:            userID,
			Status:        model.UserStatusPending,
			EmailVerified: false,
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, token).Return(verification, nil)
		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockVerificationDB.On("MarkVerificationUsed", mock.Anything, verificationID).Return(nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		out, err := domain.VerifyEmail(context.Background(), &userv1.VerifyEmailRequest{Token: token})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		assert.True(t, user.EmailVerified)
		assert.Equal(t, model.UserStatusActive, user.Status)
		mockVerificationDB.AssertExpectations(t)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("token expired", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		token := "expired-token"
		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     token,
			Purpose:   model.VerificationPurposeRegistration,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, token).Return(verification, nil)

		_, err := domain.VerifyEmail(context.Background(), &userv1.VerifyEmailRequest{Token: token})

		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("token already used", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		token := "used-token"
		usedAt := time.Now()
		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     token,
			Purpose:   model.VerificationPurposeRegistration,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			UsedAt:    &usedAt,
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, token).Return(verification, nil)

		_, err := domain.VerifyEmail(context.Background(), &userv1.VerifyEmailRequest{Token: token})

		assert.ErrorIs(t, err, ErrTokenAlreadyUsed)
	})
}

func TestUserDomain_ChangePassword(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		currentPassword := "oldpassword123"
		newPassword := "newpassword123"

		hash, _ := bcrypt.GenerateFromPassword([]byte(currentPassword), bcrypt.DefaultCost)
		hashStr := string(hash)

		user := &model.User{
			ID:           userID,
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		_, err := domain.ChangePassword(context.Background(), userID, &userv1.ChangePasswordRequest{
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		})

		assert.NoError(t, err)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		currentPassword := "wrongpassword"
		newPassword := "newpassword123"

		hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		hashStr := string(hash)

		user := &model.User{
			ID:           userID,
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.ChangePassword(context.Background(), userID, &userv1.ChangePasswordRequest{
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		})

		assert.ErrorIs(t, err, ErrIncorrectPassword)
	})
}

func TestUserDomain_SuspendUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			Status:  model.UserStatusActive,
			IsAdmin: false,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		_, err := domain.SuspendUser(context.Background(), &userv1.SuspendUserRequest{
			Id:     userID.String(),
			Reason: "violation of terms",
		})

		assert.NoError(t, err)
		assert.Equal(t, model.UserStatusSuspended, user.Status)
		assert.NotNil(t, user.SuspendedAt)
		assert.Equal(t, "violation of terms", *user.SuspendReason)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("cannot suspend admin", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			Status:  model.UserStatusActive,
			IsAdmin: true,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.SuspendUser(context.Background(), &userv1.SuspendUserRequest{
			Id:     userID.String(),
			Reason: "any reason",
		})

		assert.ErrorIs(t, err, ErrCannotSuspendAdmin)
	})
}

func TestUserDomain_ListUsers(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)

		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		in := &userv1.ListUsersRequest{Page: 1, PageSize: 20}

		expectedUsers := []*model.User{
			{ID: uuid.New(), Email: "user1@example.com"},
			{ID: uuid.New(), Email: "user2@example.com"},
		}

		mockUserDB.On("FindByFilter", mock.Anything, mock.AnythingOfType("model.UserFilter")).Return(expectedUsers, int64(2), nil)

		out, err := domain.ListUsers(context.Background(), in)

		assert.NoError(t, err)
		assert.NotNil(t, out)
		assert.Len(t, out.GetData(), 2)
		assert.Equal(t, int64(2), out.GetTotal())
		mockUserDB.AssertExpectations(t)
	})
}

func TestUserDomain_GetProfile(t *testing.T) {
	logger := zap.NewNop()

	t.Run("fallback to user data", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)

		// profileDB is nil, so fallback to user data
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:        userID,
			Name:      "Test User",
			AvatarURL: "https://example.com/avatar.jpg",
			UpdatedAt: time.Now(),
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		profile, err := domain.GetProfile(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, user.Name, profile.DisplayName)
		assert.Equal(t, user.AvatarURL, profile.AvatarUrl)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		profile, err := domain.GetProfile(context.Background(), userID)

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, profile)
	})
}

func TestUserDomain_UpdateProfile(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:   userID,
			Name: "Old Name",
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		result, err := domain.UpdateProfile(context.Background(), userID, &userv1.UpdateProfileRequest{Name: "New Name"})

		assert.NoError(t, err)
		assert.Equal(t, "New Name", result.GetName())
		mockUserDB.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		result, err := domain.UpdateProfile(context.Background(), userID, &userv1.UpdateProfileRequest{})

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, result)
	})
}

func TestUserDomain_GetPreferences(t *testing.T) {
	logger := zap.NewNop()

	t.Run("default preferences when not configured", func(t *testing.T) {
		// prefsDB is nil
		domain := NewUserDomain(nil, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		prefs, err := domain.GetPreferences(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, "light", prefs.Theme)
		assert.Equal(t, "en", prefs.Language)
		assert.Equal(t, "UTC", prefs.Timezone)
	})
}

func TestUserDomain_UpdatePreferences(t *testing.T) {
	logger := zap.NewNop()

	t.Run("no-op when not configured", func(t *testing.T) {
		// prefsDB is nil
		domain := NewUserDomain(nil, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		out, err := domain.UpdatePreferences(context.Background(), userID, &userv1.UpdatePreferencesRequest{Theme: "dark"})

		assert.NoError(t, err)
		assert.NotNil(t, out)
	})
}

func TestUserDomain_GetUserByEmail(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		email := "test@example.com"
		user := &model.User{
			ID:    uuid.New(),
			Email: email,
		}

		mockUserDB.On("FindByEmail", mock.Anything, email).Return(user, nil)

		result, err := domain.GetUserByEmail(context.Background(), email)

		assert.NoError(t, err)
		assert.Equal(t, email, result.GetEmail())
		mockUserDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		mockUserDB.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, nil)

		result, err := domain.GetUserByEmail(context.Background(), "notfound@example.com")

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, result)
	})
}

func TestUserDomain_DeleteAccount(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		password := "correctpassword"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		hashStr := string(hash)

		user := &model.User{
			ID:           userID,
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("SoftDelete", mock.Anything, userID).Return(nil)

		out, err := domain.DeleteAccount(context.Background(), userID, &userv1.DeleteAccountRequest{Password: password})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("incorrect password", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		hashStr := string(hash)

		user := &model.User{
			ID:           userID,
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.DeleteAccount(context.Background(), userID, &userv1.DeleteAccountRequest{Password: "wrongpassword"})

		assert.ErrorIs(t, err, ErrIncorrectPassword)
	})

	t.Run("password required for email user", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		hashStr := "somehash"
		user := &model.User{
			ID:           userID,
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.DeleteAccount(context.Background(), userID, &userv1.DeleteAccountRequest{Password: ""})

		assert.ErrorIs(t, err, ErrPasswordRequired)
	})
}

func TestUserDomain_ResendVerification(t *testing.T) {
	logger := zap.NewNop()

	t.Run("user not found returns nil", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		mockUserDB.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, nil)

		out, err := domain.ResendVerification(context.Background(), &userv1.ResendVerificationRequest{Email: "notfound@example.com"})

		assert.NoError(t, err) // Returns nil to not reveal if email exists
		assert.NotNil(t, out)
	})

	t.Run("already verified returns nil", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		user := &model.User{
			ID:            uuid.New(),
			Email:         "verified@example.com",
			EmailVerified: true,
		}

		mockUserDB.On("FindByEmail", mock.Anything, "verified@example.com").Return(user, nil)

		out, err := domain.ResendVerification(context.Background(), &userv1.ResendVerificationRequest{Email: "verified@example.com"})

		assert.NoError(t, err) // Returns nil for already verified
		assert.NotNil(t, out)
	})

	t.Run("sends verification email", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)
		mockEmailSender := new(MockEmailSenderPort)
		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, mockEmailSender, logger)

		user := &model.User{
			ID:            uuid.New(),
			Email:         "test@example.com",
			Name:          "Test User",
			EmailVerified: false,
		}

		mockUserDB.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockVerificationDB.On("InvalidateUserVerifications", mock.Anything, user.ID, model.VerificationPurposeRegistration).Return(nil)
		mockVerificationDB.On("CreateVerification", mock.Anything, mock.AnythingOfType("*model.EmailVerification")).Return(nil)
		mockEmailSender.On("SendVerificationEmail", mock.Anything, user.Email, user.Name, mock.AnythingOfType("string")).Return(nil)

		out, err := domain.ResendVerification(context.Background(), &userv1.ResendVerificationRequest{Email: "test@example.com"})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockEmailSender.AssertExpectations(t)
	})
}

func TestUserDomain_RequestPasswordReset(t *testing.T) {
	logger := zap.NewNop()

	t.Run("user not found returns nil", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		mockUserDB.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, nil)

		out, err := domain.RequestPasswordReset(context.Background(), &userv1.RequestPasswordResetRequest{Email: "notfound@example.com"})

		assert.NoError(t, err) // Returns nil to not reveal if email exists
		assert.NotNil(t, out)
	})

	t.Run("oauth user returns nil", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		// OAuth user has no password hash
		user := &model.User{
			ID:           uuid.New(),
			Email:        "oauth@example.com",
			PasswordHash: nil,
		}

		mockUserDB.On("FindByEmail", mock.Anything, "oauth@example.com").Return(user, nil)

		out, err := domain.RequestPasswordReset(context.Background(), &userv1.RequestPasswordResetRequest{Email: "oauth@example.com"})

		assert.NoError(t, err) // Returns nil for OAuth user
		assert.NotNil(t, out)
	})

	t.Run("sends reset email", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)
		mockEmailSender := new(MockEmailSenderPort)
		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, mockEmailSender, logger)

		hashStr := "somehash"
		user := &model.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Name:         "Test User",
			PasswordHash: &hashStr,
		}

		mockUserDB.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockVerificationDB.On("InvalidateUserVerifications", mock.Anything, user.ID, model.VerificationPurposePasswordReset).Return(nil)
		mockVerificationDB.On("CreateVerification", mock.Anything, mock.AnythingOfType("*model.EmailVerification")).Return(nil)
		mockEmailSender.On("SendPasswordResetEmail", mock.Anything, user.Email, user.Name, mock.AnythingOfType("string")).Return(nil)

		out, err := domain.RequestPasswordReset(context.Background(), &userv1.RequestPasswordResetRequest{Email: "test@example.com"})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockEmailSender.AssertExpectations(t)
	})
}

func TestUserDomain_ResetPassword(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)
		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		verificationID := uuid.New()

		verification := &model.EmailVerification{
			ID:        verificationID,
			UserID:    userID,
			Token:     "valid-token",
			Purpose:   model.VerificationPurposePasswordReset,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		user := &model.User{
			ID: userID,
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, "valid-token").Return(verification, nil)
		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockVerificationDB.On("MarkVerificationUsed", mock.Anything, verificationID).Return(nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		out, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{Token: "valid-token", NewPassword: "newpassword123"})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		assert.NotNil(t, user.PasswordHash)
		mockVerificationDB.AssertExpectations(t)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("password too short", func(t *testing.T) {
		domain := NewUserDomain(nil, nil, nil, nil, nil, nil, logger)

		_, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{Token: "token", NewPassword: "short"})

		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("wrong purpose", func(t *testing.T) {
		mockVerificationDB := new(MockVerificationDatabasePort)
		domain := NewUserDomain(nil, mockVerificationDB, nil, nil, nil, nil, logger)

		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "token",
			Purpose:   model.VerificationPurposeRegistration, // Wrong purpose
			ExpiresAt: time.Now().Add(time.Hour),
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, "token").Return(verification, nil)

		_, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{Token: "token", NewPassword: "newpassword123"})

		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestUserDomain_ReactivateUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		suspendReason := "violation"
		now := time.Now()
		user := &model.User{
			ID:            userID,
			Status:        model.UserStatusSuspended,
			SuspendedAt:   &now,
			SuspendReason: &suspendReason,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		_, err := domain.ReactivateUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.NoError(t, err)
		assert.Equal(t, model.UserStatusActive, user.Status)
		assert.Nil(t, user.SuspendedAt)
		assert.Nil(t, user.SuspendReason)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("user not suspended", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:     userID,
			Status: model.UserStatusActive,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.ReactivateUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.ErrorIs(t, err, ErrUserAlreadyActive)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		_, err := domain.ReactivateUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserDomain_SetAdminStatus(t *testing.T) {
	logger := zap.NewNop()

	t.Run("promote to admin", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			IsAdmin: false,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		_, err := domain.SetAdminStatus(context.Background(), &userv1.SetAdminStatusRequest{
			Id:      userID.String(),
			IsAdmin: true,
		})

		assert.NoError(t, err)
		assert.True(t, user.IsAdmin)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("demote from admin", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			IsAdmin: true,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

		_, err := domain.SetAdminStatus(context.Background(), &userv1.SetAdminStatusRequest{
			Id:      userID.String(),
			IsAdmin: false,
		})

		assert.NoError(t, err)
		assert.False(t, user.IsAdmin)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		_, err := domain.SetAdminStatus(context.Background(), &userv1.SetAdminStatusRequest{
			Id:      userID.String(),
			IsAdmin: true,
		})

		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserDomain_AdminDeleteUser(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			IsAdmin: false,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)
		mockUserDB.On("SoftDelete", mock.Anything, userID).Return(nil)

		_, err := domain.AdminDeleteUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.NoError(t, err)
		mockUserDB.AssertExpectations(t)
	})

	t.Run("cannot delete admin", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:      userID,
			IsAdmin: true,
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.AdminDeleteUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.ErrorIs(t, err, ErrCannotSuspendAdmin)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		_, err := domain.AdminDeleteUser(context.Background(), &userv1.GetByIDRequest{Id: userID.String()})

		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserDomain_ChangePassword_EdgeCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("new password too short", func(t *testing.T) {
		domain := NewUserDomain(nil, nil, nil, nil, nil, nil, logger)

		_, err := domain.ChangePassword(context.Background(), uuid.New(), &userv1.ChangePasswordRequest{
			CurrentPassword: "old",
			NewPassword:     "short",
		})

		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("no password set (oauth user)", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		domain := NewUserDomain(mockUserDB, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		user := &model.User{
			ID:           userID,
			PasswordHash: nil, // OAuth user
		}

		mockUserDB.On("FindByID", mock.Anything, userID).Return(user, nil)

		_, err := domain.ChangePassword(context.Background(), userID, &userv1.ChangePasswordRequest{
			CurrentPassword: "old",
			NewPassword:     "newpassword123",
		})

		assert.ErrorIs(t, err, ErrPasswordRequired)
	})
}

func TestUserDomain_GetProfile_WithProfileDB(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success with profile db", func(t *testing.T) {
		mockProfileDB := new(MockProfileDatabasePort)
		domain := NewUserDomain(nil, nil, mockProfileDB, nil, nil, nil, logger)

		userID := uuid.New()
		profile := &model.Profile{
			UserID:      userID,
			DisplayName: "Test User",
			Bio:         "Test bio",
		}

		mockProfileDB.On("GetProfile", mock.Anything, userID).Return(profile, nil)

		result, err := domain.GetProfile(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, profile.DisplayName, result.DisplayName)
		mockProfileDB.AssertExpectations(t)
	})

	t.Run("profile not found", func(t *testing.T) {
		mockProfileDB := new(MockProfileDatabasePort)
		domain := NewUserDomain(nil, nil, mockProfileDB, nil, nil, nil, logger)

		userID := uuid.New()
		mockProfileDB.On("GetProfile", mock.Anything, userID).Return(nil, nil)

		result, err := domain.GetProfile(context.Background(), userID)

		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, result)
	})
}

func TestUserDomain_GetPreferences_WithPrefsDB(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success with prefs db", func(t *testing.T) {
		mockPrefsDB := new(MockPreferencesDatabasePort)
		domain := NewUserDomain(nil, nil, nil, mockPrefsDB, nil, nil, logger)

		userID := uuid.New()
		prefs := &model.Preferences{
			UserID:   userID,
			Theme:    "dark",
			Language: "zh",
		}

		mockPrefsDB.On("GetPreferences", mock.Anything, userID).Return(prefs, nil)

		result, err := domain.GetPreferences(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, "dark", result.Theme)
		mockPrefsDB.AssertExpectations(t)
	})
}

func TestUserDomain_UpdatePreferences_WithPrefsDB(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success with prefs db", func(t *testing.T) {
		mockPrefsDB := new(MockPreferencesDatabasePort)
		domain := NewUserDomain(nil, nil, nil, mockPrefsDB, nil, nil, logger)

		userID := uuid.New()

		mockPrefsDB.On("UpdatePreferences", mock.Anything, mock.AnythingOfType("*model.Preferences")).Return(nil)

		out, err := domain.UpdatePreferences(context.Background(), userID, &userv1.UpdatePreferencesRequest{
			Theme:    "dark",
			Language: "zh",
		})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockPrefsDB.AssertExpectations(t)
	})
}

func TestUserDomain_UploadAvatar(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockAvatarStorage := new(MockAvatarStoragePort)
		domain := NewUserDomain(nil, nil, nil, nil, mockAvatarStorage, nil, logger)

		userID := uuid.New()
		data := []byte("image data")
		contentType := "image/png"
		expectedURL := "https://storage.example.com/avatars/test.png"

		mockAvatarStorage.On("Upload", mock.Anything, userID, data, contentType).Return(expectedURL, nil)

		result, err := domain.UploadAvatar(context.Background(), userID, &userv1.UploadAvatarRequest{
			Data:        data,
			ContentType: contentType,
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedURL, result.GetUrl())
		mockAvatarStorage.AssertExpectations(t)
	})

	t.Run("storage not configured", func(t *testing.T) {
		domain := NewUserDomain(nil, nil, nil, nil, nil, nil, logger)

		result, err := domain.UploadAvatar(context.Background(), uuid.New(), &userv1.UploadAvatarRequest{
			Data:        []byte("data"),
			ContentType: "image/png",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "avatar storage not configured")
	})
}

func TestUserDomain_ResetPassword_EdgeCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("token already used", func(t *testing.T) {
		mockVerificationDB := new(MockVerificationDatabasePort)
		domain := NewUserDomain(nil, mockVerificationDB, nil, nil, nil, nil, logger)

		usedAt := time.Now()
		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "used-token",
			Purpose:   model.VerificationPurposePasswordReset,
			ExpiresAt: time.Now().Add(time.Hour),
			UsedAt:    &usedAt,
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, "used-token").Return(verification, nil)

		_, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{
			Token:       "used-token",
			NewPassword: "newpassword123",
		})

		assert.ErrorIs(t, err, ErrTokenAlreadyUsed)
	})

	t.Run("token expired", func(t *testing.T) {
		mockVerificationDB := new(MockVerificationDatabasePort)
		domain := NewUserDomain(nil, mockVerificationDB, nil, nil, nil, nil, logger)

		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "expired-token",
			Purpose:   model.VerificationPurposePasswordReset,
			ExpiresAt: time.Now().Add(-time.Hour), // Expired
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, "expired-token").Return(verification, nil)

		_, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{
			Token:       "expired-token",
			NewPassword: "newpassword123",
		})

		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserDB := new(MockUserDatabasePort)
		mockVerificationDB := new(MockVerificationDatabasePort)
		domain := NewUserDomain(mockUserDB, mockVerificationDB, nil, nil, nil, nil, logger)

		userID := uuid.New()
		verification := &model.EmailVerification{
			ID:        uuid.New(),
			UserID:    userID,
			Token:     "valid-token",
			Purpose:   model.VerificationPurposePasswordReset,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		mockVerificationDB.On("GetVerificationByToken", mock.Anything, "valid-token").Return(verification, nil)
		mockUserDB.On("FindByID", mock.Anything, userID).Return(nil, nil)

		_, err := domain.ResetPassword(context.Background(), &userv1.CompletePasswordResetRequest{
			Token:       "valid-token",
			NewPassword: "newpassword123",
		})

		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}
