package billing

import (
	"time"

	billingv1 "github.com/uniedit/server/api/pb/billing"
	"github.com/uniedit/server/internal/model"
)

func toPlanPB(p *model.Plan) *billingv1.Plan {
	if p == nil {
		return nil
	}
	return &billingv1.Plan{
		Id:                     p.ID,
		Type:                   string(p.Type),
		Name:                   p.Name,
		Description:            p.Description,
		BillingCycle:           string(p.BillingCycle),
		PriceUsd:               p.PriceUSD,
		MonthlyTokens:          p.MonthlyTokens,
		DailyRequests:          int32(p.DailyRequests),
		MaxApiKeys:             int32(p.MaxAPIKeys),
		Features:               []string(p.Features),
		MonthlyChatTokens:      p.MonthlyChatTokens,
		MonthlyImageCredits:    int32(p.MonthlyImageCredits),
		MonthlyVideoMinutes:    int32(p.MonthlyVideoMinutes),
		MonthlyEmbeddingTokens: p.MonthlyEmbeddingTokens,
		GitStorageMb:           p.GitStorageMB,
		LfsStorageMb:           p.LFSStorageMB,
		MaxTeamMembers:         int32(p.MaxTeamMembers),
	}
}

func toSubscriptionPB(s *model.Subscription) *billingv1.Subscription {
	if s == nil {
		return nil
	}

	var canceledAt string
	if s.CanceledAt != nil {
		canceledAt = s.CanceledAt.UTC().Format(time.RFC3339Nano)
	}

	return &billingv1.Subscription{
		Id:                 s.ID.String(),
		PlanId:             s.PlanID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.UTC().Format(time.RFC3339Nano),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.UTC().Format(time.RFC3339Nano),
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		CanceledAt:         canceledAt,
		CreditsBalance:     s.CreditsBalance,
		CreatedAt:          s.CreatedAt.UTC().Format(time.RFC3339Nano),
		Plan:               toPlanPB(s.Plan),
	}
}

func toUsageStatsPB(s *model.UsageStats) *billingv1.UsageStats {
	if s == nil {
		return nil
	}

	out := &billingv1.UsageStats{
		TotalTokens:   s.TotalTokens,
		TotalRequests: int32(s.TotalRequests),
		TotalCostUsd:  s.TotalCostUSD,
		ByModel:       make(map[string]*billingv1.ModelUsage, len(s.ByModel)),
	}

	for k, v := range s.ByModel {
		if v == nil {
			continue
		}
		out.ByModel[k] = &billingv1.ModelUsage{
			ModelId:       v.ModelID,
			TotalTokens:   v.TotalTokens,
			TotalRequests: int32(v.TotalRequests),
			TotalCostUsd:  v.TotalCostUSD,
		}
	}

	out.ByDay = make([]*billingv1.DailyUsage, 0, len(s.ByDay))
	for _, d := range s.ByDay {
		if d == nil {
			continue
		}
		out.ByDay = append(out.ByDay, &billingv1.DailyUsage{
			Date:          d.Date,
			TotalTokens:   d.TotalTokens,
			TotalRequests: int32(d.TotalRequests),
			TotalCostUsd:  d.TotalCostUSD,
		})
	}

	return out
}

