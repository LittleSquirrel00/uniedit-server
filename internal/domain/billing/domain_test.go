package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	billingv1 "github.com/uniedit/server/api/pb/billing"
	"github.com/uniedit/server/internal/model"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type MockPlanDB struct {
	mock.Mock
}

func (m *MockPlanDB) ListActive(ctx context.Context) ([]*model.Plan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Plan), args.Error(1)
}

func (m *MockPlanDB) GetByID(ctx context.Context, id string) (*model.Plan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Plan), args.Error(1)
}

func (m *MockPlanDB) Create(ctx context.Context, plan *model.Plan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockPlanDB) Update(ctx context.Context, plan *model.Plan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

type MockSubscriptionDB struct {
	mock.Mock
}

func (m *MockSubscriptionDB) Create(ctx context.Context, sub *model.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionDB) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscription), args.Error(1)
}

func (m *MockSubscriptionDB) GetByUserIDWithPlan(ctx context.Context, userID uuid.UUID) (*model.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscription), args.Error(1)
}

func (m *MockSubscriptionDB) GetByStripeID(ctx context.Context, stripeSubID string) (*model.Subscription, error) {
	args := m.Called(ctx, stripeSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscription), args.Error(1)
}

func (m *MockSubscriptionDB) Update(ctx context.Context, sub *model.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionDB) UpdateCredits(ctx context.Context, userID uuid.UUID, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

type MockUsageDB struct {
	mock.Mock
}

func (m *MockUsageDB) Create(ctx context.Context, record *model.UsageRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockUsageDB) GetStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*model.UsageStats, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UsageStats), args.Error(1)
}

func (m *MockUsageDB) GetMonthlyTokens(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error) {
	args := m.Called(ctx, userID, start)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUsageDB) GetDailyRequests(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).(int), args.Error(1)
}

type MockQuotaCache struct {
	mock.Mock
}

func (m *MockQuotaCache) GetTokensUsed(ctx context.Context, userID uuid.UUID, periodStart time.Time) (int64, error) {
	args := m.Called(ctx, userID, periodStart)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuotaCache) IncrementTokens(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time, tokens int64) (int64, error) {
	args := m.Called(ctx, userID, periodStart, periodEnd, tokens)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuotaCache) GetRequestsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockQuotaCache) IncrementRequests(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockQuotaCache) ResetTokens(ctx context.Context, userID uuid.UUID, periodStart time.Time) error {
	args := m.Called(ctx, userID, periodStart)
	return args.Error(0)
}

// --- Tests ---

func TestBillingDomain_ListPlans(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)

		domain := NewBillingDomain(mockPlanDB, nil, nil, nil, logger)

		expectedPlans := []*model.Plan{
			{ID: "free", Name: "Free", Active: true},
			{ID: "pro", Name: "Pro", Active: true},
		}

		mockPlanDB.On("ListActive", mock.Anything).Return(expectedPlans, nil)

		out, err := domain.ListPlans(context.Background())

		assert.NoError(t, err)
		assert.Len(t, out.GetPlans(), 2)
		assert.Equal(t, "free", out.GetPlans()[0].GetId())
		mockPlanDB.AssertExpectations(t)
	})
}

func TestBillingDomain_GetPlan(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)

		domain := NewBillingDomain(mockPlanDB, nil, nil, nil, logger)

		expectedPlan := &model.Plan{ID: "pro", Name: "Pro", Active: true}

		mockPlanDB.On("GetByID", mock.Anything, "pro").Return(expectedPlan, nil)

		plan, err := domain.GetPlan(context.Background(), &billingv1.GetByIDRequest{Id: "pro"})

		assert.NoError(t, err)
		assert.Equal(t, "pro", plan.GetId())
		mockPlanDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)

		domain := NewBillingDomain(mockPlanDB, nil, nil, nil, logger)

		mockPlanDB.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		plan, err := domain.GetPlan(context.Background(), &billingv1.GetByIDRequest{Id: "nonexistent"})

		assert.ErrorIs(t, err, ErrPlanNotFound)
		assert.Nil(t, plan)
	})
}

