package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
)

// AlipayConfig holds Alipay configuration.
type AlipayConfig struct {
	AppID           string // Application ID
	PrivateKey      string // RSA2 private key (PEM format)
	AlipayPublicKey string // Alipay public key for verification (PEM format)
	IsProd          bool   // Production environment flag
	NotifyURL       string // Default notify URL
	ReturnURL       string // Default return URL
}

// AlipayProvider implements NativePaymentProvider for Alipay.
type AlipayProvider struct {
	client    *alipay.Client
	config    *AlipayConfig
	notifyURL string
	returnURL string
}

// NewAlipayProvider creates a new Alipay provider.
func NewAlipayProvider(config *AlipayConfig) (*AlipayProvider, error) {
	client, err := alipay.NewClient(config.AppID, config.PrivateKey, config.IsProd)
	if err != nil {
		return nil, fmt.Errorf("create alipay client: %w", err)
	}

	// Set public key for auto signature verification
	client.AutoVerifySign([]byte(config.AlipayPublicKey))

	return &AlipayProvider{
		client:    client,
		config:    config,
		notifyURL: config.NotifyURL,
		returnURL: config.ReturnURL,
	}, nil
}

// Name returns the provider name.
func (p *AlipayProvider) Name() string {
	return "alipay"
}

// --- Customer Management (Not supported for Alipay) ---

func (p *AlipayProvider) CreateCustomer(ctx context.Context, email, name string) (*Customer, error) {
	return nil, errors.New("alipay does not support customer management")
}

func (p *AlipayProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	return nil, errors.New("alipay does not support customer management")
}

// --- Payment Intents (Redirect to CreateNativePayment) ---

func (p *AlipayProvider) CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*PaymentIntent, error) {
	return nil, errors.New("use CreateNativePayment for Alipay payments")
}

func (p *AlipayProvider) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntent, error) {
	return nil, errors.New("use QueryPayment for Alipay payments")
}

func (p *AlipayProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	return errors.New("use ClosePayment for Alipay payments")
}

// --- Subscriptions (Not supported for Alipay in this implementation) ---

func (p *AlipayProvider) CreateSubscription(ctx context.Context, customerID, priceID string) (*Subscription, error) {
	return nil, errors.New("alipay subscriptions require separate agreement signing flow")
}

func (p *AlipayProvider) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	return nil, errors.New("alipay subscriptions not supported")
}

func (p *AlipayProvider) UpdateSubscription(ctx context.Context, subscriptionID string, params map[string]interface{}) (*Subscription, error) {
	return nil, errors.New("alipay subscriptions not supported")
}

func (p *AlipayProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return errors.New("alipay subscriptions not supported")
}

// --- Refunds ---

func (p *AlipayProvider) CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*Refund, error) {
	// For Alipay, use RefundPayment instead
	return nil, errors.New("use RefundPayment for Alipay refunds")
}

// --- Payment Methods (Not applicable for Alipay) ---

func (p *AlipayProvider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return errors.New("alipay does not support payment method attachment")
}

func (p *AlipayProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return errors.New("alipay does not support payment method detachment")
}

func (p *AlipayProvider) ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodDetails, error) {
	return nil, errors.New("alipay does not support listing payment methods")
}

func (p *AlipayProvider) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return errors.New("alipay does not support setting default payment method")
}

// --- Webhooks ---

func (p *AlipayProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	// Alipay verification is done in ParseNotify
	return nil
}

// --- Native Payment Methods ---

