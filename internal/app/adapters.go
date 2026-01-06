package app

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"github.com/uniedit/server/internal/module/order"
	"github.com/uniedit/server/internal/module/payment"
	"github.com/uniedit/server/internal/shared/events"
)

// paymentOrderAdapter adapts the order Repository to payment.OrderReader interface.
// This adapter is defined in the app package to avoid cyclic imports between modules.
type paymentOrderAdapter struct {
	repo         order.Repository
	stateMachine *order.StateMachine
}

// newPaymentOrderAdapter creates a new adapter for the payment module.
func newPaymentOrderAdapter(repo order.Repository) *paymentOrderAdapter {
	return &paymentOrderAdapter{
		repo:         repo,
		stateMachine: order.NewStateMachine(),
	}
}

// GetOrder returns order information by ID.
func (a *paymentOrderAdapter) GetOrder(ctx context.Context, id uuid.UUID) (*payment.OrderInfo, error) {
	ord, err := a.repo.GetOrderWithItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPaymentOrderInfo(ord), nil
}

// GetOrderByPaymentIntentID returns order information by Stripe PaymentIntent ID.
func (a *paymentOrderAdapter) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.OrderInfo, error) {
	ord, err := a.repo.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return nil, err
	}
	return toPaymentOrderInfo(ord), nil
}

// UpdateOrderStatus updates the order status.
func (a *paymentOrderAdapter) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	ord, err := a.repo.GetOrder(ctx, id)
	if err != nil {
		return err
	}

	// Use state machine for valid transitions
	if err := a.stateMachine.Transition(ord, order.OrderStatus(status)); err != nil {
		return err
	}

	// Set timestamp based on status
	now := time.Now()
	switch order.OrderStatus(status) {
	case order.OrderStatusPaid:
		ord.PaidAt = &now
	case order.OrderStatusCanceled:
		ord.CanceledAt = &now
	case order.OrderStatusRefunded:
		ord.RefundedAt = &now
	}

	return a.repo.UpdateOrder(ctx, ord)
}

// SetStripePaymentIntentID sets the Stripe PaymentIntent ID on an order.
func (a *paymentOrderAdapter) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	ord, err := a.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	ord.StripePaymentIntentID = paymentIntentID
	return a.repo.UpdateOrder(ctx, ord)
}

// toPaymentOrderInfo converts an order.Order to payment.OrderInfo.
func toPaymentOrderInfo(ord *order.Order) *payment.OrderInfo {
	info := &payment.OrderInfo{
		ID:            ord.ID,
		UserID:        ord.UserID,
		Type:          string(ord.Type),
		Status:        string(ord.Status),
		Total:         ord.Total,
		Currency:      ord.Currency,
		CreditsAmount: ord.CreditsAmount,
	}
	if ord.PlanID != nil {
		info.PlanID = *ord.PlanID
	}
	return info
}

// paymentBillingAdapter adapts the billing ServiceInterface to payment.BillingReader interface.
// This adapter is defined in the app package to avoid cyclic imports between modules.
type paymentBillingAdapter struct {
	service billing.ServiceInterface
}

// newPaymentBillingAdapter creates a new adapter for the payment module.
func newPaymentBillingAdapter(service billing.ServiceInterface) *paymentBillingAdapter {
	return &paymentBillingAdapter{
		service: service,
	}
}

// GetSubscription returns the user's subscription information.
func (a *paymentBillingAdapter) GetSubscription(ctx context.Context, userID uuid.UUID) (*payment.SubscriptionInfo, error) {
	sub, err := a.service.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &payment.SubscriptionInfo{
		UserID:           sub.UserID,
		PlanID:           sub.PlanID,
		Status:           string(sub.Status),
		StripeCustomerID: sub.StripeCustomerID,
	}, nil
}

// AddCredits adds credits to a user's balance.
func (a *paymentBillingAdapter) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
	return a.service.AddCredits(ctx, userID, amount, source)
}

// Compile-time checks that adapters implement required interfaces.
var (
	_ payment.OrderReader    = (*paymentOrderAdapter)(nil)
	_ payment.BillingReader  = (*paymentBillingAdapter)(nil)
	_ payment.EventPublisher = (*eventBusAdapter)(nil)
)

// eventBusAdapter adapts the events.Bus to payment.EventPublisher interface.
// This allows the payment module to publish events without depending on the events package.
type eventBusAdapter struct {
	bus *events.Bus
}

// newEventBusAdapter creates a new adapter for the event bus.
func newEventBusAdapter(bus *events.Bus) *eventBusAdapter {
	return &eventBusAdapter{bus: bus}
}

// Publish publishes a domain event to all registered handlers.
func (a *eventBusAdapter) Publish(event interface{}) {
	if e, ok := event.(events.Event); ok {
		a.bus.Publish(e)
	}
}
