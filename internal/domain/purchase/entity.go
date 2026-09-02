package purchase

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound        = errors.New("purchase: not found")
	ErrDuplicate       = errors.New("purchase: duplicate")
	ErrInvalid         = errors.New("purchase: invalid input")
	ErrConflict        = errors.New("purchase: conflict")
	ErrEmptyLines      = errors.New("purchase: at least one line item required")
	ErrInsufficientQty = errors.New("purchase: insufficient quantity")
	ErrInvalidStatus   = errors.New("purchase: invalid status transition")
)

// SupplierStatus represents the status of a supplier.
type SupplierStatus string

const (
	SupplierActive   SupplierStatus = "active"
	SupplierInactive SupplierStatus = "inactive"
)

func (s SupplierStatus) IsValid() bool {
	switch s {
	case SupplierActive, SupplierInactive:
		return true
	default:
		return false
	}
}

// Supplier represents a vendor/supplier master data.
type Supplier struct {
	ID            string         `json:"id"`
	RefNo         string         `json:"ref_no"` // Auto: NCC-XXXXX
	Name          string         `json:"name"`
	TaxCode       string         `json:"tax_code"`
	Address       string         `json:"address"`
	Phone         string         `json:"phone"`
	Email         string         `json:"email"`
	ContactPerson string         `json:"contact_person"`
	PaymentTerms  string         `json:"payment_terms"`
	CreditLimit   core.Money     `json:"credit_limit"`
	BankAccount   string         `json:"bank_account"`
	BankName      string         `json:"bank_name"`
	Status        SupplierStatus `json:"status"`
	CreatedBy     string         `json:"created_by,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedBy     string         `json:"updated_by,omitempty"`
	UpdatedAt     string         `json:"updated_at"`
}

func (s *Supplier) Clone() *Supplier {
	cp := *s
	return &cp
}

// ValidateSupplier validates supplier data.
func ValidateSupplier(s *Supplier) error {
	if s.Name == "" {
		return &core.ValidationError{Field: "name", Message: "supplier name is required"}
	}
	if s.TaxCode == "" {
		return &core.ValidationError{Field: "tax_code", Message: "tax code is required"}
	}
	if s.Status == "" {
		s.Status = SupplierActive
	}
	if !s.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// OrderStatus represents the lifecycle status of a purchase order.
type OrderStatus string

const (
	OrderDraft     OrderStatus = "draft"
	OrderConfirmed OrderStatus = "confirmed"
	OrderPartial   OrderStatus = "partial_received"
	OrderReceived  OrderStatus = "received"
	OrderCompleted OrderStatus = "completed"
	OrderCancelled OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderDraft, OrderConfirmed, OrderPartial, OrderReceived, OrderCompleted, OrderCancelled:
		return true
	default:
		return false
	}
}

// PurchaseOrder represents a purchase order (Đơn đặt hàng mua).
type PurchaseOrder struct {
	ID                   string      `json:"id"`
	RefNo                string      `json:"ref_no"` // Auto: PO-XXXXX
	SupplierCode         string      `json:"supplier_code"`
	SupplierName         string      `json:"supplier_name"`
	SupplierAddress      string      `json:"supplier_address"`
	OrderDate            string      `json:"order_date"`
	ExpectedDeliveryDate string      `json:"expected_delivery_date"`
	Status               OrderStatus `json:"status"`
	Lines                []OrderLine `json:"lines"`
	SubTotal             core.Money  `json:"sub_total"`
	DiscountRate         float64     `json:"discount_rate"`
	DiscountAmount       core.Money  `json:"discount_amount"`
	VATAmount            core.Money  `json:"vat_amount"`
	TotalAmount          core.Money  `json:"total_amount"`
	PaymentTerms         string      `json:"payment_terms"`
	DeliveryAddress      string      `json:"delivery_address"`
	Notes                string      `json:"notes,omitempty"`
	CreatedBy            string      `json:"created_by,omitempty"`
	CreatedAt            string      `json:"created_at"`
	UpdatedBy            string      `json:"updated_by,omitempty"`
	UpdatedAt            string      `json:"updated_at"`
}

// OrderLine represents a single line item on a purchase order.
type OrderLine struct {
	LineNo      int     `json:"line_no"`
	ItemCode    string  `json:"item_code"`
	ItemName    string  `json:"item_name"`
	Unit        string  `json:"unit"`
	Quantity    int64   `json:"quantity"`
	ReceivedQty int64   `json:"received_qty"`
	UnitPrice   int64   `json:"unit_price"`
	Discount    float64 `json:"discount"`
	Amount      int64   `json:"amount"`
	VATRate     float64 `json:"vat_rate"`
	VATAmount   int64   `json:"vat_amount"`
	TotalAmount int64   `json:"total_amount"`
}

func (o *PurchaseOrder) Clone() *PurchaseOrder {
	cp := *o
	if o.Lines != nil {
		cp.Lines = make([]OrderLine, len(o.Lines))
		copy(cp.Lines, o.Lines)
	}
	return &cp
}

// ValidatePurchaseOrder validates order data.
func ValidatePurchaseOrder(o *PurchaseOrder) error {
	if o.SupplierCode == "" {
		return &core.ValidationError{Field: "supplier_code", Message: "supplier code is required"}
	}
	if len(o.Lines) == 0 {
		return ErrEmptyLines
	}
	if o.OrderDate == "" {
		return &core.ValidationError{Field: "order_date", Message: "order date is required"}
	}
	if o.Status == "" {
		o.Status = OrderDraft
	}
	if !o.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	for _, line := range o.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.Quantity <= 0 {
			return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
		}
	}
	return nil
}

// ReceiptStatus represents the status of a goods receipt.
type ReceiptStatus string

const (
	ReceiptDraft     ReceiptStatus = "draft"
	ReceiptApproved  ReceiptStatus = "approved"
	ReceiptCompleted ReceiptStatus = "completed"
)

func (s ReceiptStatus) IsValid() bool {
	switch s {
	case ReceiptDraft, ReceiptApproved, ReceiptCompleted:
		return true
	default:
		return false
	}
}

// GoodsReceipt represents a goods receipt note (Phiếu nhập kho).
type GoodsReceipt struct {
	ID              string        `json:"id"`
	RefNo           string        `json:"ref_no"` // Auto: NK-XXXXX
	POID            string        `json:"po_id"`
	PORefNo         string        `json:"po_ref_no"`
	SupplierCode    string        `json:"supplier_code"`
	SupplierName    string        `json:"supplier_name"`
	WarehouseCode   string        `json:"warehouse_code"`
	ReceiptDate     string        `json:"receipt_date"`
	Status          ReceiptStatus `json:"status"`
	Lines           []ReceiptLine `json:"lines"`
	InspectionNotes string        `json:"inspection_notes"`
	Inspector       string        `json:"inspector"`
	CreatedBy       string        `json:"created_by,omitempty"`
	CreatedAt       string        `json:"created_at"`
	UpdatedBy       string        `json:"updated_by,omitempty"`
	UpdatedAt       string        `json:"updated_at"`
}

// ReceiptLine represents a single line item on a goods receipt.
type ReceiptLine struct {
	LineNo           int     `json:"line_no"`
	POLineNo         int     `json:"po_line_no"`
	ItemCode         string  `json:"item_code"`
	ItemName         string  `json:"item_name"`
	Unit             string  `json:"unit"`
	QuantityOrdered  int64   `json:"quantity_ordered"`
	QuantityReceived int64   `json:"quantity_received"`
	QuantityAccepted int64   `json:"quantity_accepted"`
	QuantityRejected int64   `json:"quantity_rejected"`
	UnitPrice        int64   `json:"unit_price"`
	Amount           int64   `json:"amount"`
	VATRate          float64 `json:"vat_rate"`
	VATAmount        int64   `json:"vat_amount"`
	TotalAmount      int64   `json:"total_amount"`
}

func (g *GoodsReceipt) Clone() *GoodsReceipt {
	cp := *g
	if g.Lines != nil {
		cp.Lines = make([]ReceiptLine, len(g.Lines))
		copy(cp.Lines, g.Lines)
	}
	return &cp
}

// ValidateGoodsReceipt validates goods receipt data.
func ValidateGoodsReceipt(g *GoodsReceipt) error {
	if g.POID == "" {
		return &core.ValidationError{Field: "po_id", Message: "purchase order ID is required"}
	}
	if g.SupplierCode == "" {
		return &core.ValidationError{Field: "supplier_code", Message: "supplier code is required"}
	}
	if len(g.Lines) == 0 {
		return ErrEmptyLines
	}
	if g.ReceiptDate == "" {
		return &core.ValidationError{Field: "receipt_date", Message: "receipt date is required"}
	}
	if g.Status == "" {
		g.Status = ReceiptDraft
	}
	if !g.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	for _, line := range g.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.QuantityReceived <= 0 {
			return &core.ValidationError{Field: "quantity_received", Message: "quantity received must be positive"}
		}
	}
	return nil
}

// InvoiceStatus represents the lifecycle status of a purchase invoice.
type InvoiceStatus string

const (
	InvoiceDraft       InvoiceStatus = "draft"
	InvoicePendingEInv InvoiceStatus = "pending_einvoice"
	InvoiceValidated   InvoiceStatus = "validated"
	InvoicePosted      InvoiceStatus = "posted"
	InvoicePaid        InvoiceStatus = "paid"
	InvoiceReconciled  InvoiceStatus = "reconciled"
)

func (s InvoiceStatus) IsValid() bool {
	switch s {
	case InvoiceDraft, InvoicePendingEInv, InvoiceValidated, InvoicePosted, InvoicePaid, InvoiceReconciled:
		return true
	default:
		return false
	}
}

// EInvoiceStatus tracks the e-invoice submission state.
type EInvoiceStatus string

const (
	EInvoiceNone      EInvoiceStatus = "none"
	EInvoicePending   EInvoiceStatus = "pending"
	EInvoiceAccepted  EInvoiceStatus = "accepted"
	EInvoiceRejected  EInvoiceStatus = "rejected"
	EInvoiceCancelled EInvoiceStatus = "cancelled"
)

func (s EInvoiceStatus) IsValid() bool {
	switch s {
	case EInvoiceNone, EInvoicePending, EInvoiceAccepted, EInvoiceRejected, EInvoiceCancelled:
		return true
	default:
		return false
	}
}

// PurchaseInvoice represents a purchase invoice (Hóa đơn mua hàng).
type PurchaseInvoice struct {
	ID                string         `json:"id"`
	RefNo             string         `json:"ref_no"` // Auto: MH-XXXXX
	SupplierInvoiceNo string         `json:"supplier_invoice_no"`
	POID              string         `json:"po_id"`
	PORefNo           string         `json:"po_ref_no"`
	GoodsReceiptID    string         `json:"goods_receipt_id"`
	GoodsReceiptRefNo string         `json:"goods_receipt_ref_no"`
	SupplierCode      string         `json:"supplier_code"`
	SupplierName      string         `json:"supplier_name"`
	SupplierTaxCode   string         `json:"supplier_tax_code"`
	SupplierAddress   string         `json:"supplier_address"`
	InvoiceDate       string         `json:"invoice_date"`
	DueDate           string         `json:"due_date"`
	Status            InvoiceStatus  `json:"status"`
	Lines             []InvoiceLine  `json:"lines"`
	SubTotal          core.Money     `json:"sub_total"`
	DiscountRate      float64        `json:"discount_rate"`
	DiscountAmount    core.Money     `json:"discount_amount"`
	VATAmount         core.Money     `json:"vat_amount"`
	TotalAmount       core.Money     `json:"total_amount"`
	EInvoiceSerial    string         `json:"e_invoice_serial,omitempty"`
	EInvoiceNumber    string         `json:"e_invoice_number,omitempty"`
	EInvoiceDate      string         `json:"e_invoice_date,omitempty"`
	EInvoiceStatus    EInvoiceStatus `json:"e_invoice_status"`
	PaidAmount        core.Money     `json:"paid_amount"`
	OutstandingAmount core.Money     `json:"outstanding_amount"`
	GLPosted          bool           `json:"gl_posted"`
	GLReference       string         `json:"gl_reference,omitempty"`
	Notes             string         `json:"notes,omitempty"`
	CreatedBy         string         `json:"created_by,omitempty"`
	CreatedAt         string         `json:"created_at"`
	UpdatedBy         string         `json:"updated_by,omitempty"`
	UpdatedAt         string         `json:"updated_at"`
}

// InvoiceLine represents a single line item on a purchase invoice.
type InvoiceLine struct {
	LineNo      int     `json:"line_no"`
	ItemCode    string  `json:"item_code"`
	ItemName    string  `json:"item_name"`
	Unit        string  `json:"unit"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	Discount    float64 `json:"discount"`
	Amount      int64   `json:"amount"`
	VATRate     float64 `json:"vat_rate"`
	VATAmount   int64   `json:"vat_amount"`
	TotalAmount int64   `json:"total_amount"`
	GLAccount   string  `json:"gl_account"`
}

