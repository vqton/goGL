# Implementation Roadmap - Tools/Equipment Module
## Công cụ, dụng cụ (CCDC)

**Version**: 1.0  
**Date**: 2026-08-29  
**Module**: Tools/Equipment  
**Compliance**: Thông tư 99/2025/TT-BTC

---

## Executive Summary

This roadmap outlines the implementation of the Tools/Equipment (CCDC) module to comply with Vietnamese accounting standards (Thông tư 99/2025/TT-BTC). The module will track tools/equipment below the 30M VND threshold with proper GL integration to Account 153.

**Total Duration**: 7 weeks  
**Priority**: P0 (Accounting compliance)

---

## Phase 1: Core Entity Enhancement (Week 1)
**Goal**: Update ToolCard entity with GL account fields and validation

### Tasks
- [ ] **T1.1**: Update ToolCard entity with new fields:
  - `AccountCode153` (string) - Account 153 detail
  - `AccountCodeExp` (string) - Expense account (623/627/641/642)
  - `Warehouse` (string) - Warehouse location
  - `SubCategory` (string) - Sub-category
  - `Unit` (string) - Unit of measure
- [ ] **T1.2**: Add validation for < 30M VND threshold
- [ ] **T1.3**: Update `ValidateToolCard` function
- [ ] **T1.4**: Update Repository interface with new fields
- [ ] **T1.5**: Update SQLite repository implementation
- [ ] **T1.6**: Write unit tests for validation
- [ ] **T1.7**: Update database migration if needed

### Acceptance Criteria
- ToolCard entity supports GL account fields
- Validation enforces < 30M VND threshold
- All existing tests pass
- New unit tests for validation

### Files to Modify
- `internal/domain/tools/entity.go`
- `internal/infrastructure/persistence/tools/repository.go`
- `internal/application/tools/service_test.go`

---

## Phase 2: Transaction Entity (Week 2)
**Goal**: Implement transaction tracking for all inventory movements

### Tasks
- [ ] **T2.1**: Create ToolTransaction entity
- [ ] **T2.2**: Define TransactionType constants (import, export, transfer, return, disposal, adjustment)
- [ ] **T2.3**: Add transaction methods to Repository interface:
  - `CreateTransaction(ctx, tx) error`
  - `FindTransactionByID(ctx, id) (*ToolTransaction, error)`
  - `ListTransactions(ctx, toolCardID, txType) ([]*ToolTransaction, error)`
- [ ] **T2.4**: Implement SQLite transaction repository
- [ ] **T2.5**: Create `tool_transactions` table migration
- [ ] **T2.6**: Write unit tests for transaction operations

### Acceptance Criteria
- ToolTransaction entity created
- Transaction CRUD operations work
- Database migration creates table
- Unit tests pass

### Files to Create/Modify
- `internal/domain/tools/transaction.go` (new)
- `internal/infrastructure/persistence/tools/repository.go`
- `internal/infrastructure/db/migrate.go`

---

## Phase 3: Import/Export Services (Week 3)
**Goal**: Implement core inventory operations with GL posting

### Tasks
- [ ] **T3.1**: Implement `Import` service method:
  - Validate tool card exists
  - Create import transaction
  - Post GL entry: Dr 153, Dr 133 / Cr 331/111/112
  - Update tool card quantity
- [ ] **T3.2**: Implement `Export` service method:
  - Validate stock availability
  - Create export transaction
  - Post GL entry: Dr 623/627/641/642 / Cr 153
  - Update tool card quantity
- [ ] **T3.3**: Implement `Transfer` service method:
  - Validate stock availability
  - Create transfer transaction
  - Update location information
- [ ] **T3.4**: Implement `Return` service method:
  - Validate stock availability
  - Create return transaction
  - Post GL entry: Dr 331/111/112 / Cr 153, Cr 133
  - Update tool card quantity
- [ ] **T3.5**: Implement `Dispose` service method:
  - Validate stock availability
  - Create disposal transaction
  - Post GL entry: Dr 632 / Cr 153; Dr 111/131 / Cr 511
  - Update tool card state
- [ ] **T3.6**: Write unit tests for all operations
- [ ] **T3.7**: Add GL posting integration

