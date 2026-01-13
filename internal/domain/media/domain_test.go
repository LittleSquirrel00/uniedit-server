package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"github.com/uniedit/server/internal/utils/common"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type MockMediaProviderDB struct {
	mock.Mock
}

func (m *MockMediaProviderDB) Create(ctx context.Context, provider *model.MediaProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockMediaProviderDB) FindByID(ctx context.Context, id uuid.UUID) (*model.MediaProvider, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MediaProvider), args.Error(1)
}

func (m *MockMediaProviderDB) FindAll(ctx context.Context) ([]*model.MediaProvider, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaProvider), args.Error(1)
}

func (m *MockMediaProviderDB) FindEnabled(ctx context.Context) ([]*model.MediaProvider, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaProvider), args.Error(1)
}

func (m *MockMediaProviderDB) Update(ctx context.Context, provider *model.MediaProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockMediaProviderDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ outbound.MediaProviderDatabasePort = (*MockMediaProviderDB)(nil)

type MockMediaModelDB struct {
	mock.Mock
}

func (m *MockMediaModelDB) Create(ctx context.Context, model *model.MediaModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockMediaModelDB) FindByID(ctx context.Context, id string) (*model.MediaModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MediaModel), args.Error(1)
}

func (m *MockMediaModelDB) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.MediaModel, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaModel), args.Error(1)
}

func (m *MockMediaModelDB) FindByCapability(ctx context.Context, capability model.MediaCapability) ([]*model.MediaModel, error) {
	args := m.Called(ctx, capability)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaModel), args.Error(1)
}

func (m *MockMediaModelDB) FindEnabled(ctx context.Context) ([]*model.MediaModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaModel), args.Error(1)
}

func (m *MockMediaModelDB) Update(ctx context.Context, model *model.MediaModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockMediaModelDB) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ outbound.MediaModelDatabasePort = (*MockMediaModelDB)(nil)

type MockMediaTaskDB struct {
	mock.Mock
}

func (m *MockMediaTaskDB) Create(ctx context.Context, task *model.MediaTask) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockMediaTaskDB) FindByID(ctx context.Context, id uuid.UUID) (*model.MediaTask, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MediaTask), args.Error(1)
}

func (m *MockMediaTaskDB) FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.MediaTask, error) {
	args := m.Called(ctx, ownerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaTask), args.Error(1)
}

func (m *MockMediaTaskDB) FindPending(ctx context.Context, limit int) ([]*model.MediaTask, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MediaTask), args.Error(1)
}

func (m *MockMediaTaskDB) Update(ctx context.Context, task *model.MediaTask) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockMediaTaskDB) UpdateStatus(ctx context.Context, id uuid.UUID, status model.MediaTaskStatus, progress int, output, errMsg string) error {
	args := m.Called(ctx, id, status, progress, output, errMsg)
	return args.Error(0)
}

func (m *MockMediaTaskDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ outbound.MediaTaskDatabasePort = (*MockMediaTaskDB)(nil)

type MockMediaHealthCache struct {
	mock.Mock
}

func (m *MockMediaHealthCache) GetHealth(ctx context.Context, providerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, providerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMediaHealthCache) SetHealth(ctx context.Context, providerID uuid.UUID, healthy bool) error {
	args := m.Called(ctx, providerID, healthy)
	return args.Error(0)
}

var _ outbound.MediaProviderHealthCachePort = (*MockMediaHealthCache)(nil)

type MockMediaVendorRegistry struct {
	mock.Mock
}

func (m *MockMediaVendorRegistry) Register(adapter outbound.MediaVendorAdapterPort) {
	m.Called(adapter)
}

func (m *MockMediaVendorRegistry) Get(providerType model.MediaProviderType) (outbound.MediaVendorAdapterPort, error) {
	args := m.Called(providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.MediaVendorAdapterPort), args.Error(1)
}

func (m *MockMediaVendorRegistry) GetForProvider(prov *model.MediaProvider) (outbound.MediaVendorAdapterPort, error) {
	args := m.Called(prov)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.MediaVendorAdapterPort), args.Error(1)
}

func (m *MockMediaVendorRegistry) All() []outbound.MediaVendorAdapterPort {
	args := m.Called()
	return args.Get(0).([]outbound.MediaVendorAdapterPort)
}

func (m *MockMediaVendorRegistry) SupportedTypes() []model.MediaProviderType {
	args := m.Called()
	return args.Get(0).([]model.MediaProviderType)
}

