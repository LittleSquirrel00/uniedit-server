package billingproto

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	billingv1 "github.com/uniedit/server/api/pb/billing"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	billingDomain billing.BillingDomain
}

func NewHandler(billingDomain billing.BillingDomain) *Handler {
	return &Handler{billingDomain: billingDomain}
}

func (h *Handler) ListPlans(c *gin.Context, _ *commonv1.Empty) (*billingv1.ListPlansResponse, error) {
	plans, err := h.billingDomain.ListPlans(c.Request.Context())
	if err != nil {
		return nil, mapBillingError(err)
	}

	out := make([]*billingv1.Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, toPlan(p))
	}
	return &billingv1.ListPlansResponse{Plans: out}, nil
}

func (h *Handler) GetPlan(c *gin.Context, in *billingv1.GetByIDRequest) (*billingv1.Plan, error) {
	if in.GetId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Plan ID is required"}
	}

	plan, err := h.billingDomain.GetPlan(c.Request.Context(), in.GetId())
	if err != nil {
		return nil, mapBillingError(err)
	}
	return toPlan(plan), nil
}

func (h *Handler) GetSubscription(c *gin.Context, _ *commonv1.Empty) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	sub, err := h.billingDomain.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return toSubscription(sub), nil
}

func (h *Handler) CreateSubscription(c *gin.Context, in *billingv1.CreateSubscriptionRequest) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}
	if in.GetPlanId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_plan_id", Message: "plan_id is required"}
	}

	sub, err := h.billingDomain.CreateSubscription(c.Request.Context(), userID, in.GetPlanId())
	if err != nil {
		return nil, mapBillingError(err)
	}

	c.Status(http.StatusCreated)
	return toSubscription(sub), nil
}

func (h *Handler) CancelSubscription(c *gin.Context, in *billingv1.CancelSubscriptionRequest) (*billingv1.Subscription, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	sub, err := h.billingDomain.CancelSubscription(c.Request.Context(), userID, in.GetImmediately())
	if err != nil {
		return nil, mapBillingError(err)
	}
	return toSubscription(sub), nil
}

func (h *Handler) GetQuotaStatus(c *gin.Context, _ *commonv1.Empty) (*billingv1.QuotaStatus, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	status, err := h.billingDomain.GetQuotaStatus(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return toQuotaStatus(status), nil
}

func (h *Handler) CheckQuota(c *gin.Context, in *billingv1.CheckQuotaRequest) (*billingv1.CheckQuotaResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	if err := h.billingDomain.CheckQuota(c.Request.Context(), userID, in.GetTaskType()); err != nil {
		c.Status(http.StatusForbidden)
		return &billingv1.CheckQuotaResponse{Allowed: false}, nil
	}

	return &billingv1.CheckQuotaResponse{Allowed: true}, nil
}

func (h *Handler) GetBalance(c *gin.Context, _ *commonv1.Empty) (*billingv1.GetBalanceResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	balance, err := h.billingDomain.GetBalance(c.Request.Context(), userID)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return &billingv1.GetBalanceResponse{Balance: balance}, nil
}

func (h *Handler) GetUsageStats(c *gin.Context, in *billingv1.GetUsageStatsRequest) (*billingv1.UsageStats, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	period := in.GetPeriod()
	if period == "" {
		period = "month"
	}

	var start, end *time.Time
	if s := in.GetStart(); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = &t
		}
	}
	if s := in.GetEnd(); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			end = &t
		}
	}

	stats, err := h.billingDomain.GetUsageStats(c.Request.Context(), userID, period, start, end)
	if err != nil {
		return nil, mapBillingError(err)
	}
	return toUsageStats(stats), nil
}

func (h *Handler) RecordUsage(c *gin.Context, in *billingv1.RecordUsageRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	providerID, err := uuid.Parse(in.GetProviderId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_provider_id", Message: "Invalid provider_id", Err: err}
	}

	input := &billing.RecordUsageInput{
		RequestID:    in.GetRequestId(),
		TaskType:     in.GetTaskType(),
		ProviderID:   providerID,
		ModelID:      in.GetModelId(),
		InputTokens:  int(in.GetInputTokens()),
		OutputTokens: int(in.GetOutputTokens()),
		CostUSD:      in.GetCostUsd(),
		LatencyMs:    int(in.GetLatencyMs()),
		Success:      in.GetSuccess(),
	}

	if err := h.billingDomain.RecordUsage(c.Request.Context(), userID, input); err != nil {
		return nil, mapBillingError(err)
	}

	return &commonv1.MessageResponse{Message: "recorded"}, nil
}

func (h *Handler) AddCredits(c *gin.Context, in *billingv1.AddCreditsRequest) (*commonv1.MessageResponse, error) {
	targetUserID, err := uuid.Parse(in.GetUserId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_user_id", Message: "Invalid user_id", Err: err}
	}
	if in.GetAmount() <= 0 {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_amount", Message: "amount must be > 0"}
	}
	if in.GetSource() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_source", Message: "source is required"}
	}

	if err := h.billingDomain.AddCredits(c.Request.Context(), targetUserID, in.GetAmount(), in.GetSource()); err != nil {
		return nil, mapBillingError(err)
	}

	return &commonv1.MessageResponse{Message: "credits added"}, nil
}

func requireUserID(c *gin.Context) (uuid.UUID, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.Nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}
	return userID, nil
}

func toPlan(p *model.Plan) *billingv1.Plan {
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

func toSubscription(s *model.Subscription) *billingv1.Subscription {
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
		Plan:               toPlan(s.Plan),
	}
}

func toQuotaStatus(s *model.QuotaStatus) *billingv1.QuotaStatus {
	if s == nil {
		return nil
	}
	return &billingv1.QuotaStatus{
		Plan:            s.Plan,
		TokensUsed:      s.TokensUsed,
		TokensLimit:     s.TokensLimit,
		TokensRemaining: s.TokensRemaining,
		RequestsToday:   int32(s.RequestsToday),
		RequestsLimit:   int32(s.RequestsLimit),
		ResetAt:         s.ResetAt.UTC().Format(time.RFC3339Nano),
	}
}

func toUsageStats(s *model.UsageStats) *billingv1.UsageStats {
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

func mapBillingError(err error) error {
	switch {
	case errors.Is(err, billing.ErrPlanNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "plan_not_found", Message: "Plan not found", Err: err}
	case errors.Is(err, billing.ErrPlanNotActive):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "plan_not_active", Message: "Plan is not active", Err: err}
	case errors.Is(err, billing.ErrSubscriptionNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "subscription_not_found", Message: "Subscription not found", Err: err}
	case errors.Is(err, billing.ErrSubscriptionExists):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "subscription_exists", Message: "Subscription already exists", Err: err}
	case errors.Is(err, billing.ErrSubscriptionCanceled):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "subscription_canceled", Message: "Subscription already canceled", Err: err}
	case errors.Is(err, billing.ErrQuotaExceeded),
		errors.Is(err, billing.ErrTokenLimitReached),
		errors.Is(err, billing.ErrRequestLimitReached):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "quota_exceeded", Message: err.Error(), Err: err}
	case errors.Is(err, billing.ErrInsufficientCredits):
		return &protohttp.HTTPError{Status: http.StatusPaymentRequired, Code: "insufficient_credits", Message: "Insufficient credits", Err: err}
	case errors.Is(err, billing.ErrInvalidCreditsAmount):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_credits_amount", Message: "Invalid credits amount", Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}

