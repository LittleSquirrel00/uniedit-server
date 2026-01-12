package ai

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	aiv1 "github.com/uniedit/server/api/pb/ai"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// ===== Mock Implementations =====

type MockProviderDB struct {
	mock.Mock
}

func (m *MockProviderDB) FindByID(ctx context.Context, id uuid.UUID) (*model.AIProvider, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIProvider), args.Error(1)
}

func (m *MockProviderDB) FindByName(ctx context.Context, name string) (*model.AIProvider, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIProvider), args.Error(1)
}

func (m *MockProviderDB) FindAll(ctx context.Context) ([]*model.AIProvider, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProvider), args.Error(1)
}

func (m *MockProviderDB) FindEnabled(ctx context.Context) ([]*model.AIProvider, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProvider), args.Error(1)
}

func (m *MockProviderDB) Create(ctx context.Context, provider *model.AIProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockProviderDB) Update(ctx context.Context, provider *model.AIProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockProviderDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProviderDB) FindByType(ctx context.Context, providerType model.AIProviderType) ([]*model.AIProvider, error) {
	args := m.Called(ctx, providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProvider), args.Error(1)
}

type MockModelDB struct {
	mock.Mock
}

func (m *MockModelDB) FindByID(ctx context.Context, id string) (*model.AIModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIModel), args.Error(1)
}

func (m *MockModelDB) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIModel, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModel), args.Error(1)
}

func (m *MockModelDB) FindEnabled(ctx context.Context) ([]*model.AIModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModel), args.Error(1)
}

func (m *MockModelDB) FindByCapability(ctx context.Context, cap model.AICapability) ([]*model.AIModel, error) {
	args := m.Called(ctx, cap)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModel), args.Error(1)
}

func (m *MockModelDB) FindByCapabilities(ctx context.Context, caps []model.AICapability) ([]*model.AIModel, error) {
	args := m.Called(ctx, caps)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModel), args.Error(1)
}

func (m *MockModelDB) Create(ctx context.Context, model *model.AIModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockModelDB) Update(ctx context.Context, model *model.AIModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockModelDB) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockModelDB) DeleteByProvider(ctx context.Context, providerID uuid.UUID) error {
	args := m.Called(ctx, providerID)
	return args.Error(0)
}

type MockAccountDB struct {
	mock.Mock
}

func (m *MockAccountDB) FindByID(ctx context.Context, id uuid.UUID) (*model.AIProviderAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIProviderAccount), args.Error(1)
}

func (m *MockAccountDB) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProviderAccount), args.Error(1)
}

func (m *MockAccountDB) FindAvailableByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProviderAccount), args.Error(1)
}

func (m *MockAccountDB) Create(ctx context.Context, account *model.AIProviderAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountDB) Update(ctx context.Context, account *model.AIProviderAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountDB) DeleteByProvider(ctx context.Context, providerID uuid.UUID) error {
	args := m.Called(ctx, providerID)
	return args.Error(0)
}

func (m *MockAccountDB) UpdateHealth(ctx context.Context, id uuid.UUID, status model.AIHealthStatus, failures int) error {
	args := m.Called(ctx, id, status, failures)
	return args.Error(0)
}

func (m *MockAccountDB) IncrementUsage(ctx context.Context, id uuid.UUID, requests int64, tokens int64, cost float64) error {
	args := m.Called(ctx, id, requests, tokens, cost)
	return args.Error(0)
}

func (m *MockAccountDB) FindActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	args := m.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIProviderAccount), args.Error(1)
}

type MockGroupDB struct {
	mock.Mock
}

func (m *MockGroupDB) FindByID(ctx context.Context, id string) (*model.AIModelGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIModelGroup), args.Error(1)
}

func (m *MockGroupDB) FindAll(ctx context.Context) ([]*model.AIModelGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModelGroup), args.Error(1)
}

func (m *MockGroupDB) FindByTaskType(ctx context.Context, taskType model.AITaskType) ([]*model.AIModelGroup, error) {
	args := m.Called(ctx, taskType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModelGroup), args.Error(1)
}

func (m *MockGroupDB) Create(ctx context.Context, group *model.AIModelGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockGroupDB) Update(ctx context.Context, group *model.AIModelGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockGroupDB) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGroupDB) FindEnabled(ctx context.Context) ([]*model.AIModelGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AIModelGroup), args.Error(1)
}

type MockHealthCache struct {
	mock.Mock
}

func (m *MockHealthCache) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, providerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockHealthCache) SetProviderHealth(ctx context.Context, providerID uuid.UUID, healthy bool, ttl time.Duration) error {
	args := m.Called(ctx, providerID, healthy, ttl)
	return args.Error(0)
}

func (m *MockHealthCache) GetAccountHealth(ctx context.Context, accountID uuid.UUID) (model.AIHealthStatus, error) {
	args := m.Called(ctx, accountID)
	return args.Get(0).(model.AIHealthStatus), args.Error(1)
}

func (m *MockHealthCache) SetAccountHealth(ctx context.Context, accountID uuid.UUID, status model.AIHealthStatus, ttl time.Duration) error {
	args := m.Called(ctx, accountID, status, ttl)
	return args.Error(0)
}

