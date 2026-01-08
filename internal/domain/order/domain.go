package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"github.com/uniedit/server/internal/shared/random"
	"go.uber.org/zap"
)

// OrderDomain defines the interface for order business logic.
type OrderDomain interface {
	// Order operations
	CreateSubscriptionOrder(ctx context.Context, userID uuid.UUID, planID string) (*model.Order, error)
	CreateTopupOrder(ctx context.Context, userID uuid.UUID, amount int64) (*model.Order, error)
	GetOrder(ctx context.Context, orderID uuid.UUID) (*model.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*model.Order, error)
	GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID, filter *model.OrderFilter, page, pageSize int) ([]*model.Order, int64, error)

	// Status transitions
	MarkAsPaid(ctx context.Context, orderID uuid.UUID) error
	MarkAsFailed(ctx context.Context, orderID uuid.UUID) error
	CancelOrder(ctx context.Context, orderID uuid.UUID, reason string) error
	MarkAsRefunded(ctx context.Context, orderID uuid.UUID) error

	// Stripe integration
	SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error

	// Invoice operations
	GenerateInvoice(ctx context.Context, orderID uuid.UUID) (*model.Invoice, error)
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*model.Invoice, error)
	ListInvoices(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error)

	// Maintenance
	ExpirePendingOrders(ctx context.Context) error
}

// orderDomain implements OrderDomain.
type orderDomain struct {
	orderDB   outbound.OrderDatabasePort
	itemDB    outbound.OrderItemDatabasePort
	invoiceDB outbound.InvoiceDatabasePort
	planDB    outbound.PlanDatabasePort
	logger    *zap.Logger
}

// NewOrderDomain creates a new order domain service.
func NewOrderDomain(
	orderDB outbound.OrderDatabasePort,
	itemDB outbound.OrderItemDatabasePort,
	invoiceDB outbound.InvoiceDatabasePort,
	planDB outbound.PlanDatabasePort,
	logger *zap.Logger,
) OrderDomain {
	return &orderDomain{
		orderDB:   orderDB,
		itemDB:    itemDB,
		invoiceDB: invoiceDB,
		planDB:    planDB,
		logger:    logger,
	}
}

func (d *orderDomain) CreateSubscriptionOrder(ctx context.Context, userID uuid.UUID, planID string) (*model.Order, error) {
	// Get plan
	plan, err := d.planDB.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %s", planID)
	}
	if !plan.Active {
		return nil, fmt.Errorf("plan is not active")
	}
	if plan.PriceUSD <= 0 {
		return nil, ErrFreePlanNotOrderable
	}

	// Generate order number
	orderNo := generateOrderNo()
	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)

	// Create order
	order := &model.Order{
		ID:        uuid.New(),
		OrderNo:   orderNo,
		UserID:    userID,
		Type:      model.OrderTypeSubscription,
		Status:    model.OrderStatusPending,
		Subtotal:  plan.PriceUSD,
		Discount:  0,
		Tax:       0,
		Total:     plan.PriceUSD,
		Currency:  "usd",
		PlanID:    &planID,
		ExpiresAt: &expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.orderDB.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Create order item
	item := &model.OrderItem{
		ID:          uuid.New(),
		OrderID:     order.ID,
		Description: fmt.Sprintf("%s - %s", plan.Name, plan.BillingCycle),
		Quantity:    1,
		UnitPrice:   plan.PriceUSD,
		Amount:      plan.PriceUSD,
	}

	if err := d.itemDB.CreateBatch(ctx, []*model.OrderItem{item}); err != nil {
		return nil, fmt.Errorf("create order item: %w", err)
	}

	order.Items = []*model.OrderItem{item}
	return order, nil
}

func (d *orderDomain) CreateTopupOrder(ctx context.Context, userID uuid.UUID, amount int64) (*model.Order, error) {
	if amount < 100 {
		return nil, ErrMinimumTopupAmount
	}

	orderNo := generateOrderNo()
	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)

	// Create order
	order := &model.Order{
		ID:            uuid.New(),
		OrderNo:       orderNo,
		UserID:        userID,
		Type:          model.OrderTypeTopup,
		Status:        model.OrderStatusPending,
		Subtotal:      amount,
		Discount:      0,
		Tax:           0,
		Total:         amount,
		Currency:      "usd",
		CreditsAmount: amount,
		ExpiresAt:     &expiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := d.orderDB.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Create order item
	dollars := float64(amount) / 100
	item := &model.OrderItem{
		ID:          uuid.New(),
		OrderID:     order.ID,
		Description: fmt.Sprintf("Credits Top-up $%.2f", dollars),
		Quantity:    1,
		UnitPrice:   amount,
		Amount:      amount,
	}

	if err := d.itemDB.CreateBatch(ctx, []*model.OrderItem{item}); err != nil {
		return nil, fmt.Errorf("create order item: %w", err)
	}

	order.Items = []*model.OrderItem{item}
	return order, nil
}