### Acceptance Criteria
- Import/Export/Transfer/Return/Dispose operations work
- GL entries posted correctly
- Stock quantities updated
- Unit tests pass

### Files to Modify
- `internal/application/tools/service.go`
- `internal/application/tools/service_test.go`

---

## Phase 4: HTTP Handlers (Week 4)
**Goal**: Implement REST API endpoints

### Tasks
- [ ] **T4.1**: Implement Import endpoint:
  - `POST /api/v1/tools/cards/:id/import`
  - Request body: quantity, unit_price, reference_no, notes
  - Response: ToolTransaction
- [ ] **T4.2**: Implement Export endpoint:
  - `POST /api/v1/tools/cards/:id/export`
  - Request body: quantity, to_department, to_person, notes
  - Response: ToolTransaction
- [ ] **T4.3**: Implement Transfer endpoint:
  - `POST /api/v1/tools/cards/:id/transfer`
  - Request body: quantity, to_location, to_department, notes
  - Response: ToolTransaction
- [ ] **T4.4**: Implement Return endpoint:
  - `POST /api/v1/tools/cards/:id/return`
  - Request body: quantity, reason, reference_no, notes
  - Response: ToolTransaction
- [ ] **T4.5**: Implement Dispose endpoint:
  - `POST /api/v1/tools/cards/:id/dispose`
  - Request body: quantity, disposal_type, sale_price, reason, notes
  - Response: ToolTransaction
- [ ] **T4.6**: Implement GetStock endpoint:
  - `GET /api/v1/tools/cards/:id/stock`
  - Response: { stock: int }
- [ ] **T4.7**: Implement ListTransactions endpoint:
  - `GET /api/v1/tools/cards/:id/transactions`
  - Query params: type, limit, offset
  - Response: []ToolTransaction
- [ ] **T4.8**: Add request validation
- [ ] **T4.9**: Add error handling
- [ ] **T4.10**: Write API tests

### Acceptance Criteria
- All API endpoints implemented
- Request validation works
- Error handling returns proper HTTP codes
- API tests pass

### Files to Modify
- `internal/interfaces/http/tools/handler.go`

---

## Phase 5: Web UI (Week 5-6)
**Goal**: Implement user interface for tool management

### Tasks
- [ ] **T5.1**: Create Tool Card List page:
  - Table with columns: Code, Name, Category, Qty, State, Actions
  - Filters: Category, State, Search
  - Pagination
- [ ] **T5.2**: Create Tool Card Detail page:
  - General information section
  - Location & assignment section
  - GL accounts section
  - Recent transactions list
  - Action buttons (Import, Export, Transfer, Return, Dispose)
- [ ] **T5.3**: Create Import form:
  - Quantity, Unit Price, Reference No, Notes
  - GL entry preview
- [ ] **T5.4**: Create Export form:
  - Quantity, To Department, To Person, Notes
  - Stock validation
- [ ] **T5.5**: Create Transfer form:
  - Quantity, To Location, To Department, Notes
- [ ] **T5.6**: Create Return form:
  - Quantity, Reason, Reference No, Notes
- [ ] **T5.7**: Create Dispose form:
  - Quantity, Disposal Type, Sale Price, Reason, Notes
- [ ] **T5.8**: Create Transaction History page:
  - List with filters
  - Export to Excel
- [ ] **T5.9**: Add responsive design
- [ ] **T5.10**: Add loading states and error handling

### Acceptance Criteria
- All pages implemented
- Forms validate input
- Responsive design works
- Error states handled

### Files to Create
- `web/templates/tools/list.html`
- `web/templates/tools/detail.html`
- `web/templates/tools/import.html`
- `web/templates/tools/export.html`
- `web/templates/tools/transfer.html`
- `web/templates/tools/return.html`
- `web/templates/tools/dispose.html`
- `web/templates/tools/transactions.html`

---

## Phase 6: Reports (Week 7)
**Goal**: Implement reporting capabilities

### Tasks
- [ ] **T6.1**: Implement Inventory List report:
  - Filter by category, state, warehouse
  - Show: Code, Name, Qty, Value, Location
