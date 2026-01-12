package paymentproto

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	paymentv1 "github.com/uniedit/server/api/pb/payment"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	paymentDomain payment.PaymentDomain
}

func NewHandler(paymentDomain payment.PaymentDomain) *Handler {
	return &Handler{paymentDomain: paymentDomain}
}

// ===== PaymentService =====

func (h *Handler) CreatePaymentIntent(c *gin.Context, in *paymentv1.CreatePaymentIntentRequest) (*paymentv1.PaymentIntentResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	orderID, err := uuid.Parse(in.GetOrderId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_order_id", Message: "Invalid order_id", Err: err}
	}

	resp, err := h.paymentDomain.CreatePaymentIntent(c.Request.Context(), orderID, userID)
	if err != nil {
		return nil, mapPaymentError(err)
	}

	return &paymentv1.PaymentIntentResponse{
		PaymentIntentId: resp.PaymentIntentID,
		ClientSecret:    resp.ClientSecret,
		Amount:          resp.Amount,
		Currency:        resp.Currency,
	}, nil
}

func (h *Handler) CreateNativePayment(c *gin.Context, in *paymentv1.CreateNativePaymentRequest) (*paymentv1.NativePaymentResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	orderID, err := uuid.Parse(in.GetOrderId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_order_id", Message: "Invalid order_id", Err: err}
	}

	resp, err := h.paymentDomain.CreateNativePayment(c.Request.Context(), &model.CreateNativePaymentRequest{
		OrderID:   orderID,
		Method:    in.GetMethod(),
		Scene:     model.PaymentScene(in.GetScene()),
		ReturnURL: in.GetReturnUrl(),
		OpenID:    in.GetOpenid(),
	}, userID)
	if err != nil {
		return nil, mapPaymentError(err)
	}

	return &paymentv1.NativePaymentResponse{
		PaymentId:    resp.PaymentID.String(),
		OrderId:      resp.OrderID.String(),
		Method:       resp.Method,
		PayUrl:       resp.PayURL,
		QrCode:       resp.QRCode,
		AppPayData:   resp.AppPayData,
		MiniPayData:  resp.MiniPayData,
		Amount:       resp.Amount,
		Currency:     resp.Currency,
		ExpireTime:   resp.ExpireTime,
	}, nil
}

func (h *Handler) GetPayment(c *gin.Context, in *paymentv1.GetByIDRequest) (*paymentv1.Payment, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	paymentID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid payment id", Err: err}
	}

	p, err := h.paymentDomain.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		return nil, mapPaymentError(err)
	}
	if p == nil {
		return nil, &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: "Payment not found"}
	}
	if p.UserID != userID {
		return nil, &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "Forbidden"}
	}

	return toPayment(p), nil
}

func (h *Handler) ListPayments(c *gin.Context, in *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	filter := model.PaymentFilter{
		UserID: &userID,
		PaginationRequest: model.PaginationRequest{
			Page:     int(in.GetPage()),
			PageSize: int(in.GetPageSize()),
		},
	}

	if in.GetOrderId() != "" {
		if id, err := uuid.Parse(in.GetOrderId()); err == nil {
			filter.OrderID = &id
		} else {
			return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_order_id", Message: "Invalid order_id", Err: err}
		}
	}
	if in.GetStatus() != "" {
		s := model.PaymentStatus(in.GetStatus())
		filter.Status = &s
	}
	if in.GetProvider() != "" {
		v := in.GetProvider()
		filter.Provider = &v
	}

	filter.DefaultPagination()
	payments, total, err := h.paymentDomain.ListPayments(c.Request.Context(), filter)
	if err != nil {
		return nil, mapPaymentError(err)
	}

	out := make([]*paymentv1.Payment, 0, len(payments))
	for _, p := range payments {
		out = append(out, toPayment(p))
	}

	totalPages := int32((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	return &paymentv1.ListPaymentsResponse{
		Data:       out,
		Total:      total,
		Page:       int32(filter.Page),
		PageSize:   int32(filter.PageSize),
		TotalPages: totalPages,
	}, nil
}

func (h *Handler) ListPaymentMethods(c *gin.Context, _ *commonv1.Empty) (*paymentv1.ListPaymentMethodsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	methods, err := h.paymentDomain.ListPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		return nil, mapPaymentError(err)
	}

	out := make([]*paymentv1.PaymentMethodInfo, 0, len(methods))
	for _, m := range methods {
		if m == nil {
			continue
		}
		out = append(out, &paymentv1.PaymentMethodInfo{
			Id:        m.ID,
			Type:      m.Type,
			CardBrand: m.CardBrand,
			CardLast4: m.CardLast4,
			ExpMonth:  int32(m.ExpMonth),
			ExpYear:   int32(m.ExpYear),
			IsDefault: m.IsDefault,
		})
	}
	return &paymentv1.ListPaymentMethodsResponse{Methods: out}, nil
}

// ===== PaymentAdminService =====

func (h *Handler) CreateRefund(c *gin.Context, in *paymentv1.CreateRefundRequest) (*commonv1.MessageResponse, error) {
	_, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	paymentID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid payment id", Err: err}
	}

	if err := h.paymentDomain.CreateRefund(c.Request.Context(), paymentID, in.GetAmount(), in.GetReason()); err != nil {
		return nil, mapPaymentError(err)
	}
	return &commonv1.MessageResponse{Message: "refund created"}, nil
}

