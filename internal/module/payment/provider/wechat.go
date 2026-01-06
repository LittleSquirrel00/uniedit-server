package provider

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
)

// WechatConfig holds WeChat Pay configuration.
type WechatConfig struct {
	AppID                 string // Application ID (公众号/小程序/APP)
	MchID                 string // Merchant ID
	APIKeyV3              string // APIv3 Key
	SerialNo              string // Certificate serial number
	PrivateKey            string // Private key (PEM format)
	WechatPublicKeySerial string // WeChat platform certificate serial (for signature verification)
	WechatPublicKey       string // WeChat platform public key (PEM format)
	IsProd                bool   // Production environment flag
	NotifyURL             string // Default notify URL
}

// WechatProvider implements NativePaymentProvider for WeChat Pay.
type WechatProvider struct {
	client    *wechat.ClientV3
	config    *WechatConfig
	notifyURL string
}

// NewWechatProvider creates a new WeChat Pay provider.
func NewWechatProvider(config *WechatConfig) (*WechatProvider, error) {
	client, err := wechat.NewClientV3(
		config.MchID,
		config.SerialNo,
		config.APIKeyV3,
		config.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create wechat client: %w", err)
	}

	// Set production mode
	if config.IsProd {
		client.SetPlatformCert([]byte(config.WechatPublicKey), config.WechatPublicKeySerial)
	}

	return &WechatProvider{
		client:    client,
		config:    config,
		notifyURL: config.NotifyURL,
	}, nil
}

// Name returns the provider name.
func (p *WechatProvider) Name() string {
	return "wechat"
}

// --- Customer Management (Not supported for WeChat) ---

func (p *WechatProvider) CreateCustomer(ctx context.Context, email, name string) (*Customer, error) {
	return nil, errors.New("wechat does not support customer management")
}

func (p *WechatProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	return nil, errors.New("wechat does not support customer management")
}

// --- Payment Intents (Redirect to CreateNativePayment) ---

func (p *WechatProvider) CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*PaymentIntent, error) {
	return nil, errors.New("use CreateNativePayment for WeChat payments")
}

func (p *WechatProvider) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntent, error) {
	return nil, errors.New("use QueryPayment for WeChat payments")
}

func (p *WechatProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	return errors.New("use ClosePayment for WeChat payments")
}

// --- Subscriptions (Not supported for WeChat in this implementation) ---

func (p *WechatProvider) CreateSubscription(ctx context.Context, customerID, priceID string) (*Subscription, error) {
	return nil, errors.New("wechat subscriptions require separate contract signing flow")
}

func (p *WechatProvider) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	return nil, errors.New("wechat subscriptions not supported")
}

func (p *WechatProvider) UpdateSubscription(ctx context.Context, subscriptionID string, params map[string]interface{}) (*Subscription, error) {
	return nil, errors.New("wechat subscriptions not supported")
}

func (p *WechatProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return errors.New("wechat subscriptions not supported")
}

// --- Refunds ---

func (p *WechatProvider) CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*Refund, error) {
	return nil, errors.New("use RefundPayment for WeChat refunds")
}

// --- Payment Methods (Not applicable for WeChat) ---

func (p *WechatProvider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return errors.New("wechat does not support payment method attachment")
}

func (p *WechatProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return errors.New("wechat does not support payment method detachment")
}

func (p *WechatProvider) ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodDetails, error) {
	return nil, errors.New("wechat does not support listing payment methods")
}

func (p *WechatProvider) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return errors.New("wechat does not support setting default payment method")
}

// --- Webhooks ---

func (p *WechatProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	// WeChat verification is done in ParseNotify
	return nil
}

// --- Native Payment Methods ---

