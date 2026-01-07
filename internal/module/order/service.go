package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"github.com/uniedit/server/internal/module/order/domain"
	"github.com/uniedit/server/internal/module/order/entity"
	"github.com/uniedit/server/internal/shared/random"
	"go.uber.org/zap"
)

// Service implements order operations.
type Service struct {
	repo        Repository
	billingRepo billing.Repository
	logger      *zap.Logger
}

// NewService creates a new order service.
func NewService(repo Repository, billingRepo billing.Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:        repo,
		billingRepo: billingRepo,
		logger:      logger,
	}
}

// CreateSubscriptionOrder creates an order for a subscription.
func (s *Service) CreateSubscriptionOrder(ctx context.Context, userID uuid.UUID, planID string) (*domain.Order, error) {
	// Get plan
	plan, err := s.billingRepo.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !plan.Active() {
		return nil, billing.ErrPlanNotActive
	}
	if plan.PriceUSD() <= 0 {
		return nil, fmt.Errorf("cannot create order for free plan")
	}

	// Generate order number
	orderNo := generateOrderNo()

	// Create domain order
	order, err := domain.NewOrder(userID, orderNo, domain.TypeSubscription, "usd")
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Set plan and expiration
	order.SetPlanID(planID)
	order.SetExpiration(time.Now().Add(30 * time.Minute))

	// Create order item
	item, err := domain.NewOrderItem(
		fmt.Sprintf("%s - %s", plan.Name(), plan.BillingCycle()),
		1,
		domain.NewMoney(plan.PriceUSD(), "usd"),
	)
	if err != nil {
		return nil, fmt.Errorf("create order item: %w", err)
	}

	// Add item to order (this also calculates totals)
	if err := order.AddItem(item); err != nil {
		return nil, fmt.Errorf("add item to order: %w", err)
	}

	// Persist order
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Persist order items
	if err := s.repo.CreateOrderItems(ctx, order.Items()); err != nil {
		return nil, fmt.Errorf("create order items: %w", err)
	}

	return order, nil
}

// CreateTopupOrder creates an order for a credits top-up.
func (s *Service) CreateTopupOrder(ctx context.Context, userID uuid.UUID, amount int64) (*domain.Order, error) {
	if amount < 100 {
		return nil, fmt.Errorf("minimum top-up is $1.00 (100 cents)")
	}

	orderNo := generateOrderNo()

	// Create domain order
	order, err := domain.NewOrder(userID, orderNo, domain.TypeTopup, "usd")
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Set credits amount and expiration
	order.SetCreditsAmount(amount)
	order.SetExpiration(time.Now().Add(30 * time.Minute))

	// Create order item
	dollars := float64(amount) / 100
	item, err := domain.NewOrderItem(
		fmt.Sprintf("Credits Top-up $%.2f", dollars),
		1,
		domain.NewMoney(amount, "usd"),
	)
	if err != nil {
		return nil, fmt.Errorf("create order item: %w", err)
	}

	// Add item to order
	if err := order.AddItem(item); err != nil {
		return nil, fmt.Errorf("add item to order: %w", err)
	}

	// Persist order
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Persist order items
	if err := s.repo.CreateOrderItems(ctx, order.Items()); err != nil {
		return nil, fmt.Errorf("create order items: %w", err)
	}

	return order, nil
}

// GetOrder returns an order by ID.
func (s *Service) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	return s.repo.GetOrderWithItems(ctx, orderID)
}

// GetOrderByNo returns an order by order number.
func (s *Service) GetOrderByNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	return s.repo.GetOrderByNo(ctx, orderNo)
}

// GetOrderByPaymentIntentID returns an order by Stripe PaymentIntent ID.
func (s *Service) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Order, error) {
	return s.repo.GetOrderByPaymentIntentID(ctx, paymentIntentID)
}

// ListOrders returns orders for a user.
func (s *Service) ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter, pagination *Pagination) ([]*domain.Order, int64, error) {
	return s.repo.ListOrders(ctx, userID, filter, pagination)
}