- [ ] **T6.2**: Implement Transaction Log report:
  - Filter by date range, type
  - Show: Date, Type, Tool, Qty, Amount, Reference
- [ ] **T6.3**: Implement GL Summary report:
  - Show Account 153 balance
  - Show transaction totals by account
- [ ] **T6.4**: Implement Stock Balance report:
  - Show current stock by tool card
  - Show stock value
- [ ] **T6.5**: Add export to Excel
- [ ] **T6.6**: Add export to PDF
- [ ] **T6.7**: Write report tests

### Acceptance Criteria
- All reports implemented
- Export to Excel/PDF works
- Reports are accurate
- Tests pass

### Files to Create/Modify
- `internal/application/tools/report.go` (new)
- `internal/interfaces/http/tools/handler.go`

---

## Phase 7: Audit Trail & Polish (Week 8)
**Goal**: Add audit trail and final polish

### Tasks
- [ ] **T7.1**: Add audit trail for all mutations:
  - Track who, what, when
  - Store old/new values
- [ ] **T7.2**: Add authorization checks:
  - Role-based access control
  - Casbin integration
- [ ] **T7.3**: Add input sanitization
- [ ] **T7.4**: Performance optimization:
  - Add indexes for common queries
  - Optimize list queries
- [ ] **T7.5**: Documentation:
  - API documentation
  - User guide
  - Developer guide
- [ ] **T7.6**: Final testing:
  - Integration tests
  - User acceptance testing

### Acceptance Criteria
- Audit trail works
- Authorization enforced
- Performance meets NFRs
- Documentation complete

### Files to Modify
- `internal/application/tools/service.go`
- `internal/interfaces/http/tools/handler.go`

---

## Dependency Graph

```
Phase 1 (Entity) ──▶ Phase 2 (Transaction) ──▶ Phase 3 (Services) ──▶ Phase 4 (HTTP)
                                    │                                        │
                                    ▼                                        ▼
                              Phase 5 (Web UI) ◀─────────────────────────────┘
                                    │
                                    ▼
                              Phase 6 (Reports)
                                    │
                                    ▼
                              Phase 7 (Polish)
```

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| GL integration complexity | HIGH | Use existing ledger module patterns |
| Performance with large datasets | MEDIUM | Add pagination, indexes |
| User adoption | MEDIUM | Provide training, intuitive UI |
| Regulatory changes | LOW | Monitor Thông tư updates |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Accounting compliance | 100% compliant with TT99 |
| GL reconciliation | Account 153 balance matches |
| Transaction accuracy | Zero posting errors |
| Test coverage | > 80% |
| API response time | < 500ms |
| User adoption | All inventory staff use system |

---

## Appendix: Technical Notes

### Database Schema Changes
```sql
-- New table for transactions
CREATE TABLE IF NOT EXISTS tool_transactions (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_tool_tx_card ON tool_transactions(json_extract(data, '$.tool_card_id'));
CREATE INDEX IF NOT EXISTS idx_tool_tx_type ON tool_transactions(json_extract(data, '$.transaction_type'));
CREATE INDEX IF NOT EXISTS idx_tool_tx_date ON tool_transactions(json_extract(data, '$.created_at'));
```

### GL Posting Pattern
```go
// Example: Import posting
func postImportGL(ctx context.Context, ledgerSvc LedgerService, tx *ToolTransaction) error {
    // Dr 153 - Tools
    ledgerSvc.Post(ctx, &Posting{
        Account: "153",
        Debit:   tx.TotalAmount,
        Credit:  0,
        Ref:     tx.ID,
    })
    
    // Dr 133 - VAT (if applicable)
    if tx.VATAmount > 0 {
        ledgerSvc.Post(ctx, &Posting{
            Account: "133",
            Debit:   tx.VATAmount,
            Credit:  0,
            Ref:     tx.ID,
        })
    }
    
    // Cr 331 - AP (or 111/112 for cash)
    ledgerSvc.Post(ctx, &Posting{
        Account: "331",
        Debit:   0,
        Credit:  tx.TotalAmount + tx.VATAmount,
        Ref:     tx.ID,
    })
    
    return nil
}
```

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
