package payment

import (
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/payment/domain"
	"github.com/uniedit/server/internal/module/payment/provider"
)

// CreatePaymentIntentRequest represents a request to create a payment intent.
type CreatePaymentIntentRequest struct {
	OrderID uuid.UUID `json:"order_id" binding:"required"`
}

// CreateNativePaymentRequest represents a request to create a native payment (Alipay/WeChat).
type CreateNativePaymentRequest struct {
	OrderID   uuid.UUID            `json:"order_id" binding:"required"`
	Method    PaymentMethod        `json:"method" binding:"required,oneof=alipay wechat"`
	Scene     provider.PaymentScene `json:"scene" binding:"required,oneof=web h5 app native mini"`
	OpenID    string               `json:"openid,omitempty"` // Required for WeChat mini/jsapi
	ReturnURL string               `json:"return_url,omitempty"`
}

// ConfirmPaymentRequest represents a request to confirm a payment.
type ConfirmPaymentRequest struct {
	PaymentIntentID string `json:"payment_intent_id" binding:"required"`
}

// AttachPaymentMethodRequest represents a request to attach a payment method.
type AttachPaymentMethodRequest struct {
	PaymentMethodID string `json:"payment_method_id" binding:"required"`
}

// RefundRequest represents a refund request.
type RefundRequest struct {
	Amount int64  `json:"amount,omitempty"` // Full refund if 0
	Reason string `json:"reason"`
}

// PaymentIntentResponse represents a payment intent for the frontend.
type PaymentIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
}

// NativePaymentResponse represents a native payment order for the frontend.
type NativePaymentResponse struct {
	PaymentID   uuid.UUID         `json:"payment_id"`
	OrderID     uuid.UUID         `json:"order_id"`
	Method      PaymentMethod     `json:"method"`
	PayURL      string            `json:"pay_url,omitempty"`      // Redirect URL for web/h5
	QRCode      string            `json:"qr_code,omitempty"`      // QR code content for native
	AppPayData  string            `json:"app_pay_data,omitempty"` // App SDK payment data
	MiniPayData map[string]string `json:"mini_pay_data,omitempty"` // Mini program payment params
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	ExpireTime  int64             `json:"expire_time"` // Unix timestamp
}

// PaymentResponse represents a payment in API responses.
type PaymentResponse struct {
	ID             uuid.UUID     `json:"id"`
	OrderID        uuid.UUID     `json:"order_id"`
	Amount         int64         `json:"amount"`
	Currency       string        `json:"currency"`
	Method         PaymentMethod `json:"method"`
	Status         PaymentStatus `json:"status"`
	Provider       string        `json:"provider"`
	FailureCode    *string       `json:"failure_code,omitempty"`
	FailureMessage *string       `json:"failure_message,omitempty"`
	RefundedAmount int64         `json:"refunded_amount"`
	SucceededAt    *time.Time    `json:"succeeded_at,omitempty"`
	FailedAt       *time.Time    `json:"failed_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

// PaymentToResponse converts a domain Payment to PaymentResponse.
func PaymentToResponse(p *domain.Payment) *PaymentResponse {
	return &PaymentResponse{
		ID:             p.ID(),
		OrderID:        p.OrderID(),
		Amount:         p.Amount(),
		Currency:       p.Currency(),
		Method:         PaymentMethod(p.Method()),
		Status:         PaymentStatus(p.Status()),
		Provider:       p.Provider(),
		FailureCode:    p.FailureCode(),
		FailureMessage: p.FailureMessage(),
		RefundedAmount: p.RefundedAmount(),
		SucceededAt:    p.SucceededAt(),
		FailedAt:       p.FailedAt(),
		CreatedAt:      p.CreatedAt(),
	}
}

// PaymentMethodInfo represents a stored payment method.
type PaymentMethodInfo struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Card     *CardInfo `json:"card,omitempty"`
	IsDefault bool   `json:"is_default"`
}

// CardInfo represents card details.
type CardInfo struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}
