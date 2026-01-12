package paymentproto

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	commonv1 "github.com/uniedit/server/api/pb/common"
	paymentv1 "github.com/uniedit/server/api/pb/payment"
	paymenthttp "github.com/uniedit/server/internal/adapter/inbound/http/payment"
	"github.com/uniedit/server/internal/transport/protohttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Handler adapts payment HTTP handlers to proto-defined interfaces.
type Handler struct {
	payment *paymenthttp.PaymentHandler
	refund  *paymenthttp.RefundHandler
	webhook *paymenthttp.WebhookHandler
}

// NewHandler creates a new payment proto adapter.
func NewHandler(
	payment *paymenthttp.PaymentHandler,
	refund *paymenthttp.RefundHandler,
	webhook *paymenthttp.WebhookHandler,
) *Handler {
	return &Handler{
		payment: payment,
		refund:  refund,
		webhook: webhook,
	}
}

// ===== PaymentService =====

func (h *Handler) CreatePaymentIntent(c *gin.Context, in *paymentv1.CreatePaymentIntentRequest) (*paymentv1.PaymentIntentResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.payment.CreatePaymentIntent(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) CreateNativePayment(c *gin.Context, in *paymentv1.CreateNativePaymentRequest) (*paymentv1.NativePaymentResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.payment.CreateNativePayment(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetPayment(c *gin.Context, in *paymentv1.GetByIDRequest) (*paymentv1.Payment, error) {
	h.payment.GetPayment(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListPayments(c *gin.Context, in *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	h.payment.ListPayments(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListPaymentMethods(c *gin.Context, in *commonv1.Empty) (*paymentv1.ListPaymentMethodsResponse, error) {
	h.payment.ListPaymentMethods(c)
	return nil, protohttp.ErrHandled
}

// ===== PaymentAdminService =====

func (h *Handler) CreateRefund(c *gin.Context, in *paymentv1.CreateRefundRequest) (*commonv1.MessageResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.refund.CreateRefund(c)
	return nil, protohttp.ErrHandled
}

// ===== WebhookService =====

func (h *Handler) StripeWebhook(c *gin.Context, _ *commonv1.Empty) (*paymentv1.StripeWebhookResponse, error) {
	h.webhook.HandleStripeWebhook(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) AlipayWebhook(c *gin.Context, _ *commonv1.Empty) (*commonv1.Empty, error) {
	h.webhook.HandleAlipayWebhook(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) WechatWebhook(c *gin.Context, _ *commonv1.Empty) (*paymentv1.WechatWebhookResponse, error) {
	h.webhook.HandleWechatWebhook(c)
	return nil, protohttp.ErrHandled
}

func resetBody(c *gin.Context, msg proto.Message) error {
	if c == nil || c.Request == nil || msg == nil {
		return nil
	}

	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return err
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	c.Request.ContentLength = int64(len(data))
	if c.Request.Header.Get("Content-Type") == "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return nil
}