func (m *MockHealthCache) InvalidateAccountHealth(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *MockHealthCache) InvalidateProviderHealth(ctx context.Context, providerID uuid.UUID) error {
	args := m.Called(ctx, providerID)
	return args.Error(0)
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

type MockVendorRegistry struct {
	mock.Mock
}

func (m *MockVendorRegistry) GetForProvider(provider *model.AIProvider) (outbound.AIVendorAdapterPort, error) {
	args := m.Called(provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.AIVendorAdapterPort), args.Error(1)
}

func (m *MockVendorRegistry) Register(adapter outbound.AIVendorAdapterPort) {
	m.Called(adapter)
}

func (m *MockVendorRegistry) Get(providerType model.AIProviderType) (outbound.AIVendorAdapterPort, error) {
	args := m.Called(providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.AIVendorAdapterPort), args.Error(1)
}

func (m *MockVendorRegistry) SupportedTypes() []model.AIProviderType {
	args := m.Called()
	return args.Get(0).([]model.AIProviderType)
}

type MockVendorAdapter struct {
	mock.Mock
}

func (m *MockVendorAdapter) Type() model.AIProviderType {
	args := m.Called()
	return args.Get(0).(model.AIProviderType)
}

func (m *MockVendorAdapter) SupportsCapability(cap model.AICapability) bool {
	args := m.Called(cap)
	return args.Bool(0)
}

func (m *MockVendorAdapter) HealthCheck(ctx context.Context, provider *model.AIProvider, apiKey string) error {
	args := m.Called(ctx, provider, apiKey)
	return args.Error(0)
}

func (m *MockVendorAdapter) Chat(ctx context.Context, req *model.AIChatRequest, mod *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIChatResponse, error) {
	args := m.Called(ctx, req, mod, p, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIChatResponse), args.Error(1)
}

func (m *MockVendorAdapter) ChatStream(ctx context.Context, req *model.AIChatRequest, mod *model.AIModel, p *model.AIProvider, apiKey string) (<-chan *model.AIChatChunk, error) {
	args := m.Called(ctx, req, mod, p, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *model.AIChatChunk), args.Error(1)
}

func (m *MockVendorAdapter) Embed(ctx context.Context, req *model.AIEmbedRequest, mod *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIEmbedResponse, error) {
	args := m.Called(ctx, req, mod, p, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AIEmbedResponse), args.Error(1)
}

// ===== Test Helpers =====

func newTestDomain(
	providerDB *MockProviderDB,
	modelDB *MockModelDB,
	accountDB *MockAccountDB,
	groupDB *MockGroupDB,
) AIDomain {
	logger := zap.NewNop()

	// Use typed nil interfaces to handle nil checks properly in domain
	var pdb outbound.AIProviderDatabasePort
	if providerDB != nil {
		pdb = providerDB
	}

	var mdb outbound.AIModelDatabasePort
	if modelDB != nil {
		mdb = modelDB
	}

	var adb outbound.AIProviderAccountDatabasePort
	if accountDB != nil {
		adb = accountDB
	}

	var gdb outbound.AIModelGroupDatabasePort
	if groupDB != nil {
		gdb = groupDB
	}

	return NewAIDomain(
		pdb,
		mdb,
		adb,
		gdb,
		nil, // healthCache
		nil, // embeddingCache
		nil, // vendorRegistry
		nil, // crypto
		nil, // usageRecorder
		DefaultConfig(),
		logger,
	)
}

func createTestProvider(id uuid.UUID, name string) *model.AIProvider {
	return &model.AIProvider{
		ID:       id,
		Name:     name,
		Type:     model.AIProviderTypeOpenAI,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "test-key",
		Enabled:  true,
		Weight:   1,
		Priority: 0,
	}
}

func createTestModel(id string, providerID uuid.UUID) *model.AIModel {
	return &model.AIModel{
		ID:              id,
		ProviderID:      providerID,
		Name:            id,
		Capabilities:    pq.StringArray{string(model.AICapabilityChat), string(model.AICapabilityStream)},
		ContextWindow:   128000,
		MaxOutputTokens: 4096,
		InputCostPer1K:  0.01,
		OutputCostPer1K: 0.03,
		Enabled:         true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestAccount(id, providerID uuid.UUID) *model.AIProviderAccount {
	return &model.AIProviderAccount{
		ID:              id,
		ProviderID:      providerID,
		Name:            "Test Account",
		EncryptedAPIKey: "encrypted-key",
		KeyPrefix:       "sk-xxx",
		Weight:          1,
		Priority:        10,
		IsActive:        true,
		HealthStatus:    model.AIHealthStatusHealthy,
	}
}

func createTestGroup(id string) *model.AIModelGroup {
	return &model.AIModelGroup{
		ID:       id,
		Name:     "Test Group",
		TaskType: model.AITaskTypeChat,
		Models:   pq.StringArray{"gpt-4", "gpt-3.5"},
		Strategy: &model.AIStrategyConfig{
			Type: model.AIStrategyPriority,
		},
		Enabled: true,
	}
}

// ===== Provider Management Tests =====

func TestAIDomain_GetProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		domain := newTestDomain(mockProviderDB, nil, nil, nil)

		providerID := uuid.New()
		expected := createTestProvider(providerID, "openai")

		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(expected, nil)

		result, err := domain.GetProvider(context.Background(), providerID)

		assert.NoError(t, err)
		assert.Equal(t, expected.ID.String(), result.GetId())
		assert.Equal(t, "openai", result.GetName())
		mockProviderDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		domain := newTestDomain(mockProviderDB, nil, nil, nil)

		providerID := uuid.New()
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(nil, nil)

		result, err := domain.GetProvider(context.Background(), providerID)

		assert.ErrorIs(t, err, ErrProviderNotFound)
		assert.Nil(t, result)
	})
}

