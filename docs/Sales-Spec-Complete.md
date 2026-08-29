# Sales Module - Complete Specification
## Bán hàng (Sales)

**Version**: 1.0
**Date**: 2026-08-29
**Module**: Sales
**Compliance**: Thông tư 99/2025/TT-BTC, Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP, VAS 14, Circular 32/2025/TT-BTC

---

## 1. Business Rules (Regulatory)

### BR-01: E-Invoice Requirement
All enterprises must issue e-invoices connected to GDT system. E-invoices cannot be cancelled; must issue credit notes for corrections.

### BR-02: Revenue Recognition
Revenue recognized when risk/reward transfers to buyer. Account 511 (domestic), 512 (export). VAT calculated separately on sales price.

### BR-03: GL Account Mapping
| Account | Name | Purpose |
|---------|------|---------|
| 511 | Doanh thu bán hàng | Sales revenue (domestic) |
| 512 | Doanh thu bán hàng hóa, dịch vụ | Revenue from goods/services |
| 131 | Phải thu khách hàng | Accounts receivable |
| 333 | Thuế GTGT phải nộp | VAT payable |
| 3331 | Thuế GTGT hàng bán | Output VAT |
| 133 | Thuế GTGT được khấu trừ | Input VAT deductible |
| 632 | Giá vốn hàng bán | Cost of goods sold |
| 6321 | Giá vốn bán thành phẩm | Finished goods COGS |
| 6322 | Giá vốn BDM | Raw material COGS |
| 156 | Hàng hóa | Finished goods inventory |
| 152 | Nguyên vật liệu | Raw material inventory |
| 1389 | Tiền ứng trước | Advance payments |

### BR-04: Transaction Types
| Transaction | GL Entry |
|-------------|----------|
| Sale (credit) | Dr 131 / Cr 511, Cr 333 |
| Sale (cash) | Dr 111/112 / Cr 511, Cr 333 |
| COGS posting | Dr 632 / Cr 156, Cr 152 |
| Return (credit note) | Dr 511, Dr 333 / Cr 131 |
| Discount | Dr 511 / Cr 131 (or reduce invoice) |
| Advance received | Dr 111/112 / Cr 1389 |
| Advance offset | Dr 1389 / Cr 131 |

### BR-05: Price Calculation
Base Price → Discount % → Unit Price → Line Total → VAT → Invoice Total

### BR-06: Delivery Tracking
Status: pending → in_transit → delivered → completed. Partial deliveries allowed. Delivery confirmation triggers COGS posting.

### BR-07: Sales Returns
Credit note issued against original invoice. Return quantity ≤ original invoice quantity. GL reversal required.

---

## 2. Data Model

### 2.1 Sales Quotation Entity

```go
package sales

type SalesQuotation struct {
    ID               string      `json:"id"`
    RefNo            string      `json:"ref_no"`           // Auto: BC-XXXXX
    CustomerCode     string      `json:"customer_code"`
    CustomerName     string      `json:"customer_name"`
    QuotationDate    string      `json:"quotation_date"`
    ValidUntil       string      `json:"valid_until"`
    Status           QuoteStatus `json:"status"`
    Lines            []QuoteLine `json:"lines"`
    SubTotal         core.Money  `json:"sub_total"`
    DiscountRate     float64     `json:"discount_rate"`
    DiscountAmount   core.Money  `json:"discount_amount"`
    VATAmount        core.Money  `json:"vat_amount"`
    TotalAmount      core.Money  `json:"total_amount"`
    Notes            string      `json:"notes,omitempty"`
    ConvertedToOrder bool        `json:"converted_to_order"`
    ConvertedOrderID string      `json:"converted_order_id,omitempty"`
    CreatedBy        string      `json:"created_by"`
    CreatedAt        string      `json:"created_at"`
    UpdatedBy        string      `json:"updated_by"`
    UpdatedAt        string      `json:"updated_at"`
}

type QuoteLine struct {
    LineNo      int    `json:"line_no"`
    ItemCode    string `json:"item_code"`
    ItemName    string `json:"item_name"`
    Unit        string `json:"unit"`
    Quantity    int64  `json:"quantity"`
    UnitPrice   int64  `json:"unit_price"`
    Discount    float64 `json:"discount"`
    Amount      int64  `json:"amount"`
    VATAmount   int64  `json:"vat_amount"`
    TotalAmount int64  `json:"total_amount"`
}

type QuoteStatus string

const (
    QuoteDraft    QuoteStatus = "draft"
    QuoteSent     QuoteStatus = "sent"
    QuoteAccepted QuoteStatus = "accepted"
    QuoteRejected QuoteStatus = "rejected"
    QuoteExpired  QuoteStatus = "expired"
)
```

