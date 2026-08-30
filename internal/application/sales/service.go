package sales

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/sales"
)

type Service interface {
	// Invoice operations
	CreateInvoice(ctx context.Context, inv *sales.SalesInvoice, actor string) (*sales.SalesInvoice, error)
	GetInvoice(ctx context.Context, id string) (*sales.SalesInvoice, error)
	UpdateInvoice(ctx context.Context, id string, patch *sales.SalesInvoice, actor string) (*sales.SalesInvoice, error)
	DeleteInvoice(ctx context.Context, id string) error
	ListInvoices(ctx context.Context, customerCode string, status sales.InvoiceStatus, limit, offset int) ([]*sales.SalesInvoice, error)

	// Order operations
	CreateOrder(ctx context.Context, o *sales.SalesOrder, actor string) (*sales.SalesOrder, error)
	GetOrder(ctx context.Context, id string) (*sales.SalesOrder, error)
	UpdateOrder(ctx context.Context, id string, patch *sales.SalesOrder, actor string) (*sales.SalesOrder, error)
	DeleteOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, customerCode string, status sales.OrderStatus, limit, offset int) ([]*sales.SalesOrder, error)
	ConfirmOrder(ctx context.Context, id string, actor string) (*sales.SalesOrder, error)
	CancelOrder(ctx context.Context, id string, reason string, actor string) (*sales.SalesOrder, error)

	// Return operations
	CreateReturn(ctx context.Context, r *sales.SalesReturn, actor string) (*sales.SalesReturn, error)
	GetReturn(ctx context.Context, id string) (*sales.SalesReturn, error)
	ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*sales.SalesReturn, error)
	ApproveReturn(ctx context.Context, id string, actor string) (*sales.SalesReturn, error)
	ReceiveReturn(ctx context.Context, id string, actor string) (*sales.SalesReturn, error)

	// Customer operations
	GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error)
}

type service struct {
	repo sales.Repository
}

func NewService(repo sales.Repository) Service {
	return &service{repo: repo}
}

// --- Invoice operations ---

func (s *service) CreateInvoice(ctx context.Context, inv *sales.SalesInvoice, actor string) (*sales.SalesInvoice, error) {
	sv := inv.Clone()
	sv.Status = sales.InvoiceDraft
	sv.EInvoiceStatus = sales.EInvoiceNone
	sv.CreatedBy = actor
	sv.UpdatedBy = actor

	if err := sales.ValidateSalesInvoice(sv); err != nil {
		return nil, err
	}

	n, err := s.repo.NextInvoiceNo(ctx)
	if err != nil {
		return nil, err
	}
	sv.RefNo = fmt.Sprintf("HD-%05d", n)
	sv.ID = core.RowID("sales_invoice", sv.RefNo)

	now := core.NowRFC3339()
	sv.CreatedAt = now
	sv.UpdatedAt = now

	if err := s.repo.CreateInvoice(ctx, sv); err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *service) GetInvoice(ctx context.Context, id string) (*sales.SalesInvoice, error) {
	return s.repo.FindInvoiceByID(ctx, id)
}