func TestAIDomain_ListProviders(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		domain := newTestDomain(mockProviderDB, nil, nil, nil)

		providers := []*model.AIProvider{
			createTestProvider(uuid.New(), "openai"),
			createTestProvider(uuid.New(), "anthropic"),
		}

		mockProviderDB.On("FindAll", mock.Anything).Return(providers, nil)

		result, err := domain.ListProviders(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result.GetData(), 2)
		mockProviderDB.AssertExpectations(t)
	})
}

func TestAIDomain_CreateProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		domain := newTestDomain(mockProviderDB, nil, nil, nil)

		in := &aiv1.CreateProviderRequest{
			Name:     "new-provider",
			Type:     string(model.AIProviderTypeGeneric),
			BaseUrl:   "https://api.example.com",
			ApiKey:    "sk-test",
			Enabled:   true,
			Weight:    1,
			Priority:  10,
			RateLimit: &aiv1.RateLimitConfig{Rpm: 60, Tpm: 1000, DailyLimit: 10000},
		}

		mockProviderDB.On("Create", mock.Anything, mock.AnythingOfType("*model.AIProvider")).Return(nil)

		out, err := domain.CreateProvider(context.Background(), in)

		assert.NoError(t, err)
		assert.NotEmpty(t, out.GetId())
		assert.Equal(t, "new-provider", out.GetName())
		mockProviderDB.AssertExpectations(t)
	})
}

func TestAIDomain_UpdateProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		domain := newTestDomain(mockProviderDB, nil, nil, nil)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		provider.Enabled = true

		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockProviderDB.On("Update", mock.Anything, provider).Return(nil)

		out, err := domain.UpdateProvider(context.Background(), providerID, &aiv1.UpdateProviderRequest{
			Id:      providerID.String(),
			Name:    &commonv1.StringValue{Value: "openai-new"},
			Enabled: &commonv1.BoolValue{Value: false},
		})

		assert.NoError(t, err)
		assert.Equal(t, providerID.String(), out.GetId())
		assert.Equal(t, "openai-new", out.GetName())
		assert.False(t, out.GetEnabled())
		mockProviderDB.AssertExpectations(t)
	})
}

func TestAIDomain_DeleteProvider(t *testing.T) {
	t.Run("success cascades", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(mockProviderDB, mockModelDB, mockAccountDB, nil)

		providerID := uuid.New()

		mockModelDB.On("DeleteByProvider", mock.Anything, providerID).Return(nil)
		mockAccountDB.On("DeleteByProvider", mock.Anything, providerID).Return(nil)
		mockProviderDB.On("Delete", mock.Anything, providerID).Return(nil)

		resp, err := domain.DeleteProvider(context.Background(), providerID)

		assert.NoError(t, err)
		assert.Equal(t, "provider deleted", resp.GetMessage())
		mockModelDB.AssertExpectations(t)
		mockAccountDB.AssertExpectations(t)
		mockProviderDB.AssertExpectations(t)
	})
}

// ===== Model Management Tests =====

func TestAIDomain_GetModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		expected := createTestModel("gpt-4", uuid.New())
		mockModelDB.On("FindByID", mock.Anything, "gpt-4").Return(expected, nil)

		result, err := domain.GetModel(context.Background(), "gpt-4")

		assert.NoError(t, err)
		assert.Equal(t, "gpt-4", result.GetId())
		mockModelDB.AssertExpectations(t)
	})
}

func TestAIDomain_ListModels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		models := []*model.AIModel{
			createTestModel("gpt-4", uuid.New()),
			createTestModel("gpt-3.5", uuid.New()),
		}

		mockModelDB.On("FindEnabled", mock.Anything).Return(models, nil)

		result, err := domain.ListModels(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result.GetData(), 2)
		mockModelDB.AssertExpectations(t)
	})
}

func TestAIDomain_ListModelsByCapability(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		models := []*model.AIModel{
			createTestModel("gpt-4-vision", uuid.New()),
		}

		mockModelDB.On("FindByCapability", mock.Anything, model.AICapabilityVision).Return(models, nil)

		result, err := domain.ListModelsByCapability(context.Background(), model.AICapabilityVision)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockModelDB.AssertExpectations(t)
	})
}

func TestAIDomain_CreateModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		providerID := uuid.New()
		mockModelDB.On("Create", mock.Anything, mock.AnythingOfType("*model.AIModel")).Return(nil)

		out, err := domain.CreateModel(context.Background(), &aiv1.CreateModelRequest{
			Id:               "new-model",
			ProviderId:       providerID.String(),
			Name:             "new-model",
			Capabilities:     []string{string(model.AICapabilityChat)},
			ContextWindow:    128000,
			MaxOutputTokens:  4096,
			InputCostPer_1K:  0.01,
			OutputCostPer_1K: 0.03,
			Enabled:          true,
		})

		assert.NoError(t, err)
		assert.Equal(t, "new-model", out.GetId())
		mockModelDB.AssertExpectations(t)
	})
}

func TestAIDomain_UpdateModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		m := createTestModel("gpt-4", uuid.New())
		mockModelDB.On("FindByID", mock.Anything, "gpt-4").Return(m, nil)
		mockModelDB.On("Update", mock.Anything, m).Return(nil)

		out, err := domain.UpdateModel(context.Background(), "gpt-4", &aiv1.UpdateModelRequest{
			Id:      "gpt-4",
			Enabled: &commonv1.BoolValue{Value: false},
		})

		assert.NoError(t, err)
		assert.Equal(t, "gpt-4", out.GetId())
		assert.False(t, out.GetEnabled())
		mockModelDB.AssertExpectations(t)
	})
}

