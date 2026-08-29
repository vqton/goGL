# Implementation Roadmap - Sales Module
## Bán hàng (Sales)

**Version**: 1.0
**Date**: 2026-08-29
**Module**: Sales
**Compliance**: Thông tư 99/2025/TT-BTC, Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP, Circular 32/2025/TT-BTC

---

## Executive Summary

This roadmap outlines the implementation of the Sales Module to comply with Vietnamese accounting standards and e-invoicing regulations. The module manages the complete order-to-cash cycle including quotations, orders, deliveries, invoicing, returns, and revenue recognition.

**Total Duration**: 10 weeks
**Priority**: P0 (Accounting compliance + E-invoicing)

---

## Phase 1: Core Entity Enhancement (Week 1-2)
**Goal**: Enhance SalesInvoice entity and create Order/Quotation/Return entities

### Tasks
- [ ] **T1.1**: Enhance SalesInvoice entity with full fields:
  - Customer details (name, address, tax code)
  - E-invoice fields (serial, number, date, status)
  - Payment tracking (paid, outstanding)
  - GL posting fields (gl_posted, gl_reference)
  - Due date
- [ ] **T1.2**: Create SalesQuotation entity with QuoteLine
- [ ] **T1.3**: Create SalesOrder entity with OrderLine and delivery tracking
- [ ] **T1.4**: Create SalesReturn entity with ReturnLine
- [ ] **T1.5**: Define all status enums (QuoteStatus, OrderStatus, InvoiceStatus, etc.)
- [ ] **T1.6**: Update Repository interface with full CRUD for all entities
- [ ] **T1.7**: Implement SQLite repository for all entities
- [ ] **T1.8**: Add database migrations for new tables
- [ ] **T1.9**: Write unit tests for entity validation

### Acceptance Criteria
- All 4 entities fully defined
- Repository interface supports all operations
- SQLite implementation complete
- Database migrations created
- Unit tests pass

### Files to Modify/Create
- `internal/domain/sales/entity.go` (MODIFY: enhance SalesInvoice)
- `internal/domain/sales/quotation.go` (CREATE)
- `internal/domain/sales/order.go` (CREATE)
- `internal/domain/sales/return.go` (CREATE)
- `internal/infrastructure/persistence/sales/repository.go` (MODIFY)
- `internal/infrastructure/db/migrate.go` (MODIFY)

---

## Phase 2: Service Layer (Week 3-4)
**Goal**: Implement business logic for all sales operations

### Tasks
- [ ] **T2.1**: Implement Quotation service:
  - CreateQuotation, GetQuotation, UpdateQuotation, DeleteQuotation
  - ListQuotations with filters
  - SendQuotation, AcceptQuotation, RejectQuotation
  - Auto-generate quotation code (BC-XXXXX)
- [ ] **T2.2**: Implement Order service:
  - CreateOrder, GetOrder, UpdateOrder, DeleteOrder
  - ListOrders with filters
  - ConfirmOrder, CancelOrder
  - Auto-generate order code (DH-XXXXX)
- [ ] **T2.3**: Implement Invoice service:
  - CreateInvoice, GetInvoice, UpdateInvoice
  - ListInvoices with filters
  - IssueInvoice (e-invoice submission)
  - VoidInvoice (with credit note)
  - Auto-generate invoice code (HD-XXXXX)
- [ ] **T2.4**: Implement Return service:
  - CreateReturn, GetReturn
  - ListReturns with filters
  - ApproveReturn, ReceiveReturn
  - Auto-generate return code (PH-XXXXX)
- [ ] **T2.5**: Implement Delivery service:
  - DeliverOrder (partial delivery support)
  - Update delivery status
  - Update delivered quantities
- [ ] **T2.6**: Add validation functions for all entities
- [ ] **T2.7**: Implement price calculation logic:
  - Line total = quantity × unit price × (1 - discount%)
  - Subtotal = sum of line totals
  - VAT = subtotal × VAT rate
  - Total = subtotal + VAT
- [ ] **T2.8**: Write unit tests for all services