// ===== WebhookService =====

func (h *Handler) StripeWebhook(c *gin.Context, _ *commonv1.Empty) (*paymentv1.StripeWebhookResponse, error) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_payload", Message: "Failed to read request body", Err: err}
	}

	signature := c.GetHeader("Stripe-Signature")
	if err := h.paymentDomain.VerifyWebhookSignature(payload, signature); err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_signature", Message: "Webhook signature verification failed", Err: err}
	}

	return &paymentv1.StripeWebhookResponse{Received: true}, nil
}

func (h *Handler) AlipayWebhook(c *gin.Context, _ *commonv1.Empty) (*commonv1.Empty, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return nil, protohttp.ErrHandled
	}

	headers := make(map[string]string, len(c.Request.Header))
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := h.paymentDomain.HandleNativePaymentNotify(c.Request.Context(), "alipay", body, headers)
	if err != nil {
		c.String(http.StatusInternalServerError, "fail")
		return nil, protohttp.ErrHandled
	}

	c.String(http.StatusOK, resp)
	return nil, protohttp.ErrHandled
}

func (h *Handler) WechatWebhook(c *gin.Context, _ *commonv1.Empty) (*paymentv1.WechatWebhookResponse, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return &paymentv1.WechatWebhookResponse{Code: "FAIL", Message: "failed to read body"}, nil
	}

	headers := make(map[string]string, len(c.Request.Header))
	for key := range c.Request.Header {
		headers[key] = c.GetHeader(key)
	}

	resp, err := h.paymentDomain.HandleNativePaymentNotify(c.Request.Context(), "wechat", body, headers)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return &paymentv1.WechatWebhookResponse{Code: "FAIL", Message: err.Error()}, nil
	}

	return &paymentv1.WechatWebhookResponse{Code: "SUCCESS", Message: resp}, nil
}

func requireUserID(c *gin.Context) (uuid.UUID, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.Nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}
	return userID, nil
}

func toPayment(p *model.Payment) *paymentv1.Payment {
	if p == nil {
		return nil
	}

	failureCode := ""
	if p.FailureCode != nil {
		failureCode = *p.FailureCode
	}
	failureMessage := ""
	if p.FailureMessage != nil {
		failureMessage = *p.FailureMessage
	}
	succeededAt := ""
	if p.SucceededAt != nil {
		succeededAt = formatTime(*p.SucceededAt)
	}
	failedAt := ""
	if p.FailedAt != nil {
		failedAt = formatTime(*p.FailedAt)
	}

	return &paymentv1.Payment{
		Id:                 p.ID.String(),
		OrderId:            p.OrderID.String(),
		UserId:             p.UserID.String(),
		Amount:             p.Amount,
		Currency:           p.Currency,
		Method:             string(p.Method),
		Status:             string(p.Status),
		Provider:           p.Provider,
		StripePaymentIntentId: p.StripePaymentIntentID,
		StripeChargeId:     p.StripeChargeID,
		TradeNo:            p.TradeNo,
		PayerId:            p.PayerID,
		FailureCode:        failureCode,
		FailureMessage:     failureMessage,
		RefundedAmount:     p.RefundedAmount,
		SucceededAt:        succeededAt,
		FailedAt:           failedAt,
		CreatedAt:          formatTime(p.CreatedAt),
		UpdatedAt:          formatTime(p.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func mapPaymentError(err error) error {
	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "payment_not_found", Message: "Payment not found", Err: err}
	case errors.Is(err, payment.ErrForbidden):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "Forbidden", Err: err}
	case errors.Is(err, payment.ErrOrderNotPending):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "order_not_pending", Message: "Order is not pending", Err: err}
	case errors.Is(err, payment.ErrPaymentNotSucceeded):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "payment_not_succeeded", Message: "Payment is not succeeded", Err: err}
	case errors.Is(err, payment.ErrInvalidRefundAmount):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_refund_amount", Message: "Invalid refund amount", Err: err}
	case errors.Is(err, payment.ErrProviderNotAvailable):
		return &protohttp.HTTPError{Status: http.StatusServiceUnavailable, Code: "provider_unavailable", Message: "Payment provider not available", Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}

