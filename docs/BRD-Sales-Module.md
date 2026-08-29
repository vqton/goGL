# Business Requirements Document (BRD)
## Sales Module (Bán hàng / Doanh thu bán hàng)

**Document ID**: BRD-SALES-001
**Version**: 1.0
**Date**: 2026-08-29
**Author**: BA Lead (20+ years) + Chief Accountant (20+ years, CPA Vietnam)
**Compliance**: Thông tư 99/2025/TT-BTC, Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP, VAS 14, Circular 32/2025/TT-BTC

---

## 1. Executive Summary

### 1.1 Purpose
This BRD defines the business requirements for a **Sales Module (Bán hàng)** that complies with Vietnamese accounting standards and e-invoicing regulations. The module manages the complete order-to-cash cycle including quotations, sales orders, delivery tracking, invoicing, returns, and revenue recognition with proper GL integration.

### 1.2 Business Context
According to **Thông tư 99/2025/TT-BTC** (effective from 01/01/2026):
- **Account 511** - Doanh thu bán hàng: Sales revenue
- **Account 512** - Doanh thu bán hàng hóa, dịch vụ: Revenue from goods/services
- **Account 131** - Phải thu khách hàng: Accounts receivable
- **Account 333** - Thuế GTGT: VAT payable
- **Account 632** - Giá vốn hàng bán: Cost of goods sold

**Decree 123/2020/NĐ-CP** (amended by Decree 70/2025/NĐ-CP): Mandatory e-invoicing for all enterprises since July 1, 2022. Must support connection to Vietnam's General Department of Taxation (GDT) e-invoice system.

### 1.3 Scope
| In Scope | Out of Scope |
|----------|--------------|
| Sales quotations/proposals | Manufacturing (handled by costing module) |
| Sales order management | Inventory stock management (handled by inventory module) |
| Delivery tracking | Accounts receivable collections (handled by ledger module) |
| E-invoice issuance via GDT | Tax calculation logic (handled by tax module) |
| Revenue recognition (GL 511/512) | Payment processing (handled by cash/bank modules) |
| Sales returns & credit notes | CRM (customer profiles managed in masterdata) |
| Customer price lists | |
| Sales commission tracking | |
| Sales reports & analytics | |

---

## 2. Business Rules (Regulatory)

### BR-01: E-Invoice Requirement
**Source**: Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP, Circular 32/2025/TT-BTC

- All enterprises **must** issue e-invoices for sales transactions
- E-invoices must be connected to GDT's e-invoice system
- Each e-invoice receives a unique serial + number from GDT
- E-invoices cannot be cancelled; must issue credit notes for corrections
- E-invoice data must be preserved for minimum 10 years (Luật Kế toán 88/2015)
- Transition: Paper invoices may still be used for pre-July 2022 transactions only

### BR-02: Revenue Recognition
**Source**: Thông tư 99/2025/TT-BTC, VAS 14

| Aspect | Requirement |
|--------|-------------|
| **Timing** | Revenue recognized when risk/reward transfers to buyer |
| **Account** | 511 - Doanh thu bán hàng (domestic), 512 - Export revenue |
| **VAT** | Separate calculation on sales price |
| **Discounts** | Deducted from revenue |
| **Returns** | Reverse via credit note (khấu trừ doanh thu) |

### BR-03: GL Account Mapping
**Source**: Thông tư 99/2025/TT-BTC

| Account | Name | Purpose |
|---------|------|---------|
| **511** | Doanh thu bán hàng | Sales revenue (domestic) |
| **512** | Doanh thu bán hàng hóa, dịch vụ | Revenue from goods/services |
| **5111** | Doanh thu bán thành phẩm | Finished goods revenue |
| **5112** | Doanh thu bán ngu一个个物料 | Raw material revenue |
| **5113** | Doanh thu bán dịch vụ | Service revenue |
| **131** | Phải thu khách hàng | Accounts receivable |
| **333** | Thuế GTGT phải nộp | VAT payable |
| **3331** | Thuế GTGT hàng bán | Output VAT |
| **3332** | Thuế GTGT hàng nhập | Input VAT |
| **133** | Thuế GTGT được khấu trừ | Input VAT deductible |
| **632** | Giá vốn hàng bán | Cost of goods sold |
| **6321** | Giá vốn bán thành phẩm | Finished goods COGS |
| **6322** | Giá vốn BDM | Raw material COGS |
| **156** | Hàng hóa | Finished goods inventory |
| **152** | Nguyên vật liệu | Raw material inventory |
| **515** | Doanh thu tài chính | Financial revenue |

