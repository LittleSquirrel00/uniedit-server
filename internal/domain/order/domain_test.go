package order

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/uniedit/server/internal/model"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type MockOrderDB struct {
	mock.Mock
}

func (m *MockOrderDB) Create(ctx context.Context, order *model.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderDB) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderDB) GetByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderDB) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	args := m.Called(ctx, orderNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderDB) GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Order, error) {
	args := m.Called(ctx, paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderDB) List(ctx context.Context, userID uuid.UUID, filter *model.OrderFilter, page, pageSize int) ([]*model.Order, int64, error) {
	args := m.Called(ctx, userID, filter, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.Order), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrderDB) Update(ctx context.Context, order *model.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderDB) ListPendingExpired(ctx context.Context) ([]*model.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Order), args.Error(1)
}

type MockOrderItemDB struct {
	mock.Mock
}

func (m *MockOrderItemDB) CreateBatch(ctx context.Context, items []*model.OrderItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockOrderItemDB) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.OrderItem, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.OrderItem), args.Error(1)
}

type MockInvoiceDB struct {
	mock.Mock
}

func (m *MockInvoiceDB) Create(ctx context.Context, invoice *model.Invoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *MockInvoiceDB) GetByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Invoice), args.Error(1)
}

func (m *MockInvoiceDB) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Invoice, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Invoice), args.Error(1)
}

func (m *MockInvoiceDB) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Invoice), args.Error(1)
}

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

// --- Tests ---

func TestOrderDomain_CreateSubscriptionOrder(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockItemDB := new(MockOrderItemDB)
		mockPlanDB := new(MockPlanDB)

		domain := NewOrderDomain(mockOrderDB, mockItemDB, nil, mockPlanDB, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:           "pro",
			Name:         "Pro",
			BillingCycle: "monthly",
			PriceUSD:     1999,
			Active:       true,
		}

		mockPlanDB.On("GetByID", mock.Anything, "pro").Return(plan, nil)
		mockOrderDB.On("Create", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)
		mockItemDB.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*model.OrderItem")).Return(nil)

		order, err := domain.CreateSubscriptionOrder(context.Background(), userID, "pro")

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, model.OrderTypeSubscription, order.Type)
		assert.Equal(t, model.OrderStatusPending, order.Status)
		assert.Equal(t, int64(1999), order.Total)
		mockPlanDB.AssertExpectations(t)
		mockOrderDB.AssertExpectations(t)
		mockItemDB.AssertExpectations(t)
	})

	t.Run("free plan not orderable", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockItemDB := new(MockOrderItemDB)
		mockPlanDB := new(MockPlanDB)

		domain := NewOrderDomain(mockOrderDB, mockItemDB, nil, mockPlanDB, logger)

		userID := uuid.New()
		plan := &model.Plan{
			ID:       "free",
			Name:     "Free",
			PriceUSD: 0,
			Active:   true,
		}

		mockPlanDB.On("GetByID", mock.Anything, "free").Return(plan, nil)

		order, err := domain.CreateSubscriptionOrder(context.Background(), userID, "free")

		assert.ErrorIs(t, err, ErrFreePlanNotOrderable)
		assert.Nil(t, order)
	})
}

func TestOrderDomain_CreateTopupOrder(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockItemDB := new(MockOrderItemDB)

		domain := NewOrderDomain(mockOrderDB, mockItemDB, nil, nil, logger)

		userID := uuid.New()

		mockOrderDB.On("Create", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)
		mockItemDB.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*model.OrderItem")).Return(nil)

		order, err := domain.CreateTopupOrder(context.Background(), userID, 5000) // $50.00

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, model.OrderTypeTopup, order.Type)
		assert.Equal(t, int64(5000), order.CreditsAmount)
		mockOrderDB.AssertExpectations(t)
		mockItemDB.AssertExpectations(t)
	})

	t.Run("minimum amount", func(t *testing.T) {
		domain := NewOrderDomain(nil, nil, nil, nil, logger)

		order, err := domain.CreateTopupOrder(context.Background(), uuid.New(), 50) // $0.50

		assert.ErrorIs(t, err, ErrMinimumTopupAmount)
		assert.Nil(t, order)
	})
}

