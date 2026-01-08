package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// invoiceAdapter implements outbound.InvoiceDatabasePort.
type invoiceAdapter struct {
	db *gorm.DB
}

// NewInvoiceAdapter creates a new invoice database adapter.
func NewInvoiceAdapter(db *gorm.DB) outbound.InvoiceDatabasePort {
	return &invoiceAdapter{db: db}
}

func (a *invoiceAdapter) Create(ctx context.Context, invoice *model.Invoice) error {
	return a.db.WithContext(ctx).Create(invoice).Error
}

func (a *invoiceAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	var invoice model.Invoice
	err := a.db.WithContext(ctx).First(&invoice, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invoice, nil
}

func (a *invoiceAdapter) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Invoice, error) {
	var invoice model.Invoice
	err := a.db.WithContext(ctx).First(&invoice, "order_id = ?", orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invoice, nil
}

func (a *invoiceAdapter) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error) {
	var invoices []*model.Invoice
	err := a.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Find(&invoices).Error
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

// Compile-time check
var _ outbound.InvoiceDatabasePort = (*invoiceAdapter)(nil)
