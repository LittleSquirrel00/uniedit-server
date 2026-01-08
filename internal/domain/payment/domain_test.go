package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// --- Mock Implementations ---

type MockPaymentDatabasePort struct {
	mock.Mock
}

func (m *MockPaymentDatabasePort) Create(ctx context.Context, payment *model.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentDatabasePort) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentDatabasePort) FindByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Payment, error) {
	args := m.Called(ctx, paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentDatabasePort) FindByTradeNo(ctx context.Context, tradeNo string) (*model.Payment, error) {
	args := m.Called(ctx, tradeNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentDatabasePort) FindByFilter(ctx context.Context, filter model.PaymentFilter) ([]*model.Payment, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*model.Payment), args.Get(1).(int64), args.Error(2)
}

func (m *MockPaymentDatabasePort) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.Payment, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).([]*model.Payment), args.Error(1)
}

func (m *MockPaymentDatabasePort) Update(ctx context.Context, payment *model.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

type MockWebhookEventDatabasePort struct {
	mock.Mock
}

func (m *MockWebhookEventDatabasePort) Create(ctx context.Context, event *model.WebhookEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockWebhookEventDatabasePort) Exists(ctx context.Context, provider, eventID string) (bool, error) {
	args := m.Called(ctx, provider, eventID)
	return args.Bool(0), args.Error(1)
}

func (m *MockWebhookEventDatabasePort) MarkProcessed(ctx context.Context, id uuid.UUID, err error) error {
	args := m.Called(ctx, id, err)
	return args.Error(0)
}

type MockPaymentProviderPort struct {
	mock.Mock
}

func (m *MockPaymentProviderPort) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPaymentProviderPort) CreateCustomer(ctx context.Context, email, name string) (*model.ProviderCustomer, error) {
	args := m.Called(ctx, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderCustomer), args.Error(1)
}

func (m *MockPaymentProviderPort) GetCustomer(ctx context.Context, customerID string) (*model.ProviderCustomer, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderCustomer), args.Error(1)
}

func (m *MockPaymentProviderPort) CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*model.ProviderPaymentIntent, error) {
	args := m.Called(ctx, amount, currency, customerID, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderPaymentIntent), args.Error(1)
}

func (m *MockPaymentProviderPort) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*model.ProviderPaymentIntent, error) {
	args := m.Called(ctx, paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderPaymentIntent), args.Error(1)
}

func (m *MockPaymentProviderPort) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	args := m.Called(ctx, paymentIntentID)
	return args.Error(0)
}

func (m *MockPaymentProviderPort) CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*model.ProviderRefund, error) {
	args := m.Called(ctx, chargeID, amount, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderRefund), args.Error(1)
}

func (m *MockPaymentProviderPort) ListPaymentMethods(ctx context.Context, customerID string) ([]*model.PaymentMethodInfo, error) {
	args := m.Called(ctx, customerID)
	return args.Get(0).([]*model.PaymentMethodInfo), args.Error(1)
}

func (m *MockPaymentProviderPort) VerifyWebhookSignature(payload []byte, signature string) error {
	args := m.Called(payload, signature)
	return args.Error(0)
}

type MockPaymentProviderRegistryPort struct {
	mock.Mock
}

func (m *MockPaymentProviderRegistryPort) Get(name string) (outbound.PaymentProviderPort, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.PaymentProviderPort), args.Error(1)
}

func (m *MockPaymentProviderRegistryPort) GetNative(name string) (outbound.NativePaymentProviderPort, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.NativePaymentProviderPort), args.Error(1)
}

func (m *MockPaymentProviderRegistryPort) GetNativeByMethod(method string) (outbound.NativePaymentProviderPort, error) {
	args := m.Called(method)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(outbound.NativePaymentProviderPort), args.Error(1)
}

func (m *MockPaymentProviderRegistryPort) Register(provider outbound.PaymentProviderPort) {
	m.Called(provider)
}

func (m *MockPaymentProviderRegistryPort) RegisterNative(provider outbound.NativePaymentProviderPort) {
	m.Called(provider)
}

type MockOrderReaderPort struct {
	mock.Mock
}

func (m *MockOrderReaderPort) GetOrder(ctx context.Context, id uuid.UUID) (*outbound.PaymentOrderInfo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.PaymentOrderInfo), args.Error(1)
}

func (m *MockOrderReaderPort) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*outbound.PaymentOrderInfo, error) {
	args := m.Called(ctx, paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.PaymentOrderInfo), args.Error(1)
}

func (m *MockOrderReaderPort) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockOrderReaderPort) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	args := m.Called(ctx, orderID, paymentIntentID)
	return args.Error(0)
}

type MockBillingReaderPort struct {
	mock.Mock
}

func (m *MockBillingReaderPort) GetSubscription(ctx context.Context, userID uuid.UUID) (*outbound.PaymentSubscriptionInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.PaymentSubscriptionInfo), args.Error(1)
}

func (m *MockBillingReaderPort) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
	args := m.Called(ctx, userID, amount, source)
	return args.Error(0)
}

// --- Tests ---