func TestOrderDomain_MarkAsPaid(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockInvoiceDB := new(MockInvoiceDB)

		domain := NewOrderDomain(mockOrderDB, nil, mockInvoiceDB, nil, logger)

		orderID := uuid.New()
		userID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			UserID: userID,
			Status: model.OrderStatusPending,
			Total:  1999,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)
		mockInvoiceDB.On("GetByOrderID", mock.Anything, orderID).Return(nil, nil)
		mockInvoiceDB.On("Create", mock.Anything, mock.AnythingOfType("*model.Invoice")).Return(nil)

		err := domain.MarkAsPaid(context.Background(), orderID)

		assert.NoError(t, err)
		assert.Equal(t, model.OrderStatusPaid, order.Status)
		assert.NotNil(t, order.PaidAt)
		mockOrderDB.AssertExpectations(t)
	})

	t.Run("invalid transition", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusCanceled, // Already canceled
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)

		err := domain.MarkAsPaid(context.Background(), orderID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state transition")
	})
}

func TestOrderDomain_CancelOrder(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPending,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)

		err := domain.CancelOrder(context.Background(), orderID, "user requested")

		assert.NoError(t, err)
		assert.Equal(t, model.OrderStatusCanceled, order.Status)
		assert.NotNil(t, order.CanceledAt)
		mockOrderDB.AssertExpectations(t)
	})

	t.Run("already paid", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPaid,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)

		err := domain.CancelOrder(context.Background(), orderID, "user requested")

		assert.ErrorIs(t, err, ErrOrderNotCancelable)
	})
}

func TestOrderDomain_MarkAsRefunded(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		paidAt := time.Now()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPaid,
			PaidAt: &paidAt,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)

		err := domain.MarkAsRefunded(context.Background(), orderID)

		assert.NoError(t, err)
		assert.Equal(t, model.OrderStatusRefunded, order.Status)
		assert.NotNil(t, order.RefundedAt)
		mockOrderDB.AssertExpectations(t)
	})

	t.Run("not paid", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPending,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)

		err := domain.MarkAsRefunded(context.Background(), orderID)

		assert.ErrorIs(t, err, ErrOrderNotRefundable)
	})
}

func TestOrderDomain_GenerateInvoice(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockInvoiceDB := new(MockInvoiceDB)

		domain := NewOrderDomain(mockOrderDB, nil, mockInvoiceDB, nil, logger)

		orderID := uuid.New()
		userID := uuid.New()
		paidAt := time.Now()
		order := &model.Order{
			ID:       orderID,
			UserID:   userID,
			Status:   model.OrderStatusPaid,
			Total:    1999,
			Currency: "usd",
			PaidAt:   &paidAt,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockInvoiceDB.On("GetByOrderID", mock.Anything, orderID).Return(nil, nil)
		mockInvoiceDB.On("Create", mock.Anything, mock.AnythingOfType("*model.Invoice")).Return(nil)

		invoice, err := domain.GenerateInvoice(context.Background(), orderID)

		assert.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.Equal(t, orderID, invoice.OrderID)
		assert.Equal(t, int64(1999), invoice.Amount)
		mockOrderDB.AssertExpectations(t)
		mockInvoiceDB.AssertExpectations(t)
	})

	t.Run("order not paid", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPending,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)

		invoice, err := domain.GenerateInvoice(context.Background(), orderID)

		assert.ErrorIs(t, err, ErrOrderNotPaid)
		assert.Nil(t, invoice)
	})

	t.Run("invoice already exists", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		mockInvoiceDB := new(MockInvoiceDB)

		domain := NewOrderDomain(mockOrderDB, nil, mockInvoiceDB, nil, logger)

		orderID := uuid.New()
		paidAt := time.Now()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPaid,
			PaidAt: &paidAt,
		}
		existingInvoice := &model.Invoice{
			ID:      uuid.New(),
			OrderID: orderID,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockInvoiceDB.On("GetByOrderID", mock.Anything, orderID).Return(existingInvoice, nil)

		invoice, err := domain.GenerateInvoice(context.Background(), orderID)

		assert.NoError(t, err)
		assert.Equal(t, existingInvoice, invoice)
	})
}

func TestOrderDomain_ExpirePendingOrders(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)

		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		expiredOrders := []*model.Order{
			{ID: uuid.New(), Status: model.OrderStatusPending},
			{ID: uuid.New(), Status: model.OrderStatusPending},
		}

		mockOrderDB.On("ListPendingExpired", mock.Anything).Return(expiredOrders, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil).Times(2)

		err := domain.ExpirePendingOrders(context.Background())

		assert.NoError(t, err)
		for _, ord := range expiredOrders {
			assert.Equal(t, model.OrderStatusFailed, ord.Status)
		}
		mockOrderDB.AssertExpectations(t)
	})
}

