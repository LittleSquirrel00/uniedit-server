package payment

import "github.com/uniedit/server/internal/module/payment/domain"

// Type aliases for backward compatibility with existing code.
type PaymentStatus = domain.PaymentStatus
type PaymentMethod = domain.PaymentMethod

// Status constants for convenience.
const (
	PaymentStatusPending    = domain.StatusPending
	PaymentStatusProcessing = domain.StatusProcessing
	PaymentStatusSucceeded  = domain.StatusSucceeded
	PaymentStatusFailed     = domain.StatusFailed
	PaymentStatusCanceled   = domain.StatusCanceled
	PaymentStatusRefunded   = domain.StatusRefunded
)

// Method constants for convenience.
const (
	PaymentMethodCard   = domain.MethodCard
	PaymentMethodAlipay = domain.MethodAlipay
	PaymentMethodWechat = domain.MethodWechat
)
