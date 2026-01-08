package billing

// Plan represents a subscription plan.
// Plan is a value object - it describes plan configuration and is mostly read-only.
type Plan struct {
	id            string
	planType      PlanType
	name          string
	description   string
	billingCycle  BillingCycle
	priceUSD      int64    // In cents
	stripePriceID string
	features      []string
	active        bool
	displayOrder  int

	// Token quotas
	monthlyTokens          int64
	dailyRequests          int
	maxAPIKeys             int
	monthlyChatTokens      int64
	monthlyImageCredits    int
	monthlyVideoMinutes    int
	monthlyEmbeddingTokens int64

	// Storage quotas
	gitStorageMB int64
	lfsStorageMB int64

	// Team quota
	maxTeamMembers int
}

// NewPlan creates a new Plan with the given parameters.
func NewPlan(id string, planType PlanType, name string, priceUSD int64) *Plan {
	return &Plan{
		id:       id,
		planType: planType,
		name:     name,
		priceUSD: priceUSD,
		active:   true,
	}
}

// RestorePlan recreates a Plan from persisted data.
func RestorePlan(
	id string,
	planType PlanType,
	name, description string,
	billingCycle BillingCycle,
	priceUSD int64,
	stripePriceID string,
	monthlyTokens int64,
	dailyRequests, maxAPIKeys int,
	features []string,
	active bool,
	displayOrder int,
	monthlyChatTokens int64,
	monthlyImageCredits, monthlyVideoMinutes int,
	monthlyEmbeddingTokens int64,
	gitStorageMB, lfsStorageMB int64,
	maxTeamMembers int,
) *Plan {
	return &Plan{
		id:                     id,
		planType:               planType,
		name:                   name,
		description:            description,
		billingCycle:           billingCycle,
		priceUSD:               priceUSD,
		stripePriceID:          stripePriceID,
		monthlyTokens:          monthlyTokens,
		dailyRequests:          dailyRequests,
		maxAPIKeys:             maxAPIKeys,
		features:               features,
		active:                 active,
		displayOrder:           displayOrder,
		monthlyChatTokens:      monthlyChatTokens,
		monthlyImageCredits:    monthlyImageCredits,
		monthlyVideoMinutes:    monthlyVideoMinutes,
		monthlyEmbeddingTokens: monthlyEmbeddingTokens,
		gitStorageMB:           gitStorageMB,
		lfsStorageMB:           lfsStorageMB,
		maxTeamMembers:         maxTeamMembers,
	}
}

// ID returns the plan ID.
func (p *Plan) ID() string {
	return p.id
}

// Type returns the plan type.
func (p *Plan) Type() PlanType {
	return p.planType
}

// Name returns the plan name.
func (p *Plan) Name() string {
	return p.name
}

// Description returns the plan description.
func (p *Plan) Description() string {
	return p.description
}

// BillingCycle returns the billing cycle.
func (p *Plan) BillingCycle() BillingCycle {
	return p.billingCycle
}

// PriceUSD returns the price in cents.
func (p *Plan) PriceUSD() int64 {
	return p.priceUSD
}

// StripePriceID returns the Stripe price ID.
func (p *Plan) StripePriceID() string {
	return p.stripePriceID
}

// MonthlyTokens returns the monthly token limit. -1 means unlimited.
func (p *Plan) MonthlyTokens() int64 {
	return p.monthlyTokens
}

// DailyRequests returns the daily request limit. -1 means unlimited.
func (p *Plan) DailyRequests() int {
	return p.dailyRequests
}

// MaxAPIKeys returns the maximum number of API keys.
func (p *Plan) MaxAPIKeys() int {
	return p.maxAPIKeys
}

// Features returns the plan features.
func (p *Plan) Features() []string {
	result := make([]string, len(p.features))
	copy(result, p.features)
	return result
}

// Active returns whether the plan is active.
func (p *Plan) Active() bool {
	return p.active
}

// DisplayOrder returns the display order.
func (p *Plan) DisplayOrder() int {
	return p.displayOrder
}

// MonthlyChatTokens returns the monthly chat token limit.
func (p *Plan) MonthlyChatTokens() int64 {
	return p.monthlyChatTokens
}

// MonthlyImageCredits returns the monthly image credits.
func (p *Plan) MonthlyImageCredits() int {
	return p.monthlyImageCredits
}

// MonthlyVideoMinutes returns the monthly video minutes.
func (p *Plan) MonthlyVideoMinutes() int {
	return p.monthlyVideoMinutes
}

// MonthlyEmbeddingTokens returns the monthly embedding tokens.
func (p *Plan) MonthlyEmbeddingTokens() int64 {
	return p.monthlyEmbeddingTokens
}

// GitStorageMB returns the git storage limit in MB. -1 means unlimited.
func (p *Plan) GitStorageMB() int64 {
	return p.gitStorageMB
}

// LFSStorageMB returns the LFS storage limit in MB. -1 means unlimited.
func (p *Plan) LFSStorageMB() int64 {
	return p.lfsStorageMB
}

// MaxTeamMembers returns the maximum team members. -1 means unlimited.
func (p *Plan) MaxTeamMembers() int {
	return p.maxTeamMembers
}

// --- Quota Check Methods ---

// IsUnlimitedTokens returns true if the plan has unlimited tokens.
func (p *Plan) IsUnlimitedTokens() bool {
	return p.monthlyTokens == -1
}

// IsUnlimitedRequests returns true if the plan has unlimited daily requests.
func (p *Plan) IsUnlimitedRequests() bool {
	return p.dailyRequests == -1
}

// IsUnlimitedChatTokens returns true if chat tokens are unlimited.
func (p *Plan) IsUnlimitedChatTokens() bool {
	return p.monthlyChatTokens == -1
}

// IsUnlimitedImageCredits returns true if image credits are unlimited.
func (p *Plan) IsUnlimitedImageCredits() bool {
	return p.monthlyImageCredits == -1
}

// IsUnlimitedVideoMinutes returns true if video minutes are unlimited.
func (p *Plan) IsUnlimitedVideoMinutes() bool {
	return p.monthlyVideoMinutes == -1
}

// IsUnlimitedEmbeddingTokens returns true if embedding tokens are unlimited.
func (p *Plan) IsUnlimitedEmbeddingTokens() bool {
	return p.monthlyEmbeddingTokens == -1
}

// IsUnlimitedGitStorage returns true if git storage is unlimited.
func (p *Plan) IsUnlimitedGitStorage() bool {
	return p.gitStorageMB == -1
}

// IsUnlimitedLFSStorage returns true if LFS storage is unlimited.
func (p *Plan) IsUnlimitedLFSStorage() bool {
	return p.lfsStorageMB == -1
}

// IsUnlimitedTeamMembers returns true if team members are unlimited.
func (p *Plan) IsUnlimitedTeamMembers() bool {
	return p.maxTeamMembers == -1
}

// GetEffectiveChatTokenLimit returns the effective chat token limit.
// If MonthlyChatTokens is 0, falls back to MonthlyTokens for backward compatibility.
func (p *Plan) GetEffectiveChatTokenLimit() int64 {
	if p.monthlyChatTokens == 0 {
		return p.monthlyTokens
	}
	return p.monthlyChatTokens
}
