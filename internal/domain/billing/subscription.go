package billing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Subscription is the aggregate root for user subscriptions.
// It encapsulates all business rules related to subscriptions and credits.
type Subscription struct {
	id                   uuid.UUID
	userID               uuid.UUID
	planID               string
	status               SubscriptionStatus
	stripeCustomerID     string
	stripeSubscriptionID string
	currentPeriodStart   time.Time
	currentPeriodEnd     time.Time
	cancelAtPeriodEnd    bool
	canceledAt           *time.Time
	creditsBalance       int64
	createdAt            time.Time
	updatedAt            time.Time

	// Loaded relation
	plan *Plan
}

// NewSubscription creates a new Subscription with the given parameters.
func NewSubscription(userID uuid.UUID, planID string) (*Subscription, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if planID == "" {
		return nil, fmt.Errorf("plan ID cannot be empty")
	}

	now := time.Now()
	periodEnd := endOfMonth(now)

	return &Subscription{
		id:                 uuid.New(),
		userID:             userID,
		planID:             planID,
		status:             StatusActive,
		currentPeriodStart: now,
		currentPeriodEnd:   periodEnd,
		createdAt:          now,
		updatedAt:          now,
	}, nil
}

// RestoreSubscription recreates a Subscription from persisted data.
func RestoreSubscription(
	id, userID uuid.UUID,
	planID string,
	status SubscriptionStatus,
	stripeCustomerID, stripeSubscriptionID string,
	currentPeriodStart, currentPeriodEnd time.Time,
	cancelAtPeriodEnd bool,
	canceledAt *time.Time,
	creditsBalance int64,
	createdAt, updatedAt time.Time,
	plan *Plan,
) *Subscription {
	return &Subscription{
		id:                   id,
		userID:               userID,
		planID:               planID,
		status:               status,
		stripeCustomerID:     stripeCustomerID,
		stripeSubscriptionID: stripeSubscriptionID,
		currentPeriodStart:   currentPeriodStart,
		currentPeriodEnd:     currentPeriodEnd,
		cancelAtPeriodEnd:    cancelAtPeriodEnd,
		canceledAt:           canceledAt,
		creditsBalance:       creditsBalance,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
		plan:                 plan,
	}
}

// --- Getters ---

// ID returns the subscription ID.
func (s *Subscription) ID() uuid.UUID {
	return s.id
}

// UserID returns the user ID.
func (s *Subscription) UserID() uuid.UUID {
	return s.userID
}

// PlanID returns the plan ID.
func (s *Subscription) PlanID() string {
	return s.planID
}

// Status returns the subscription status.
func (s *Subscription) Status() SubscriptionStatus {
	return s.status
}

// StripeCustomerID returns the Stripe customer ID.
func (s *Subscription) StripeCustomerID() string {
	return s.stripeCustomerID
}

// StripeSubscriptionID returns the Stripe subscription ID.
func (s *Subscription) StripeSubscriptionID() string {
	return s.stripeSubscriptionID
}

// CurrentPeriodStart returns the current billing period start.
func (s *Subscription) CurrentPeriodStart() time.Time {
	return s.currentPeriodStart
}

// CurrentPeriodEnd returns the current billing period end.
func (s *Subscription) CurrentPeriodEnd() time.Time {
	return s.currentPeriodEnd
}

// CancelAtPeriodEnd returns whether the subscription will cancel at period end.
func (s *Subscription) CancelAtPeriodEnd() bool {
	return s.cancelAtPeriodEnd
}

// CanceledAt returns when the subscription was canceled.
func (s *Subscription) CanceledAt() *time.Time {
	return s.canceledAt
}

// CreditsBalance returns the credits balance in cents.
func (s *Subscription) CreditsBalance() int64 {
	return s.creditsBalance
}

// CreatedAt returns when the subscription was created.
func (s *Subscription) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns when the subscription was last updated.
func (s *Subscription) UpdatedAt() time.Time {
	return s.updatedAt
}

// Plan returns the associated plan (may be nil if not loaded).
func (s *Subscription) Plan() *Plan {
	return s.plan
}

// --- Status Query Methods ---

// IsActive returns true if the subscription is active or trialing.
func (s *Subscription) IsActive() bool {
	return s.status.IsActive()
}

// IsCanceled returns true if the subscription is canceled.
func (s *Subscription) IsCanceled() bool {
	return s.status == StatusCanceled
}

// --- Domain Methods ---

// Cancel cancels the subscription.
// If immediately is true, cancels now. Otherwise, marks to cancel at period end.
func (s *Subscription) Cancel(immediately bool) error {
	if s.IsCanceled() {
		return fmt.Errorf("subscription already canceled")
	}

	now := time.Now()
	s.canceledAt = &now
	s.updatedAt = now

	if immediately {
		s.status = StatusCanceled
		s.planID = "free" // Downgrade to free plan
	} else {
		s.cancelAtPeriodEnd = true
	}

	return nil
}

// Reactivate reactivates a canceled subscription.
func (s *Subscription) Reactivate() error {
	if !s.IsCanceled() && !s.cancelAtPeriodEnd {
		return fmt.Errorf("subscription is not canceled")
	}

	s.cancelAtPeriodEnd = false
	s.canceledAt = nil
	s.status = StatusActive
	s.updatedAt = time.Now()

	return nil
}

// AddCredits adds credits to the balance.
func (s *Subscription) AddCredits(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.creditsBalance += amount
	s.updatedAt = time.Now()
	return nil
}

// DeductCredits deducts credits from the balance.
func (s *Subscription) DeductCredits(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if s.creditsBalance < amount {
		return fmt.Errorf("insufficient credits: have %d, need %d", s.creditsBalance, amount)
	}

	s.creditsBalance -= amount
	s.updatedAt = time.Now()
	return nil
}

// HasSufficientCredits checks if the subscription has enough credits.
func (s *Subscription) HasSufficientCredits(amount int64) bool {
	return s.creditsBalance >= amount
}

// UpdateFromStripe updates the subscription from Stripe webhook data.
func (s *Subscription) UpdateFromStripe(status SubscriptionStatus, periodStart, periodEnd time.Time, cancelAtPeriodEnd bool) {
	s.status = status
	s.currentPeriodStart = periodStart
	s.currentPeriodEnd = periodEnd
	s.cancelAtPeriodEnd = cancelAtPeriodEnd
	s.updatedAt = time.Now()
}

// SetStripeIDs sets the Stripe customer and subscription IDs.
func (s *Subscription) SetStripeIDs(customerID, subscriptionID string) {
	s.stripeCustomerID = customerID
	s.stripeSubscriptionID = subscriptionID
	s.updatedAt = time.Now()
}

// SetPlan sets the plan and plan ID.
func (s *Subscription) SetPlan(plan *Plan) {
	s.plan = plan
	s.planID = plan.ID()
	s.updatedAt = time.Now()
}

// RenewPeriod starts a new billing period.
func (s *Subscription) RenewPeriod(start, end time.Time) {
	s.currentPeriodStart = start
	s.currentPeriodEnd = end
	s.updatedAt = time.Now()
}

// --- Helpers ---

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
}