var _ outbound.MediaVendorRegistryPort = (*MockMediaVendorRegistry)(nil)

type MockMediaCrypto struct {
	mock.Mock
}

func (m *MockMediaCrypto) Encrypt(plaintext string) (string, error) {
	args := m.Called(plaintext)
	return args.String(0), args.Error(1)
}

func (m *MockMediaCrypto) Decrypt(ciphertext string) (string, error) {
	args := m.Called(ciphertext)
	return args.String(0), args.Error(1)
}

var _ outbound.MediaCryptoPort = (*MockMediaCrypto)(nil)

type MockMediaVendorAdapter struct {
	mock.Mock
}

func (m *MockMediaVendorAdapter) Type() model.MediaProviderType {
	args := m.Called()
	return args.Get(0).(model.MediaProviderType)
}

func (m *MockMediaVendorAdapter) SupportsCapability(cap model.MediaCapability) bool {
	args := m.Called(cap)
	return args.Bool(0)
}

func (m *MockMediaVendorAdapter) HealthCheck(ctx context.Context, prov *model.MediaProvider, apiKey string) error {
	args := m.Called(ctx, prov, apiKey)
	return args.Error(0)
}

func (m *MockMediaVendorAdapter) GenerateImage(ctx context.Context, req *model.ImageRequest, mdl *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.ImageResponse, error) {
	args := m.Called(ctx, req, mdl, prov, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ImageResponse), args.Error(1)
}

func (m *MockMediaVendorAdapter) GenerateVideo(ctx context.Context, req *model.VideoRequest, mdl *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.VideoResponse, error) {
	args := m.Called(ctx, req, mdl, prov, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.VideoResponse), args.Error(1)
}

func (m *MockMediaVendorAdapter) GetVideoStatus(ctx context.Context, taskID string, prov *model.MediaProvider, apiKey string) (*model.VideoStatus, error) {
	args := m.Called(ctx, taskID, prov, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.VideoStatus), args.Error(1)
}

var _ outbound.MediaVendorAdapterPort = (*MockMediaVendorAdapter)(nil)

// --- Tests ---