func (p *PurchaseInvoice) Clone() *PurchaseInvoice {
	cp := *p
	if p.Lines != nil {
		cp.Lines = make([]InvoiceLine, len(p.Lines))
		copy(cp.Lines, p.Lines)
	}
	return &cp
}

// ValidatePurchaseInvoice validates invoice data.
func ValidatePurchaseInvoice(p *PurchaseInvoice) error {
	if p.SupplierCode == "" {
		return &core.ValidationError{Field: "supplier_code", Message: "supplier code is required"}
	}
	if len(p.Lines) == 0 {
		return ErrEmptyLines
	}
	if p.InvoiceDate == "" {
		return &core.ValidationError{Field: "invoice_date", Message: "invoice date is required"}
	}
	if p.Status == "" {
		p.Status = InvoiceDraft
	}
	if !p.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	if p.EInvoiceStatus == "" {
		p.EInvoiceStatus = EInvoiceNone
	}
	if !p.EInvoiceStatus.IsValid() {
		return &core.ValidationError{Field: "e_invoice_status", Message: "invalid e-invoice status"}
	}
	for _, line := range p.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.Quantity <= 0 {
			return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
		}
	}
	return nil
}

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentDraft      PaymentStatus = "draft"
	PaymentApproved   PaymentStatus = "approved"
	PaymentProcessed  PaymentStatus = "processed"
	PaymentReconciled PaymentStatus = "reconciled"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentDraft, PaymentApproved, PaymentProcessed, PaymentReconciled:
		return true
	default:
		return false
	}
}

