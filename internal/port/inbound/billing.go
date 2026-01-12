package inbound

import (
	"context"

	"github.com/google/uuid"
	billingv1 "github.com/uniedit/server/api/pb/billing"
	commonv1 "github.com/uniedit/server/api/pb/common"
)

// BillingDomain defines the billing inbound port with pb request/response pass-through.
type BillingDomain interface {
	ListPlans(ctx context.Context) (*billingv1.ListPlansResponse, error)
	GetPlan(ctx context.Context, in *billingv1.GetByIDRequest) (*billingv1.Plan, error)

	GetSubscription(ctx context.Context, userID uuid.UUID) (*billingv1.Subscription, error)
	CreateSubscription(ctx context.Context, userID uuid.UUID, in *billingv1.CreateSubscriptionRequest) (*billingv1.Subscription, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID, in *billingv1.CancelSubscriptionRequest) (*billingv1.Subscription, error)

	GetQuotaStatus(ctx context.Context, userID uuid.UUID) (*billingv1.QuotaStatus, error)
	CheckQuota(ctx context.Context, userID uuid.UUID, in *billingv1.CheckQuotaRequest) error

	GetBalance(ctx context.Context, userID uuid.UUID) (*billingv1.GetBalanceResponse, error)
	GetUsageStats(ctx context.Context, userID uuid.UUID, in *billingv1.GetUsageStatsRequest) (*billingv1.UsageStats, error)

	RecordUsage(ctx context.Context, userID uuid.UUID, in *billingv1.RecordUsageRequest) (*commonv1.MessageResponse, error)
	AddCredits(ctx context.Context, in *billingv1.AddCreditsRequest) (*commonv1.MessageResponse, error)
}