// CreateNativePayment creates a payment order for WeChat Pay.
func (p *WechatProvider) CreateNativePayment(ctx context.Context, scene PaymentScene, orderID string, amount int64, subject, description, notifyURL, returnURL string, metadata map[string]string) (*NativePaymentOrder, error) {
	if notifyURL == "" {
		notifyURL = p.notifyURL
	}

	// WeChat uses cents (分)
	expireTime := time.Now().Add(30 * time.Minute)

	bm := make(gopay.BodyMap)
	bm.Set("appid", p.config.AppID)
	bm.Set("mchid", p.config.MchID)
	bm.Set("description", subject)
	bm.Set("out_trade_no", orderID)
	bm.Set("time_expire", expireTime.Format(time.RFC3339))
	bm.Set("notify_url", notifyURL)
	bm.SetBodyMap("amount", func(am gopay.BodyMap) {
		am.Set("total", amount)
		am.Set("currency", "CNY")
	})

	// Add metadata as attach
	if len(metadata) > 0 {
		attachData, _ := json.Marshal(metadata)
		bm.Set("attach", string(attachData))
	}

	result := &NativePaymentOrder{
		OrderID:    orderID,
		Amount:     amount,
		Currency:   "CNY",
		ExpireTime: expireTime.Unix(),
	}

	switch scene {
	case PaymentSceneNative:
		// Native payment (QR code)
		resp, err := p.client.V3TransactionNative(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create native payment: %w", err)
		}
		if resp.Code != wechat.Success {
			return nil, fmt.Errorf("wechat error: %d - %s", resp.Code, resp.Error)
		}
		result.QRCode = resp.Response.CodeUrl

	case PaymentSceneH5:
		// H5 payment (mobile web)
		bm.SetBodyMap("scene_info", func(sm gopay.BodyMap) {
			sm.Set("payer_client_ip", "127.0.0.1") // Should be actual client IP
			sm.SetBodyMap("h5_info", func(h5 gopay.BodyMap) {
				h5.Set("type", "Wap")
			})
		})
		resp, err := p.client.V3TransactionH5(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create h5 payment: %w", err)
		}
		if resp.Code != wechat.Success {
			return nil, fmt.Errorf("wechat error: %d - %s", resp.Code, resp.Error)
		}
		result.PayURL = resp.Response.H5Url

	case PaymentSceneApp:
		// APP payment
		resp, err := p.client.V3TransactionApp(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create app payment: %w", err)
		}
		if resp.Code != wechat.Success {
			return nil, fmt.Errorf("wechat error: %d - %s", resp.Code, resp.Error)
		}
		// Build app payment params
		appParams, err := p.client.PaySignOfApp(p.config.AppID, resp.Response.PrepayId)
		if err != nil {
			return nil, fmt.Errorf("sign app payment: %w", err)
		}
		appData, _ := json.Marshal(appParams)
		result.AppPayData = string(appData)

	case PaymentSceneMini:
		// Mini program payment (requires openid)
		openid := ""
		if metadata != nil {
			openid = metadata["openid"]
		}
		if openid == "" {
			return nil, errors.New("openid is required for mini program payment")
		}
		bm.SetBodyMap("payer", func(pm gopay.BodyMap) {
			pm.Set("openid", openid)
		})
		resp, err := p.client.V3TransactionJsapi(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create jsapi payment: %w", err)
		}
		if resp.Code != wechat.Success {
			return nil, fmt.Errorf("wechat error: %d - %s", resp.Code, resp.Error)
		}
		// Build mini program payment params
		miniParams, err := p.client.PaySignOfJSAPI(p.config.AppID, resp.Response.PrepayId)
		if err != nil {
			return nil, fmt.Errorf("sign mini payment: %w", err)
		}
		result.MiniPayData = map[string]string{
			"timeStamp": miniParams.TimeStamp,
			"nonceStr":  miniParams.NonceStr,
			"package":   miniParams.Package,
			"signType":  miniParams.SignType,
			"paySign":   miniParams.PaySign,
		}

	case PaymentSceneWeb:
		// JSAPI payment (公众号) - requires openid
		openid := ""
		if metadata != nil {
			openid = metadata["openid"]
		}
		if openid == "" {
			return nil, errors.New("openid is required for JSAPI payment")
		}
		bm.SetBodyMap("payer", func(pm gopay.BodyMap) {
			pm.Set("openid", openid)
		})
		resp, err := p.client.V3TransactionJsapi(ctx, bm)
		if err != nil {
			return nil, fmt.Errorf("create jsapi payment: %w", err)
		}
		if resp.Code != wechat.Success {
			return nil, fmt.Errorf("wechat error: %d - %s", resp.Code, resp.Error)
		}
		// Build JSAPI payment params
		jsapiParams, err := p.client.PaySignOfJSAPI(p.config.AppID, resp.Response.PrepayId)
		if err != nil {
			return nil, fmt.Errorf("sign jsapi payment: %w", err)
		}
		appData, _ := json.Marshal(jsapiParams)
		result.AppPayData = string(appData)

	default:
		return nil, fmt.Errorf("unsupported payment scene: %s", scene)
	}

	return result, nil
}