### 2.2 Sales Order Entity

```go
package sales

type SalesOrder struct {
    ID               string        `json:"id"`
    RefNo            string        `json:"ref_no"`           // Auto: DH-XXXXX
    QuoteID          string        `json:"quote_id,omitempty"`
    CustomerCode     string        `json:"customer_code"`
    CustomerName     string        `json:"customer_name"`
    OrderDate        string        `json:"order_date"`
    DeliveryDate     string        `json:"delivery_date"`
    Status           OrderStatus   `json:"status"`
    Lines            []OrderLine   `json:"lines"`
    SubTotal         core.Money    `json:"sub_total"`
    DiscountRate     float64       `json:"discount_rate"`
    DiscountAmount   core.Money    `json:"discount_amount"`
    VATAmount        core.Money    `json:"vat_amount"`
    TotalAmount      core.Money    `json:"total_amount"`
    DeliveryAddress  string        `json:"delivery_address"`
    PaymentTerms     string        `json:"payment_terms"`
    Notes            string        `json:"notes,omitempty"`
    DeliveryStatus   DeliveryStatus `json:"delivery_status"`
    DeliveredAmount  core.Money     `json:"delivered_amount"`
    InvoicedAmount   core.Money     `json:"invoiced_amount"`
    CreatedBy        string         `json:"created_by"`
    CreatedAt        string         `json:"created_at"`
    UpdatedBy        string         `json:"updated_by"`
    UpdatedAt        string         `json:"updated_at"`
}

type OrderLine struct {
    LineNo       int    `json:"line_no"`
    ItemCode     string `json:"item_code"`
    ItemName     string `json:"item_name"`
    Unit         string `json:"unit"`
    Quantity     int64  `json:"quantity"`
    DeliveredQty int64  `json:"delivered_qty"`
    InvoicedQty  int64  `json:"invoiced_qty"`
    UnitPrice    int64  `json:"unit_price"`
    Discount     float64 `json:"discount"`
    Amount       int64  `json:"amount"`
    VATAmount    int64  `json:"vat_amount"`
    TotalAmount  int64  `json:"total_amount"`
}

type OrderStatus string

const (
    OrderDraft      OrderStatus = "draft"
    OrderConfirmed  OrderStatus = "confirmed"
    OrderPartialDel OrderStatus = "partial_delivery"
    OrderDelivered  OrderStatus = "delivered"
    OrderPartialInv OrderStatus = "partial_invoice"
    OrderInvoiced   OrderStatus = "invoiced"
    OrderCompleted  OrderStatus = "completed"
    OrderCancelled  OrderStatus = "cancelled"
)

type DeliveryStatus string

const (
    DeliveryPending   DeliveryStatus = "pending"
    DeliveryPartial   DeliveryStatus = "partial"
    DeliveryInTransit DeliveryStatus = "in_transit"
    DeliveryDelivered DeliveryStatus = "delivered"
    DeliveryCompleted DeliveryStatus = "completed"
)
```

### 2.3 Sales Invoice Entity