func TestDomain_GenerateImage(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockMediaProviderDB)
		mockModelDB := new(MockMediaModelDB)
		mockTaskDB := new(MockMediaTaskDB)
		mockHealthCache := new(MockMediaHealthCache)
		mockVendorReg := new(MockMediaVendorRegistry)
		mockCrypto := new(MockMediaCrypto)
		mockAdapter := new(MockMediaVendorAdapter)

		domain := NewDomain(
			mockProviderDB,
			mockModelDB,
			mockTaskDB,
			nil,
			mockHealthCache,
			mockVendorReg,
			mockCrypto,
			nil,
			logger,
		)

		userID := uuid.New()
		providerID := uuid.New()

		mediaModel := &model.MediaModel{
			ID:           "dall-e-3",
			ProviderID:   providerID,
			Name:         "DALL-E 3",
			Capabilities: []model.MediaCapability{model.MediaCapabilityImage},
			Enabled:      true,
		}

		provider := &model.MediaProvider{
			ID:           providerID,
			Name:         "OpenAI",
			Type:         model.MediaProviderTypeOpenAI,
			BaseURL:      "https://api.openai.com",
			EncryptedKey: "encrypted-key",
			Enabled:      true,
		}

		expectedResp := &model.ImageResponse{
			Images: []*model.GeneratedImage{
				{URL: "https://example.com/image.png"},
			},
			Model:     "dall-e-3",
			CreatedAt: time.Now().Unix(),
		}

		mockModelDB.On("FindByID", mock.Anything, "dall-e-3").Return(mediaModel, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockHealthCache.On("GetHealth", mock.Anything, providerID).Return(true, nil)
		mockCrypto.On("Decrypt", "encrypted-key").Return("sk-test-key", nil)
		mockVendorReg.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("GenerateImage", mock.Anything, mock.AnythingOfType("*model.ImageRequest"), mediaModel, provider, "sk-test-key").Return(expectedResp, nil)

		input := &mediav1.GenerateImageRequest{
			Prompt: "A beautiful sunset",
			Model:  "dall-e-3",
		}

		output, err := domain.GenerateImage(context.Background(), userID, input)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.GetImages(), 1)
		assert.Equal(t, "dall-e-3", output.GetModel())
		mockModelDB.AssertExpectations(t)
		mockProviderDB.AssertExpectations(t)
		mockAdapter.AssertExpectations(t)
	})

	t.Run("empty prompt", func(t *testing.T) {
		domain := NewDomain(nil, nil, nil, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		input := &mediav1.GenerateImageRequest{Prompt: ""}

		output, err := domain.GenerateImage(context.Background(), userID, input)

		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, output)
	})

	t.Run("model not found", func(t *testing.T) {
		mockModelDB := new(MockMediaModelDB)

		domain := NewDomain(
			nil,
			mockModelDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		mockModelDB.On("FindByID", mock.Anything, "unknown-model").Return(nil, nil)

		input := &mediav1.GenerateImageRequest{Prompt: "Test prompt", Model: "unknown-model"}

		output, err := domain.GenerateImage(context.Background(), userID, input)

		assert.ErrorIs(t, err, ErrModelNotFound)
		assert.Nil(t, output)
	})

	t.Run("provider unhealthy", func(t *testing.T) {
		mockProviderDB := new(MockMediaProviderDB)
		mockModelDB := new(MockMediaModelDB)
		mockHealthCache := new(MockMediaHealthCache)

		domain := NewDomain(
			mockProviderDB,
			mockModelDB,
			nil,
			nil,
			mockHealthCache,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		providerID := uuid.New()

		mediaModel := &model.MediaModel{
			ID:           "dall-e-3",
			ProviderID:   providerID,
			Capabilities: []model.MediaCapability{model.MediaCapabilityImage},
		}

		provider := &model.MediaProvider{
			ID:      providerID,
			Enabled: true,
		}

		mockModelDB.On("FindByID", mock.Anything, "dall-e-3").Return(mediaModel, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockHealthCache.On("GetHealth", mock.Anything, providerID).Return(false, nil)

		input := &mediav1.GenerateImageRequest{Prompt: "Test prompt", Model: "dall-e-3"}

		output, err := domain.GenerateImage(context.Background(), userID, input)

		assert.ErrorIs(t, err, ErrProviderUnhealthy)
		assert.Nil(t, output)
	})
}

func TestDomain_GenerateVideo(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		mockTaskDB.On("Create", mock.Anything, mock.AnythingOfType("*model.MediaTask")).Return(nil)

		input := &mediav1.GenerateVideoRequest{
			Prompt:   "A beautiful sunset video",
			Duration: 10,
		}

		output, err := domain.GenerateVideo(context.Background(), userID, input)

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotEmpty(t, output.GetTaskId())
		assert.Equal(t, string(model.VideoStatePending), output.GetStatus())
		mockTaskDB.AssertExpectations(t)
	})

	t.Run("invalid input", func(t *testing.T) {
		domain := NewDomain(nil, nil, nil, nil, nil, nil, nil, nil, logger)

		userID := uuid.New()
		input := &mediav1.GenerateVideoRequest{}

		output, err := domain.GenerateVideo(context.Background(), userID, input)

		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, output)
	})
}

func TestDomain_GetVideoStatus(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		taskID := uuid.New()

		task := &model.MediaTask{
			ID:        taskID,
			OwnerID:   userID,
			Type:      "video_generation",
			Status:    model.MediaTaskStatusRunning,
			Progress:  50,
			CreatedAt: time.Now(),
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

		output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, string(model.VideoStateProcessing), output.GetStatus())
		assert.Equal(t, int32(50), output.GetProgress())
	})

	t.Run("task not found", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		taskID := uuid.New()

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(nil, nil)

		output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.ErrorIs(t, err, ErrTaskNotFound)
		assert.Nil(t, output)
	})

	t.Run("task not owned", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		taskID := uuid.New()

		task := &model.MediaTask{
			ID:      taskID,
			OwnerID: otherUserID, // Different owner
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

		output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.ErrorIs(t, err, ErrTaskNotOwned)
		assert.Nil(t, output)
	})
}

func TestDomain_GetTask(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		taskID := uuid.New()
		now := time.Now()

		task := &model.MediaTask{
			ID:        taskID,
			OwnerID:   userID,
			Type:      "image_generation",
			Status:    model.MediaTaskStatusCompleted,
			Progress:  100,
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

		output, err := domain.GetTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, taskID.String(), output.GetId())
		assert.Equal(t, string(model.MediaTaskStatusCompleted), output.GetStatus())
	})
}