func TestOrderDomain_GetOrder(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		userID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			UserID: userID,
			Status: model.OrderStatusPending,
			Total:  1999,
		}

		mockOrderDB.On("GetByIDWithItems", mock.Anything, orderID).Return(order, nil)

		result, err := domain.GetOrder(context.Background(), orderID)

		assert.NoError(t, err)
		assert.Equal(t, orderID, result.ID)
		assert.Equal(t, int64(1999), result.Total)
		mockOrderDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		mockOrderDB.On("GetByIDWithItems", mock.Anything, orderID).Return(nil, nil)

		result, err := domain.GetOrder(context.Background(), orderID)

		assert.ErrorIs(t, err, ErrOrderNotFound)
		assert.Nil(t, result)
	})
}

func TestOrderDomain_GetOrderByNo(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderNo := "ORD20240101001"
		order := &model.Order{
			ID:      uuid.New(),
			OrderNo: orderNo,
			Status:  model.OrderStatusPending,
		}

		mockOrderDB.On("GetByOrderNo", mock.Anything, orderNo).Return(order, nil)

		result, err := domain.GetOrderByNo(context.Background(), orderNo)

		assert.NoError(t, err)
		assert.Equal(t, orderNo, result.OrderNo)
		mockOrderDB.AssertExpectations(t)
	})
}

func TestOrderDomain_GetOrderByPaymentIntentID(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		paymentIntentID := "pi_test123"
		order := &model.Order{
			ID:                    uuid.New(),
			StripePaymentIntentID: paymentIntentID,
			Status:                model.OrderStatusPending,
		}

		mockOrderDB.On("GetByPaymentIntentID", mock.Anything, paymentIntentID).Return(order, nil)

		result, err := domain.GetOrderByPaymentIntentID(context.Background(), paymentIntentID)

		assert.NoError(t, err)
		assert.Equal(t, paymentIntentID, result.StripePaymentIntentID)
		mockOrderDB.AssertExpectations(t)
	})
}

func TestOrderDomain_ListOrders(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		userID := uuid.New()
		orders := []*model.Order{
			{ID: uuid.New(), UserID: userID, Status: model.OrderStatusPending},
			{ID: uuid.New(), UserID: userID, Status: model.OrderStatusPaid},
		}

		mockOrderDB.On("List", mock.Anything, userID, (*model.OrderFilter)(nil), 1, 20).Return(orders, int64(2), nil)

		result, total, err := domain.ListOrders(context.Background(), userID, nil, 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, result, 2)
		mockOrderDB.AssertExpectations(t)
	})
}

func TestOrderDomain_MarkAsFailed(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPending,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)

		err := domain.MarkAsFailed(context.Background(), orderID)

		assert.NoError(t, err)
		assert.Equal(t, model.OrderStatusFailed, order.Status)
		mockOrderDB.AssertExpectations(t)
	})

	t.Run("already paid cannot fail", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPaid,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)

		err := domain.MarkAsFailed(context.Background(), orderID)

		assert.Error(t, err)
	})
}

func TestOrderDomain_SetStripePaymentIntentID(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockOrderDB := new(MockOrderDB)
		domain := NewOrderDomain(mockOrderDB, nil, nil, nil, logger)

		orderID := uuid.New()
		paymentIntentID := "pi_test123"
		order := &model.Order{
			ID:     orderID,
			Status: model.OrderStatusPending,
		}

		mockOrderDB.On("GetByID", mock.Anything, orderID).Return(order, nil)
		mockOrderDB.On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).Return(nil)

		err := domain.SetStripePaymentIntentID(context.Background(), orderID, paymentIntentID)

		assert.NoError(t, err)
		assert.Equal(t, paymentIntentID, order.StripePaymentIntentID)
		mockOrderDB.AssertExpectations(t)
	})
}

func TestOrderDomain_GetInvoice(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockInvoiceDB := new(MockInvoiceDB)
		domain := NewOrderDomain(nil, nil, mockInvoiceDB, nil, logger)

		invoiceID := uuid.New()
		invoice := &model.Invoice{
			ID:     invoiceID,
			Amount: 1999,
		}

		mockInvoiceDB.On("GetByID", mock.Anything, invoiceID).Return(invoice, nil)

		result, err := domain.GetInvoice(context.Background(), invoiceID)

		assert.NoError(t, err)
		assert.Equal(t, invoiceID, result.ID)
		mockInvoiceDB.AssertExpectations(t)
	})
}

func TestOrderDomain_ListInvoices(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockInvoiceDB := new(MockInvoiceDB)
		domain := NewOrderDomain(nil, nil, mockInvoiceDB, nil, logger)

		userID := uuid.New()
		invoices := []*model.Invoice{
			{ID: uuid.New(), UserID: userID, Amount: 1999},
			{ID: uuid.New(), UserID: userID, Amount: 2999},
		}

		mockInvoiceDB.On("ListByUserID", mock.Anything, userID).Return(invoices, nil)

		result, err := domain.ListInvoices(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockInvoiceDB.AssertExpectations(t)
	})
}