```go
package sales

type SalesInvoice struct {
    ID              string          `json:"id"`
    RefNo           string          `json:"ref_no"`           // Auto: HD-XXXXX
    OrderID         string          `json:"order_id,omitempty"`
    CustomerCode    string          `json:"customer_code"`
    CustomerName    string          `json:"customer_name"`
    CustomerAddress string          `json:"customer_address"`
    CustomerTaxCode string          `json:"customer_tax_code"`
    InvoiceDate     string          `json:"invoice_date"`
    DueDate         string          `json:"due_date"`
    Status          InvoiceStatus   `json:"status"`
    Lines           []InvoiceLine   `json:"lines"`
    SubTotal        core.Money      `json:"sub_total"`
    DiscountRate    float64         `json:"discount_rate"`
    DiscountAmount  core.Money      `json:"discount_amount"`
    VATAmount       core.Money      `json:"vat_amount"`
    TotalAmount     core.Money      `json:"total_amount"`
    EInvoiceSerial  string          `json:"e_invoice_serial,omitempty"`
    EInvoiceNumber  string          `json:"e_invoice_number,omitempty"`
    EInvoiceDate    string          `json:"e_invoice_date,omitempty"`
    EInvoiceStatus  EInvoiceStatus  `json:"e_invoice_status"`
    PaidAmount      core.Money      `json:"paid_amount"`
    OutstandingAmt  core.Money      `json:"outstanding_amount"`
    GLPosted        bool            `json:"gl_posted"`
    GLReference     string          `json:"gl_reference,omitempty"`
    Notes           string          `json:"notes,omitempty"`
    CreatedBy       string          `json:"created_by"`
    CreatedAt       string          `json:"created_at"`
    UpdatedBy       string          `json:"updated_by"`
    UpdatedAt       string          `json:"updated_at"`
}

type InvoiceLine struct {
    LineNo      int     `json:"line_no"`
    ItemCode    string  `json:"item_code"`
    ItemName    string  `json:"item_name"`
    Unit        string  `json:"unit"`
    Quantity    int64   `json:"quantity"`
    UnitPrice   int64   `json:"unit_price"`
    Discount    float64 `json:"discount"`
    Amount      int64   `json:"amount"`
    VATAmount   int64   `json:"vat_amount"`
    TotalAmount int64   `json:"total_amount"`
    COGSAccount string  `json:"cogs_account"`
    RevenueAcct string  `json:"revenue_account"`
}

type InvoiceStatus string

const (
    InvoiceDraft     InvoiceStatus = "draft"
    InvoicePending   InvoiceStatus = "pending"
    InvoiceIssued    InvoiceStatus = "issued"
    InvoicePartial   InvoiceStatus = "partial_paid"
    InvoicePaid      InvoiceStatus = "paid"
    InvoiceOverdue   InvoiceStatus = "overdue"
    InvoiceVoided    InvoiceStatus = "voided"
    InvoiceReturned  InvoiceStatus = "returned"
)

type EInvoiceStatus string

const (
    EInvoiceNone      EInvoiceStatus = "none"
    EInvoicePending   EInvoiceStatus = "pending"
    EInvoiceAccepted  EInvoiceStatus = "accepted"
    EInvoiceRejected  EInvoiceStatus = "rejected"
    EInvoiceCancelled EInvoiceStatus = "cancelled"
)
```

### 2.4 Sales Return Entity

```go
package sales

type SalesReturn struct {
    ID           string         `json:"id"`
    RefNo        string         `json:"ref_no"`           // Auto: PH-XXXXX
    InvoiceID    string         `json:"invoice_id"`
    CustomerCode string         `json:"customer_code"`
    ReturnDate   string         `json:"return_date"`
    Reason       ReturnReason   `json:"reason"`
    Status       ReturnStatus   `json:"status"`
    Lines        []ReturnLine   `json:"lines"`
    SubTotal     core.Money     `json:"sub_total"`
    VATAmount    core.Money     `json:"vat_amount"`
    TotalAmount  core.Money     `json:"total_amount"`
    CreditNoteNo string         `json:"credit_note_no"`
    GLPosted     bool           `json:"gl_posted"`
    GLReference  string         `json:"gl_reference,omitempty"`
    Notes        string         `json:"notes,omitempty"`
    CreatedBy    string         `json:"created_by"`
    CreatedAt    string         `json:"created_at"`
    UpdatedBy    string         `json:"updated_by"`
    UpdatedAt    string         `json:"updated_at"`
}

type ReturnLine struct {
    LineNo      int    `json:"line_no"`
    ItemCode    string `json:"item_code"`
    ItemName    string `json:"item_name"`
    Unit        string `json:"unit"`
    Quantity    int64  `json:"quantity"`
    UnitPrice   int64  `json:"unit_price"`
    Amount      int64  `json:"amount"`
    VATAmount   int64  `json:"vat_amount"`
    TotalAmount int64  `json:"total_amount"`
}

type ReturnReason string

const (
    ReturnDefective ReturnReason = "defective"
    ReturnWrongItem ReturnReason = "wrong_item"
    ReturnDamaged   ReturnReason = "damaged"
    ReturnCustomer  ReturnReason = "customer_request"
    ReturnExpired   ReturnReason = "expired"
)

type ReturnStatus string

const (
    ReturnDraft     ReturnStatus = "draft"
    ReturnApproved  ReturnStatus = "approved"
    ReturnReceived  ReturnStatus = "received"
    ReturnIssued    ReturnStatus = "credit_note_issued"
    ReturnCompleted ReturnStatus = "completed"
)
```