### Acceptance Criteria
- All CRUD operations work
- Business logic validates correctly
- Price calculations accurate
- Auto-code generation works
- Unit tests pass (coverage > 80%)

### Files to Modify/Create
- `internal/application/sales/service.go` (MODIFY: implement all methods)
- `internal/application/sales/quotation.go` (CREATE)
- `internal/application/sales/order.go` (CREATE)
- `internal/application/sales/return.go` (CREATE)
- `internal/application/sales/invoice.go` (CREATE)
- `internal/application/sales/delivery.go` (CREATE)
- `internal/application/sales/service_test.go` (CREATE)

---

## Phase 3: GL Integration (Week 5)
**Goal**: Implement GL posting for all sales transactions

### Tasks
- [ ] **T3.1**: Implement revenue posting (511/512):
  - Post on invoice issuance
  - Post credit note on return
  - Post void reversal
- [ ] **T3.2**: Implement COGS posting (632):
  - Post on delivery confirmation
  - Post reversal on return
- [ ] **T3.3**: Implement VAT posting (333):
  - Post output VAT on invoice
  - Reverse VAT on credit note
- [ ] **T3.4**: Implement receivables posting (131):
  - Post on invoice issuance
  - Reverse on credit note
- [ ] **T3.5**: Implement advance payment handling:
  - Post advance received (Dr 111/112 / Cr 1389)
  - Post advance offset (Dr 1389 / Cr 131)
- [ ] **T3.6**: Add GL posting status tracking
- [ ] **T3.7**: Write integration tests for GL posting

### Acceptance Criteria
- All GL entries posted correctly
- Debits equal credits for all transactions
- GL posting status tracked
- Integration tests pass

### Files to Modify/Create
- `internal/application/sales/gl_posting.go` (CREATE)
- `internal/application/sales/service_test.go` (MODIFY)

---

## Phase 4: HTTP Handlers (Week 6)
**Goal**: Implement REST API endpoints

### Tasks
- [ ] **T4.1**: Implement Quotation endpoints:
  - POST `/api/v1/sales/quotations`
  - GET `/api/v1/sales/quotations`
  - GET `/api/v1/sales/quotations/:id`
  - PUT `/api/v1/sales/quotations/:id`
  - DELETE `/api/v1/sales/quotations/:id`
  - POST `/api/v1/sales/quotations/:id/send`
  - POST `/api/v1/sales/quotations/:id/accept`
  - POST `/api/v1/sales/quotations/:id/reject`
- [ ] **T4.2**: Implement Order endpoints:
  - POST `/api/v1/sales/orders`
  - GET `/api/v1/sales/orders`
  - GET `/api/v1/sales/orders/:id`
  - PUT `/api/v1/sales/orders/:id`
  - DELETE `/api/v1/sales/orders/:id`
  - POST `/api/v1/sales/orders/:id/confirm`
  - POST `/api/v1/sales/orders/:id/cancel`
  - POST `/api/v1/sales/orders/:id/deliver`
- [ ] **T4.3**: Implement Invoice endpoints:
  - POST `/api/v1/sales/invoices`
  - GET `/api/v1/sales/invoices`
  - GET `/api/v1/sales/invoices/:id`
  - PUT `/api/v1/sales/invoices/:id`
  - POST `/api/v1/sales/invoices/:id/issue`
  - POST `/api/v1/sales/invoices/:id/void`
- [ ] **T4.4**: Implement Return endpoints:
  - POST `/api/v1/sales/returns`
  - GET `/api/v1/sales/returns`
  - GET `/api/v1/sales/returns/:id`
  - POST `/api/v1/sales/returns/:id/approve`
  - POST `/api/v1/sales/returns/:id/receive`
- [ ] **T4.5**: Implement Customer endpoints:
  - GET `/api/v1/sales/customers/:code/balance`
  - GET `/api/v1/sales/customers/:code/outstanding`
- [ ] **T4.6**: Add request validation middleware
- [ ] **T4.7**: Add error handling
- [ ] **T4.8**: Write API tests

### Acceptance Criteria
- All API endpoints implemented
- Request validation works
- Error handling returns proper HTTP codes
- API tests pass

