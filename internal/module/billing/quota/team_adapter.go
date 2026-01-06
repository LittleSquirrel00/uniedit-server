package quota

import (
	"context"

	"github.com/google/uuid"
)

// TeamQuotaAdapter provides team quota information for the collaboration module.
type TeamQuotaAdapter struct {
	checker *TeamQuotaChecker
}

// NewTeamQuotaAdapter creates a new team quota adapter.
func NewTeamQuotaAdapter(checker *TeamQuotaChecker) *TeamQuotaAdapter {
	return &TeamQuotaAdapter{
		checker: checker,
	}
}

// GetTeamMemberLimit returns the team member limit for a team owner.
// Returns -1 for unlimited.
func (a *TeamQuotaAdapter) GetTeamMemberLimit(ctx context.Context, teamOwnerID uuid.UUID) (int, error) {
	sub, err := a.checker.billingService.GetSubscription(ctx, teamOwnerID)
	if err != nil {
		return -1, nil // Default to unlimited on error
	}

	plan := sub.Plan
	if plan == nil || plan.IsUnlimitedTeamMembers() {
		return -1, nil
	}

	return plan.MaxTeamMembers, nil
}

// CheckTeamMemberQuota checks if the team owner has quota for adding more members.
func (a *TeamQuotaAdapter) CheckTeamMemberQuota(ctx context.Context, teamOwnerID uuid.UUID, currentCount, pendingInvites int) error {
	return a.checker.CheckTeamMemberQuota(ctx, teamOwnerID, currentCount, pendingInvites)
}

// GetTeamMemberQuotaStatus returns the team member quota status for a team owner.
func (a *TeamQuotaAdapter) GetTeamMemberQuotaStatus(ctx context.Context, teamOwnerID uuid.UUID, currentCount, pendingInvites int) (*TeamMemberQuotaStatus, error) {
	return a.checker.GetTeamMemberQuotaStatus(ctx, teamOwnerID, currentCount, pendingInvites)
}