### 2.5 Repository Interface

```go
package sales

type Repository interface {
    // Quotation CRUD
    CreateQuotation(ctx context.Context, q *SalesQuotation) error
    FindQuotationByID(ctx context.Context, id string) (*SalesQuotation, error)
    UpdateQuotation(ctx context.Context, q *SalesQuotation) error
    DeleteQuotation(ctx context.Context, id string) error
    ListQuotations(ctx context.Context, customerCode string, status QuoteStatus, limit, offset int) ([]*SalesQuotation, error)
    NextQuotationNo(ctx context.Context) (int64, error)

    // Order CRUD
    CreateOrder(ctx context.Context, o *SalesOrder) error
    FindOrderByID(ctx context.Context, id string) (*SalesOrder, error)
    UpdateOrder(ctx context.Context, o *SalesOrder) error
    DeleteOrder(ctx context.Context, id string) error
    ListOrders(ctx context.Context, customerCode string, status OrderStatus, limit, offset int) ([]*SalesOrder, error)
    NextOrderNo(ctx context.Context) (int64, error)

    // Invoice CRUD
    CreateInvoice(ctx context.Context, inv *SalesInvoice) error
    FindInvoiceByID(ctx context.Context, id string) (*SalesInvoice, error)
    UpdateInvoice(ctx context.Context, inv *SalesInvoice) error
    ListInvoices(ctx context.Context, customerCode string, status InvoiceStatus, limit, offset int) ([]*SalesInvoice, error)
    NextInvoiceNo(ctx context.Context) (int64, error)

    // Return CRUD
    CreateReturn(ctx context.Context, r *SalesReturn) error
    FindReturnByID(ctx context.Context, id string) (*SalesReturn, error)
    UpdateReturn(ctx context.Context, r *SalesReturn) error
    ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*SalesReturn, error)
    NextReturnNo(ctx context.Context) (int64, error)

    // Delivery tracking
    UpdateDeliveryStatus(ctx context.Context, orderID string, status DeliveryStatus, deliveredAmount core.Money) error
    UpdateLineDeliveredQty(ctx context.Context, orderID string, lineNo int, qty int64) error

    // Invoice payment tracking
    UpdateInvoicePayment(ctx context.Context, invoiceID string, paidAmount core.Money) error

    // Customer balance
    GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error)
    GetCustomerOutstanding(ctx context.Context, customerCode string) (core.Money, error)
}
```

---

## 3. Service Layer

### 3.1 Service Interface