func (s *service) UpdateInvoice(ctx context.Context, id string, patch *sales.SalesInvoice, actor string) (*sales.SalesInvoice, error) {
	cur, err := s.repo.FindInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status != sales.InvoiceDraft {
		return nil, sales.ErrConflict
	}

	if patch.CustomerCode != "" {
		cur.CustomerCode = patch.CustomerCode
	}
	if patch.CustomerName != "" {
		cur.CustomerName = patch.CustomerName
	}
	if patch.CustomerAddress != "" {
		cur.CustomerAddress = patch.CustomerAddress
	}
	if patch.CustomerTaxCode != "" {
		cur.CustomerTaxCode = patch.CustomerTaxCode
	}
	if patch.InvoiceDate != "" {
		cur.InvoiceDate = patch.InvoiceDate
	}
	if patch.DueDate != "" {
		cur.DueDate = patch.DueDate
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	if len(patch.Lines) > 0 {
		cur.Lines = patch.Lines
	}

	if err := sales.ValidateSalesInvoice(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateInvoice(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) DeleteInvoice(ctx context.Context, id string) error {
	inv, err := s.repo.FindInvoiceByID(ctx, id)
	if err != nil {
		return err
	}
	if inv.Status != sales.InvoiceDraft {
		return sales.ErrConflict
	}
	return s.repo.DeleteInvoice(ctx, id)
}

func (s *service) ListInvoices(ctx context.Context, customerCode string, status sales.InvoiceStatus, limit, offset int) ([]*sales.SalesInvoice, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListInvoices(ctx, customerCode, status, limit, offset)
}

// --- Order operations ---

func (s *service) CreateOrder(ctx context.Context, o *sales.SalesOrder, actor string) (*sales.SalesOrder, error) {
	od := o.Clone()
	od.Status = sales.OrderDraft
	od.DeliveryStatus = sales.DeliveryPending
	od.CreatedBy = actor
	od.UpdatedBy = actor

	if err := sales.ValidateSalesOrder(od); err != nil {
		return nil, err
	}

	n, err := s.repo.NextOrderNo(ctx)
	if err != nil {
		return nil, err
	}
	od.RefNo = fmt.Sprintf("DH-%05d", n)
	od.ID = core.RowID("sales_order", od.RefNo)

	now := core.NowRFC3339()
	od.CreatedAt = now
	od.UpdatedAt = now

	if err := s.repo.CreateOrder(ctx, od); err != nil {
		return nil, err
	}
	return od, nil
}

func (s *service) GetOrder(ctx context.Context, id string) (*sales.SalesOrder, error) {
	return s.repo.FindOrderByID(ctx, id)
}

func (s *service) UpdateOrder(ctx context.Context, id string, patch *sales.SalesOrder, actor string) (*sales.SalesOrder, error) {
	cur, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status != sales.OrderDraft {
		return nil, sales.ErrConflict
	}

	if patch.CustomerCode != "" {
		cur.CustomerCode = patch.CustomerCode
	}
	if patch.CustomerName != "" {
		cur.CustomerName = patch.CustomerName
	}
	if patch.OrderDate != "" {
		cur.OrderDate = patch.OrderDate
	}
	if patch.DeliveryDate != "" {
		cur.DeliveryDate = patch.DeliveryDate
	}
	if patch.DeliveryAddress != "" {
		cur.DeliveryAddress = patch.DeliveryAddress
	}
	if patch.PaymentTerms != "" {
		cur.PaymentTerms = patch.PaymentTerms
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	if len(patch.Lines) > 0 {
		cur.Lines = patch.Lines
	}

	if err := sales.ValidateSalesOrder(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateOrder(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) DeleteOrder(ctx context.Context, id string) error {
	o, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return err
	}
	if o.Status != sales.OrderDraft {
		return sales.ErrConflict
	}
	return s.repo.DeleteOrder(ctx, id)
}

func (s *service) ListOrders(ctx context.Context, customerCode string, status sales.OrderStatus, limit, offset int) ([]*sales.SalesOrder, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListOrders(ctx, customerCode, status, limit, offset)
}

func (s *service) ConfirmOrder(ctx context.Context, id string, actor string) (*sales.SalesOrder, error) {
	o, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status != sales.OrderDraft {
		return nil, sales.ErrConflict
	}

	o.Status = sales.OrderConfirmed
	o.UpdatedBy = actor
	o.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *service) CancelOrder(ctx context.Context, id string, reason string, actor string) (*sales.SalesOrder, error) {
	o, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status == sales.OrderDelivered || o.Status == sales.OrderCompleted || o.Status == sales.OrderCancelled {
		return nil, sales.ErrConflict
	}

	o.Status = sales.OrderCancelled
	o.Notes = reason
	o.UpdatedBy = actor
	o.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// --- Return operations ---

func (s *service) CreateReturn(ctx context.Context, r *sales.SalesReturn, actor string) (*sales.SalesReturn, error) {
	// Validate that the source invoice exists
	inv, err := s.repo.FindInvoiceByID(ctx, r.InvoiceID)
	if err != nil {
		return nil, err
	}
	_ = inv // Could validate return qty vs invoice qty here

	ret := r.Clone()
	ret.Status = sales.ReturnDraft
	ret.CreatedBy = actor
	ret.UpdatedBy = actor

	if err := sales.ValidateSalesReturn(ret); err != nil {
		return nil, err
	}

	n, err := s.repo.NextReturnNo(ctx)
	if err != nil {
		return nil, err
	}
	ret.RefNo = fmt.Sprintf("PH-%05d", n)
	ret.ID = core.RowID("sales_return", ret.RefNo)

	now := core.NowRFC3339()
	ret.CreatedAt = now
	ret.UpdatedAt = now

	if err := s.repo.CreateReturn(ctx, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *service) GetReturn(ctx context.Context, id string) (*sales.SalesReturn, error) {
	return s.repo.FindReturnByID(ctx, id)
}

func (s *service) ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*sales.SalesReturn, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListReturns(ctx, customerCode, limit, offset)
}

func (s *service) ApproveReturn(ctx context.Context, id string, actor string) (*sales.SalesReturn, error) {
	r, err := s.repo.FindReturnByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Status != sales.ReturnDraft {
		return nil, sales.ErrConflict
	}

	r.Status = sales.ReturnApproved
	r.UpdatedBy = actor
	r.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateReturn(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *service) ReceiveReturn(ctx context.Context, id string, actor string) (*sales.SalesReturn, error) {
	r, err := s.repo.FindReturnByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Status != sales.ReturnApproved {
		return nil, sales.ErrConflict
	}

	r.Status = sales.ReturnReceived
	r.UpdatedBy = actor
	r.UpdatedAt = core.NowRFC3339()

	if err := s.repo.UpdateReturn(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// --- Customer operations ---

func (s *service) GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error) {
	return s.repo.GetCustomerBalance(ctx, customerCode)
}