func TestAIDomain_DeleteModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		mockModelDB.On("Delete", mock.Anything, "gpt-4").Return(nil)

		resp, err := domain.DeleteModel(context.Background(), "gpt-4")

		assert.NoError(t, err)
		assert.Equal(t, "model deleted", resp.GetMessage())
		mockModelDB.AssertExpectations(t)
	})
}

// ===== Account Management Tests =====

func TestAIDomain_GetAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		accountID := uuid.New()
		expected := createTestAccount(accountID, uuid.New())
		mockAccountDB.On("FindByID", mock.Anything, accountID).Return(expected, nil)

		result, err := domain.GetAccount(context.Background(), accountID)

		assert.NoError(t, err)
		assert.Equal(t, accountID, result.ID)
		mockAccountDB.AssertExpectations(t)
	})

	t.Run("no account db", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		result, err := domain.GetAccount(context.Background(), uuid.New())

		assert.ErrorIs(t, err, ErrAccountNotFound)
		assert.Nil(t, result)
	})
}

func TestAIDomain_ListAccounts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		providerID := uuid.New()
		accounts := []*model.AIProviderAccount{
			createTestAccount(uuid.New(), providerID),
			createTestAccount(uuid.New(), providerID),
		}

		mockAccountDB.On("FindByProvider", mock.Anything, providerID).Return(accounts, nil)

		result, err := domain.ListAccounts(context.Background(), providerID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockAccountDB.AssertExpectations(t)
	})

	t.Run("no account db returns nil", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		result, err := domain.ListAccounts(context.Background(), uuid.New())

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestAIDomain_CreateAccount(t *testing.T) {
	t.Run("success with encryption", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		mockCrypto := new(MockCrypto)
		logger := zap.NewNop()

		domain := NewAIDomain(
			nil, nil, mockAccountDB, nil,
			nil, nil, nil, mockCrypto, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		account := &model.AIProviderAccount{
			ProviderID: providerID,
			Name:       "New Account",
			IsActive:   true,
		}

		mockCrypto.On("Encrypt", "sk-test-key").Return("encrypted-key", nil)
		mockAccountDB.On("Create", mock.Anything, mock.AnythingOfType("*model.AIProviderAccount")).Return(nil)

		err := domain.CreateAccount(context.Background(), account, "sk-test-key")

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, account.ID)
		assert.Equal(t, "encrypted-key", account.EncryptedAPIKey)
		mockCrypto.AssertExpectations(t)
		mockAccountDB.AssertExpectations(t)
	})

	t.Run("no account db", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		err := domain.CreateAccount(context.Background(), &model.AIProviderAccount{}, "key")

		assert.ErrorIs(t, err, ErrAdapterNotFound)
	})
}

func TestAIDomain_UpdateAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		account := createTestAccount(uuid.New(), uuid.New())
		account.IsActive = false
		mockAccountDB.On("Update", mock.Anything, account).Return(nil)

		err := domain.UpdateAccount(context.Background(), account)

		assert.NoError(t, err)
		mockAccountDB.AssertExpectations(t)
	})
}

func TestAIDomain_DeleteAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		accountID := uuid.New()
		mockAccountDB.On("Delete", mock.Anything, accountID).Return(nil)

		err := domain.DeleteAccount(context.Background(), accountID)

		assert.NoError(t, err)
		mockAccountDB.AssertExpectations(t)
	})
}

func TestAIDomain_ResetAccountHealth(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		accountID := uuid.New()
		mockAccountDB.On("UpdateHealth", mock.Anything, accountID, model.AIHealthStatusHealthy, 0).Return(nil)

		err := domain.ResetAccountHealth(context.Background(), accountID)

		assert.NoError(t, err)
		mockAccountDB.AssertExpectations(t)
	})
}

// ===== Group Management Tests =====

func TestAIDomain_GetGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(nil, nil, nil, mockGroupDB)

		expected := createTestGroup("chat-default")
		mockGroupDB.On("FindByID", mock.Anything, "chat-default").Return(expected, nil)

		result, err := domain.GetGroup(context.Background(), "chat-default")

		assert.NoError(t, err)
		assert.Equal(t, "chat-default", result.ID)
		mockGroupDB.AssertExpectations(t)
	})
}

func TestAIDomain_ListGroups(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(nil, nil, nil, mockGroupDB)

		groups := []*model.AIModelGroup{
			createTestGroup("chat-default"),
			createTestGroup("embedding-default"),
		}

		mockGroupDB.On("FindAll", mock.Anything).Return(groups, nil)

		result, err := domain.ListGroups(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockGroupDB.AssertExpectations(t)
	})
}

func TestAIDomain_CreateGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(nil, nil, nil, mockGroupDB)

		group := createTestGroup("new-group")
		mockGroupDB.On("Create", mock.Anything, group).Return(nil)

		err := domain.CreateGroup(context.Background(), group)

		assert.NoError(t, err)
		mockGroupDB.AssertExpectations(t)
	})
}

func TestAIDomain_UpdateGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(nil, nil, nil, mockGroupDB)

		group := createTestGroup("chat-default")
		group.Enabled = false
		mockGroupDB.On("Update", mock.Anything, group).Return(nil)

		err := domain.UpdateGroup(context.Background(), group)

		assert.NoError(t, err)
		mockGroupDB.AssertExpectations(t)
	})
}

func TestAIDomain_DeleteGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(nil, nil, nil, mockGroupDB)

		mockGroupDB.On("Delete", mock.Anything, "chat-default").Return(nil)

		err := domain.DeleteGroup(context.Background(), "chat-default")

		assert.NoError(t, err)
		mockGroupDB.AssertExpectations(t)
	})
}