func (d *orderDomain) GetOrder(ctx context.Context, orderID uuid.UUID) (*model.Order, error) {
	order, err := d.orderDB.GetByIDWithItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (d *orderDomain) GetOrderByNo(ctx context.Context, orderNo string) (*model.Order, error) {
	order, err := d.orderDB.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (d *orderDomain) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Order, error) {
	order, err := d.orderDB.GetByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (d *orderDomain) ListOrders(ctx context.Context, userID uuid.UUID, filter *model.OrderFilter, page, pageSize int) ([]*model.Order, int64, error) {
	return d.orderDB.List(ctx, userID, filter, page, pageSize)
}

func (d *orderDomain) MarkAsPaid(ctx context.Context, orderID uuid.UUID) error {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	if !order.Status.CanTransitionTo(model.OrderStatusPaid) {
		return fmt.Errorf("%w: cannot transition from %s to paid", ErrInvalidTransition, order.Status)
	}

	now := time.Now()
	order.Status = model.OrderStatusPaid
	order.PaidAt = &now
	order.UpdatedAt = now

	if err := d.orderDB.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Generate invoice
	if _, err := d.GenerateInvoice(ctx, orderID); err != nil {
		d.logger.Error("failed to generate invoice", zap.Error(err), zap.String("order_id", orderID.String()))
	}

	return nil
}

func (d *orderDomain) MarkAsFailed(ctx context.Context, orderID uuid.UUID) error {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	if !order.Status.CanTransitionTo(model.OrderStatusFailed) {
		return fmt.Errorf("%w: cannot transition from %s to failed", ErrInvalidTransition, order.Status)
	}

	order.Status = model.OrderStatusFailed
	order.UpdatedAt = time.Now()

	return d.orderDB.Update(ctx, order)
}

func (d *orderDomain) CancelOrder(ctx context.Context, orderID uuid.UUID, reason string) error {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	if !order.Status.CanTransitionTo(model.OrderStatusCanceled) {
		return ErrOrderNotCancelable
	}

	now := time.Now()
	order.Status = model.OrderStatusCanceled
	order.CanceledAt = &now
	order.UpdatedAt = now

	if err := d.orderDB.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	d.logger.Info("order canceled",
		zap.String("order_id", orderID.String()),
		zap.String("reason", reason),
	)

	return nil
}

func (d *orderDomain) MarkAsRefunded(ctx context.Context, orderID uuid.UUID) error {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	if !order.Status.CanTransitionTo(model.OrderStatusRefunded) {
		return ErrOrderNotRefundable
	}

	now := time.Now()
	order.Status = model.OrderStatusRefunded
	order.RefundedAt = &now
	order.UpdatedAt = now

	return d.orderDB.Update(ctx, order)
}

func (d *orderDomain) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	order.StripePaymentIntentID = paymentIntentID
	order.UpdatedAt = time.Now()

	return d.orderDB.Update(ctx, order)
}

func (d *orderDomain) GenerateInvoice(ctx context.Context, orderID uuid.UUID) (*model.Invoice, error) {
	order, err := d.orderDB.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	if !order.IsPaid() {
		return nil, ErrOrderNotPaid
	}

	// Check if invoice already exists
	existing, err := d.invoiceDB.GetByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		return existing, nil
	}

	invoiceNo := generateInvoiceNo()
	now := time.Now()

	invoice := &model.Invoice{
		ID:        uuid.New(),
		InvoiceNo: invoiceNo,
		OrderID:   orderID,
		UserID:    order.UserID,
		Amount:    order.Total,
		Currency:  order.Currency,
		Status:    "paid",
		IssuedAt:  now,
		DueAt:     now,
		PaidAt:    order.PaidAt,
		CreatedAt: now,
	}

	if err := d.invoiceDB.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	return invoice, nil
}

func (d *orderDomain) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*model.Invoice, error) {
	invoice, err := d.invoiceDB.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	return invoice, nil
}

func (d *orderDomain) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error) {
	return d.invoiceDB.ListByUserID(ctx, userID)
}

func (d *orderDomain) ExpirePendingOrders(ctx context.Context) error {
	orders, err := d.orderDB.ListPendingExpired(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if !order.Status.CanTransitionTo(model.OrderStatusFailed) {
			continue
		}

		order.Status = model.OrderStatusFailed
		order.UpdatedAt = time.Now()

		if err := d.orderDB.Update(ctx, order); err != nil {
			d.logger.Error("failed to update expired order", zap.Error(err), zap.String("order_id", order.ID.String()))
			continue
		}

		d.logger.Info("order expired", zap.String("order_id", order.ID.String()))
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
