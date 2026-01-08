package billing

import "errors"

var (
	// Plan errors
	ErrPlanNotFound  = errors.New("plan not found")
	ErrPlanNotActive = errors.New("plan is not active")

	// Subscription errors
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrSubscriptionExists    = errors.New("subscription already exists")
	ErrSubscriptionCanceled  = errors.New("subscription already canceled")
	ErrSubscriptionNotActive = errors.New("subscription is not active")

	// Quota errors
	ErrQuotaExceeded     = errors.New("quota exceeded")
	ErrTokenLimitReached = errors.New("monthly token limit reached")
	ErrRequestLimitReached = errors.New("daily request limit reached")

	// Credits errors
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrInvalidCreditsAmount = errors.New("invalid credits amount")

	// Stripe errors
	ErrStripeCustomerNotFound = errors.New("stripe customer not found")
	ErrStripeError            = errors.New("stripe operation failed")
)