// QueryPayment queries the payment status.
func (p *WechatProvider) QueryPayment(ctx context.Context, orderID, tradeNo string) (*NotifyResult, error) {
	var resp *wechat.QueryOrderRsp
	var err error

	if tradeNo != "" {
		resp, err = p.client.V3TransactionQueryOrder(ctx, wechat.TransactionId, tradeNo)
	} else {
		resp, err = p.client.V3TransactionQueryOrder(ctx, wechat.OutTradeNo, orderID)
	}

	if err != nil {
		return nil, fmt.Errorf("query payment: %w", err)
	}

	if resp.Code != wechat.Success {
		return nil, fmt.Errorf("wechat query error: %d - %s", resp.Code, resp.Error)
	}

	// Parse pay time
	var payTime int64
	if resp.Response.SuccessTime != "" {
		t, err := time.Parse(time.RFC3339, resp.Response.SuccessTime)
		if err == nil {
			payTime = t.Unix()
		}
	}

	var amount int64
	if resp.Response.Amount != nil {
		amount = int64(resp.Response.Amount.Total)
	}

	var payerID string
	if resp.Response.Payer != nil {
		payerID = resp.Response.Payer.Openid
	}

	return &NotifyResult{
		TradeNo:    resp.Response.TransactionId,
		OutTradeNo: resp.Response.OutTradeNo,
		Amount:     amount,
		Status:     mapWechatTradeStatus(resp.Response.TradeState),
		PayerID:    payerID,
		PayTime:    payTime,
	}, nil
}

// ClosePayment closes an unpaid order.
func (p *WechatProvider) ClosePayment(ctx context.Context, orderID, tradeNo string) error {
	resp, err := p.client.V3TransactionCloseOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("close payment: %w", err)
	}

	if resp.Code != wechat.Success {
		return fmt.Errorf("wechat close error: %d - %s", resp.Code, resp.Error)
	}

	return nil
}

// RefundPayment creates a refund.
func (p *WechatProvider) RefundPayment(ctx context.Context, orderID, tradeNo, refundID string, refundAmount, totalAmount int64, reason string) (*RefundResult, error) {
	bm := make(gopay.BodyMap)
	if tradeNo != "" {
		bm.Set("transaction_id", tradeNo)
	} else {
		bm.Set("out_trade_no", orderID)
	}
	bm.Set("out_refund_no", refundID)
	if reason != "" {
		bm.Set("reason", reason)
	}
	bm.SetBodyMap("amount", func(am gopay.BodyMap) {
		am.Set("refund", refundAmount)
		am.Set("total", totalAmount)
		am.Set("currency", "CNY")
	})

	resp, err := p.client.V3Refund(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("refund payment: %w", err)
	}

	if resp.Code != wechat.Success {
		return nil, fmt.Errorf("wechat refund error: %d - %s", resp.Code, resp.Error)
	}

	var refundTime int64
	if resp.Response.SuccessTime != "" {
		t, err := time.Parse(time.RFC3339, resp.Response.SuccessTime)
		if err == nil {
			refundTime = t.Unix()
		}
	}

	return &RefundResult{
		RefundNo:    resp.Response.RefundId,
		OutRefundNo: resp.Response.OutRefundNo,
		Amount:      int64(resp.Response.Amount.Refund),
		Status:      mapWechatRefundStatus(resp.Response.Status),
		RefundTime:  refundTime,
	}, nil
}