// PaymentMethod represents the payment method.
type PaymentMethod string

const (
	PaymentBankTransfer PaymentMethod = "bank_transfer"
	PaymentCash         PaymentMethod = "cash"
	PaymentCheque       PaymentMethod = "cheque"
)

func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentBankTransfer, PaymentCash, PaymentCheque:
		return true
	default:
		return false
	}
}

// Payment represents a payment to a supplier (Thanh toán).
type Payment struct {
	ID              string               `json:"id"`
	RefNo           string               `json:"ref_no"` // Auto: TT-XXXXX
	SupplierCode    string               `json:"supplier_code"`
	SupplierName    string               `json:"supplier_name"`
	PaymentDate     string               `json:"payment_date"`
	PaymentMethod   PaymentMethod        `json:"payment_method"`
	BankAccount     string               `json:"bank_account"`
	BankName        string               `json:"bank_name"`
	ChequeNumber    string               `json:"cheque_number"`
	Amount          core.Money           `json:"amount"`
	AppliedInvoices []PaymentApplication `json:"applied_invoices"`
	Status          PaymentStatus        `json:"status"`
	ApprovedBy      string               `json:"approved_by"`
	Notes           string               `json:"notes,omitempty"`
	CreatedBy       string               `json:"created_by,omitempty"`
	CreatedAt       string               `json:"created_at"`
	UpdatedBy       string               `json:"updated_by,omitempty"`
	UpdatedAt       string               `json:"updated_at"`
}

