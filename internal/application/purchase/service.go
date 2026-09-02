package purchase

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

// Service provides purchase-module business operations.
type Service interface {
	// Supplier operations
	CreateSupplier(ctx context.Context, s *purchase.Supplier, actor string) (*purchase.Supplier, error)
	GetSupplier(ctx context.Context, id string) (*purchase.Supplier, error)
	UpdateSupplier(ctx context.Context, id string, patch *purchase.Supplier, actor string) (*purchase.Supplier, error)
	DeleteSupplier(ctx context.Context, id string) error
	ListSuppliers(ctx context.Context, name string, status purchase.SupplierStatus, limit, offset int) ([]*purchase.Supplier, error)

	// Purchase Order operations
	CreateOrder(ctx context.Context, o *purchase.PurchaseOrder, actor string) (*purchase.PurchaseOrder, error)
	GetOrder(ctx context.Context, id string) (*purchase.PurchaseOrder, error)
	UpdateOrder(ctx context.Context, id string, patch *purchase.PurchaseOrder, actor string) (*purchase.PurchaseOrder, error)
	DeleteOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, supplierCode string, status purchase.OrderStatus, limit, offset int) ([]*purchase.PurchaseOrder, error)
	ConfirmOrder(ctx context.Context, id string, actor string) (*purchase.PurchaseOrder, error)
	CancelOrder(ctx context.Context, id string, reason string, actor string) (*purchase.PurchaseOrder, error)

	// Goods Receipt operations
	CreateReceipt(ctx context.Context, g *purchase.GoodsReceipt, actor string) (*purchase.GoodsReceipt, error)
	GetReceipt(ctx context.Context, id string) (*purchase.GoodsReceipt, error)
	ApproveReceipt(ctx context.Context, id string, actor string) (*purchase.GoodsReceipt, error)
	ListReceipts(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.GoodsReceipt, error)

	// Purchase Invoice operations
	CreateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice, actor string) (*purchase.PurchaseInvoice, error)
	GetInvoice(ctx context.Context, id string) (*purchase.PurchaseInvoice, error)
	UpdateInvoice(ctx context.Context, id string, patch *purchase.PurchaseInvoice, actor string) (*purchase.PurchaseInvoice, error)
	DeleteInvoice(ctx context.Context, id string) error
	ListInvoices(ctx context.Context, supplierCode string, status purchase.InvoiceStatus, limit, offset int) ([]*purchase.PurchaseInvoice, error)
	PostInvoice(ctx context.Context, id string, actor string) (*purchase.PurchaseInvoice, error)

	// Payment operations
	CreatePayment(ctx context.Context, p *purchase.Payment, actor string) (*purchase.Payment, error)
	GetPayment(ctx context.Context, id string) (*purchase.Payment, error)
	ApprovePayment(ctx context.Context, id string, actor string) (*purchase.Payment, error)
	ListPayments(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.Payment, error)

	// Supplier balance
	GetSupplierBalance(ctx context.Context, supplierCode string) (core.Money, error)
}

type service struct {
	repo purchase.Repository
}

func NewService(repo purchase.Repository) Service {
	return &service{repo: repo}
}

// --- Supplier operations ---

