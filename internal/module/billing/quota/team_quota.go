package quota

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"go.uber.org/zap"
)

// Team quota errors
var (
	ErrTeamMemberQuotaExceeded = fmt.Errorf("team member quota exceeded")
)

// TeamQuotaChecker checks team member quotas.
type TeamQuotaChecker struct {
	billingService billing.ServiceInterface
	logger         *zap.Logger
}

// NewTeamQuotaChecker creates a new team quota checker.
func NewTeamQuotaChecker(billingService billing.ServiceInterface, logger *zap.Logger) *TeamQuotaChecker {
	return &TeamQuotaChecker{
		billingService: billingService,
		logger:         logger,
	}
}

// CheckTeamMemberQuota checks if team owner has quota for more members.
// teamOwnerID is the user ID of the team owner (whose subscription determines quota).
// currentCount is the current number of team members (including owner).
// pendingInvites is the number of pending invitations.
func (c *TeamQuotaChecker) CheckTeamMemberQuota(ctx context.Context, teamOwnerID uuid.UUID, currentCount, pendingInvites int) error {
	sub, err := c.billingService.GetSubscription(ctx, teamOwnerID)
	if err != nil {
		c.logger.Warn("failed to get subscription, allowing request", zap.Error(err))
		return nil
	}

	plan := sub.Plan()
	if plan == nil || plan.IsUnlimitedTeamMembers() {
		return nil
	}

	// Total committed = current members + pending invites
	totalCommitted := currentCount + pendingInvites

	if totalCommitted >= plan.MaxTeamMembers() {
		return ErrTeamMemberQuotaExceeded
	}

	return nil
}

// GetTeamMemberQuotaStatus returns the team member quota status.
func (c *TeamQuotaChecker) GetTeamMemberQuotaStatus(ctx context.Context, teamOwnerID uuid.UUID, currentCount, pendingInvites int) (*TeamMemberQuotaStatus, error) {
	sub, err := c.billingService.GetSubscription(ctx, teamOwnerID)
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	plan := sub.Plan()
	if plan == nil {
		return &TeamMemberQuotaStatus{
			Current:   currentCount,
			Pending:   pendingInvites,
			Limit:     -1,
			Remaining: -1,
			Unlimited: true,
		}, nil
	}

	status := &TeamMemberQuotaStatus{
		Current:   currentCount,
		Pending:   pendingInvites,
		Limit:     plan.MaxTeamMembers(),
		Unlimited: plan.IsUnlimitedTeamMembers(),
	}

	if !status.Unlimited {
		status.Remaining = plan.MaxTeamMembers() - currentCount - pendingInvites
		if status.Remaining < 0 {
			status.Remaining = 0
			status.OverQuota = true
		}
	} else {
		status.Remaining = -1
	}

	return status, nil
}

// TeamMemberQuotaStatus represents the team member quota status.
type TeamMemberQuotaStatus struct {
	Current   int  `json:"current"`    // Current member count
	Pending   int  `json:"pending"`    // Pending invitations
	Limit     int  `json:"limit"`      // Max allowed (-1 for unlimited)
	Remaining int  `json:"remaining"`  // Remaining slots (-1 for unlimited)
	Unlimited bool `json:"unlimited"`  // Whether quota is unlimited
	OverQuota bool `json:"over_quota"` // Whether currently over quota (after downgrade)
}
