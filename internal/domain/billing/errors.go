package billing

import "errors"

// Domain errors for billing.
var (
	// Plan errors
	ErrPlanNotFound  = errors.New("plan not found")
	ErrPlanNotActive = errors.New("plan is not active")
	ErrInvalidPlan   = errors.New("invalid plan")

	// Subscription errors
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionExists   = errors.New("subscription already exists")
	ErrSubscriptionCanceled = errors.New("subscription is canceled")
	ErrCannotDowngrade      = errors.New("cannot downgrade to this plan")
	ErrSamePlan             = errors.New("already subscribed to this plan")

	// Quota errors
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrTokenQuotaExceeded   = errors.New("monthly token quota exceeded")
	ErrRequestQuotaExceeded = errors.New("daily request quota exceeded")

	// Credits errors
	ErrInsufficientCredits = errors.New("insufficient credits")
)