```go
package sales

type Service interface {
    // Quotation operations
    CreateQuotation(ctx context.Context, q *SalesQuotation, actor string) (*SalesQuotation, error)
    GetQuotation(ctx context.Context, id string) (*SalesQuotation, error)
    UpdateQuotation(ctx context.Context, id string, patch *SalesQuotation, actor string) (*SalesQuotation, error)
    DeleteQuotation(ctx context.Context, id string) error
    ListQuotations(ctx context.Context, customerCode string, status QuoteStatus, limit, offset int) ([]*SalesQuotation, error)
    SendQuotation(ctx context.Context, id string, actor string) (*SalesQuotation, error)
    AcceptQuotation(ctx context.Context, id string, actor string) (*SalesOrder, error)
    RejectQuotation(ctx context.Context, id string, reason string, actor string) (*SalesQuotation, error)

    // Order operations
    CreateOrder(ctx context.Context, o *SalesOrder, actor string) (*SalesOrder, error)
    GetOrder(ctx context.Context, id string) (*SalesOrder, error)
    UpdateOrder(ctx context.Context, id string, patch *SalesOrder, actor string) (*SalesOrder, error)
    DeleteOrder(ctx context.Context, id string) error
    ListOrders(ctx context.Context, customerCode string, status OrderStatus, limit, offset int) ([]*SalesOrder, error)
    ConfirmOrder(ctx context.Context, id string, actor string) (*SalesOrder, error)
    CancelOrder(ctx context.Context, id string, reason string, actor string) (*SalesOrder, error)

    // Delivery operations
    DeliverOrder(ctx context.Context, orderID string, lines []DeliveryLine, actor string) (*SalesOrder, error)

    // Invoice operations
    CreateInvoice(ctx context.Context, inv *SalesInvoice, actor string) (*SalesInvoice, error)
    GetInvoice(ctx context.Context, id string) (*SalesInvoice, error)
    UpdateInvoice(ctx context.Context, id string, patch *SalesInvoice, actor string) (*SalesInvoice, error)
    ListInvoices(ctx context.Context, customerCode string, status InvoiceStatus, limit, offset int) ([]*SalesInvoice, error)
    IssueInvoice(ctx context.Context, id string, actor string) (*SalesInvoice, error)
    VoidInvoice(ctx context.Context, id string, reason string, actor string) (*SalesInvoice, error)

    // Return operations
    CreateReturn(ctx context.Context, r *SalesReturn, actor string) (*SalesReturn, error)
    GetReturn(ctx context.Context, id string) (*SalesReturn, error)
    ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*SalesReturn, error)
    ApproveReturn(ctx context.Context, id string, actor string) (*SalesReturn, error)
    ReceiveReturn(ctx context.Context, id string, actor string) (*SalesReturn, error)

    // Customer operations
    GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error)
    GetCustomerOutstanding(ctx context.Context, customerCode string) (core.Money, error)
}

type DeliveryLine struct {
    LineNo   int   `json:"line_no"`
    ItemCode string `json:"item_code"`
    Quantity int64  `json:"quantity"`
}
```

### 3.2 Validation Functions

```go
func ValidateSalesInvoice(inv *SalesInvoice) error {
    if inv.CustomerCode == "" {
        return &ValidationError{Field: "customer_code", Message: "customer code is required"}
    }
    if len(inv.Lines) == 0 {
        return &ValidationError{Field: "lines", Message: "at least one line item required"}
    }
    for i, line := range inv.Lines {
        if line.ItemCode == "" {
            return &ValidationError{Field: fmt.Sprintf("lines[%d].item_code", i), Message: "item code is required"}
        }
        if line.Quantity <= 0 {
            return &ValidationError{Field: fmt.Sprintf("lines[%d].quantity", i), Message: "quantity must be positive"}
        }
        if line.UnitPrice < 0 {
            return &ValidationError{Field: fmt.Sprintf("lines[%d].unit_price", i), Message: "unit price must be non-negative"}
        }
    }
    if inv.InvoiceDate == "" {
        return &ValidationError{Field: "invoice_date", Message: "invoice date is required"}
    }
    // Validate line totals
    for i, line := range inv.Lines {
        expectedAmount := line.Quantity * int64(line.UnitPrice) * int64(100-line.Discount) / 100
        if line.Amount != expectedAmount {
            return &ValidationError{Field: fmt.Sprintf("lines[%d].amount", i), Message: "amount mismatch"}
        }
    }
    return nil
}
```

---

## 4. HTTP Handler