func TestPaymentDomain_CreatePaymentIntent(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)
		mockStripeProvider := new(MockPaymentProviderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		orderID := uuid.New()
		userID := uuid.New()

		order := &outbound.PaymentOrderInfo{
			ID:       orderID,
			UserID:   userID,
			Type:     "topup",
			Status:   "pending",
			Total:    10000,
			Currency: "usd",
		}

		paymentIntent := &model.ProviderPaymentIntent{
			ID:           "pi_test123",
			ClientSecret: "pi_test123_secret",
			Amount:       10000,
			Currency:     "usd",
		}

		mockOrderReader.On("GetOrder", mock.Anything, orderID).Return(order, nil)
		mockBillingReader.On("GetSubscription", mock.Anything, userID).Return(nil, nil)
		mockProviderReg.On("Get", "stripe").Return(mockStripeProvider, nil)
		mockStripeProvider.On("Name").Return("stripe")
		mockStripeProvider.On("CreatePaymentIntent", mock.Anything, int64(10000), "usd", "", mock.Anything).Return(paymentIntent, nil)
		mockOrderReader.On("SetStripePaymentIntentID", mock.Anything, orderID, "pi_test123").Return(nil)
		mockPaymentDB.On("Create", mock.Anything, mock.Anything).Return(nil)

		resp, err := domain.CreatePaymentIntent(context.Background(), orderID, userID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "pi_test123", resp.PaymentIntentID)
		assert.Equal(t, "pi_test123_secret", resp.ClientSecret)
		mockOrderReader.AssertExpectations(t)
		mockProviderReg.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		orderID := uuid.New()
		userID := uuid.New()

		mockOrderReader.On("GetOrder", mock.Anything, orderID).Return(nil, errors.New("not found"))

		_, err := domain.CreatePaymentIntent(context.Background(), orderID, userID)

		assert.Error(t, err)
	})

	t.Run("forbidden - not owner", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		orderID := uuid.New()
		userID := uuid.New()
		otherUserID := uuid.New()

		order := &outbound.PaymentOrderInfo{
			ID:     orderID,
			UserID: otherUserID,
			Status: "pending",
		}

		mockOrderReader.On("GetOrder", mock.Anything, orderID).Return(order, nil)

		_, err := domain.CreatePaymentIntent(context.Background(), orderID, userID)

		assert.ErrorIs(t, err, ErrForbidden)
	})

	t.Run("order not pending", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		orderID := uuid.New()
		userID := uuid.New()

		order := &outbound.PaymentOrderInfo{
			ID:     orderID,
			UserID: userID,
			Status: "paid",
		}

		mockOrderReader.On("GetOrder", mock.Anything, orderID).Return(order, nil)

		_, err := domain.CreatePaymentIntent(context.Background(), orderID, userID)

		assert.ErrorIs(t, err, ErrOrderNotPending)
	})
}

func TestPaymentDomain_HandlePaymentSucceeded(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil, // No event publisher, will use fallback
			"https://api.example.com",
			logger,
		)

		paymentIntentID := "pi_test123"
		chargeID := "ch_test123"
		orderID := uuid.New()
		userID := uuid.New()

		order := &outbound.PaymentOrderInfo{
			ID:       orderID,
			UserID:   userID,
			Type:     "topup",
			Status:   "pending",
			Total:    10000,
			Currency: "usd",
			CreditsAmount: 1000,
		}

		payment := &model.Payment{
			ID:                    uuid.New(),
			OrderID:               orderID,
			UserID:                userID,
			Amount:                10000,
			Currency:              "usd",
			Status:                model.PaymentStatusPending,
			Provider:              "stripe",
			StripePaymentIntentID: paymentIntentID,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}

		mockOrderReader.On("GetOrderByPaymentIntentID", mock.Anything, paymentIntentID).Return(order, nil)
		mockPaymentDB.On("FindByPaymentIntentID", mock.Anything, paymentIntentID).Return(payment, nil)
		mockPaymentDB.On("Update", mock.Anything, mock.Anything).Return(nil)
		mockOrderReader.On("UpdateOrderStatus", mock.Anything, orderID, "paid").Return(nil)
		mockBillingReader.On("AddCredits", mock.Anything, userID, int64(1000), "topup").Return(nil)

		err := domain.HandlePaymentSucceeded(context.Background(), paymentIntentID, chargeID)

		assert.NoError(t, err)
		mockPaymentDB.AssertExpectations(t)
		mockOrderReader.AssertExpectations(t)
	})

	t.Run("already succeeded", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentIntentID := "pi_test123"
		chargeID := "ch_test123"
		orderID := uuid.New()
		userID := uuid.New()

		order := &outbound.PaymentOrderInfo{
			ID:     orderID,
			UserID: userID,
		}

		payment := &model.Payment{
			ID:      uuid.New(),
			Status:  model.PaymentStatusSucceeded,
		}

		mockOrderReader.On("GetOrderByPaymentIntentID", mock.Anything, paymentIntentID).Return(order, nil)
		mockPaymentDB.On("FindByPaymentIntentID", mock.Anything, paymentIntentID).Return(payment, nil)

		err := domain.HandlePaymentSucceeded(context.Background(), paymentIntentID, chargeID)

		assert.NoError(t, err) // Should return nil for idempotency
	})
}