func TestBillingDomain_CreateSubscription(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(mockPlanDB, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		plan := &model.Plan{ID: "pro", Name: "Pro", Active: true}
		createdSub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "pro",
			Status: model.SubscriptionStatusActive,
			Plan:   plan,
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)
		mockPlanDB.On("GetByID", mock.Anything, "pro").Return(plan, nil)
		mockSubDB.On("Create", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)
		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(createdSub, nil)

		sub, err := domain.CreateSubscription(context.Background(), userID, &billingv1.CreateSubscriptionRequest{PlanId: "pro"})

		assert.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, "pro", sub.GetPlanId())
		mockSubDB.AssertExpectations(t)
		mockPlanDB.AssertExpectations(t)
	})

	t.Run("subscription exists", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(mockPlanDB, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		existingSub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "free",
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(existingSub, nil)

		sub, err := domain.CreateSubscription(context.Background(), userID, &billingv1.CreateSubscriptionRequest{PlanId: "pro"})

		assert.ErrorIs(t, err, ErrSubscriptionExists)
		assert.Nil(t, sub)
	})

	t.Run("plan not found", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(mockPlanDB, mockSubDB, nil, nil, logger)

		userID := uuid.New()

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)
		mockPlanDB.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

		sub, err := domain.CreateSubscription(context.Background(), userID, &billingv1.CreateSubscriptionRequest{PlanId: "nonexistent"})

		assert.ErrorIs(t, err, ErrPlanNotFound)
		assert.Nil(t, sub)
	})

	t.Run("plan not active", func(t *testing.T) {
		mockPlanDB := new(MockPlanDB)
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(mockPlanDB, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		inactivePlan := &model.Plan{ID: "legacy", Name: "Legacy", Active: false}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)
		mockPlanDB.On("GetByID", mock.Anything, "legacy").Return(inactivePlan, nil)

		sub, err := domain.CreateSubscription(context.Background(), userID, &billingv1.CreateSubscriptionRequest{PlanId: "legacy"})

		assert.ErrorIs(t, err, ErrPlanNotActive)
		assert.Nil(t, sub)
	})
}

func TestBillingDomain_CancelSubscription(t *testing.T) {
	logger := zap.NewNop()

	t.Run("cancel at period end", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "pro",
			Status: model.SubscriptionStatusActive,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockSubDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

		result, err := domain.CancelSubscription(context.Background(), userID, &billingv1.CancelSubscriptionRequest{Immediately: false})

		assert.NoError(t, err)
		assert.True(t, result.GetCancelAtPeriodEnd())
		assert.NotEmpty(t, result.GetCanceledAt())
		mockSubDB.AssertExpectations(t)
	})

	t.Run("cancel immediately", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "pro",
			Status: model.SubscriptionStatusActive,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockSubDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

		result, err := domain.CancelSubscription(context.Background(), userID, &billingv1.CancelSubscriptionRequest{Immediately: true})

		assert.NoError(t, err)
		assert.Equal(t, string(model.SubscriptionStatusCanceled), result.GetStatus())
		assert.Equal(t, "free", result.GetPlanId())
		mockSubDB.AssertExpectations(t)
	})

	t.Run("already canceled", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "pro",
			Status: model.SubscriptionStatusCanceled,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)

		result, err := domain.CancelSubscription(context.Background(), userID, &billingv1.CancelSubscriptionRequest{Immediately: false})

		assert.ErrorIs(t, err, ErrSubscriptionCanceled)
		assert.Nil(t, result)
	})
}

func TestBillingDomain_CheckQuota(t *testing.T) {
	logger := zap.NewNop()

	t.Run("quota available", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)

		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:            "pro",
			MonthlyTokens: 1000000,
			DailyRequests: 1000,
		}
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "pro",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			Plan:               plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("GetTokensUsed", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(int64(500000), nil)
		mockQuotaCache.On("GetRequestsToday", mock.Anything, userID).Return(500, nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.NoError(t, err)
		mockSubDB.AssertExpectations(t)
		mockQuotaCache.AssertExpectations(t)
	})

	t.Run("token limit reached", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)

		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:            "free",
			MonthlyTokens: 100000,
			DailyRequests: 100,
		}
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "free",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			Plan:               plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("GetTokensUsed", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(int64(100000), nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.ErrorIs(t, err, ErrTokenLimitReached)
	})

	t.Run("request limit reached", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)

		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:            "free",
			MonthlyTokens: 100000,
			DailyRequests: 100,
		}
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "free",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			Plan:               plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("GetTokensUsed", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(int64(50000), nil)
		mockQuotaCache.On("GetRequestsToday", mock.Anything, userID).Return(100, nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.ErrorIs(t, err, ErrRequestLimitReached)
	})

	t.Run("unlimited plan", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)

		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:            "enterprise",
			MonthlyTokens: -1, // Unlimited
			DailyRequests: -1, // Unlimited
		}
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "enterprise",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			Plan:               plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.NoError(t, err)
		mockSubDB.AssertExpectations(t)
	})
}

