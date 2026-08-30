package sales

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound         = errors.New("sales: not found")
	ErrDuplicate        = errors.New("sales: duplicate")
	ErrInvalid          = errors.New("sales: invalid input")
	ErrConflict         = errors.New("sales: conflict")
	ErrInsufficientQty  = errors.New("sales: insufficient quantity")
	ErrInvalidQuantity  = errors.New("sales: invalid quantity")
	ErrEmptyLines       = errors.New("sales: at least one line item required")
	ErrReturnExceedsQty = errors.New("sales: return quantity exceeds invoice quantity")
)

// InvoiceStatus represents the lifecycle status of a sales invoice.
type InvoiceStatus string

const (
	InvoiceDraft    InvoiceStatus = "draft"
	InvoicePending  InvoiceStatus = "pending" // Awaiting e-invoice
	InvoiceIssued   InvoiceStatus = "issued"  // E-invoice issued
	InvoicePartial  InvoiceStatus = "partial_paid"
	InvoicePaid     InvoiceStatus = "paid"
	InvoiceOverdue  InvoiceStatus = "overdue"
	InvoiceVoided   InvoiceStatus = "voided"
	InvoiceReturned InvoiceStatus = "returned"
)

func (s InvoiceStatus) IsValid() bool {
	switch s {
	case InvoiceDraft, InvoicePending, InvoiceIssued, InvoicePartial,
		InvoicePaid, InvoiceOverdue, InvoiceVoided, InvoiceReturned:
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

// InvoiceLine represents a single line item on a sales invoice.
type InvoiceLine struct {
	LineNo      int     `json:"line_no"`
	ItemCode    string  `json:"item_code"`
	ItemName    string  `json:"item_name"`
	Unit        string  `json:"unit"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"` // VND
	Discount    float64 `json:"discount"`   // 0-100%
	Amount      int64   `json:"amount"`     // quantity * unit_price * (1-discount/100)
	VATAmount   int64   `json:"vat_amount"`
	TotalAmount int64   `json:"total_amount"`    // amount + vat_amount
	COGSAccount string  `json:"cogs_account"`    // 6321, 6322
	RevenueAcct string  `json:"revenue_account"` // 5111, 5112, 5113
}

// SalesInvoice represents a sales invoice (Hóa đơn bán hàng / E-Invoice).
type SalesInvoice struct {
	ID              string        `json:"id"`
	RefNo           string        `json:"ref_no"` // Auto: HD-XXXXX
	OrderID         string        `json:"order_id,omitempty"`
	CustomerCode    string        `json:"customer_code"`
	CustomerName    string        `json:"customer_name"`
	CustomerAddress string        `json:"customer_address"`
	CustomerTaxCode string        `json:"customer_tax_code"`
	InvoiceDate     string        `json:"invoice_date"`
	DueDate         string        `json:"due_date"`
	Status          InvoiceStatus `json:"status"`
	Lines           []InvoiceLine `json:"lines"`
	SubTotal        core.Money    `json:"sub_total"`
	DiscountRate    float64       `json:"discount_rate"`
	DiscountAmount  core.Money    `json:"discount_amount"`
	VATAmount       core.Money    `json:"vat_amount"`
	TotalAmount     core.Money    `json:"total_amount"`
	// E-invoice fields
	EInvoiceSerial string         `json:"e_invoice_serial,omitempty"`
	EInvoiceNumber string         `json:"e_invoice_number,omitempty"`
	EInvoiceDate   string         `json:"e_invoice_date,omitempty"`
	EInvoiceStatus EInvoiceStatus `json:"e_invoice_status"`
	// Payment tracking
	PaidAmount     core.Money `json:"paid_amount"`
	OutstandingAmt core.Money `json:"outstanding_amount"`
	// GL posting
	GLPosted    bool   `json:"gl_posted"`
	GLReference string `json:"gl_reference,omitempty"`
	Notes       string `json:"notes,omitempty"`
	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedBy string `json:"updated_by,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

func (i *SalesInvoice) Clone() *SalesInvoice {
	cp := *i
	if i.Lines != nil {
		cp.Lines = make([]InvoiceLine, len(i.Lines))
		copy(cp.Lines, i.Lines)
	}
	return &cp
}

// ValidateSalesInvoice validates invoice data per Thông tư 99/2025/TT-BTC.
func ValidateSalesInvoice(inv *SalesInvoice) error {
	if inv.CustomerCode == "" {
		return &core.ValidationError{Field: "customer_code", Message: "customer code is required"}
	}
	if len(inv.Lines) == 0 {
		return ErrEmptyLines
	}
	if inv.InvoiceDate == "" {
		return &core.ValidationError{Field: "invoice_date", Message: "invoice date is required"}
	}
	if inv.Status == "" {
		inv.Status = InvoiceDraft
	}
	if !inv.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	if inv.EInvoiceStatus == "" {
		inv.EInvoiceStatus = EInvoiceNone
	}
	if !inv.EInvoiceStatus.IsValid() {
		return &core.ValidationError{Field: "e_invoice_status", Message: "invalid e-invoice status"}
	}
	// Validate line items
	for i, line := range inv.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.Quantity <= 0 {
			return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
		}
		if line.UnitPrice < 0 {
			return &core.ValidationError{Field: "unit_price", Message: "unit price must be non-negative"}
		}
		_ = i // line number validation if needed
	}
	return nil
}

// Repository defines the persistence interface for sales entities.
type Repository interface {
	// Invoice CRUD
	CreateInvoice(ctx context.Context, inv *SalesInvoice) error
	FindInvoiceByID(ctx context.Context, id string) (*SalesInvoice, error)
	UpdateInvoice(ctx context.Context, inv *SalesInvoice) error
	DeleteInvoice(ctx context.Context, id string) error
	ListInvoices(ctx context.Context, customerCode string, status InvoiceStatus, limit, offset int) ([]*SalesInvoice, error)
	NextInvoiceNo(ctx context.Context) (int64, error)

	// Order CRUD
	CreateOrder(ctx context.Context, o *SalesOrder) error
	FindOrderByID(ctx context.Context, id string) (*SalesOrder, error)
	UpdateOrder(ctx context.Context, o *SalesOrder) error
	DeleteOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, customerCode string, status OrderStatus, limit, offset int) ([]*SalesOrder, error)
	NextOrderNo(ctx context.Context) (int64, error)

	// Return CRUD
	CreateReturn(ctx context.Context, r *SalesReturn) error
	FindReturnByID(ctx context.Context, id string) (*SalesReturn, error)
	UpdateReturn(ctx context.Context, r *SalesReturn) error
	ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*SalesReturn, error)
	NextReturnNo(ctx context.Context) (int64, error)

	// Customer balance
	GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error)
}