// CreateNativePayment creates a payment order for Alipay.
func (p *AlipayProvider) CreateNativePayment(ctx context.Context, scene PaymentScene, orderID string, amount int64, subject, description, notifyURL, returnURL string, metadata map[string]string) (*NativePaymentOrder, error) {
	if notifyURL == "" {
		notifyURL = p.notifyURL
	}
	if returnURL == "" {
		returnURL = p.returnURL
	}

	// Convert cents to yuan (Alipay uses yuan with 2 decimal places)
	amountStr := fmt.Sprintf("%.2f", float64(amount)/100)

	// Set timeout (30 minutes)
	expireTime := time.Now().Add(30 * time.Minute)
	timeoutExpress := "30m"

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", orderID)
	bm.Set("total_amount", amountStr)
	bm.Set("subject", subject)
	bm.Set("timeout_express", timeoutExpress)

	if description != "" {
		bm.Set("body", description)
	}

	// Add metadata as passback_params
	if len(metadata) > 0 {
		passbackData, _ := json.Marshal(metadata)
		bm.Set("passback_params", string(passbackData))
	}

	result := &NativePaymentOrder{
		OrderID:    orderID,
		Amount:     amount,
		Currency:   "CNY",
		ExpireTime: expireTime.Unix(),
	}

	switch scene {
	case PaymentSceneWeb:
		// PC web payment (redirect)
		bm.Set("product_code", "FAST_INSTANT_TRADE_PAY")
		payURL, err := p.client.TradePagePay(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create web payment: %w", err)
		}
		result.PayURL = payURL

	case PaymentSceneH5:
		// Mobile H5 payment
		bm.Set("product_code", "QUICK_WAP_WAY")
		payURL, err := p.client.TradeWapPay(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create h5 payment: %w", err)
		}
		result.PayURL = payURL

	case PaymentSceneApp:
		// App SDK payment
		bm.Set("product_code", "QUICK_MSECURITY_PAY")
		payStr, err := p.client.TradeAppPay(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create app payment: %w", err)
		}
		result.AppPayData = payStr

	case PaymentSceneNative:
		// QR code payment (precreate)
		bm.Set("product_code", "FACE_TO_FACE_PAYMENT")
		resp, err := p.client.TradePrecreate(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create native payment: %w", err)
		}
		if resp.Response.Code != "10000" {
			return nil, fmt.Errorf("alipay error: %s - %s", resp.Response.Code, resp.Response.Msg)
		}
		result.QRCode = resp.Response.QrCode
		result.TradeNo = resp.Response.OutTradeNo

	default:
		return nil, fmt.Errorf("unsupported payment scene: %s", scene)
	}

	return result, nil
}

// QueryPayment queries the payment status.
func (p *AlipayProvider) QueryPayment(ctx context.Context, orderID, tradeNo string) (*NotifyResult, error) {
	bm := make(gopay.BodyMap)
	if tradeNo != "" {
		bm.Set("trade_no", tradeNo)
	} else {
		bm.Set("out_trade_no", orderID)
	}

	resp, err := p.client.TradeQuery(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("query payment: %w", err)
	}

	if resp.Response.Code != "10000" {
		return nil, fmt.Errorf("alipay query error: %s - %s", resp.Response.Code, resp.Response.Msg)
	}

	// Parse amount from string
	amount, _ := strconv.ParseFloat(resp.Response.TotalAmount, 64)
	amountCents := int64(amount * 100)

	// Parse pay time
	var payTime int64
	if resp.Response.SendPayDate != "" {
		t, err := time.Parse("2006-01-02 15:04:05", resp.Response.SendPayDate)
		if err == nil {
			payTime = t.Unix()
		}
	}

	return &NotifyResult{
		TradeNo:    resp.Response.TradeNo,
		OutTradeNo: resp.Response.OutTradeNo,
		Amount:     amountCents,
		Status:     mapAlipayTradeStatus(resp.Response.TradeStatus),
		PayerID:    resp.Response.BuyerUserId,
		PayTime:    payTime,
	}, nil
}

// ClosePayment closes an unpaid order.
func (p *AlipayProvider) ClosePayment(ctx context.Context, orderID, tradeNo string) error {
	bm := make(gopay.BodyMap)
	if tradeNo != "" {
		bm.Set("trade_no", tradeNo)
	} else {
		bm.Set("out_trade_no", orderID)
	}

	resp, err := p.client.TradeClose(ctx, bm)
	if err != nil {
		return fmt.Errorf("close payment: %w", err)
	}

	if resp.Response.Code != "10000" {
		return fmt.Errorf("alipay close error: %s - %s", resp.Response.Code, resp.Response.Msg)
	}

	return nil
}