func TestBillingDomain_AddCredits(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:             uuid.New(),
			UserID:         userID,
			CreditsBalance: 1000,
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)
		mockSubDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

		out, err := domain.AddCredits(context.Background(), &billingv1.AddCreditsRequest{
			UserId: userID.String(),
			Amount: 500,
			Source: "purchase",
		})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockSubDB.AssertExpectations(t)
	})

	t.Run("invalid amount", func(t *testing.T) {
		domain := NewBillingDomain(nil, nil, nil, nil, logger)

		_, err := domain.AddCredits(context.Background(), &billingv1.AddCreditsRequest{
			UserId: uuid.New().String(),
			Amount: -100,
			Source: "refund",
		})

		assert.ErrorIs(t, err, ErrInvalidCreditsAmount)
	})
}

func TestBillingDomain_DeductCredits(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:             uuid.New(),
			UserID:         userID,
			CreditsBalance: 1000,
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)
		mockSubDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)

		err := domain.DeductCredits(context.Background(), userID, 500, "usage")

		assert.NoError(t, err)
		mockSubDB.AssertExpectations(t)
	})

	t.Run("insufficient credits", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)

		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:             uuid.New(),
			UserID:         userID,
			CreditsBalance: 100,
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)

		err := domain.DeductCredits(context.Background(), userID, 500, "usage")

		assert.ErrorIs(t, err, ErrInsufficientCredits)
	})
}

func TestBillingDomain_GetSubscription(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		plan := &model.Plan{ID: "pro", Name: "Pro"}
		sub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			PlanID: "pro",
			Status: model.SubscriptionStatusActive,
			Plan:   plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)

		result, err := domain.GetSubscription(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, "pro", result.GetPlanId())
		mockSubDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(nil, nil)

		result, err := domain.GetSubscription(context.Background(), userID)

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
		assert.Nil(t, result)
	})
}

func TestBillingDomain_GetQuotaStatus(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)
		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:            "pro",
			Name:          "Pro",
			MonthlyTokens: 1000000,
			DailyRequests: 1000,
		}
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "pro",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			CreditsBalance:     5000,
			Plan:               plan,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("GetTokensUsed", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(int64(500000), nil)
		mockQuotaCache.On("GetRequestsToday", mock.Anything, userID).Return(500, nil)

		result, err := domain.GetQuotaStatus(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1000000), result.GetTokensLimit())
		assert.Equal(t, int64(500000), result.GetTokensUsed())
		assert.Equal(t, int32(1000), result.GetRequestsLimit())
		assert.Equal(t, int32(500), result.GetRequestsToday())
		mockSubDB.AssertExpectations(t)
		mockQuotaCache.AssertExpectations(t)
	})
}

func TestBillingDomain_ConsumeQuota(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)
		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			PlanID:             "pro",
			Status:             model.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			CurrentPeriodEnd:   time.Now().AddDate(0, 0, 15),
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("IncrementTokens", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), int64(1000)).Return(int64(501000), nil)
		mockQuotaCache.On("IncrementRequests", mock.Anything, userID).Return(501, nil)

		err := domain.ConsumeQuota(context.Background(), userID, 1000)

		assert.NoError(t, err)
		mockSubDB.AssertExpectations(t)
		mockQuotaCache.AssertExpectations(t)
	})
}