func TestDomain_ListTasks(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		now := time.Now()

		tasks := []*model.MediaTask{
			{ID: uuid.New(), OwnerID: userID, Type: "image", Status: model.MediaTaskStatusCompleted, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), OwnerID: userID, Type: "video", Status: model.MediaTaskStatusRunning, CreatedAt: now, UpdatedAt: now},
		}

		mockTaskDB.On("FindByOwner", mock.Anything, userID, 20, 0).Return(tasks, nil)

		output, err := domain.ListTasks(context.Background(), userID, &mediav1.ListTasksRequest{Limit: 20, Offset: 0})

		assert.NoError(t, err)
		assert.Len(t, output.GetTasks(), 2)
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()

		mockTaskDB.On("FindByOwner", mock.Anything, userID, 100, 0).Return([]*model.MediaTask{}, nil)

		_, err := domain.ListTasks(context.Background(), userID, &mediav1.ListTasksRequest{Limit: 200, Offset: 0})

		assert.NoError(t, err)
		mockTaskDB.AssertCalled(t, "FindByOwner", mock.Anything, userID, 100, 0)
	})
}

func TestDomain_CancelTask(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		taskID := uuid.New()

		task := &model.MediaTask{
			ID:       taskID,
			OwnerID:  userID,
			Status:   model.MediaTaskStatusPending,
			Progress: 0,
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)
		mockTaskDB.On("UpdateStatus", mock.Anything, taskID, model.MediaTaskStatusCancelled, 0, "", "cancelled by user").Return(nil)

		_, err := domain.CancelTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.NoError(t, err)
		mockTaskDB.AssertExpectations(t)
	})

	t.Run("task already completed", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		taskID := uuid.New()

		task := &model.MediaTask{
			ID:      taskID,
			OwnerID: userID,
			Status:  model.MediaTaskStatusCompleted,
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

		_, err := domain.CancelTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.ErrorIs(t, err, ErrTaskAlreadyCompleted)
	})

	t.Run("task not owned", func(t *testing.T) {
		mockTaskDB := new(MockMediaTaskDB)

		domain := NewDomain(
			nil,
			nil,
			mockTaskDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		userID := uuid.New()
		otherUserID := uuid.New()
		taskID := uuid.New()

		task := &model.MediaTask{
			ID:      taskID,
			OwnerID: otherUserID,
			Status:  model.MediaTaskStatusPending,
		}

		mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

		_, err := domain.CancelTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

		assert.ErrorIs(t, err, ErrTaskNotOwned)
	})
}

func TestDomain_GetProvider(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockMediaProviderDB)

		domain := NewDomain(
			mockProviderDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		providerID := uuid.New()
		provider := &model.MediaProvider{
			ID:      providerID,
			Name:    "OpenAI",
			Type:    model.MediaProviderTypeOpenAI,
			Enabled: true,
		}

		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)

		result, err := domain.GetProvider(context.Background(), &mediav1.GetByIDRequest{Id: providerID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "OpenAI", result.GetName())
	})

	t.Run("not found", func(t *testing.T) {
		mockProviderDB := new(MockMediaProviderDB)

		domain := NewDomain(
			mockProviderDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		providerID := uuid.New()
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(nil, nil)

		result, err := domain.GetProvider(context.Background(), &mediav1.GetByIDRequest{Id: providerID.String()})

		assert.ErrorIs(t, err, ErrProviderNotFound)
		assert.Nil(t, result)
	})
}

func TestDomain_ListProviders(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockMediaProviderDB)

		domain := NewDomain(
			mockProviderDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		providers := []*model.MediaProvider{
			{ID: uuid.New(), Name: "OpenAI"},
			{ID: uuid.New(), Name: "Anthropic"},
		}

		mockProviderDB.On("FindAll", mock.Anything).Return(providers, nil)

		result, err := domain.ListProviders(context.Background(), &commonv1.Empty{})

		assert.NoError(t, err)
		assert.Len(t, result.GetProviders(), 2)
	})
}