### Files to Modify/Create
- `internal/interfaces/http/sales/handler.go` (MODIFY)
- `internal/interfaces/http/sales/quotation.go` (CREATE)
- `internal/interfaces/http/sales/order.go` (CREATE)
- `internal/interfaces/http/sales/invoice.go` (CREATE)
- `internal/interfaces/http/sales/return.go` (CREATE)

---

## Phase 5: E-Invoice Integration (Week 7)
**Goal**: Implement GDT e-invoice system integration

### Tasks
- [ ] **T5.1**: Implement GDT API client:
  - Connect to GDT e-invoice system
  - Handle authentication
  - Handle request/response format
- [ ] **T5.2**: Implement e-invoice submission:
  - Submit invoice data to GDT
  - Handle GDT validation
  - Receive serial + number
- [ ] **T5.3**: Implement e-invoice response handling:
  - Parse GDT response
  - Store serial + number
  - Update invoice status
- [ ] **T5.4**: Implement credit note issuance:
  - Submit credit note to GDT
  - Receive credit note serial + number
  - Link to original invoice
- [ ] **T5.5**: Implement e-invoice status tracking:
  - Track submission status
  - Handle GDT rejection
  - Retry logic
- [ ] **T5.6**: Add e-invoice data preservation:
  - Store for 10 years minimum
  - Audit trail
- [ ] **T5.7**: Write integration tests

### Acceptance Criteria
- GDT connection works
- E-invoices submitted successfully
- Serial + number received
- Credit notes work
- Integration tests pass

### Files to Modify/Create
- `internal/infrastructure/einvoice/gdt.go` (CREATE)
- `internal/infrastructure/einvoice/types.go` (CREATE)
- `internal/application/sales/einvoice.go` (CREATE)

---

## Phase 6: Web UI (Week 8-9)
**Goal**: Implement user interface for sales management

### Tasks
- [ ] **T6.1**: Create Sales Invoice List page:
  - Table with columns: Ref No, Customer, Date, Total, Status, Actions
  - Filters: Customer, Status, Date Range
  - Pagination
  - Export to Excel
- [ ] **T6.2**: Create Sales Invoice Detail page:
  - General information section
  - Line items table
  - Totals section (subtotal, VAT, total)
  - E-invoice information
  - GL posting details
  - Action buttons (Issue, Void, Return, Print)
- [ ] **T6.3**: Create Create Invoice form:
  - Customer selection
  - Line items editor
  - VAT calculation
  - GL entry preview
  - Save Draft / Save & Issue buttons
- [ ] **T6.4**: Create Order List & Detail pages
- [ ] **T6.5**: Create Quotation List & Detail pages
- [ ] **T6.6**: Create Return List & Detail pages
- [ ] **T6.7**: Create Customer Balance page
- [ ] **T6.8**: Add responsive design
- [ ] **T6.9**: Add loading states and error handling
- [ ] **T6.10**: Add print invoice functionality

### Acceptance Criteria
- All pages implemented
- Forms validate input
- Responsive design works
- Error states handled
- Print functionality works

### Files to Create
- `web/templates/sales/invoice_list.html`
- `web/templates/sales/invoice_detail.html`
- `web/templates/sales/invoice_create.html`
- `web/templates/sales/order_list.html`
- `web/templates/sales/order_detail.html`
- `web/templates/sales/quotation_list.html`
- `web/templates/sales/quotation_detail.html`
- `web/templates/sales/return_list.html`
- `web/templates/sales/return_detail.html`
- `web/templates/sales/customer_balance.html`

---

## Phase 7: Reports & Polish (Week 10)
**Goal**: Implement reporting capabilities and final polish

### Tasks
- [ ] **T7.1**: Implement Revenue Report:
  - By customer
  - By item
  - By period (day/week/month/quarter/year)
- [ ] **T7.2**: Implement Outstanding Receivables Report:
  - By customer
  - By aging (current, 30, 60, 90+ days)
- [ ] **T7.3**: Implement E-Invoice Status Report:
  - Submitted, accepted, rejected
  - By date range
- [ ] **T7.4**: Implement Sales Performance Report:
  - Top customers
  - Top items
  - Sales trends
