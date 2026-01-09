package orderhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/port/inbound"
)

// InvoiceHandler handles invoice HTTP requests.
type InvoiceHandler struct {
	orderDomain order.OrderDomain
}

// NewInvoiceHandler creates a new invoice handler.
func NewInvoiceHandler(orderDomain order.OrderDomain) *InvoiceHandler {
	return &InvoiceHandler{orderDomain: orderDomain}
}

// RegisterRoutes registers invoice routes.
func (h *InvoiceHandler) RegisterRoutes(r *gin.RouterGroup) {
	invoices := r.Group("/invoices")
	{
		invoices.GET("", h.ListInvoices)
		invoices.GET("/:id", h.GetInvoice)
		invoices.GET("/:id/download", h.DownloadInvoice)
	}
}

// GetInvoice handles GET /invoices/:id.
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	invoiceIDStr := c.Param("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	invoice, err := h.orderDomain.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check ownership
	if invoice.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, invoice.ToResponse())
}

// ListInvoices handles GET /invoices.
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	invoices, err := h.orderDomain.ListInvoices(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}

	responses := make([]*gin.H, len(invoices))
	for i, inv := range invoices {
		resp := inv.ToResponse()
		responses[i] = &gin.H{
			"id":         resp.ID,
			"invoice_no": resp.InvoiceNo,
			"order_id":   resp.OrderID,
			"amount":     resp.Amount,
			"currency":   resp.Currency,
			"status":     resp.Status,
			"pdf_url":    resp.PDFURL,
			"issued_at":  resp.IssuedAt,
			"due_at":     resp.DueAt,
			"paid_at":    resp.PaidAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"invoices": responses})
}

// DownloadInvoice handles GET /invoices/:id/download.
func (h *InvoiceHandler) DownloadInvoice(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	invoiceIDStr := c.Param("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	invoice, err := h.orderDomain.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check ownership
	if invoice.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if invoice.PDFURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice PDF not available"})
		return
	}

	c.Redirect(http.StatusFound, invoice.PDFURL)
}

// Compile-time check
var _ inbound.InvoiceHttpPort = (*InvoiceHandler)(nil)