### BR-04: Transaction Types
**Source**: Thông tư 99/2025/TT-BTC

| Transaction | GL Entry |
|-------------|----------|
| **Sale (credit)** | Dr 131 / Cr 511, Cr 333 |
| **Sale (cash)** | Dr 111/112 / Cr 511, Cr 333 |
| **Sale (bank transfer)** | Dr 112 / Cr 511, Cr 333 |
| **COGS posting** | Dr 632 / Cr 156 (finished goods), Cr 152 (raw materials) |
| **Return (credit note)** | Dr 511, Dr 333 / Cr 131 |
| **Return (cash refund)** | Dr 511, Dr 333 / Cr 111/112 |
| **Discount** | Dr 511 / Cr 131 (or reduce invoice amount) |
| **Advance payment received** | Dr 111/112 / Cr 1389 |
| **Advance payment offset** | Dr 1389 / Cr 131 |

### BR-05: Price Calculation
**Source**: Thông tư 99/2025/TT-BTC

- **Base Price**: From price list (bảng giá)
- **Discount %**: By customer tier, quantity, promotion
- **Unit Price**: Base Price × (1 - Discount%)
- **Line Total**: Unit Price × Quantity
- **VAT Rate**: 0%, 5%, 8%, 10% (as per tax module)
- **Invoice Total**: Sum(Line Totals) + Total VAT

### BR-06: Delivery Tracking
**Source**: Industry best practice (MISA, Fast, Bravo)

- Each sales order generates delivery request
- Delivery status tracked: pending → in_transit → delivered → completed
- Partial deliveries allowed
- Delivery confirmation triggers COGS posting
- Proof of delivery (POD) record required

### BR-07: Sales Returns (Đổi trả hàng)
**Source**: Thông tư 99/2025/TT-BTC

- Credit note issued against original invoice
- Reason must be recorded (defective, wrong item, customer change, etc.)
- Return quantity cannot exceed original invoice quantity
- GL reversal: Dr 511, Dr 333 / Cr 131 (or Cr 111/112)
- Returned goods re-enter inventory (Dr 156 / Cr 632)

---

## 3. Functional Requirements

### 3.1 Core Entities