// PaymentApplication links a payment to a specific invoice.
type PaymentApplication struct {
	InvoiceID     string `json:"invoice_id"`
	InvoiceRefNo  string `json:"invoice_ref_no"`
	AmountApplied int64  `json:"amount_applied"`
}

func (p *Payment) Clone() *Payment {
	cp := *p
	if p.AppliedInvoices != nil {
		cp.AppliedInvoices = make([]PaymentApplication, len(p.AppliedInvoices))
		copy(cp.AppliedInvoices, p.AppliedInvoices)
	}
	return &cp
}

// ValidatePayment validates payment data.
func ValidatePayment(p *Payment) error {
	if p.SupplierCode == "" {
		return &core.ValidationError{Field: "supplier_code", Message: "supplier code is required"}
	}
	if p.PaymentDate == "" {
		return &core.ValidationError{Field: "payment_date", Message: "payment date is required"}
	}
	if p.PaymentMethod == "" {
		return &core.ValidationError{Field: "payment_method", Message: "payment method is required"}
	}
	if !p.PaymentMethod.IsValid() {
		return &core.ValidationError{Field: "payment_method", Message: "invalid payment method"}
	}
	if p.Amount.AmountMinor <= 0 {
		return &core.ValidationError{Field: "amount", Message: "amount must be positive"}
	}
	if len(p.AppliedInvoices) == 0 {
		return &core.ValidationError{Field: "applied_invoices", Message: "at least one invoice must be applied"}
	}
	if p.Status == "" {
		p.Status = PaymentDraft
	}
	if !p.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// Repository defines the persistence interface for purchase entities.
type Repository interface {
	// Supplier CRUD
	CreateSupplier(ctx context.Context, s *Supplier) error
	FindSupplierByID(ctx context.Context, id string) (*Supplier, error)
	UpdateSupplier(ctx context.Context, s *Supplier) error
	DeleteSupplier(ctx context.Context, id string) error
	ListSuppliers(ctx context.Context, name string, status SupplierStatus, limit, offset int) ([]*Supplier, error)
	NextSupplierNo(ctx context.Context) (int64, error)

	// Purchase Order CRUD
	CreateOrder(ctx context.Context, o *PurchaseOrder) error
	FindOrderByID(ctx context.Context, id string) (*PurchaseOrder, error)
	UpdateOrder(ctx context.Context, o *PurchaseOrder) error
	DeleteOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, supplierCode string, status OrderStatus, limit, offset int) ([]*PurchaseOrder, error)
	NextOrderNo(ctx context.Context) (int64, error)

	// Goods Receipt CRUD
	CreateReceipt(ctx context.Context, r *GoodsReceipt) error
	FindReceiptByID(ctx context.Context, id string) (*GoodsReceipt, error)
	UpdateReceipt(ctx context.Context, r *GoodsReceipt) error
	ListReceipts(ctx context.Context, supplierCode string, limit, offset int) ([]*GoodsReceipt, error)
	HasReceiptsForOrder(ctx context.Context, poID string) (bool, error)
	NextReceiptNo(ctx context.Context) (int64, error)

	// Purchase Invoice CRUD
	CreateInvoice(ctx context.Context, inv *PurchaseInvoice) error
	FindInvoiceByID(ctx context.Context, id string) (*PurchaseInvoice, error)
	UpdateInvoice(ctx context.Context, inv *PurchaseInvoice) error
	DeleteInvoice(ctx context.Context, id string) error
	ListInvoices(ctx context.Context, supplierCode string, status InvoiceStatus, limit, offset int) ([]*PurchaseInvoice, error)
	NextInvoiceNo(ctx context.Context) (int64, error)

	// Payment CRUD
	CreatePayment(ctx context.Context, p *Payment) error
	FindPaymentByID(ctx context.Context, id string) (*Payment, error)
	UpdatePayment(ctx context.Context, p *Payment) error
	ListPayments(ctx context.Context, supplierCode string, limit, offset int) ([]*Payment, error)
	NextPaymentNo(ctx context.Context) (int64, error)

	// Supplier balance
	GetSupplierBalance(ctx context.Context, supplierCode string) (core.Money, error)
}