// QueryRefund queries the refund status.
func (p *WechatProvider) QueryRefund(ctx context.Context, orderID, refundID string) (*RefundResult, error) {
	resp, err := p.client.V3RefundQuery(ctx, refundID, nil)
	if err != nil {
		return nil, fmt.Errorf("query refund: %w", err)
	}

	if resp.Code != wechat.Success {
		return nil, fmt.Errorf("wechat refund query error: %d - %s", resp.Code, resp.Error)
	}

	var refundTime int64
	if resp.Response.SuccessTime != "" {
		t, err := time.Parse(time.RFC3339, resp.Response.SuccessTime)
		if err == nil {
			refundTime = t.Unix()
		}
	}

	return &RefundResult{
		RefundNo:    resp.Response.RefundId,
		OutRefundNo: resp.Response.OutRefundNo,
		Amount:      int64(resp.Response.Amount.Refund),
		Status:      mapWechatRefundStatus(resp.Response.Status),
		RefundTime:  refundTime,
	}, nil
}

// ParseNotify parses and verifies the async notification.
func (p *WechatProvider) ParseNotify(ctx context.Context, body []byte, headers map[string]string) (*NotifyResult, error) {
	// Convert body to *http.Request for gopay SDK
	req, err := http.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set WeChat signature headers
	req.Header.Set("Wechatpay-Timestamp", headers["Wechatpay-Timestamp"])
	req.Header.Set("Wechatpay-Nonce", headers["Wechatpay-Nonce"])
	req.Header.Set("Wechatpay-Signature", headers["Wechatpay-Signature"])
	req.Header.Set("Wechatpay-Serial", headers["Wechatpay-Serial"])

	// Parse notification
	notifyReq, err := wechat.V3ParseNotify(req)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}

	// Parse WeChat public key for signature verification
	wxPublicKey, err := parseRSAPublicKey(p.config.WechatPublicKey)
	if err != nil {
		return nil, fmt.Errorf("parse wechat public key: %w", err)
	}

	// Verify signature
	err = notifyReq.VerifySignByPK(wxPublicKey)
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}

	// Decrypt the payment result
	resource, err := notifyReq.DecryptPayCipherText(p.config.APIKeyV3)
	if err != nil {
		return nil, fmt.Errorf("decrypt resource: %w", err)
	}

	// Parse pay time
	var payTime int64
	if resource.SuccessTime != "" {
		t, err := time.Parse(time.RFC3339, resource.SuccessTime)
		if err == nil {
			payTime = t.Unix()
		}
	}

	var amount int64
	if resource.Amount != nil {
		amount = int64(resource.Amount.Total)
	}

	var payerID string
	if resource.Payer != nil {
		payerID = resource.Payer.Openid
	}

	rawData, _ := json.Marshal(resource)

	// WeChat expects JSON response
	successResp, _ := json.Marshal(map[string]string{
		"code":    "SUCCESS",
		"message": "OK",
	})

	return &NotifyResult{
		TradeNo:     resource.TransactionId,
		OutTradeNo:  resource.OutTradeNo,
		Amount:      amount,
		Status:      mapWechatTradeStatus(resource.TradeState),
		PayerID:     payerID,
		PayTime:     payTime,
		RawData:     string(rawData),
		SuccessResp: string(successResp),
	}, nil
}

// parseRSAPublicKey parses PEM encoded RSA public key.
func parseRSAPublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Try parsing as PKIX public key first
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try parsing as certificate
		cert, certErr := x509.ParseCertificate(block.Bytes)
		if certErr != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate does not contain RSA public key")
		}
		return rsaKey, nil
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaKey, nil
}

// mapWechatTradeStatus maps WeChat trade status to our standard status.
func mapWechatTradeStatus(status string) string {
	switch status {
	case "NOTPAY":
		return "pending"
	case "CLOSED":
		return "closed"
	case "SUCCESS":
		return "success"
	case "REFUND":
		return "refunded"
	case "PAYERROR":
		return "failed"
	case "USERPAYING":
		return "processing"
	default:
		return status
	}
}

// mapWechatRefundStatus maps WeChat refund status.
func mapWechatRefundStatus(status string) string {
	switch status {
	case "SUCCESS":
		return "success"
	case "CLOSED":
		return "closed"
	case "PROCESSING":
		return "processing"
	case "ABNORMAL":
		return "failed"
	default:
		return status
	}
}

// Ensure strconv is used
var _ = strconv.Itoa
