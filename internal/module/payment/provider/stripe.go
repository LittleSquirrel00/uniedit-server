package provider

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeProvider implements the Provider interface for Stripe.
type StripeProvider struct {
	apiKey        string
	webhookSecret string
}

// StripeConfig holds Stripe configuration.
type StripeConfig struct {
	APIKey        string
	WebhookSecret string
}

// NewStripeProvider creates a new Stripe provider.
func NewStripeProvider(config *StripeConfig) *StripeProvider {
	stripe.Key = config.APIKey
	return &StripeProvider{
		apiKey:        config.APIKey,
		webhookSecret: config.WebhookSecret,
	}
}

// Name returns the provider name.
func (p *StripeProvider) Name() string {
	return "stripe"
}

// --- Customer Management ---

func (p *StripeProvider) CreateCustomer(ctx context.Context, email, name string) (*Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	c, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}
	return &Customer{
		ID:    c.ID,
		Email: c.Email,
	}, nil
}

func (p *StripeProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	c, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	return &Customer{
		ID:    c.ID,
		Email: c.Email,
	}, nil
}

// --- Payment Intents ---

func (p *StripeProvider) CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	if customerID != "" {
		params.Customer = stripe.String(customerID)
	}
	if len(metadata) > 0 {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = v
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}

	return &PaymentIntent{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       pi.Amount,
		Currency:     string(pi.Currency),
		Status:       string(pi.Status),
	}, nil
}

func (p *StripeProvider) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntent, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("get payment intent: %w", err)
	}

	result := &PaymentIntent{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       pi.Amount,
		Currency:     string(pi.Currency),
		Status:       string(pi.Status),
	}
	if pi.Metadata != nil {
		result.Metadata = pi.Metadata
	}
	return result, nil
}

func (p *StripeProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	_, err := paymentintent.Cancel(paymentIntentID, nil)
	if err != nil {
		return fmt.Errorf("cancel payment intent: %w", err)
	}
	return nil
}

// --- Subscriptions ---

func (p *StripeProvider) CreateSubscription(ctx context.Context, customerID, priceID string) (*Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(priceID)},
		},
	}
	sub, err := subscription.New(params)
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return mapStripeSubscription(sub), nil
}

func (p *StripeProvider) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return mapStripeSubscription(sub), nil
}

func (p *StripeProvider) UpdateSubscription(ctx context.Context, subscriptionID string, params map[string]interface{}) (*Subscription, error) {
	updateParams := &stripe.SubscriptionParams{}
	if cancelAtPeriodEnd, ok := params["cancel_at_period_end"].(bool); ok {
		updateParams.CancelAtPeriodEnd = stripe.Bool(cancelAtPeriodEnd)
	}
	sub, err := subscription.Update(subscriptionID, updateParams)
	if err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}
	return mapStripeSubscription(sub), nil
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	if immediately {
		_, err := subscription.Cancel(subscriptionID, nil)
		return err
	}
	_, err := subscription.Update(subscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	return err
}

// --- Refunds ---

func (p *StripeProvider) CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*Refund, error) {
	params := &stripe.RefundParams{
		Charge: stripe.String(chargeID),
	}
	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}
	if reason != "" {
		params.Reason = stripe.String(reason)
	}
	r, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("create refund: %w", err)
	}
	return &Refund{
		ID:       r.ID,
		ChargeID: r.Charge.ID,
		Amount:   r.Amount,
		Status:   string(r.Status),
		Reason:   string(r.Reason),
	}, nil
}

// --- Payment Methods ---

func (p *StripeProvider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	_, err := paymentmethod.Attach(paymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	})
	return err
}

func (p *StripeProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := paymentmethod.Detach(paymentMethodID, nil)
	return err
}

func (p *StripeProvider) ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodDetails, error) {
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"),
	}
	i := paymentmethod.List(params)

	var methods []*PaymentMethodDetails
	for i.Next() {
		pm := i.PaymentMethod()
		method := &PaymentMethodDetails{
			ID:   pm.ID,
			Type: string(pm.Type),
		}
		if pm.Card != nil {
			method.CardBrand = string(pm.Card.Brand)
			method.CardLast4 = pm.Card.Last4
			method.ExpMonth = int(pm.Card.ExpMonth)
			method.ExpYear = int(pm.Card.ExpYear)
		}
		methods = append(methods, method)
	}
	if err := i.Err(); err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}

	return methods, nil
}

func (p *StripeProvider) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	_, err := customer.Update(customerID, &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	})
	return err
}

// --- Webhooks ---

func (p *StripeProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	_, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	return err
}

// --- Helpers ---

func mapStripeSubscription(sub *stripe.Subscription) *Subscription {
	return &Subscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}
}