// MarkAsPaid marks an order as paid.
func (s *Service) MarkAsPaid(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Use domain method for state transition
	if err := order.MarkAsPaid(); err != nil {
		return err
	}

	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Generate invoice
	if _, err := s.GenerateInvoice(ctx, orderID); err != nil {
		s.logger.Error("failed to generate invoice", zap.Error(err), zap.String("order_id", orderID.String()))
	}

	return nil
}

// CancelOrder cancels a pending order.
func (s *Service) CancelOrder(ctx context.Context, orderID uuid.UUID, reason string) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Use domain method for state transition
	if err := order.Cancel(); err != nil {
		return ErrOrderNotCancelable
	}

	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	s.logger.Info("order canceled",
		zap.String("order_id", orderID.String()),
		zap.String("reason", reason),
	)

	return nil
}

// MarkAsRefunded marks an order as refunded.
func (s *Service) MarkAsRefunded(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Use domain method for state transition
	if err := order.Refund(); err != nil {
		return ErrOrderNotRefundable
	}

	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	return nil
}

// MarkAsFailed marks an order as failed.
func (s *Service) MarkAsFailed(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Use domain method for state transition
	if err := order.MarkAsFailed(); err != nil {
		return err
	}

	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	return nil
}

// SetStripePaymentIntentID sets the Stripe PaymentIntent ID on an order.
func (s *Service) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	order.SetStripePaymentIntentID(paymentIntentID)
	return s.repo.UpdateOrder(ctx, order)
}

// GenerateInvoice generates an invoice for a paid order.
func (s *Service) GenerateInvoice(ctx context.Context, orderID uuid.UUID) (*entity.InvoiceEntity, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsPaid() {
		return nil, ErrOrderNotPaid
	}

	// Check if invoice already exists
	existing, err := s.repo.GetInvoiceByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		return existing, nil
	}

	invoiceNo := generateInvoiceNo()
	now := time.Now()

	invoice := &entity.InvoiceEntity{
		ID:        uuid.New(),
		InvoiceNo: invoiceNo,
		OrderID:   orderID,
		UserID:    order.UserID(),
		Amount:    order.Total().Amount(),
		Currency:  order.Currency(),
		Status:    "paid",
		IssuedAt:  now,
		DueAt:     now,
		PaidAt:    order.PaidAt(),
	}

	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	return invoice, nil
}

// GetInvoice returns an invoice by ID.
func (s *Service) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*entity.InvoiceEntity, error) {
	return s.repo.GetInvoice(ctx, invoiceID)
}

// ListInvoices returns all invoices for a user.
func (s *Service) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*entity.InvoiceEntity, error) {
	return s.repo.ListInvoices(ctx, userID)
}

// ExpirePendingOrders expires orders that have passed their expiration time.
func (s *Service) ExpirePendingOrders(ctx context.Context) error {
	orders, err := s.repo.ListPendingExpiredOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		// Use domain method for state transition
		if err := order.MarkAsFailed(); err != nil {
			s.logger.Error("failed to expire order", zap.Error(err), zap.String("order_id", order.ID().String()))
			continue
		}
		if err := s.repo.UpdateOrder(ctx, order); err != nil {
			s.logger.Error("failed to update expired order", zap.Error(err), zap.String("order_id", order.ID().String()))
			continue
		}
		s.logger.Info("order expired", zap.String("order_id", order.ID().String()))
	}

	return nil
}

// --- Helpers ---

func generateOrderNo() string {
	now := time.Now()
	suffix := random.UpperAlphaNum(5)
	return fmt.Sprintf("ORD-%s-%s", now.Format("20060102"), suffix)
}

func generateInvoiceNo() string {
	now := time.Now()
	suffix := random.UpperAlphaNum(5)
	return fmt.Sprintf("INV-%s-%s", now.Format("20060102"), suffix)
}