#### 3.1.1 Sales Quotation (Báo giá bán hàng)
```go
type SalesQuotation struct {
    ID              string        `json:"id"`
    RefNo           string        `json:"ref_no"`           // Auto: BC-XXXXX
    CustomerCode    string        `json:"customer_code"`
    CustomerName    string        `json:"customer_name"`
    QuotationDate   string        `json:"quotation_date"`
    ValidUntil      string        `json:"valid_until"`
    Status          QuoteStatus   `json:"status"`           // draft, sent, accepted, rejected, expired
    Lines           []QuoteLine   `json:"lines"`
    SubTotal        core.Money    `json:"sub_total"`
    DiscountRate    float64       `json:"discount_rate"`    // 0-100%
    DiscountAmount  core.Money    `json:"discount_amount"`
    VATAmount       core.Money    `json:"vat_amount"`
    TotalAmount     core.Money    `json:"total_amount"`
    Notes           string        `json:"notes,omitempty"`
    ConvertedToOrder bool         `json:"converted_to_order"`
    ConvertedOrderID string       `json:"converted_order_id,omitempty"`
    CreatedBy       string        `json:"created_by"`
    CreatedAt       string        `json:"created_at"`
    UpdatedBy       string        `json:"updated_by"`
    UpdatedAt       string        `json:"updated_at"`
}

type QuoteLine struct {
    LineNo      int         `json:"line_no"`
    ItemCode    string      `json:"item_code"`
    ItemName    string      `json:"item_name"`
    Unit        string      `json:"unit"`
    Quantity    int64       `json:"quantity"`
    UnitPrice   int64       `json:"unit_price"`       // VND
    Discount    float64     `json:"discount"`          // %
    Amount      int64       `json:"amount"`            // quantity * unit_price * (1-discount)
    VATAmount   int64       `json:"vat_amount"`
    TotalAmount int64       `json:"total_amount"`
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

#### 3.1.2 Sales Order (Đơn hàng bán)
```go
type SalesOrder struct {
    ID              string        `json:"id"`
    RefNo           string        `json:"ref_no"`           // Auto: DH-XXXXX
    QuoteID         string        `json:"quote_id,omitempty"` // Source quotation
    CustomerCode    string        `json:"customer_code"`
    CustomerName    string        `json:"customer_name"`
    OrderDate       string        `json:"order_date"`
    DeliveryDate    string        `json:"delivery_date"`
    Status          OrderStatus   `json:"status"`
    Lines           []OrderLine   `json:"lines"`
    SubTotal        core.Money    `json:"sub_total"`
    DiscountRate    float64       `json:"discount_rate"`
    DiscountAmount  core.Money    `json:"discount_amount"`
    VATAmount       core.Money    `json:"vat_amount"`
    TotalAmount     core.Money    `json:"total_amount"`
    DeliveryAddress string        `json:"delivery_address"`
    PaymentTerms    string        `json:"payment_terms"`
    Notes           string        `json:"notes,omitempty"`
    // Delivery tracking
    DeliveryStatus  DeliveryStatus `json:"delivery_status"`
    DeliveredAmount core.Money     `json:"delivered_amount"`
    InvoicedAmount  core.Money     `json:"invoiced_amount"`
    CreatedBy       string         `json:"created_by"`
    CreatedAt       string         `json:"created_at"`
    UpdatedBy       string         `json:"updated_by"`
    UpdatedAt       string         `json:"updated_at"`
}

type OrderLine struct {
    LineNo          int         `json:"line_no"`
    ItemCode        string      `json:"item_code"`
    ItemName        string      `json:"item_name"`
    Unit            string      `json:"unit"`
    Quantity        int64       `json:"quantity"`
    DeliveredQty    int64       `json:"delivered_qty"`     // Partial delivery tracking
    InvoicedQty     int64       `json:"invoiced_qty"`
    UnitPrice       int64       `json:"unit_price"`
    Discount        float64     `json:"discount"`
    Amount          int64       `json:"amount"`
    VATAmount       int64       `json:"vat_amount"`
    TotalAmount     int64       `json:"total_amount"`
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
    DeliveryPending    DeliveryStatus = "pending"
    DeliveryPartial    DeliveryStatus = "partial"
    DeliveryInTransit  DeliveryStatus = "in_transit"
    DeliveryDelivered  DeliveryStatus = "delivered"
    DeliveryCompleted  DeliveryStatus = "completed"
)
```

#### 3.1.3 Sales Invoice (Hóa đơn bán hàng / E-Invoice)
```go
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
    // E-invoice fields
    EInvoiceSerial  string          `json:"e_invoice_serial,omitempty"`
    EInvoiceNumber  string          `json:"e_invoice_number,omitempty"`
    EInvoiceDate    string          `json:"e_invoice_date,omitempty"`
    EInvoiceStatus  EInvoiceStatus  `json:"e_invoice_status"`
    // Payment tracking
    PaidAmount      core.Money      `json:"paid_amount"`
    OutstandingAmt  core.Money      `json:"outstanding_amount"`
    // GL posting
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
    Amount      int64   `json:"amount"`       // quantity * unit_price * (1-discount)
    VATAmount   int64   `json:"vat_amount"`
    TotalAmount int64   `json:"total_amount"`
    COGSAccount string  `json:"cogs_account"` // 6321, 6322
    RevenueAcct string  `json:"revenue_account"` // 5111, 5112, 5113
}

type InvoiceStatus string

