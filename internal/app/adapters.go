package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"github.com/uniedit/server/internal/module/order"
	"github.com/uniedit/server/internal/module/order/domain"
	"github.com/uniedit/server/internal/module/payment"
	"github.com/uniedit/server/internal/shared/events"
)

// paymentOrderAdapter adapts the order Repository to payment.OrderReader interface.
// This adapter is defined in the app package to avoid cyclic imports between modules.
type paymentOrderAdapter struct {
	repo order.Repository
}

// newPaymentOrderAdapter creates a new adapter for the payment module.
func newPaymentOrderAdapter(repo order.Repository) *paymentOrderAdapter {
	return &paymentOrderAdapter{
		repo: repo,
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

	// Use domain methods for state transitions
	switch domain.OrderStatus(status) {
	case domain.StatusPaid:
		if err := ord.MarkAsPaid(); err != nil {
			return err
		}
	case domain.StatusCanceled:
		if err := ord.Cancel(); err != nil {
			return err
		}
	case domain.StatusRefunded:
		if err := ord.Refund(); err != nil {
			return err
		}
	case domain.StatusFailed:
		if err := ord.MarkAsFailed(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported status transition: %s", status)
	}

	return a.repo.UpdateOrder(ctx, ord)
}

// SetStripePaymentIntentID sets the Stripe PaymentIntent ID on an order.
func (a *paymentOrderAdapter) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	ord, err := a.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	ord.SetStripePaymentIntentID(paymentIntentID)
	return a.repo.UpdateOrder(ctx, ord)
}

// toPaymentOrderInfo converts a domain.Order to payment.OrderInfo.
func toPaymentOrderInfo(ord *domain.Order) *payment.OrderInfo {
	info := &payment.OrderInfo{
		ID:            ord.ID(),
		UserID:        ord.UserID(),
		Type:          string(ord.Type()),
		Status:        string(ord.Status()),
		Total:         ord.Total().Amount(),
		Currency:      ord.Currency(),
		CreditsAmount: ord.CreditsAmount(),
	}
	if ord.PlanID() != nil {
		info.PlanID = *ord.PlanID()
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
		UserID:           sub.UserID(),
		PlanID:           sub.PlanID(),
		Status:           string(sub.Status()),
		StripeCustomerID: sub.StripeCustomerID(),
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