func TestPaymentDomain_CreateRefund(t *testing.T) {
	logger := zap.NewNop()

	t.Run("full refund", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)
		mockStripeProvider := new(MockPaymentProviderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentID := uuid.New()
		orderID := uuid.New()

		payment := &model.Payment{
			ID:             paymentID,
			OrderID:        orderID,
			Amount:         10000,
			Status:         model.PaymentStatusSucceeded,
			Provider:       "stripe",
			StripeChargeID: "ch_test123",
			RefundedAmount: 0,
		}

		refund := &model.ProviderRefund{
			ID:       "re_test123",
			ChargeID: "ch_test123",
			Amount:   10000,
			Status:   "succeeded",
		}

		mockPaymentDB.On("FindByID", mock.Anything, paymentID).Return(payment, nil)
		mockProviderReg.On("Get", "stripe").Return(mockStripeProvider, nil)
		mockStripeProvider.On("CreateRefund", mock.Anything, "ch_test123", int64(10000), "customer request").Return(refund, nil)
		mockPaymentDB.On("Update", mock.Anything, mock.Anything).Return(nil)
		mockOrderReader.On("UpdateOrderStatus", mock.Anything, orderID, "refunded").Return(nil)

		err := domain.CreateRefund(context.Background(), paymentID, 0, "customer request")

		assert.NoError(t, err)
		mockPaymentDB.AssertExpectations(t)
		mockProviderReg.AssertExpectations(t)
	})

	t.Run("payment not succeeded", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentID := uuid.New()

		payment := &model.Payment{
			ID:     paymentID,
			Status: model.PaymentStatusPending,
		}

		mockPaymentDB.On("FindByID", mock.Anything, paymentID).Return(payment, nil)

		err := domain.CreateRefund(context.Background(), paymentID, 0, "customer request")

		assert.ErrorIs(t, err, ErrPaymentNotSucceeded)
	})

	t.Run("invalid refund amount", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentID := uuid.New()

		payment := &model.Payment{
			ID:             paymentID,
			Amount:         10000,
			Status:         model.PaymentStatusSucceeded,
			RefundedAmount: 5000,
		}

		mockPaymentDB.On("FindByID", mock.Anything, paymentID).Return(payment, nil)

		// Try to refund more than remaining
		err := domain.CreateRefund(context.Background(), paymentID, 6000, "customer request")

		assert.ErrorIs(t, err, ErrInvalidRefundAmount)
	})
}

func TestPaymentDomain_GetPayment(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentID := uuid.New()
		payment := &model.Payment{
			ID:     paymentID,
			Amount: 10000,
			Status: model.PaymentStatusSucceeded,
		}

		mockPaymentDB.On("FindByID", mock.Anything, paymentID).Return(payment, nil)

		result, err := domain.GetPayment(context.Background(), paymentID)

		assert.NoError(t, err)
		assert.Equal(t, paymentID, result.ID)
		assert.Equal(t, int64(10000), result.Amount)
	})

	t.Run("not found", func(t *testing.T) {
		mockPaymentDB := new(MockPaymentDatabasePort)
		mockWebhookDB := new(MockWebhookEventDatabasePort)
		mockProviderReg := new(MockPaymentProviderRegistryPort)
		mockOrderReader := new(MockOrderReaderPort)
		mockBillingReader := new(MockBillingReaderPort)

		domain := NewPaymentDomain(
			mockPaymentDB,
			mockWebhookDB,
			mockProviderReg,
			mockOrderReader,
			mockBillingReader,
			nil,
			"https://api.example.com",
			logger,
		)

		paymentID := uuid.New()

		mockPaymentDB.On("FindByID", mock.Anything, paymentID).Return(nil, nil)

		result, err := domain.GetPayment(context.Background(), paymentID)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestPaymentStatus_Transitions(t *testing.T) {
	t.Run("pending can transition to processing", func(t *testing.T) {
		assert.True(t, model.PaymentStatusPending.CanTransitionTo(model.PaymentStatusProcessing))
	})

	t.Run("pending can transition to succeeded", func(t *testing.T) {
		assert.True(t, model.PaymentStatusPending.CanTransitionTo(model.PaymentStatusSucceeded))
	})

	t.Run("pending can transition to failed", func(t *testing.T) {
		assert.True(t, model.PaymentStatusPending.CanTransitionTo(model.PaymentStatusFailed))
	})

	t.Run("succeeded can transition to refunded", func(t *testing.T) {
		assert.True(t, model.PaymentStatusSucceeded.CanTransitionTo(model.PaymentStatusRefunded))
	})

	t.Run("succeeded cannot transition to pending", func(t *testing.T) {
		assert.False(t, model.PaymentStatusSucceeded.CanTransitionTo(model.PaymentStatusPending))
	})

	t.Run("failed is terminal", func(t *testing.T) {
		assert.True(t, model.PaymentStatusFailed.IsTerminal())
		assert.False(t, model.PaymentStatusFailed.CanTransitionTo(model.PaymentStatusPending))
	})
}