// RefundPayment creates a refund.
func (p *AlipayProvider) RefundPayment(ctx context.Context, orderID, tradeNo, refundID string, refundAmount, totalAmount int64, reason string) (*RefundResult, error) {
	bm := make(gopay.BodyMap)
	if tradeNo != "" {
		bm.Set("trade_no", tradeNo)
	} else {
		bm.Set("out_trade_no", orderID)
	}
	bm.Set("out_request_no", refundID)
	bm.Set("refund_amount", fmt.Sprintf("%.2f", float64(refundAmount)/100))

	if reason != "" {
		bm.Set("refund_reason", reason)
	}

	resp, err := p.client.TradeRefund(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("refund payment: %w", err)
	}

	if resp.Response.Code != "10000" {
		return nil, fmt.Errorf("alipay refund error: %s - %s", resp.Response.Code, resp.Response.Msg)
	}

	// Parse refund amount
	refundAmt, _ := strconv.ParseFloat(resp.Response.RefundFee, 64)

	return &RefundResult{
		RefundNo:    resp.Response.TradeNo,
		OutRefundNo: refundID,
		Amount:      int64(refundAmt * 100),
		Status:      "success",
		RefundTime:  time.Now().Unix(),
	}, nil
}

// QueryRefund queries the refund status.
func (p *AlipayProvider) QueryRefund(ctx context.Context, orderID, refundID string) (*RefundResult, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", orderID)
	bm.Set("out_request_no", refundID)

	resp, err := p.client.TradeFastPayRefundQuery(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("query refund: %w", err)
	}

	if resp.Response.Code != "10000" {
		return nil, fmt.Errorf("alipay refund query error: %s - %s", resp.Response.Code, resp.Response.Msg)
	}

	// Parse refund amount
	refundAmt, _ := strconv.ParseFloat(resp.Response.RefundAmount, 64)

	status := "pending"
	if resp.Response.RefundStatus == "REFUND_SUCCESS" {
		status = "success"
	}

	return &RefundResult{
		RefundNo:    resp.Response.TradeNo,
		OutRefundNo: resp.Response.OutRequestNo,
		Amount:      int64(refundAmt * 100),
		Status:      status,
	}, nil
}

// ParseNotify parses and verifies the async notification.
func (p *AlipayProvider) ParseNotify(ctx context.Context, body []byte, headers map[string]string) (*NotifyResult, error) {
	// Convert body to *http.Request for gopay SDK
	// Alipay sends form-urlencoded data
	req, err := http.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Parse notification data
	notifyReq, err := alipay.ParseNotifyToBodyMap(req)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}

	// Verify signature
	ok, err := alipay.VerifySign(p.config.AlipayPublicKey, notifyReq)
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return nil, errors.New("invalid signature")
	}

	// Parse amount
	totalAmount := notifyReq.Get("total_amount")
	amount, _ := strconv.ParseFloat(totalAmount, 64)
	amountCents := int64(amount * 100)

	// Parse pay time
	var payTime int64
	gmtPayment := notifyReq.Get("gmt_payment")
	if gmtPayment != "" {
		t, err := time.Parse("2006-01-02 15:04:05", gmtPayment)
		if err == nil {
			payTime = t.Unix()
		}
	}

	rawData, _ := json.Marshal(notifyReq)

	return &NotifyResult{
		TradeNo:     notifyReq.Get("trade_no"),
		OutTradeNo:  notifyReq.Get("out_trade_no"),
		Amount:      amountCents,
		Status:      mapAlipayTradeStatus(notifyReq.Get("trade_status")),
		PayerID:     notifyReq.Get("buyer_id"),
		PayTime:     payTime,
		RawData:     string(rawData),
		SuccessResp: "success", // Alipay expects "success" as response
	}, nil
}

// mapAlipayTradeStatus maps Alipay trade status to our standard status.
func mapAlipayTradeStatus(status string) string {
	switch status {
	case "WAIT_BUYER_PAY":
		return "pending"
	case "TRADE_CLOSED":
		return "closed"
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return "success"
	default:
		return status
	}
}
