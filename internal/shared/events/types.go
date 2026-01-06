package events

import "github.com/google/uuid"

// Payment event type constants.
const (
	PaymentSucceededType = "PaymentSucceeded"
	PaymentFailedType    = "PaymentFailed"
)

// Order type constants for event handlers.
const (
	OrderTypeTopup        = "topup"
	OrderTypeSubscription = "subscription"
	OrderTypeUpgrade      = "upgrade"
)

// PaymentSucceededEvent is emitted when a payment is successfully processed.
// This is defined in the events package to avoid cyclic imports.
type PaymentSucceededEvent struct {
	BaseEvent

	// PaymentID is the unique identifier of the payment.
	PaymentID uuid.UUID `json:"payment_id"`

	// OrderID is the ID of the order this payment is for.
	OrderID uuid.UUID `json:"order_id"`

	// UserID is the ID of the user who made the payment.
	UserID uuid.UUID `json:"user_id"`

	// Amount is the payment amount in smallest currency unit (e.g., cents).
	Amount int64 `json:"amount"`

	// Currency is the ISO currency code (e.g., "usd", "cny").
	Currency string `json:"currency"`

	// Provider is the payment provider name (e.g., "stripe", "alipay", "wechat").
	Provider string `json:"provider"`

	// OrderType is the type of order (e.g., "topup", "subscription").
	OrderType string `json:"order_type"`

	// CreditsAmount is the credits amount for topup orders.
	CreditsAmount int64 `json:"credits_amount,omitempty"`

	// PlanID is the plan ID for subscription orders.
	PlanID string `json:"plan_id,omitempty"`
}

// NewPaymentSucceededEvent creates a new PaymentSucceededEvent.
func NewPaymentSucceededEvent(
	paymentID, orderID, userID uuid.UUID,
	amount int64,
	currency, provider, orderType string,
	creditsAmount int64,
	planID string,
) *PaymentSucceededEvent {
	return &PaymentSucceededEvent{
		BaseEvent:     NewBaseEvent(PaymentSucceededType, paymentID, "Payment"),
		PaymentID:     paymentID,
		OrderID:       orderID,
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		Provider:      provider,
		OrderType:     orderType,
		CreditsAmount: creditsAmount,
		PlanID:        planID,
	}
}

// PaymentFailedEvent is emitted when a payment fails.
type PaymentFailedEvent struct {
	BaseEvent

	// PaymentID is the unique identifier of the payment.
	PaymentID uuid.UUID `json:"payment_id"`

	// OrderID is the ID of the order this payment was for.
	OrderID uuid.UUID `json:"order_id"`

	// UserID is the ID of the user.
	UserID uuid.UUID `json:"user_id"`

	// FailureCode is the error code from the payment provider.
	FailureCode string `json:"failure_code,omitempty"`

	// FailureMessage is a human-readable error message.
	FailureMessage string `json:"failure_message,omitempty"`

	// Provider is the payment provider name.
	Provider string `json:"provider"`
}

// NewPaymentFailedEvent creates a new PaymentFailedEvent.
func NewPaymentFailedEvent(
	paymentID, orderID, userID uuid.UUID,
	failureCode, failureMessage, provider string,
) *PaymentFailedEvent {
	return &PaymentFailedEvent{
		BaseEvent:      NewBaseEvent(PaymentFailedType, paymentID, "Payment"),
		PaymentID:      paymentID,
		OrderID:        orderID,
		UserID:         userID,
		FailureCode:    failureCode,
		FailureMessage: failureMessage,
		Provider:       provider,
	}
}
