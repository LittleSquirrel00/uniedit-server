package inbound

import "github.com/gin-gonic/gin"

// OrderHttpPort defines HTTP handler interface for order operations.
type OrderHttpPort interface {
	// CreateSubscriptionOrder handles POST /orders/subscription
	CreateSubscriptionOrder(c *gin.Context)

	// CreateTopupOrder handles POST /orders/topup
	CreateTopupOrder(c *gin.Context)

	// GetOrder handles GET /orders/:id
	GetOrder(c *gin.Context)

	// ListOrders handles GET /orders
	ListOrders(c *gin.Context)

	// CancelOrder handles POST /orders/:id/cancel
	CancelOrder(c *gin.Context)
}

// InvoiceHttpPort defines HTTP handler interface for invoice operations.
type InvoiceHttpPort interface {
	// GetInvoice handles GET /invoices/:id
	GetInvoice(c *gin.Context)

	// ListInvoices handles GET /invoices
	ListInvoices(c *gin.Context)

	// DownloadInvoice handles GET /invoices/:id/download
	DownloadInvoice(c *gin.Context)
}