### 4.1 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sales/quotations` | Create quotation |
| GET | `/api/v1/sales/quotations` | List quotations |
| GET | `/api/v1/sales/quotations/:id` | Get quotation by ID |
| PUT | `/api/v1/sales/quotations/:id` | Update quotation |
| DELETE | `/api/v1/sales/quotations/:id` | Delete quotation |
| POST | `/api/v1/sales/quotations/:id/send` | Send quotation to customer |
| POST | `/api/v1/sales/quotations/:id/accept` | Accept & convert to order |
| POST | `/api/v1/sales/quotations/:id/reject` | Reject quotation |
| POST | `/api/v1/sales/orders` | Create order |
| GET | `/api/v1/sales/orders` | List orders |
| GET | `/api/v1/sales/orders/:id` | Get order by ID |
| PUT | `/api/v1/sales/orders/:id` | Update order |
| DELETE | `/api/v1/sales/orders/:id` | Delete order |
| POST | `/api/v1/sales/orders/:id/confirm` | Confirm order |
| POST | `/api/v1/sales/orders/:id/cancel` | Cancel order |
| POST | `/api/v1/sales/orders/:id/deliver` | Record delivery |
| POST | `/api/v1/sales/invoices` | Create invoice |
| GET | `/api/v1/sales/invoices` | List invoices |
| GET | `/api/v1/sales/invoices/:id` | Get invoice by ID |
| PUT | `/api/v1/sales/invoices/:id` | Update invoice |
| POST | `/api/v1/sales/invoices/:id/issue` | Issue e-invoice |
| POST | `/api/v1/sales/invoices/:id/void` | Void invoice |
| POST | `/api/v1/sales/returns` | Create return |
| GET | `/api/v1/sales/returns` | List returns |
| GET | `/api/v1/sales/returns/:id` | Get return by ID |
| POST | `/api/v1/sales/returns/:id/approve` | Approve return |
| POST | `/api/v1/sales/returns/:id/receive` | Receive returned goods |
| GET | `/api/v1/sales/customers/:code/balance` | Get customer balance |
| GET | `/api/v1/sales/customers/:code/outstanding` | Get customer outstanding |

### 4.2 Request/Response Examples

#### Create Invoice
```json
// Request
POST /api/v1/sales/invoices
{
    "customer_code": "KH-001",
    "customer_name": "ABC Company",
    "customer_address": "123 Nguyen Hue, District 1, HCMC",
    "customer_tax_code": "0123456789",
    "invoice_date": "2026-08-29",
    "due_date": "2026-09-29",
    "lines": [
        {
            "item_code": "SP-001",
            "item_name": "Widget A",
            "unit": "pcs",
            "quantity": 100,
            "unit_price": 500000,
            "discount": 5,
            "cogs_account": "6321",
            "revenue_account": "5111"
        }
    ],
    "discount_rate": 0
}

// Response
{
    "data": {
        "id": "inv_abc123",
        "ref_no": "HD-00001",
        "customer_code": "KH-001",
        "sub_total": {"amount_minor": 47500000, "currency": "VND"},
        "vat_amount": {"amount_minor": 4750000, "currency": "VND"},
        "total_amount": {"amount_minor": 52250000, "currency": "VND"},
        "status": "draft",
        "e_invoice_status": "none",
        "created_at": "2026-08-29T10:30:00Z"
    }
}
```

#### Issue E-Invoice
```json
// Request
POST /api/v1/sales/invoices/inv_abc123/issue

// Response
{
    "data": {
        "id": "inv_abc123",
        "ref_no": "HD-00001",
        "status": "issued",
        "e_invoice_serial": "1C25TCH",
        "e_invoice_number": "000001",
        "e_invoice_date": "2026-08-29T10:35:00Z",
        "e_invoice_status": "accepted",
        "gl_posted": true,
        "gl_reference": "gl_ref_123"
    }
}
```

---

## 5. UI/UX Wireframes