// ===== Public API Tests =====

func TestAIDomain_ListEnabledModels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		models := []*model.AIModel{
			createTestModel("gpt-4", uuid.New()),
		}

		mockModelDB.On("FindEnabled", mock.Anything).Return(models, nil)

		result, err := domain.ListEnabledModels(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result.GetModels(), 1)
		mockModelDB.AssertExpectations(t)
	})
}

// ===== Health Monitoring Tests =====

func TestAIDomain_IsProviderHealthy(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		domain.healthMu.Lock()
		domain.healthStatus[providerID] = true
		domain.healthMu.Unlock()

		assert.True(t, domain.IsProviderHealthy(providerID))
	})

	t.Run("unhealthy", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		domain.healthMu.Lock()
		domain.healthStatus[providerID] = false
		domain.healthMu.Unlock()

		assert.False(t, domain.IsProviderHealthy(providerID))
	})

	t.Run("unknown", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		// No health status set

		assert.False(t, domain.IsProviderHealthy(providerID))
	})
}

func TestAIDomain_IsAccountHealthy(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		accountID := uuid.New()
		domain.healthMu.Lock()
		domain.accountHealth[accountID] = model.AIHealthStatusHealthy
		domain.healthMu.Unlock()

		assert.True(t, domain.IsAccountHealthy(accountID))
	})

	t.Run("degraded can serve", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		accountID := uuid.New()
		domain.healthMu.Lock()
		domain.accountHealth[accountID] = model.AIHealthStatusDegraded
		domain.healthMu.Unlock()

		assert.True(t, domain.IsAccountHealthy(accountID))
	})

	t.Run("unhealthy cannot serve", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		accountID := uuid.New()
		domain.healthMu.Lock()
		domain.accountHealth[accountID] = model.AIHealthStatusUnhealthy
		domain.healthMu.Unlock()

		assert.False(t, domain.IsAccountHealthy(accountID))
	})
}

// ===== Chat Tests =====

func TestAIDomain_Chat(t *testing.T) {
	t.Run("empty messages error", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		req := &aiv1.ChatRequest{Model: "gpt-4"}

		result, err := domain.Chat(context.Background(), uuid.New(), req)

		assert.ErrorIs(t, err, ErrEmptyMessages)
		assert.Nil(t, result)
	})
}

func TestAIDomain_ChatStream(t *testing.T) {
	t.Run("empty messages error", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		req := &aiv1.ChatRequest{Model: "gpt-4", Stream: true}

		chunks, info, err := domain.ChatStream(context.Background(), uuid.New(), req)

		assert.ErrorIs(t, err, ErrEmptyMessages)
		assert.Nil(t, chunks)
		assert.Nil(t, info)
	})
}

// ===== Embed Tests =====

func TestAIDomain_Embed(t *testing.T) {
	t.Run("empty input error", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		req := &model.AIEmbedRequest{
			Model: "text-embedding-3-small",
			Input: []string{},
		}

		result, err := domain.Embed(context.Background(), uuid.New(), req)

		assert.ErrorIs(t, err, ErrEmptyInput)
		assert.Nil(t, result)
	})
}

// ===== Routing Tests =====

func TestAIDomain_Route(t *testing.T) {
	t.Run("no candidates error", func(t *testing.T) {
		mockModelDB := new(MockModelDB)
		domain := newTestDomain(nil, mockModelDB, nil, nil)

		// Route calls FindByCapabilities because NewAIRoutingContext sets TaskType="chat"
		// which makes RequiredCapabilities() return [AICapabilityChat]
		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return([]*model.AIModel{}, nil)

		routingCtx := model.NewAIRoutingContext()
		result, err := domain.Route(context.Background(), routingCtx)

		assert.ErrorIs(t, err, ErrNoAvailableModels)
		assert.Nil(t, result)
	})

	t.Run("success with candidates", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(mockProviderDB, mockModelDB, mockAccountDB, nil)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		models := []*model.AIModel{
			createTestModel("gpt-4", providerID),
		}

		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return(models, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockAccountDB.On("FindAvailableByProvider", mock.Anything, providerID).Return([]*model.AIProviderAccount{}, nil)

		routingCtx := model.NewAIRoutingContext()
		result, err := domain.Route(context.Background(), routingCtx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "gpt-4", result.Model.ID)
		assert.Equal(t, provider.APIKey, result.APIKey)
	})

	t.Run("with group", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		mockGroupDB := new(MockGroupDB)
		domain := newTestDomain(mockProviderDB, mockModelDB, mockAccountDB, mockGroupDB)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")

		group := &model.AIModelGroup{
			ID:       "chat-default",
			Name:     "Chat Default",
			TaskType: model.AITaskTypeChat,
			Models:   pq.StringArray{"gpt-4"},
			Strategy: &model.AIStrategyConfig{Type: model.AIStrategyPriority},
			Enabled:  true,
		}

		gpt4 := createTestModel("gpt-4", providerID)

		mockGroupDB.On("FindByID", mock.Anything, "chat-default").Return(group, nil)
		mockModelDB.On("FindByID", mock.Anything, "gpt-4").Return(gpt4, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockAccountDB.On("FindAvailableByProvider", mock.Anything, providerID).Return([]*model.AIProviderAccount{}, nil)

		routingCtx := model.NewAIRoutingContext()
		routingCtx.GroupID = "chat-default"

		result, err := domain.Route(context.Background(), routingCtx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "gpt-4", result.Model.ID)
	})
}