func TestBillingDomain_GetUsageStats(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success with period", func(t *testing.T) {
		mockUsageDB := new(MockUsageDB)
		domain := NewBillingDomain(nil, nil, mockUsageDB, nil, logger)

		userID := uuid.New()
		stats := &model.UsageStats{
			TotalTokens:   1000000,
			TotalRequests: 1000,
			TotalCostUSD:  50.0,
		}

		mockUsageDB.On("GetStats", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

		result, err := domain.GetUsageStats(context.Background(), userID, &billingv1.GetUsageStatsRequest{Period: "month"})

		assert.NoError(t, err)
		assert.Equal(t, int32(1000), result.GetTotalRequests())
		assert.Equal(t, int64(1000000), result.GetTotalTokens())
		mockUsageDB.AssertExpectations(t)
	})

	t.Run("success with custom time range", func(t *testing.T) {
		mockUsageDB := new(MockUsageDB)
		domain := NewBillingDomain(nil, nil, mockUsageDB, nil, logger)

		userID := uuid.New()
		start := time.Now().AddDate(0, -1, 0)
		end := time.Now()
		stats := &model.UsageStats{
			TotalTokens:   500000,
			TotalRequests: 500,
		}

		mockUsageDB.On("GetStats", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

		result, err := domain.GetUsageStats(context.Background(), userID, &billingv1.GetUsageStatsRequest{
			Start: start.UTC().Format(time.RFC3339),
			End:   end.UTC().Format(time.RFC3339),
		})

		assert.NoError(t, err)
		assert.Equal(t, int32(500), result.GetTotalRequests())
		mockUsageDB.AssertExpectations(t)
	})
}

func TestBillingDomain_RecordUsage(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockUsageDB := new(MockUsageDB)
		mockQuotaCache := new(MockQuotaCache)
		domain := NewBillingDomain(nil, mockSubDB, mockUsageDB, mockQuotaCache, logger)

		userID := uuid.New()
		providerID := uuid.New()
		in := &billingv1.RecordUsageRequest{
			RequestId:    "req_123",
			TaskType:     "chat",
			ProviderId:   providerID.String(),
			ModelId:      "gpt-4",
			InputTokens:  100,
			OutputTokens: 200,
			CostUsd:      0.07,
			LatencyMs:    500,
			Success:      true,
		}

		sub := &model.Subscription{
			ID:                 uuid.New(),
			UserID:             userID,
			CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
			CurrentPeriodEnd:   time.Now().AddDate(0, 0, 15),
		}

		mockUsageDB.On("Create", mock.Anything, mock.AnythingOfType("*model.UsageRecord")).Return(nil)
		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)
		mockQuotaCache.On("IncrementTokens", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), int64(300)).Return(int64(300), nil)
		mockQuotaCache.On("IncrementRequests", mock.Anything, userID).Return(1, nil)

		out, err := domain.RecordUsage(context.Background(), userID, in)

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockUsageDB.AssertExpectations(t)
	})

	t.Run("failed request does not consume quota", func(t *testing.T) {
		mockUsageDB := new(MockUsageDB)
		domain := NewBillingDomain(nil, nil, mockUsageDB, nil, logger)

		userID := uuid.New()
		providerID := uuid.New()
		in := &billingv1.RecordUsageRequest{
			RequestId:    "req_456",
			TaskType:     "chat",
			ProviderId:   providerID.String(),
			ModelId:      "gpt-4",
			InputTokens:  100,
			OutputTokens: 0,
			CostUsd:      0,
			LatencyMs:    100,
			Success:      false,
		}

		mockUsageDB.On("Create", mock.Anything, mock.AnythingOfType("*model.UsageRecord")).Return(nil)

		out, err := domain.RecordUsage(context.Background(), userID, in)

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockUsageDB.AssertExpectations(t)
	})
}

func TestBillingDomain_GetBalance(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:             uuid.New(),
			UserID:         userID,
			CreditsBalance: 5000,
		}

		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(sub, nil)

		out, err := domain.GetBalance(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, int64(5000), out.GetBalance())
		mockSubDB.AssertExpectations(t)
	})

	t.Run("no subscription", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)

		out, err := domain.GetBalance(context.Background(), userID)

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
		assert.Nil(t, out)
	})
}

