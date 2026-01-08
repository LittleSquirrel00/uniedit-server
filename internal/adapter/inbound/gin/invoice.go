package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/port/inbound"
)

// invoiceHandler implements inbound.InvoiceHttpPort.
type invoiceHandler struct {
	orderDomain order.OrderDomain
}

// NewInvoiceHandler creates a new invoice HTTP handler.
func NewInvoiceHandler(orderDomain order.OrderDomain) inbound.InvoiceHttpPort {
	return &invoiceHandler{orderDomain: orderDomain}
}

func (h *invoiceHandler) GetInvoice(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		if err == order.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
		return
	}

	// Check ownership
	if invoice.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, invoice.ToResponse())
}

func (h *invoiceHandler) ListInvoices(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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

func (h *invoiceHandler) DownloadInvoice(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
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
		if err == order.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
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
var _ inbound.InvoiceHttpPort = (*invoiceHandler)(nil)