// ===== Cost Calculation Tests =====

func TestAIDomain_CalculateCost(t *testing.T) {
	domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

	t.Run("nil usage returns zero", func(t *testing.T) {
		m := createTestModel("gpt-4", uuid.New())
		cost := domain.calculateCost(m, nil)
		assert.Equal(t, float64(0), cost)
	})

	t.Run("calculates cost correctly", func(t *testing.T) {
		m := createTestModel("gpt-4", uuid.New())
		m.InputCostPer1K = 0.01  // $0.01 per 1K input tokens
		m.OutputCostPer1K = 0.03 // $0.03 per 1K output tokens

		usage := &model.AIUsage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		}

		cost := domain.calculateCost(m, usage)

		// Expected: (1000/1000 * 0.01) + (500/1000 * 0.03) = 0.01 + 0.015 = 0.025
		assert.InDelta(t, 0.025, cost, 0.0001)
	})
}

// ===== Config Tests =====

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
}

// ===== SyncModels Tests =====

func TestAIDomain_SyncModels(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		resp, err := domain.SyncModels(context.Background(), uuid.New())

		assert.NoError(t, err)
		assert.Equal(t, "synced", resp.GetMessage())
	})
}

// ===== GetAccountStats Tests =====

func TestAIDomain_GetAccountStats(t *testing.T) {
	t.Run("returns nil with account db", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		stats, err := domain.GetAccountStats(context.Background(), uuid.New(), 7)

		assert.NoError(t, err)
		assert.Nil(t, stats)
	})

	t.Run("returns error without account db", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		stats, err := domain.GetAccountStats(context.Background(), uuid.New(), 7)

		assert.ErrorIs(t, err, ErrAdapterNotFound)
		assert.Nil(t, stats)
	})
}

// ===== ListAccountsByProvider Tests =====

func TestAIDomain_ListAccountsByProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(nil, nil, mockAccountDB, nil)

		providerID := uuid.New()
		accounts := []*model.AIProviderAccount{
			createTestAccount(uuid.New(), providerID),
		}

		mockAccountDB.On("FindByProvider", mock.Anything, providerID).Return(accounts, nil)

		result, err := domain.ListAccountsByProvider(context.Background(), providerID)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockAccountDB.AssertExpectations(t)
	})

	t.Run("no account db returns nil", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		result, err := domain.ListAccountsByProvider(context.Background(), uuid.New())

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

// ===== SelectAccount Tests =====

func TestAIDomain_SelectAccount(t *testing.T) {
	t.Run("empty accounts returns nil", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		result := domain.selectAccount([]*model.AIProviderAccount{})

		assert.Nil(t, result)
	})

	t.Run("selects highest priority", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		accounts := []*model.AIProviderAccount{
			{ID: uuid.New(), ProviderID: providerID, Priority: 10},
			{ID: uuid.New(), ProviderID: providerID, Priority: 50},
			{ID: uuid.New(), ProviderID: providerID, Priority: 30},
		}

		result := domain.selectAccount(accounts)

		assert.NotNil(t, result)
		assert.Equal(t, 50, result.Priority)
	})

	t.Run("single account returns it", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		account := &model.AIProviderAccount{ID: uuid.New(), Priority: 10}
		result := domain.selectAccount([]*model.AIProviderAccount{account})

		assert.Equal(t, account.ID, result.ID)
	})
}

// ===== BuildRoutingContext Tests =====

func TestAIDomain_BuildRoutingContext(t *testing.T) {
	t.Run("basic chat request", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		req := &model.AIChatRequest{
			Model: "gpt-4",
			Messages: []*model.AIChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := domain.buildRoutingContext(req)

		assert.Equal(t, string(model.AITaskTypeChat), ctx.TaskType)
		assert.False(t, ctx.RequireStream)
		assert.False(t, ctx.RequireVision)
		assert.False(t, ctx.RequireTools)
		assert.Equal(t, []string{"gpt-4"}, ctx.PreferredModels)
	})

	t.Run("stream request", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		req := &model.AIChatRequest{
			Model:  "gpt-4",
			Stream: true,
			Messages: []*model.AIChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := domain.buildRoutingContext(req)

		assert.True(t, ctx.RequireStream)
	})

	t.Run("request with tools", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		req := &model.AIChatRequest{
			Model: "gpt-4",
			Messages: []*model.AIChatMessage{
				{Role: "user", Content: "Hello"},
			},
			Tools: []*model.AITool{
				{Type: "function", Function: &model.AIFunction{Name: "test"}},
			},
		}

		ctx := domain.buildRoutingContext(req)

		assert.True(t, ctx.RequireTools)
	})

	t.Run("auto model selection", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		req := &model.AIChatRequest{
			Model: "auto",
			Messages: []*model.AIChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := domain.buildRoutingContext(req)

		assert.Len(t, ctx.PreferredModels, 0)
	})
}

// ===== UpdateAccount Error Tests =====

func TestAIDomain_UpdateAccount_NoAccountDB(t *testing.T) {
	domain := newTestDomain(nil, nil, nil, nil)

	err := domain.UpdateAccount(context.Background(), &model.AIProviderAccount{})

	assert.ErrorIs(t, err, ErrAdapterNotFound)
}

// ===== DeleteAccount Error Tests =====

func TestAIDomain_DeleteAccount_NoAccountDB(t *testing.T) {
	domain := newTestDomain(nil, nil, nil, nil)

	err := domain.DeleteAccount(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrAdapterNotFound)
}

// ===== ResetAccountHealth Error Tests =====

