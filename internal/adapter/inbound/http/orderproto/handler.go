package orderproto

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	orderv1 "github.com/uniedit/server/api/pb/order"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	orderDomain order.OrderDomain
}

func NewHandler(orderDomain order.OrderDomain) *Handler {
	return &Handler{orderDomain: orderDomain}
}

func (h *Handler) CreateSubscriptionOrder(c *gin.Context, in *orderv1.CreateSubscriptionOrderRequest) (*orderv1.Order, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	ord, err := h.orderDomain.CreateSubscriptionOrder(c.Request.Context(), userID, in.GetPlanId())
	if err != nil {
		return nil, mapOrderError(err)
	}

	c.Status(http.StatusCreated)
	return toOrder(ord), nil
}

func (h *Handler) CreateTopupOrder(c *gin.Context, in *orderv1.CreateTopupOrderRequest) (*orderv1.Order, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	ord, err := h.orderDomain.CreateTopupOrder(c.Request.Context(), userID, in.GetAmount())
	if err != nil {
		return nil, mapOrderError(err)
	}

	c.Status(http.StatusCreated)
	return toOrder(ord), nil
}

func (h *Handler) ListOrders(c *gin.Context, in *orderv1.ListOrdersRequest) (*orderv1.OrderListResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	page := int(in.GetPage())
	pageSize := int(in.GetPageSize())
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var filter *model.OrderFilter
	if statusStr := in.GetStatus(); statusStr != "" {
		status := model.OrderStatus(statusStr)
		if status.IsValid() {
			filter = &model.OrderFilter{Status: &status}
		}
	}
	if typeStr := in.GetType(); typeStr != "" {
		t := model.OrderType(typeStr)
		if t.IsValid() {
			if filter == nil {
				filter = &model.OrderFilter{}
			}
			filter.Type = &t
		}
	}

	orders, total, err := h.orderDomain.ListOrders(c.Request.Context(), userID, filter, page, pageSize)
	if err != nil {
		return nil, mapOrderError(err)
	}

	out := make([]*orderv1.Order, 0, len(orders))
	for _, o := range orders {
		out = append(out, toOrder(o))
	}

	totalPages := int32(total) / int32(pageSize)
	if int32(total)%int32(pageSize) != 0 {
		totalPages++
	}

	return &orderv1.OrderListResponse{
		Orders:     out,
		Total:      total,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalPages: totalPages,
	}, nil
}

func (h *Handler) GetOrder(c *gin.Context, in *orderv1.GetByIDRequest) (*orderv1.Order, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	orderID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid order ID", Err: err}
	}

	ord, err := h.orderDomain.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	if ord.UserID != userID {
		return nil, &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}
	return toOrder(ord), nil
}

func (h *Handler) CancelOrder(c *gin.Context, in *orderv1.CancelOrderRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	orderID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid order ID", Err: err}
	}

	ord, err := h.orderDomain.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	if ord.UserID != userID {
		return nil, &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}

	if err := h.orderDomain.CancelOrder(c.Request.Context(), orderID, in.GetReason()); err != nil {
		return nil, mapOrderError(err)
	}
	return &commonv1.MessageResponse{Message: "order canceled"}, nil
}

func (h *Handler) ListInvoices(c *gin.Context, _ *commonv1.Empty) (*orderv1.InvoiceListResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	invoices, err := h.orderDomain.ListInvoices(c.Request.Context(), userID)
	if err != nil {
		return nil, mapOrderError(err)
	}

	out := make([]*orderv1.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, toInvoice(inv))
	}

	return &orderv1.InvoiceListResponse{Invoices: out}, nil
}

func (h *Handler) GetInvoice(c *gin.Context, in *orderv1.GetByIDRequest) (*orderv1.Invoice, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	invoiceID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid invoice ID", Err: err}
	}

	inv, err := h.orderDomain.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	if inv.UserID != userID {
		return nil, &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}

	return toInvoice(inv), nil
}

func (h *Handler) DownloadInvoice(c *gin.Context, in *orderv1.GetByIDRequest) (*orderv1.DownloadInvoiceResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	invoiceID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid invoice ID", Err: err}
	}

	inv, err := h.orderDomain.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	if inv.UserID != userID {
		return nil, &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "access denied"}
	}
	if inv.PDFURL == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: "invoice PDF not available"}
	}

	c.Redirect(http.StatusFound, inv.PDFURL)
	return nil, protohttp.ErrHandled
}

func mapOrderError(err error) error {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "order_not_found", Message: "Order not found", Err: err}
	case errors.Is(err, order.ErrOrderNotCancelable):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "order_not_cancelable", Message: "Order cannot be canceled", Err: err}
	case errors.Is(err, order.ErrOrderNotRefundable):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "order_not_refundable", Message: "Order cannot be refunded", Err: err}
	case errors.Is(err, order.ErrMinimumTopupAmount):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "minimum_topup_amount", Message: "Minimum top-up is $1.00", Err: err}
	case errors.Is(err, order.ErrFreePlanNotOrderable):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "free_plan_not_orderable", Message: "Cannot create order for free plan", Err: err}
	case errors.Is(err, order.ErrInvoiceNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "invoice_not_found", Message: "Invoice not found", Err: err}
	default:
		return err
	}
}

func toOrder(o *model.Order) *orderv1.Order {
	if o == nil {
		return nil
	}

	paidAt := formatTimePtr(o.PaidAt)
	canceledAt := formatTimePtr(o.CanceledAt)
	refundedAt := formatTimePtr(o.RefundedAt)
	expiresAt := formatTimePtr(o.ExpiresAt)

	items := make([]*orderv1.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, &orderv1.OrderItem{
			Id:          it.ID.String(),
			Description: it.Description,
			Quantity:    int32(it.Quantity),
			UnitPrice:   it.UnitPrice,
			Amount:      it.Amount,
		})
	}

	planID := ""
	if o.PlanID != nil {
		planID = *o.PlanID
	}

	return &orderv1.Order{
		Id:            o.ID.String(),
		OrderNo:       o.OrderNo,
		Type:          string(o.Type),
		Status:        string(o.Status),
		Subtotal:      o.Subtotal,
		Discount:      o.Discount,
		Tax:           o.Tax,
		Total:         o.Total,
		Currency:      o.Currency,
		PlanId:        planID,
		CreditsAmount: o.CreditsAmount,
		PaidAt:        paidAt,
		CanceledAt:    canceledAt,
		RefundedAt:    refundedAt,
		ExpiresAt:     expiresAt,
		CreatedAt:     o.CreatedAt.UTC().Format(time.RFC3339Nano),
		Items:         items,
	}
}

func toInvoice(i *model.Invoice) *orderv1.Invoice {
	if i == nil {
		return nil
	}
	paidAt := formatTimePtr(i.PaidAt)
	return &orderv1.Invoice{
		Id:        i.ID.String(),
		InvoiceNo: i.InvoiceNo,
		OrderId:   i.OrderID.String(),
		Amount:    i.Amount,
		Currency:  i.Currency,
		Status:    i.Status,
		PdfUrl:    i.PDFURL,
		IssuedAt:  i.IssuedAt.UTC().Format(time.RFC3339Nano),
		DueAt:     i.DueAt.UTC().Format(time.RFC3339Nano),
		PaidAt:    paidAt,
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