- [ ] **T7.5**: Add export to Excel
- [ ] **T7.6**: Add export to PDF
- [ ] **T7.7**: Add audit trail for all mutations
- [ ] **T7.8**: Add authorization checks (Casbin)
- [ ] **T7.9**: Performance optimization:
  - Add indexes for common queries
  - Optimize list queries
- [ ] **T7.10**: Documentation:
  - API documentation
  - User guide
  - Developer guide

### Acceptance Criteria
- All reports implemented
- Export to Excel/PDF works
- Reports are accurate
- Audit trail works
- Authorization enforced
- Performance meets NFRs
- Documentation complete

### Files to Create/Modify
- `internal/application/sales/report.go` (CREATE)
- `internal/interfaces/http/sales/report.go` (CREATE)

---

## Dependency Graph

```
Phase 1 (Entity) ──▶ Phase 2 (Service) ──▶ Phase 3 (GL) ──▶ Phase 4 (HTTP)
                          │                      │                │
                          ▼                      ▼                ▼
                    Phase 5 (E-Invoice) ◀───────┘                │
                          │                                      │
                          ▼                                      ▼
                    Phase 6 (Web UI) ◀───────────────────────────┘
                          │
                          ▼
                    Phase 7 (Reports)
```

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| GDT API complexity | HIGH | Start with mock, integrate early |
| E-invoice compliance changes | MEDIUM | Monitor Decree updates |
| Performance with large datasets | MEDIUM | Add pagination, indexes |
| User adoption | MEDIUM | Provide training, intuitive UI |
| Regulatory changes | LOW | Monitor Thông tư updates |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Accounting compliance | 100% compliant with TT99 |
| E-invoice compliance | 100% compliant with Decree 123/70 |
| GL reconciliation | Account 511/512 balance matches |
| Transaction accuracy | Zero posting errors |
| Test coverage | > 80% |
| API response time | < 500ms |
| User adoption | All sales staff use system |

---

## Appendix: Technical Notes

### Database Schema Changes
```sql
-- Enhanced sales_invoices table
CREATE TABLE IF NOT EXISTS sales_invoices (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Sales orders table
CREATE TABLE IF NOT EXISTS sales_orders (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Sales quotations table
CREATE TABLE IF NOT EXISTS sales_quotations (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Sales returns table
CREATE TABLE IF NOT EXISTS sales_returns (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_invoice_customer ON sales_invoices(json_extract(data, '$.customer_code'));
CREATE INDEX IF NOT EXISTS idx_invoice_status ON sales_invoices(json_extract(data, '$.status'));
CREATE INDEX IF NOT EXISTS idx_invoice_date ON sales_invoices(json_extract(data, '$.invoice_date'));
CREATE INDEX IF NOT EXISTS idx_order_customer ON sales_orders(json_extract(data, '$.customer_code'));
CREATE INDEX IF NOT EXISTS idx_order_status ON sales_orders(json_extract(data, '$.status'));
CREATE INDEX IF NOT EXISTS idx_quotation_customer ON sales_quotations(json_extract(data, '$.customer_code'));
CREATE INDEX IF NOT EXISTS idx_return_customer ON sales_returns(json_extract(data, '$.customer_code'));
```

### GL Posting Pattern
```go
// Example: Invoice issuance posting
func postInvoiceGL(ctx context.Context, ledgerSvc LedgerService, inv *SalesInvoice) error {
    // Dr 131 - Receivable
    ledgerSvc.Post(ctx, &Posting{
        Account: "131",
        Debit:   inv.TotalAmount.AmountMinor,
        Credit:  0,
        Ref:     inv.ID,
    })

    // Cr 511 - Revenue
    ledgerSvc.Post(ctx, &Posting{
        Account: "511",
        Debit:   0,
        Credit:  inv.SubTotal.AmountMinor,
        Ref:     inv.ID,
    })

    // Cr 333 - VAT Payable
    ledgerSvc.Post(ctx, &Posting{
        Account: "333",
        Debit:   0,
        Credit:  inv.VATAmount.AmountMinor,
        Ref:     inv.ID,
    })

    return nil
}
```

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