func TestBillingDomain_UpdateSubscriptionFromStripe(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		mockQuotaCache := new(MockQuotaCache)
		domain := NewBillingDomain(nil, mockSubDB, nil, mockQuotaCache, logger)

		stripeSubID := "sub_test123"
		sub := &model.Subscription{
			ID:                   uuid.New(),
			UserID:               uuid.New(),
			StripeSubscriptionID: stripeSubID,
			Status:               model.SubscriptionStatusActive,
			CurrentPeriodStart:   time.Now().AddDate(0, -1, 0),
		}

		periodStart := time.Now()
		periodEnd := time.Now().AddDate(0, 1, 0)

		mockSubDB.On("GetByStripeID", mock.Anything, stripeSubID).Return(sub, nil)
		mockSubDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil)
		mockQuotaCache.On("ResetTokens", mock.Anything, sub.UserID, mock.AnythingOfType("time.Time")).Return(nil)

		err := domain.UpdateSubscriptionFromStripe(context.Background(), stripeSubID, model.SubscriptionStatusCanceled, periodStart, periodEnd, false)

		assert.NoError(t, err)
		assert.Equal(t, model.SubscriptionStatusCanceled, sub.Status)
		mockSubDB.AssertExpectations(t)
	})

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		stripeSubID := "sub_nonexistent"
		mockSubDB.On("GetByStripeID", mock.Anything, stripeSubID).Return(nil, nil)

		err := domain.UpdateSubscriptionFromStripe(context.Background(), stripeSubID, model.SubscriptionStatusActive, time.Now(), time.Now().AddDate(0, 1, 0), false)

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
	})
}

func TestBillingDomain_GetQuotaStatus_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(nil, nil)

		result, err := domain.GetQuotaStatus(context.Background(), userID)

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
		assert.Nil(t, result)
	})
}

func TestBillingDomain_ConsumeQuota_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)

		err := domain.ConsumeQuota(context.Background(), userID, 1000)

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
	})
}

func TestBillingDomain_GetUsageStats_Periods(t *testing.T) {
	logger := zap.NewNop()

	t.Run("day period", func(t *testing.T) {
		mockUsageDB := new(MockUsageDB)
		domain := NewBillingDomain(nil, nil, mockUsageDB, nil, logger)

		userID := uuid.New()
		stats := &model.UsageStats{
			TotalTokens:   100000,
			TotalRequests: 100,
		}

		mockUsageDB.On("GetStats", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

		result, err := domain.GetUsageStats(context.Background(), userID, &billingv1.GetUsageStatsRequest{Period: "day"})

		assert.NoError(t, err)
		assert.Equal(t, int32(100), result.GetTotalRequests())
		mockUsageDB.AssertExpectations(t)
	})

	t.Run("week period", func(t *testing.T) {
		mockUsageDB := new(MockUsageDB)
		domain := NewBillingDomain(nil, nil, mockUsageDB, nil, logger)

		userID := uuid.New()
		stats := &model.UsageStats{
			TotalTokens:   500000,
			TotalRequests: 500,
		}

		mockUsageDB.On("GetStats", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

		result, err := domain.GetUsageStats(context.Background(), userID, &billingv1.GetUsageStatsRequest{Period: "week"})

		assert.NoError(t, err)
		assert.Equal(t, int32(500), result.GetTotalRequests())
		mockUsageDB.AssertExpectations(t)
	})
}

func TestBillingDomain_DeductCredits_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)

		err := domain.DeductCredits(context.Background(), userID, 500, "usage")

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
	})

	t.Run("invalid amount", func(t *testing.T) {
		domain := NewBillingDomain(nil, nil, nil, nil, logger)

		err := domain.DeductCredits(context.Background(), uuid.New(), -100, "refund")

		assert.ErrorIs(t, err, ErrInvalidCreditsAmount)
	})
}

func TestBillingDomain_AddCredits_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserID", mock.Anything, userID).Return(nil, nil)

		_, err := domain.AddCredits(context.Background(), &billingv1.AddCreditsRequest{
			UserId: userID.String(),
			Amount: 500,
			Source: "purchase",
		})

		assert.ErrorIs(t, err, ErrSubscriptionNotFound)
	})
}

func TestBillingDomain_CheckQuota_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("subscription not active", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		sub := &model.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			Status: model.SubscriptionStatusCanceled,
		}

		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(sub, nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.ErrorIs(t, err, ErrQuotaExceeded)
	})

	t.Run("subscription not found", func(t *testing.T) {
		mockSubDB := new(MockSubscriptionDB)
		domain := NewBillingDomain(nil, mockSubDB, nil, nil, logger)

		userID := uuid.New()
		mockSubDB.On("GetByUserIDWithPlan", mock.Anything, userID).Return(nil, nil)

		err := domain.CheckQuota(context.Background(), userID, &billingv1.CheckQuotaRequest{TaskType: "chat"})

		assert.ErrorIs(t, err, ErrQuotaExceeded)
	})
}