### 5.1 Sales Invoice List Page
```
┌─────────────────────────────────────────────────────────────────────┐
│ Sales Invoices / Hóa đơn bán hàng                               [+]│
├─────────────────────────────────────────────────────────────────────┤
│ Filters:                                                            │
│ Customer: [__________▼]  Status: [All ▼]  Date: [____] to [____]   │
│ Search: [______________] [🔍]                                       │
├─────────────────────────────────────────────────────────────────────┤
│ Ref No   │ Customer     │ Date       │ Total      │ Status  │ Action│
│----------|--------------|------------|------------|---------|-------│
│ HD-00001 │ ABC Company  │ 2026-08-29 │ 52,250,000 │ Issued  │ 👁️ ✏️│
│ HD-00002 │ XYZ Corp     │ 2026-08-28 │ 23,100,000 │ Paid    │ 👁️ ✏️│
│ HD-00003 │ DEF Ltd      │ 2026-08-27 │ 115,500,000│ Overdue │ 👁️ ✏️│
├─────────────────────────────────────────────────────────────────────┤
│ Total: 190,850,000 VND | Showing 1-3 of 3 | < 1 >                │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 Sales Invoice Detail Page
```
┌─────────────────────────────────────────────────────────────────────┐
│ Invoice: HD-00001 - ABC Company                          [E-Invoice]│
├─────────────────────────────────────────────────────────────────────┤
│ General Information:                                                │
│ Ref No: HD-00001          Invoice Date: 2026-08-29                 │
│ Customer: ABC Company      Due Date: 2026-09-29                    │
│ Address: 123 Nguyen Hue    Status: Issued                          │
│ Tax Code: 0123456789       E-Invoice: 1C25TCH-000001              │
├─────────────────────────────────────────────────────────────────────┤
│ Line Items:                                                         │
│ # │ Item      │ Qty  │ Price   │ Disc │ VAT     │ Amount          │
│---│-----------|------|---------|------|---------|-----------------│
│ 1 │ Widget A  │ 100  │ 500,000 │ 5%   │ 4,750,000│ 47,500,000     │
├─────────────────────────────────────────────────────────────────────┤
│ Sub Total: 47,500,000 VND                                          │
│ VAT (10%):  4,750,000 VND                                          │
│ Total:      52,250,000 VND                                         │
│ Paid:       0 VND                                                  │
│ Outstanding: 52,250,000 VND                                        │
├─────────────────────────────────────────────────────────────────────┤
│ GL Posting:                                                         │
│ Dr 131 - Receivable:     52,250,000 VND                           │
│ Cr 511 - Revenue:        47,500,000 VND                           │
│ Cr 333 - VAT Payable:     4,750,000 VND                           │
├─────────────────────────────────────────────────────────────────────┤
│ [Print] [Issue E-Invoice] [Void] [Return] [Payment History]        │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 Create Invoice Form
```
┌─────────────────────────────────────────────────────────────────────┐
│ New Sales Invoice                                            [X]    │
├─────────────────────────────────────────────────────────────────────┤
│ Customer: [KH-001 ▼] ABC Company                                   │
│ Address: 123 Nguyen Hue, District 1, HCMC                          │
│ Tax Code: 0123456789                                               │
│ Invoice Date: [2026-08-29]  Due Date: [2026-09-29]                 │
├─────────────────────────────────────────────────────────────────────┤
│ Line Items:                                                         │
│ + Add Line                                                         │
│ # │ Item Code │ Item Name │ Qty │ Unit Price │ Disc │ Amount       │
│ 1 │ [SP-001▼] │ Widget A  │ [100]│ [500,000] │ [5]% │ 47,500,000  │
├─────────────────────────────────────────────────────────────────────┤
│ Sub Total: 47,500,000 VND                                          │
│ VAT (10%):  4,750,000 VND                                          │
│ Total:      52,250,000 VND                                         │
├─────────────────────────────────────────────────────────────────────┤
│ GL Entry Preview:                                                   │
│ Dr 131 - Receivable:     52,250,000 VND                           │
│ Cr 511 - Revenue:        47,500,000 VND                           │
│ Cr 333 - VAT Payable:     4,750,000 VND                           │
├─────────────────────────────────────────────────────────────────────┤
│ Notes: [________________________________________________]          │
│                              [Cancel]  [Save Draft]  [Save & Issue]│
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Process Flows

### 6.1 Order-to-Cash Flow
```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Quotation│───▶│  Order   │───▶│ Delivery │───▶│ Invoice  │───▶│ Payment  │
│          │    │          │    │          │    │          │    │          │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼
  BC-XXXXX       DH-XXXXX       [delivery]      HD-XXXXX      [payment]
```

### 6.2 Invoice Issuance with E-Invoice
```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Create   │───▶│ Validate │───▶│ Submit   │───▶│ Issue    │
│ Invoice  │    │ Data     │    │ to GDT   │    │ Invoice  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │
     ▼               ▼               ▼               ▼
  Draft          Validation      GDT Accept      Posted GL
                 Passed          Serial+Number   Entry
