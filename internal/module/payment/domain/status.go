package domain

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusProcessing PaymentStatus = "processing"
	StatusSucceeded  PaymentStatus = "succeeded"
	StatusFailed     PaymentStatus = "failed"
	StatusCanceled   PaymentStatus = "canceled"
	StatusRefunded   PaymentStatus = "refunded"
)

// IsTerminal returns true if the status is a terminal state.
func (s PaymentStatus) IsTerminal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCanceled || s == StatusRefunded
}

// IsSucceeded returns true if the status is succeeded.
func (s PaymentStatus) IsSucceeded() bool {
	return s == StatusSucceeded
}

// CanTransitionTo returns true if the status can transition to the target status.
func (s PaymentStatus) CanTransitionTo(target PaymentStatus) bool {
	switch s {
	case StatusPending:
		return target == StatusProcessing || target == StatusSucceeded || target == StatusFailed || target == StatusCanceled
	case StatusProcessing:
		return target == StatusSucceeded || target == StatusFailed || target == StatusCanceled
	case StatusSucceeded:
		return target == StatusRefunded
	case StatusFailed, StatusCanceled, StatusRefunded:
		return false // Terminal states
	default:
		return false
	}
}

// PaymentMethod represents a payment method type.
type PaymentMethod string

const (
	MethodCard   PaymentMethod = "card"
	MethodAlipay PaymentMethod = "alipay"
	MethodWechat PaymentMethod = "wechat"
)

// IsNative returns true if this is a native China payment method.
func (m PaymentMethod) IsNative() bool {
	return m == MethodAlipay || m == MethodWechat
}

// Provider represents a payment provider.
type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderAlipay Provider = "alipay"
	ProviderWechat Provider = "wechat"
)