func TestAIDomain_ResetAccountHealth_NoAccountDB(t *testing.T) {
	domain := newTestDomain(nil, nil, nil, nil)

	err := domain.ResetAccountHealth(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrAdapterNotFound)
}

// ===== DeleteProvider With Nil AccountDB =====

func TestAIDomain_DeleteProvider_NoAccountDB(t *testing.T) {
	mockProviderDB := new(MockProviderDB)
	mockModelDB := new(MockModelDB)
	domain := newTestDomain(mockProviderDB, mockModelDB, nil, nil)

	providerID := uuid.New()

	mockModelDB.On("DeleteByProvider", mock.Anything, providerID).Return(nil)
	mockProviderDB.On("Delete", mock.Anything, providerID).Return(nil)

	resp, err := domain.DeleteProvider(context.Background(), providerID)

	assert.NoError(t, err)
	assert.Equal(t, "provider deleted", resp.GetMessage())
	mockModelDB.AssertExpectations(t)
	mockProviderDB.AssertExpectations(t)
}

// ===== Resolve API Key Tests =====

func TestAIDomain_ResolveAPIKey(t *testing.T) {
	t.Run("uses provider API key when no accounts", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		domain := newTestDomain(mockProviderDB, mockModelDB, mockAccountDB, nil).(*aiDomain)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		provider.APIKey = "provider-api-key"

		mockAccountDB.On("FindAvailableByProvider", mock.Anything, providerID).Return([]*model.AIProviderAccount{}, nil)

		result := &model.AIRoutingResult{
			Provider: provider,
			Model:    createTestModel("gpt-4", providerID),
		}

		err := domain.resolveAPIKey(context.Background(), result)

		assert.NoError(t, err)
		assert.Equal(t, "provider-api-key", result.APIKey)
		assert.Nil(t, result.AccountID)
	})

	t.Run("uses account API key when available", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		mockCrypto := new(MockCrypto)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, mockModelDB, mockAccountDB, nil,
			nil, nil, nil, mockCrypto, nil,
			DefaultConfig(), logger,
		).(*aiDomain)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")

		account := createTestAccount(uuid.New(), providerID)
		account.EncryptedAPIKey = "encrypted-key"

		mockAccountDB.On("FindAvailableByProvider", mock.Anything, providerID).Return([]*model.AIProviderAccount{account}, nil)
		mockCrypto.On("Decrypt", "encrypted-key").Return("decrypted-api-key", nil)

		result := &model.AIRoutingResult{
			Provider: provider,
			Model:    createTestModel("gpt-4", providerID),
		}

		err := domain.resolveAPIKey(context.Background(), result)

		assert.NoError(t, err)
		assert.Equal(t, "decrypted-api-key", result.APIKey)
		assert.NotNil(t, result.AccountID)
		assert.Equal(t, account.ID.String(), *result.AccountID)
	})

	t.Run("no account db uses provider key", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		provider.APIKey = "provider-key"

		result := &model.AIRoutingResult{
			Provider: provider,
			Model:    createTestModel("gpt-4", providerID),
		}

		err := domain.resolveAPIKey(context.Background(), result)

		assert.NoError(t, err)
		assert.Equal(t, "provider-key", result.APIKey)
		assert.Nil(t, result.AccountID)
	})
}

// ===== Route with Account Selection =====

func TestAIDomain_Route_WithAccountSelection(t *testing.T) {
	t.Run("selects account when available", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockAccountDB := new(MockAccountDB)
		mockCrypto := new(MockCrypto)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, mockModelDB, mockAccountDB, nil,
			nil, nil, nil, mockCrypto, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		accountID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		models := []*model.AIModel{createTestModel("gpt-4", providerID)}
		account := createTestAccount(accountID, providerID)

		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return(models, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockAccountDB.On("FindAvailableByProvider", mock.Anything, providerID).Return([]*model.AIProviderAccount{account}, nil)
		mockCrypto.On("Decrypt", "encrypted-key").Return("decrypted-key", nil)

		routingCtx := model.NewAIRoutingContext()
		result, err := domain.Route(context.Background(), routingCtx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "decrypted-key", result.APIKey)
		assert.NotNil(t, result.AccountID)
		assert.Equal(t, accountID.String(), *result.AccountID)
	})
}

// ===== ProviderHealthCheck Tests =====

func TestAIDomain_ProviderHealthCheck(t *testing.T) {
	t.Run("healthy provider", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockRegistry := new(MockVendorRegistry)
		mockAdapter := new(MockVendorAdapter)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, nil, nil, nil,
			nil, nil, mockRegistry, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")

		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockRegistry.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("HealthCheck", mock.Anything, provider, provider.APIKey).Return(nil)

		resp, err := domain.ProviderHealthCheck(context.Background(), providerID)

		assert.NoError(t, err)
		assert.True(t, resp.GetHealthy())
	})

	t.Run("unhealthy provider", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockRegistry := new(MockVendorRegistry)
		mockAdapter := new(MockVendorAdapter)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, nil, nil, nil,
			nil, nil, mockRegistry, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")

		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockRegistry.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("HealthCheck", mock.Anything, provider, provider.APIKey).Return(assert.AnError)

		resp, err := domain.ProviderHealthCheck(context.Background(), providerID)

		assert.NoError(t, err)
		assert.False(t, resp.GetHealthy())
	})

	t.Run("provider not found", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, nil, nil, nil,
			nil, nil, nil, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(nil, nil)

		resp, err := domain.ProviderHealthCheck(context.Background(), providerID)

		assert.ErrorIs(t, err, ErrProviderNotFound)
		assert.Nil(t, resp)
	})
}