func (s *service) CreateSupplier(ctx context.Context, sup *purchase.Supplier, actor string) (*purchase.Supplier, error) {
	sp := sup.Clone()
	sp.Status = purchase.SupplierActive
	sp.CreatedBy = actor
	sp.UpdatedBy = actor

	if err := purchase.ValidateSupplier(sp); err != nil {
		return nil, err
	}

	n, err := s.repo.NextSupplierNo(ctx)
	if err != nil {
		return nil, err
	}
	sp.RefNo = fmt.Sprintf("NCC-%05d", n)
	sp.ID = core.RowID("supplier", sp.RefNo)

	now := core.NowRFC3339()
	sp.CreatedAt = now
	sp.UpdatedAt = now

	if err := s.repo.CreateSupplier(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

func (s *service) GetSupplier(ctx context.Context, id string) (*purchase.Supplier, error) {
	return s.repo.FindSupplierByID(ctx, id)
}

func (s *service) UpdateSupplier(ctx context.Context, id string, patch *purchase.Supplier, actor string) (*purchase.Supplier, error) {
	existing, err := s.repo.FindSupplierByID(ctx, id)
	if err != nil {
		return nil, err
	}
	existing.Name = patch.Name
	existing.TaxCode = patch.TaxCode
	existing.Phone = patch.Phone
	existing.Email = patch.Email
	existing.Address = patch.Address
	existing.BankAccount = patch.BankAccount
	existing.BankName = patch.BankName
	existing.CreditLimit = patch.CreditLimit
	existing.ContactPerson = patch.ContactPerson
	existing.PaymentTerms = patch.PaymentTerms
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()

	if err := purchase.ValidateSupplier(existing); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSupplier(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) DeleteSupplier(ctx context.Context, id string) error {
	return s.repo.DeleteSupplier(ctx, id)
}

func (s *service) ListSuppliers(ctx context.Context, name string, status purchase.SupplierStatus, limit, offset int) ([]*purchase.Supplier, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListSuppliers(ctx, name, status, limit, offset)
}

// --- Purchase Order operations ---

func (s *service) CreateOrder(ctx context.Context, o *purchase.PurchaseOrder, actor string) (*purchase.PurchaseOrder, error) {
	od := o.Clone()
	od.Status = purchase.OrderDraft
	od.CreatedBy = actor
	od.UpdatedBy = actor

	if err := purchase.ValidatePurchaseOrder(od); err != nil {
		return nil, err
	}

	n, err := s.repo.NextOrderNo(ctx)
	if err != nil {
		return nil, err
	}
	od.RefNo = fmt.Sprintf("PO-%05d", n)
	od.ID = core.RowID("purchase_order", od.RefNo)

	now := core.NowRFC3339()
	od.CreatedAt = now
	od.UpdatedAt = now

	if err := s.repo.CreateOrder(ctx, od); err != nil {
		return nil, err
	}
	return od, nil
}

func (s *service) GetOrder(ctx context.Context, id string) (*purchase.PurchaseOrder, error) {
	return s.repo.FindOrderByID(ctx, id)
}

func (s *service) UpdateOrder(ctx context.Context, id string, patch *purchase.PurchaseOrder, actor string) (*purchase.PurchaseOrder, error) {
	existing, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.OrderDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.SupplierCode = patch.SupplierCode
	existing.SupplierName = patch.SupplierName
	existing.OrderDate = patch.OrderDate
	existing.ExpectedDeliveryDate = patch.ExpectedDeliveryDate
	existing.PaymentTerms = patch.PaymentTerms
	existing.DeliveryAddress = patch.DeliveryAddress
	existing.Notes = patch.Notes
	existing.Lines = patch.Lines
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()

	if err := purchase.ValidatePurchaseOrder(existing); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateOrder(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) DeleteOrder(ctx context.Context, id string) error {
	existing, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status != purchase.OrderDraft {
		return purchase.ErrInvalidStatus
	}
	return s.repo.DeleteOrder(ctx, id)
}

func (s *service) ListOrders(ctx context.Context, supplierCode string, status purchase.OrderStatus, limit, offset int) ([]*purchase.PurchaseOrder, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListOrders(ctx, supplierCode, status, limit, offset)
}

func (s *service) ConfirmOrder(ctx context.Context, id string, actor string) (*purchase.PurchaseOrder, error) {
	existing, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.OrderDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.Status = purchase.OrderConfirmed
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()
	if err := s.repo.UpdateOrder(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) CancelOrder(ctx context.Context, id string, reason string, actor string) (*purchase.PurchaseOrder, error) {
	existing, err := s.repo.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status == purchase.OrderReceived || existing.Status == purchase.OrderCompleted {
		return nil, purchase.ErrInvalidStatus
	}
	hasReceipts, err := s.repo.HasReceiptsForOrder(ctx, existing.ID)
	if err != nil {
		return nil, err
	}
	if hasReceipts {
		return nil, purchase.ErrInvalidStatus
	}
	existing.Status = purchase.OrderCancelled
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()
	if err := s.repo.UpdateOrder(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// --- Goods Receipt operations ---

func (s *service) CreateReceipt(ctx context.Context, g *purchase.GoodsReceipt, actor string) (*purchase.GoodsReceipt, error) {
	gr := g.Clone()
	gr.Status = purchase.ReceiptDraft
	gr.CreatedBy = actor
	gr.UpdatedBy = actor

	if err := purchase.ValidateGoodsReceipt(gr); err != nil {
		return nil, err
	}

	n, err := s.repo.NextReceiptNo(ctx)
	if err != nil {
		return nil, err
	}
	gr.RefNo = fmt.Sprintf("NK-%05d", n)
	gr.ID = core.RowID("goods_receipt", gr.RefNo)

	now := core.NowRFC3339()
	gr.CreatedAt = now
	gr.UpdatedAt = now

	if err := s.repo.CreateReceipt(ctx, gr); err != nil {
		return nil, err
	}
	return gr, nil
}

func (s *service) GetReceipt(ctx context.Context, id string) (*purchase.GoodsReceipt, error) {
	return s.repo.FindReceiptByID(ctx, id)
}

func (s *service) ApproveReceipt(ctx context.Context, id string, actor string) (*purchase.GoodsReceipt, error) {
	existing, err := s.repo.FindReceiptByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.ReceiptDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.Status = purchase.ReceiptApproved
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()
	if err := s.repo.UpdateReceipt(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) ListReceipts(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.GoodsReceipt, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListReceipts(ctx, supplierCode, limit, offset)
}

// --- Purchase Invoice operations ---

func (s *service) CreateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice, actor string) (*purchase.PurchaseInvoice, error) {
	iv := inv.Clone()
	iv.Status = purchase.InvoiceDraft
	iv.EInvoiceStatus = purchase.EInvoiceNone
	iv.CreatedBy = actor
	iv.UpdatedBy = actor

	if err := purchase.ValidatePurchaseInvoice(iv); err != nil {
		return nil, err
	}

	n, err := s.repo.NextInvoiceNo(ctx)
	if err != nil {
		return nil, err
	}
	iv.RefNo = fmt.Sprintf("MH-%05d", n)
	iv.ID = core.RowID("purchase_invoice", iv.RefNo)

	now := core.NowRFC3339()
	iv.CreatedAt = now
	iv.UpdatedAt = now

	if err := s.repo.CreateInvoice(ctx, iv); err != nil {
		return nil, err
	}
	return iv, nil
}

func (s *service) GetInvoice(ctx context.Context, id string) (*purchase.PurchaseInvoice, error) {
	return s.repo.FindInvoiceByID(ctx, id)
}

func (s *service) UpdateInvoice(ctx context.Context, id string, patch *purchase.PurchaseInvoice, actor string) (*purchase.PurchaseInvoice, error) {
	existing, err := s.repo.FindInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.InvoiceDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.SupplierCode = patch.SupplierCode
	existing.SupplierName = patch.SupplierName
	existing.SupplierTaxCode = patch.SupplierTaxCode
	existing.InvoiceDate = patch.InvoiceDate
	existing.SupplierInvoiceNo = patch.SupplierInvoiceNo
	existing.GoodsReceiptID = patch.GoodsReceiptID
	existing.GoodsReceiptRefNo = patch.GoodsReceiptRefNo
	existing.POID = patch.POID
	existing.PORefNo = patch.PORefNo
	existing.Notes = patch.Notes
	existing.Lines = patch.Lines
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()

	if err := purchase.ValidatePurchaseInvoice(existing); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateInvoice(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) DeleteInvoice(ctx context.Context, id string) error {
	existing, err := s.repo.FindInvoiceByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status != purchase.InvoiceDraft {
		return purchase.ErrInvalidStatus
	}
	return s.repo.DeleteInvoice(ctx, id)
}

func (s *service) ListInvoices(ctx context.Context, supplierCode string, status purchase.InvoiceStatus, limit, offset int) ([]*purchase.PurchaseInvoice, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListInvoices(ctx, supplierCode, status, limit, offset)
}

func (s *service) PostInvoice(ctx context.Context, id string, actor string) (*purchase.PurchaseInvoice, error) {
	existing, err := s.repo.FindInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.InvoiceDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.Status = purchase.InvoicePendingEInv
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()
	if err := s.repo.UpdateInvoice(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// --- Payment operations ---

func (s *service) CreatePayment(ctx context.Context, p *purchase.Payment, actor string) (*purchase.Payment, error) {
	pm := p.Clone()
	pm.Status = purchase.PaymentDraft
	pm.CreatedBy = actor
	pm.UpdatedBy = actor

	if err := purchase.ValidatePayment(pm); err != nil {
		return nil, err
	}

	n, err := s.repo.NextPaymentNo(ctx)
	if err != nil {
		return nil, err
	}
	pm.RefNo = fmt.Sprintf("TT-%05d", n)
	pm.ID = core.RowID("purchase_payment", pm.RefNo)

	now := core.NowRFC3339()
	pm.CreatedAt = now
	pm.UpdatedAt = now

	if err := s.repo.CreatePayment(ctx, pm); err != nil {
		return nil, err
	}
	return pm, nil
}

func (s *service) GetPayment(ctx context.Context, id string) (*purchase.Payment, error) {
	return s.repo.FindPaymentByID(ctx, id)
}

func (s *service) ApprovePayment(ctx context.Context, id string, actor string) (*purchase.Payment, error) {
	existing, err := s.repo.FindPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status != purchase.PaymentDraft {
		return nil, purchase.ErrInvalidStatus
	}
	existing.Status = purchase.PaymentApproved
	existing.UpdatedBy = actor
	existing.UpdatedAt = core.NowRFC3339()
	if err := s.repo.UpdatePayment(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *service) ListPayments(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.Payment, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListPayments(ctx, supplierCode, limit, offset)
}

// --- Supplier balance ---

func (s *service) GetSupplierBalance(ctx context.Context, supplierCode string) (core.Money, error) {
	return s.repo.GetSupplierBalance(ctx, supplierCode)
}