const (
    InvoiceDraft     InvoiceStatus = "draft"
    InvoicePending   InvoiceStatus = "pending"     // Awaiting e-invoice
    InvoiceIssued    InvoiceStatus = "issued"       // E-invoice issued
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

#### 3.1.4 Sales Return (Trả hàng bán)
```go
type SalesReturn struct {
    ID              string          `json:"id"`
    RefNo           string          `json:"ref_no"`           // Auto: PH-XXXXX
    InvoiceID       string          `json:"invoice_id"`
    CustomerCode    string          `json:"customer_code"`
    ReturnDate      string          `json:"return_date"`
    Reason          ReturnReason    `json:"reason"`
    Status          ReturnStatus    `json:"status"`
    Lines           []ReturnLine    `json:"lines"`
    SubTotal        core.Money      `json:"sub_total"`
    VATAmount       core.Money      `json:"vat_amount"`
    TotalAmount     core.Money      `json:"total_amount"`
    CreditNoteNo    string          `json:"credit_note_no"`   // E-invoice credit note
    GLPosted        bool            `json:"gl_posted"`
    GLReference     string          `json:"gl_reference,omitempty"`
    Notes           string          `json:"notes,omitempty"`
    CreatedBy       string          `json:"created_by"`
    CreatedAt       string          `json:"created_at"`
    UpdatedBy       string          `json:"updated_by"`
    UpdatedAt       string          `json:"updated_at"`
}

type ReturnLine struct {
    LineNo      int     `json:"line_no"`
    ItemCode    string  `json:"item_code"`
    ItemName    string  `json:"item_name"`
    Unit        string  `json:"unit"`
    Quantity    int64   `json:"quantity"`
    UnitPrice   int64   `json:"unit_price"`
    Amount      int64   `json:"amount"`
    VATAmount   int64   `json:"vat_amount"`
    TotalAmount int64   `json:"total_amount"`
}

type ReturnReason string

const (
    ReturnDefective  ReturnReason = "defective"
    ReturnWrongItem  ReturnReason = "wrong_item"
    ReturnDamaged    ReturnReason = "damaged"
    ReturnCustomer   ReturnReason = "customer_request"
    ReturnExpired    ReturnReason = "expired"
)

type ReturnStatus string

const (
    ReturnDraft    ReturnStatus = "draft"
    ReturnApproved ReturnStatus = "approved"
    ReturnReceived ReturnStatus = "received"
    ReturnIssued   ReturnStatus = "credit_note_issued"
    ReturnCompleted ReturnStatus = "completed"
)
```

### 3.2 Functional Requirements Matrix

| ID | Requirement | Priority | Current State |
|----|-------------|----------|---------------|
| FR-01 | Sales quotation management | P1 | ❌ Not implemented |
| FR-02 | Sales order management | P0 | ❌ Not implemented |
| FR-03 | Sales invoice creation | P0 | ⚠️ Stub only (7 fields) |
| FR-04 | E-invoice integration with GDT | P0 | ❌ Not implemented |
| FR-05 | Sales return & credit note | P0 | ❌ Not implemented |
| FR-06 | Revenue GL posting (511/512) | P0 | ❌ Not implemented |
| FR-07 | COGS posting (632) | P0 | ❌ Not implemented |
| FR-08 | VAT calculation on sales | P0 | ❌ Not implemented |
| FR-09 | Delivery tracking | P1 | ❌ Not implemented |
| FR-10 | Customer price lists | P1 | ❌ Not implemented |
| FR-11 | Sales commission tracking | P2 | ❌ Not implemented |
| FR-12 | Sales reports & analytics | P1 | ❌ Not implemented |
| FR-13 | Customer balance inquiry | P1 | ❌ Not implemented |
| FR-14 | Multi-currency support | P2 | ❌ Not implemented |
| FR-15 | Print invoice / e-invoice PDF | P1 | ❌ Not implemented |

---

## 4. Non-Functional Requirements

### 4.1 Compliance
- **NFR-01**: Must comply with Thông tư 99/2025/TT-BTC
- **NFR-02**: Must support e-invoicing per Decree 123/2020 & Decree 70/2025
- **NFR-03**: Must maintain audit trail for regulatory compliance
- **NFR-04**: Must preserve all accounting data for 10 years minimum
- **NFR-05**: Must support Vietnamese language interfaces

### 4.2 Performance
- **NFR-06**: Invoice creation < 500ms
- **NFR-07**: E-invoice submission < 3s (GDT API dependent)
- **NFR-08**: Support concurrent users (minimum 50)

### 4.3 Data Integrity
- **NFR-09**: All transactions must be atomic
- **NFR-10**: Invoice totals must equal sum of line items
- **NFR-11**: Prevent deletion of issued invoices
- **NFR-12**: Prevent double-invoicing of same order

---

## 5. Integration Requirements

### 5.1 GL Integration
- Post revenue to Account 511/512 on invoice issuance
- Post COGS to Account 632 on delivery confirmation
- Post VAT to Account 333 on invoice issuance
- Post receivables to Account 131 on invoice issuance
- Monthly reconciliation with GL

### 5.2 E-Invoice Integration
- Connect to GDT e-invoice system (Circular 32/2025/TT-BTC)
- Submit invoice data for validation
- Receive serial + number from GDT
- Store e-invoice response
- Support credit note issuance
- Preserve e-invoice data for 10 years

### 5.3 Inventory Integration
- Query stock availability before order confirmation
- Trigger stock deduction on delivery
- Return goods re-enter inventory on credit note

### 5.4 Master Data Integration
- Customer information from masterdata module
- Item/product catalog from masterdata module
- Price lists from masterdata module
- VAT rates from tax module

### 5.5 Cash/Bank Integration
- Record payment receipts
- Link payments to invoices
- Support advance payment handling

---

## 6. Assumptions & Constraints

### 6.1 Assumptions
1. Customer master data exists in masterdata module
2. Item/product catalog exists in masterdata module
3. VAT calculation logic exists in tax module
4. GL posting mechanism exists in ledger module
5. Users have basic accounting knowledge

### 6.2 Constraints
1. Must use existing SQLite database infrastructure
2. Must follow existing 4-layer architecture pattern
3. Must integrate with existing Casbin authorization
4. Must comply with Decree 123/2020/NĐ-CP (e-invoicing)
5. Must comply with Thông tư 99/2025/TT-BTC (accounting)

---

## 7. Success Criteria

| Criterion | Metric |
|-----------|--------|
| Accounting compliance | 100% compliant with Thông tư 99/2025/TT-BTC |
| E-invoice compliance | 100% compliant with Decree 123/2020 & Decree 70/2025 |
| GL reconciliation | Account 511/512 balance matches invoice totals |
| Transaction accuracy | Zero posting errors |
| User adoption | All sales staff use the system |
| Audit readiness | All required reports available |

---

## 8. Recommendations

### 8.1 Immediate Actions (P0)
1. **Enhance SalesInvoice entity** with e-invoice fields, customer details, delivery tracking
2. **Implement SalesOrder entity** with full order lifecycle
3. **Add e-invoice integration** with GDT system
4. **Add GL posting** for revenue (511/512), COGS (632), VAT (333), receivables (131)

### 8.2 Short-term Actions (P1)
1. **Implement Sales Return** with credit note issuance
2. **Implement Sales Quotation** with order conversion
3. **Add delivery tracking** with partial delivery support
4. **Add sales reports** (revenue by customer, item, period)

### 8.3 Long-term Actions (P2)
1. **Sales commission tracking**
2. **Multi-currency support**
3. **Advanced analytics & BI integration**
4. **CRM integration** for customer lifecycle

---

## Appendix A: Reference Documents

1. **Thông tư 99/2025/TT-BTC** - Hệ thống tài khoản kế toán doanh nghiệp
2. **Decree 123/2020/NĐ-CP** - Hoá đơn điện tử
3. **Decree 70/2025/NĐ-CP** - Sửa đổi Decree 123
4. **Circular 32/2025/TT-BTC** - Hướng dẫn e-invoice
5. **VAS 14** - Doanh thu và các khoản giảm trừ doanh thu
6. **Luật Kế toán 88/2015** - Luật kế toán Việt Nam

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