// ===== UpdateProviderHealth Tests =====

func TestAIDomain_UpdateProviderHealth(t *testing.T) {
	t.Run("updates health status", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil).(*aiDomain)

		providerID := uuid.New()
		domain.updateProviderHealth(providerID, true)

		assert.True(t, domain.IsProviderHealthy(providerID))

		domain.updateProviderHealth(providerID, false)
		assert.False(t, domain.IsProviderHealthy(providerID))
	})

	t.Run("with health cache", func(t *testing.T) {
		mockHealthCache := new(MockHealthCache)
		logger := zap.NewNop()

		domain := NewAIDomain(
			nil, nil, nil, nil,
			mockHealthCache, nil, nil, nil, nil,
			DefaultConfig(), logger,
		).(*aiDomain)

		providerID := uuid.New()
		mockHealthCache.On("SetProviderHealth", mock.Anything, providerID, true, mock.Anything).Return(nil)

		domain.updateProviderHealth(providerID, true)

		mockHealthCache.AssertExpectations(t)
	})
}

// ===== StopHealthMonitor Tests =====

func TestAIDomain_StopHealthMonitor(t *testing.T) {
	t.Run("stops without starting", func(t *testing.T) {
		domain := newTestDomain(nil, nil, nil, nil)

		// Should not panic
		domain.StopHealthMonitor()
	})
}

// ===== Chat Success Path Tests =====

func TestAIDomain_Chat_Success(t *testing.T) {
	t.Run("successful chat completion", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockRegistry := new(MockVendorRegistry)
		mockAdapter := new(MockVendorAdapter)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, mockModelDB, nil, nil,
			nil, nil, mockRegistry, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		m := createTestModel("gpt-4", providerID)
		userID := uuid.New()

		req := &aiv1.ChatRequest{
			Model: "gpt-4",
			Messages: []*aiv1.ChatMessage{
				{Role: "user", Content: structpb.NewStringValue("Hello")},
			},
		}

		expectedResp := &model.AIChatResponse{
			ID:    "chat-123",
			Model: "gpt-4",
			Message: &model.AIChatMessage{
				Role:    "assistant",
				Content: "Hi there!",
			},
			Usage: &model.AIUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return([]*model.AIModel{m}, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockRegistry.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("Chat", mock.Anything, mock.AnythingOfType("*model.AIChatRequest"), m, provider, provider.APIKey).Return(expectedResp, nil)

		resp, err := domain.Chat(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.GetMessage())
		assert.NotNil(t, resp.GetMessage().GetContent())
		assert.Equal(t, "Hi there!", resp.GetMessage().GetContent().GetStringValue())
	})
}

// ===== ChatStream Success Path Tests =====

func TestAIDomain_ChatStream_Success(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockRegistry := new(MockVendorRegistry)
		mockAdapter := new(MockVendorAdapter)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, mockModelDB, nil, nil,
			nil, nil, mockRegistry, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		m := createTestModel("gpt-4", providerID)
		userID := uuid.New()

		req := &aiv1.ChatRequest{
			Model:  "gpt-4",
			Stream: true,
			Messages: []*aiv1.ChatMessage{
				{Role: "user", Content: structpb.NewStringValue("Hello")},
			},
		}

		chunkChan := make(chan *model.AIChatChunk, 1)
		chunkChan <- &model.AIChatChunk{
			ID:    "chunk-1",
			Model: "gpt-4",
			Delta: &model.AIDelta{Content: "Hi"},
		}
		close(chunkChan)

		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return([]*model.AIModel{m}, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockRegistry.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("ChatStream", mock.Anything, mock.AnythingOfType("*model.AIChatRequest"), m, provider, provider.APIKey).Return((<-chan *model.AIChatChunk)(chunkChan), nil)

		chunks, info, err := domain.ChatStream(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, chunks)
		assert.NotNil(t, info)

		// Consume chunks
		for range chunks {
		}
	})
}

// ===== Embed Success Path Tests =====

func TestAIDomain_Embed_Success(t *testing.T) {
	t.Run("successful embedding", func(t *testing.T) {
		mockProviderDB := new(MockProviderDB)
		mockModelDB := new(MockModelDB)
		mockRegistry := new(MockVendorRegistry)
		mockAdapter := new(MockVendorAdapter)
		logger := zap.NewNop()

		domain := NewAIDomain(
			mockProviderDB, mockModelDB, nil, nil,
			nil, nil, mockRegistry, nil, nil,
			DefaultConfig(), logger,
		)

		providerID := uuid.New()
		provider := createTestProvider(providerID, "openai")
		// Model needs AICapabilityChat because RequiredCapabilities() always includes it
		m := createTestModel("text-embedding-3-small", providerID)
		userID := uuid.New()

		req := &model.AIEmbedRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		expectedResp := &model.AIEmbedResponse{
			Model: "text-embedding-3-small",
			Embeddings: [][]float64{
				{0.1, 0.2, 0.3},
			},
			Usage: &model.AIUsage{
				PromptTokens: 2,
				TotalTokens:  2,
			},
		}

		mockModelDB.On("FindByCapabilities", mock.Anything, mock.Anything).Return([]*model.AIModel{m}, nil)
		mockProviderDB.On("FindByID", mock.Anything, providerID).Return(provider, nil)
		mockRegistry.On("GetForProvider", provider).Return(mockAdapter, nil)
		mockAdapter.On("Embed", mock.Anything, req, m, provider, provider.APIKey).Return(expectedResp, nil)

		resp, err := domain.Embed(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Embeddings, 1)
	})
}