func TestDomain_ListModels(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockMediaModelDB)

		domain := NewDomain(
			nil,
			mockModelDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		models := []*model.MediaModel{
			{ID: "dall-e-3", Capabilities: []model.MediaCapability{model.MediaCapabilityImage}},
			{ID: "dall-e-2", Capabilities: []model.MediaCapability{model.MediaCapabilityImage}},
		}

		mockModelDB.On("FindByCapability", mock.Anything, model.MediaCapabilityImage).Return(models, nil)

		result, err := domain.ListModels(context.Background(), &mediav1.ListModelsRequest{Capability: string(model.MediaCapabilityImage)})

		assert.NoError(t, err)
		assert.Len(t, result.GetModels(), 2)
	})

	t.Run("empty capability returns empty list", func(t *testing.T) {
		mockModelDB := new(MockMediaModelDB)

		domain := NewDomain(
			nil,
			mockModelDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		result, err := domain.ListModels(context.Background(), &mediav1.ListModelsRequest{})

		assert.NoError(t, err)
		assert.Len(t, result.GetModels(), 0)
	})
}

func TestTaskStatusToVideoState(t *testing.T) {
	tests := []struct {
		input    model.MediaTaskStatus
		expected model.VideoState
	}{
		{model.MediaTaskStatusPending, model.VideoStatePending},
		{model.MediaTaskStatusRunning, model.VideoStateProcessing},
		{model.MediaTaskStatusCompleted, model.VideoStateCompleted},
		{model.MediaTaskStatusFailed, model.VideoStateFailed},
		{model.MediaTaskStatusCancelled, model.VideoStateFailed},
		{"unknown", model.VideoStatePending}, // default case
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := taskStatusToVideoState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDomain_GetVideoStatus_Completed(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	taskID := uuid.New()

	task := &model.MediaTask{
		ID:        taskID,
		OwnerID:   userID,
		Type:      "video_generation",
		Status:    model.MediaTaskStatusCompleted,
		Progress:  100,
		Output:    common.NewString(`{"url":"https://example.com/video.mp4","duration":10}`),
		CreatedAt: time.Now(),
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

	output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, string(model.VideoStateCompleted), output.GetStatus())
	assert.Equal(t, int32(100), output.GetProgress())
	assert.NotNil(t, output.GetVideo())
}

func TestDomain_GetVideoStatus_Failed(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	taskID := uuid.New()

	task := &model.MediaTask{
		ID:        taskID,
		OwnerID:   userID,
		Type:      "video_generation",
		Status:    model.MediaTaskStatusFailed,
		Error:     "generation failed",
		CreatedAt: time.Now(),
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

	output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, string(model.VideoStateFailed), output.GetStatus())
	assert.Equal(t, "generation failed", output.GetError())
}

func TestDomain_GetVideoStatus_InvalidTaskID(t *testing.T) {
	logger := zap.NewNop()
	domain := NewDomain(nil, nil, nil, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()

	output, err := domain.GetVideoStatus(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: "invalid-uuid"})

	assert.ErrorIs(t, err, ErrInvalidInput)
	assert.Nil(t, output)
}

func TestDomain_CancelTask_AlreadyCancelled(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	taskID := uuid.New()

	task := &model.MediaTask{
		ID:      taskID,
		OwnerID: userID,
		Status:  model.MediaTaskStatusCancelled,
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

	_, err := domain.CancelTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.ErrorIs(t, err, ErrTaskAlreadyCancelled)
}

func TestDomain_CancelTask_Failed(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	taskID := uuid.New()

	task := &model.MediaTask{
		ID:      taskID,
		OwnerID: userID,
		Status:  model.MediaTaskStatusFailed,
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

	_, err := domain.CancelTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.ErrorIs(t, err, ErrTaskAlreadyCompleted)
}

func TestDomain_GetTask_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	taskID := uuid.New()

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(nil, nil)

	output, err := domain.GetTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.ErrorIs(t, err, ErrTaskNotFound)
	assert.Nil(t, output)
}

func TestDomain_GetTask_NotOwned(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	otherUserID := uuid.New()
	taskID := uuid.New()

	task := &model.MediaTask{
		ID:      taskID,
		OwnerID: otherUserID,
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)

	output, err := domain.GetTask(context.Background(), userID, &mediav1.GetByTaskIDRequest{TaskId: taskID.String()})

	assert.ErrorIs(t, err, ErrTaskNotOwned)
	assert.Nil(t, output)
}

func TestDomain_GenerateImage_CapabilityNotSupported(t *testing.T) {
	logger := zap.NewNop()
	mockModelDB := new(MockMediaModelDB)

	domain := NewDomain(nil, mockModelDB, nil, nil, nil, nil, nil, nil, logger)

	userID := uuid.New()
	providerID := uuid.New()

	// Model that only supports video, not image
	mediaModel := &model.MediaModel{
		ID:           "video-model",
		ProviderID:   providerID,
		Capabilities: []model.MediaCapability{model.MediaCapabilityVideo},
		Enabled:      true,
	}

	mockModelDB.On("FindByID", mock.Anything, "video-model").Return(mediaModel, nil)

	input := &mediav1.GenerateImageRequest{Prompt: "Test prompt", Model: "video-model"}

	output, err := domain.GenerateImage(context.Background(), userID, input)

	assert.ErrorIs(t, err, ErrCapabilityNotSupported)
	assert.Nil(t, output)
}

func TestDomain_GenerateImage_AutoSelectModel(t *testing.T) {
	logger := zap.NewNop()
	mockProviderDB := new(MockMediaProviderDB)
	mockModelDB := new(MockMediaModelDB)
	mockHealthCache := new(MockMediaHealthCache)
	mockVendorReg := new(MockMediaVendorRegistry)
	mockCrypto := new(MockMediaCrypto)
	mockAdapter := new(MockMediaVendorAdapter)

	domain := NewDomain(
		mockProviderDB,
		mockModelDB,
		nil,
		nil,
		mockHealthCache,
		mockVendorReg,
		mockCrypto,
		nil,
		logger,
	)

	userID := uuid.New()
	providerID := uuid.New()

	mediaModel := &model.MediaModel{
		ID:           "dall-e-3",
		ProviderID:   providerID,
		Name:         "DALL-E 3",
		Capabilities: []model.MediaCapability{model.MediaCapabilityImage},
		Enabled:      true,
	}

	provider := &model.MediaProvider{
		ID:           providerID,
		Name:         "OpenAI",
		Type:         model.MediaProviderTypeOpenAI,
		BaseURL:      "https://api.openai.com",
		EncryptedKey: "encrypted-key",
		Enabled:      true,
	}

	expectedResp := &model.ImageResponse{
		Images: []*model.GeneratedImage{
			{URL: "https://example.com/image.png"},
		},
		Model:     "dall-e-3",
		CreatedAt: time.Now().Unix(),
	}

	// Auto-select returns models by capability
	mockModelDB.On("FindByCapability", mock.Anything, model.MediaCapabilityImage).Return([]*model.MediaModel{mediaModel}, nil)
	mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
	mockHealthCache.On("GetHealth", mock.Anything, providerID).Return(true, nil)
	mockCrypto.On("Decrypt", "encrypted-key").Return("sk-test-key", nil)
	mockVendorReg.On("GetForProvider", provider).Return(mockAdapter, nil)
	mockAdapter.On("GenerateImage", mock.Anything, mock.AnythingOfType("*model.ImageRequest"), mediaModel, provider, "sk-test-key").Return(expectedResp, nil)

	input := &mediav1.GenerateImageRequest{
		Prompt: "A beautiful sunset",
		Model:  "auto", // Auto-select
	}

	output, err := domain.GenerateImage(context.Background(), userID, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.GetImages(), 1)
}

func TestDomain_FindModelWithCapability_NoModels(t *testing.T) {
	logger := zap.NewNop()
	mockModelDB := new(MockMediaModelDB)

	domain := NewDomain(nil, mockModelDB, nil, nil, nil, nil, nil, nil, logger)

	mockModelDB.On("FindByCapability", mock.Anything, model.MediaCapabilityImage).Return([]*model.MediaModel{}, nil)

	userID := uuid.New()
	input := &mediav1.GenerateImageRequest{Prompt: "Test prompt", Model: "auto"}

	output, err := domain.GenerateImage(context.Background(), userID, input)

	assert.ErrorIs(t, err, ErrModelNotFound)
	assert.Nil(t, output)
}

func TestDomain_FindModelWithCapability_NoHealthyProvider(t *testing.T) {
	logger := zap.NewNop()
	mockProviderDB := new(MockMediaProviderDB)
	mockModelDB := new(MockMediaModelDB)
	mockHealthCache := new(MockMediaHealthCache)

	domain := NewDomain(
		mockProviderDB,
		mockModelDB,
		nil,
		nil,
		mockHealthCache,
		nil,
		nil,
		nil,
		logger,
	)

	userID := uuid.New()
	providerID := uuid.New()

	mediaModel := &model.MediaModel{
		ID:           "dall-e-3",
		ProviderID:   providerID,
		Capabilities: []model.MediaCapability{model.MediaCapabilityImage},
		Enabled:      true,
	}

	provider := &model.MediaProvider{
		ID:      providerID,
		Enabled: true,
	}

	mockModelDB.On("FindByCapability", mock.Anything, model.MediaCapabilityImage).Return([]*model.MediaModel{mediaModel}, nil)
	mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
	mockHealthCache.On("GetHealth", mock.Anything, providerID).Return(false, nil) // Unhealthy

	input := &mediav1.GenerateImageRequest{Prompt: "Test prompt", Model: "auto"}

	output, err := domain.GenerateImage(context.Background(), userID, input)

	assert.ErrorIs(t, err, ErrNoHealthyProvider)
	assert.Nil(t, output)
}

func TestDomain_ExecuteVideoTask_Success(t *testing.T) {
	logger := zap.NewNop()
	mockProviderDB := new(MockMediaProviderDB)
	mockModelDB := new(MockMediaModelDB)
	mockTaskDB := new(MockMediaTaskDB)
	mockHealthCache := new(MockMediaHealthCache)
	mockVendorReg := new(MockMediaVendorRegistry)
	mockCrypto := new(MockMediaCrypto)
	mockAdapter := new(MockMediaVendorAdapter)

	config := DefaultConfig()
	config.VideoPollInterval = 1 * time.Millisecond // Speed up for testing

	domain := NewDomain(
		mockProviderDB,
		mockModelDB,
		mockTaskDB,
		nil,
		mockHealthCache,
		mockVendorReg,
		mockCrypto,
		config,
		logger,
	)

	taskID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()

	task := &model.MediaTask{
		ID:        taskID,
		OwnerID:   userID,
		Type:      "video_generation",
		Status:    model.MediaTaskStatusPending,
		Input:     common.NewString(`{"prompt":"test video","model":"video-model"}`),
		CreatedAt: time.Now(),
	}

	mediaModel := &model.MediaModel{
		ID:           "video-model",
		ProviderID:   providerID,
		Capabilities: []model.MediaCapability{model.MediaCapabilityVideo},
		Enabled:      true,
	}

	provider := &model.MediaProvider{
		ID:           providerID,
		Type:         model.MediaProviderTypeOpenAI,
		EncryptedKey: "encrypted-key",
		Enabled:      true,
	}

	videoResp := &model.VideoResponse{
		TaskID: "provider-task-123",
	}

	completedStatus := &model.VideoStatus{
		Status:   model.VideoStateCompleted,
		Progress: 100,
		Video:    &model.GeneratedVideo{URL: "https://example.com/video.mp4"},
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)
	mockTaskDB.On("UpdateStatus", mock.Anything, taskID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockModelDB.On("FindByID", mock.Anything, "video-model").Return(mediaModel, nil)
	mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
	mockHealthCache.On("GetHealth", mock.Anything, providerID).Return(true, nil)
	mockCrypto.On("Decrypt", "encrypted-key").Return("sk-test-key", nil)
	mockVendorReg.On("GetForProvider", provider).Return(mockAdapter, nil)
	mockAdapter.On("GenerateVideo", mock.Anything, mock.AnythingOfType("*model.VideoRequest"), mediaModel, provider, "sk-test-key").Return(videoResp, nil)
	mockAdapter.On("GetVideoStatus", mock.Anything, "provider-task-123", provider, "sk-test-key").Return(completedStatus, nil)

	err := domain.ExecuteVideoTask(context.Background(), taskID)

	assert.NoError(t, err)
}

func TestDomain_ExecuteVideoTask_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	taskID := uuid.New()
	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(nil, nil)

	err := domain.ExecuteVideoTask(context.Background(), taskID)

	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestDomain_ExecuteVideoTask_InvalidInput(t *testing.T) {
	logger := zap.NewNop()
	mockTaskDB := new(MockMediaTaskDB)

	domain := NewDomain(nil, nil, mockTaskDB, nil, nil, nil, nil, nil, logger)

	taskID := uuid.New()
	task := &model.MediaTask{
		ID:     taskID,
		Status: model.MediaTaskStatusPending,
		Input:  common.NewString("invalid json"),
	}

	mockTaskDB.On("FindByID", mock.Anything, taskID).Return(task, nil)
	mockTaskDB.On("UpdateStatus", mock.Anything, taskID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := domain.ExecuteVideoTask(context.Background(), taskID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal input")
}
