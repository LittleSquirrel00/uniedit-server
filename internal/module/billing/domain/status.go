package domain

import "fmt"

// PlanType represents the type of subscription plan.
type PlanType string

const (
	PlanTypeFree       PlanType = "free"
	PlanTypePro        PlanType = "pro"
	PlanTypeTeam       PlanType = "team"
	PlanTypeEnterprise PlanType = "enterprise"
)

// String returns the string representation of the plan type.
func (p PlanType) String() string {
	return string(p)
}

// IsValid checks if the plan type is valid.
func (p PlanType) IsValid() bool {
	switch p {
	case PlanTypeFree, PlanTypePro, PlanTypeTeam, PlanTypeEnterprise:
		return true
	}
	return false
}

// BillingCycle represents the billing period.
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// String returns the string representation of the billing cycle.
func (b BillingCycle) String() string {
	return string(b)
}

// IsValid checks if the billing cycle is valid.
func (b BillingCycle) IsValid() bool {
	switch b {
	case BillingCycleMonthly, BillingCycleYearly:
		return true
	}
	return false
}

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	StatusTrialing   SubscriptionStatus = "trialing"
	StatusActive     SubscriptionStatus = "active"
	StatusPastDue    SubscriptionStatus = "past_due"
	StatusCanceled   SubscriptionStatus = "canceled"
	StatusIncomplete SubscriptionStatus = "incomplete"
)

// String returns the string representation of the status.
func (s SubscriptionStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid.
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case StatusTrialing, StatusActive, StatusPastDue, StatusCanceled, StatusIncomplete:
		return true
	}
	return false
}

// IsActive returns true if the subscription is active or trialing.
func (s SubscriptionStatus) IsActive() bool {
	return s == StatusActive || s == StatusTrialing
}

// ErrInvalidStatusTransition is returned when a status transition is not allowed.
var ErrInvalidStatusTransition = fmt.Errorf("invalid status transition")