```

### 6.3 Sales Return Flow
```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Create   │───▶│ Approve  │───▶│ Receive  │───▶│ Issue    │
│ Return   │    │ Return   │    │ Goods    │    │ Credit   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │
     ▼               ▼               ▼               ▼
  Draft          Approved        Received        Credit Note
                 by Manager      to Warehouse    Issued
```

---

## 7. User Journeys

### 7.1 Journey 1: Create & Issue Invoice
```
1. User navigates to Sales Invoices
2. User clicks [+] to create new invoice
3. User selects customer from dropdown
4. System auto-fills customer details
5. User adds line items (item, qty, price)
6. System calculates totals (subtotal, VAT, total)
7. User reviews GL entry preview
8. User clicks [Save Draft]
9. User clicks [Issue E-Invoice]
10. System submits to GDT
11. System receives serial + number
12. System posts GL entry (Dr 131, Cr 511, Cr 333)
13. System marks invoice as "issued"
14. User sees updated invoice with e-invoice details
```

### 7.2 Journey 2: Process Sales Return
```
1. User navigates to Sales Invoice
2. User clicks [Return] on invoice detail
3. System displays Return form
4. User selects items to return
5. User enters return reason
6. User clicks [Create Return]
7. System creates return (status: draft)
8. Manager approves return
9. Warehouse receives returned goods
10. System issues credit note (e-invoice)
11. System posts GL reversal (Dr 511, Dr 333 / Cr 131)
12. System re-enters goods to inventory (Dr 156 / Cr 632)
```

---

## 8. Implementation Roadmap

### Phase 1: Core Entity (Week 1-2)
- [ ] Enhance SalesInvoice entity with full fields
- [ ] Create SalesOrder entity
- [ ] Create SalesQuotation entity
- [ ] Create SalesReturn entity
- [ ] Update Repository interfaces
- [ ] Implement SQLite repository
- [ ] Unit tests

### Phase 2: Service Layer (Week 3-4)
- [ ] Implement Invoice service (CRUD, issue, void)
- [ ] Implement Order service (CRUD, confirm, cancel)
- [ ] Implement Quotation service (CRUD, send, accept, reject)
- [ ] Implement Return service (CRUD, approve, receive)
- [ ] Add validation functions
- [ ] Unit tests

### Phase 3: GL Integration (Week 5)
- [ ] Implement revenue posting (511/512)
- [ ] Implement COGS posting (632)
- [ ] Implement VAT posting (333)
- [ ] Implement receivables posting (131)
- [ ] Add credit note posting
- [ ] Integration tests

### Phase 4: HTTP Handlers (Week 6)
- [ ] Implement all API endpoints
- [ ] Add request validation
- [ ] Add error handling
- [ ] API tests

### Phase 5: E-Invoice Integration (Week 7)
- [ ] Implement GDT API client
- [ ] Implement e-invoice submission
- [ ] Implement e-invoice response handling
- [ ] Implement credit note issuance
- [ ] Integration tests

### Phase 6: Web UI (Week 8-9)
- [ ] Invoice list page
- [ ] Invoice detail page
- [ ] Create invoice form
- [ ] Order management pages
- [ ] Quotation management pages
- [ ] Return management pages
- [ ] Customer balance inquiry

### Phase 7: Reports & Polish (Week 10)
- [ ] Revenue report by customer/item/period
- [ ] Outstanding receivables report
- [ ] E-invoice status report
- [ ] Sales commission report
- [ ] API documentation
- [ ] User documentation

---

## 9. Verification Checklist

- [ ] All transactions comply with Thông tư 99/2025/TT-BTC
- [ ] E-invoices comply with Decree 123/2020 & Decree 70/2025
- [ ] GL Account 511/512 postings are correct
- [ ] COGS postings (632) are correct
- [ ] VAT postings (333) are correct
- [ ] E-invoice data preserved for 10 years
- [ ] All mutations have audit trail
- [ ] Unit tests pass (coverage > 80%)
- [ ] Integration tests pass
- [ ] API documentation complete
- [ ] User documentation complete

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
