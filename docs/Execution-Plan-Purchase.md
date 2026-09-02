# Execution Plan: Purchase Module

## Version History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-08-30 | BA Lead + Chief Accountant | Initial execution plan |

---

## Overview

| Metric | Value |
|--------|-------|
| **Total Duration** | 12 weeks |
| **Total Effort** | 60 person-days |
| **Team Size** | 1 developer |
| **Daily Capacity** | 6-8 hours |

---

## Week 1: Supplier Entity & Repository

### Day 1 (Monday): Supplier Entity

**Tasks:**
- [ ] Create `internal/domain/purchase/supplier.go`
- [ ] Define Supplier struct with 15+ fields
- [ ] Define SupplierStatus type (active/inactive)
- [ ] Add validation logic
- [ ] Add Clone() method

**Files to Create:**
```go
// internal/domain/purchase/supplier.go
type Supplier struct {
    ID              string       `json:"id"`
    RefNo           string       `json:"ref_no"`        // NCC-00001
    Name            string       `json:"name"`
    TaxCode         string       `json:"tax_code"`
    Address         string       `json:"address"`
    Phone           string       `json:"phone"`
    Email           string       `json:"email"`
    ContactPerson   string       `json:"contact_person"`
    PaymentTerms    string       `json:"payment_terms"`
    CreditLimit     core.Money   `json:"credit_limit"`
    BankAccount     string       `json:"bank_account"`
    BankName        string       `json:"bank_name"`
    Status          SupplierStatus `json:"status"`
    CreatedBy       string       `json:"created_by"`
    CreatedAt       string       `json:"created_at"`
    UpdatedBy       string       `json:"updated_by"`
    UpdatedAt       string       `json:"updated_at"`
}
```

**Tests to Write:**
- TestValidateSupplier_Success
- TestValidateSupplier_EmptyName
- TestValidateSupplier_EmptyTaxCode
- TestSupplierStatus_IsValid
- TestCloneSupplier

**Estimated Time:** 6 hours

---

### Day 2 (Tuesday): Supplier Repository Interface

**Tasks:**
- [ ] Add Supplier methods to Repository interface in `entity.go`
- [ ] Define CRUD methods
- [ ] Define NextSupplierNo method

**Methods to Add:**
```go
// Repository interface additions
CreateSupplier(ctx context.Context, s *Supplier) error
FindSupplierByID(ctx context.Context, id string) (*Supplier, error)
UpdateSupplier(ctx context.Context, s *Supplier) error
DeleteSupplier(ctx context.Context, id string) error
ListSuppliers(ctx context.Context, name string, status SupplierStatus, limit, offset int) ([]*Supplier, error)
NextSupplierNo(ctx context.Context) (int64, error)
```

**Estimated Time:** 4 hours

---

### Day 3 (Wednesday): Supplier SQLite Repository

**Tasks:**
- [ ] Create `internal/infrastructure/persistence/supplier/repository.go`
- [ ] Implement CreateSupplier
- [ ] Implement FindSupplierByID
- [ ] Implement UpdateSupplier
- [ ] Implement DeleteSupplier
- [ ] Implement ListSuppliers with filters
- [ ] Implement NextSupplierNo with sequence

**Repository Pattern:**
```go
func (r *sqliteRepository) CreateSupplier(ctx context.Context, s *Supplier) error {
    data, err := json.Marshal(s)
    if err != nil {
        return fmt.Errorf("supplier: marshal: %w", err)
    }
    _, err = r.db.ExecContext(ctx,
        `INSERT INTO suppliers (id, data) VALUES (?, ?)
         ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
        s.ID, string(data))
    return err
}
```

**Tests to Write:**
- TestCreateSupplier_Success
- TestCreateSupplier_Duplicate
- TestFindSupplierByID_Success
- TestFindSupplierByID_NotFound
- TestUpdateSupplier_Success
- TestDeleteSupplier_Success
- TestListSuppliers_Success
- TestListSuppliers_FilterByName
- TestListSuppliers_FilterByStatus
- TestNextSupplierNo

**Estimated Time:** 8 hours

---

### Day 4 (Thursday): Supplier Service

**Tasks:**
- [ ] Create `internal/application/purchase/service.go`
- [ ] Implement CreateSupplier
- [ ] Implement GetSupplier
- [ ] Implement UpdateSupplier
- [ ] Implement DeleteSupplier
- [ ] Implement ListSuppliers
- [ ] Add validation logic

**Service Pattern:**
```go
func (s *service) CreateSupplier(ctx context.Context, supplier *Supplier, actor string) (*Supplier, error) {
    sp := supplier.Clone()
    sp.Status = SupplierActive
    sp.CreatedBy = actor
    sp.UpdatedBy = actor
    
    if err := ValidateSupplier(sp); err != nil {
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
```

**Tests to Write:**
- TestCreateSupplier_Success
- TestCreateSupplier_EmptyName
- TestGetSupplier_Success
- TestGetSupplier_NotFound
- TestUpdateSupplier_Success
- TestDeleteSupplier_Success
- TestListSuppliers_Success
- TestListSuppliers_FilterByName
- TestListSuppliers_FilterByStatus

**Estimated Time:** 8 hours

---

### Day 5 (Friday): Supplier Handler

**Tasks:**
- [ ] Create `internal/interfaces/http/purchase/handler.go`
- [ ] Implement createSupplier
- [ ] Implement getSupplier
- [ ] Implement updateSupplier
- [ ] Implement deleteSupplier
- [ ] Implement listSuppliers
- [ ] Add error mapping

**Handler Pattern:**
```go
func (h *Handler) createSupplier(c *gin.Context) {
    var input purchase.Supplier
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
        return
    }
    result, err := h.svc.CreateSupplier(c.Request.Context(), &input, h.actor(c))
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusCreated, result)
}
```

**Tests to Write:**
- TestCreateSupplier_Success
- TestCreateSupplier_InvalidJSON
- TestGetSupplier_Success
- TestGetSupplier_NotFound
- TestUpdateSupplier_Success
- TestDeleteSupplier_Success
- TestListSuppliers_Success

**Estimated Time:** 6 hours

---

## Week 2: Supplier Integration & Migrations

### Day 1 (Monday): DB Migrations

**Tasks:**
- [ ] Add `suppliers` table to `migrate.go`
- [ ] Add `supplier_sequences` table
- [ ] Test migrations

**Migration Code:**
```go
var tables = []string{
    // ... existing tables
    "suppliers",
    "supplier_sequences",
}
```

**Estimated Time:** 2 hours

---

### Day 2 (Tuesday): Wire in main.go

**Tasks:**
- [ ] Import purchase packages
- [ ] Create supplier repository
- [ ] Create supplier service
- [ ] Create supplier handler
- [ ] Register routes

**Wire Code:**
```go
// In main.go
perssupplier "goGL/internal/infrastructure/persistence/purchase"
httpsupplier "goGL/internal/interfaces/http/purchase"

// In setup
supplierRepo := perssupplier.NewSqliteRepository(sqlDB)
supplierSvc := purchase.NewService(supplierRepo)
httpsupplier.NewHandler(supplierSvc).Register(v1)
```

**Estimated Time:** 2 hours

---

### Day 3 (Wednesday): E2E Testing

**Tasks:**
- [ ] Test Create Supplier via API
- [ ] Test Get Supplier via API
- [ ] Test Update Supplier via API
- [ ] Test Delete Supplier via API
- [ ] Test List Suppliers via API

**Test Script:**
```bash
# Create supplier
curl -X POST http://localhost:8080/api/v1/purchase/suppliers \
  -H "Content-Type: application/json" \
  -d '{"name":"ABC Company","tax_code":"0123456789"}'

# Get supplier
curl http://localhost:8080/api/v1/purchase/suppliers/NCC-00001

# List suppliers
curl http://localhost:8080/api/v1/purchase/suppliers
```

**Estimated Time:** 4 hours

---

### Day 4 (Thursday): Code Review

**Tasks:**
- [ ] Review all Week 1-2 code
- [ ] Fix any issues
- [ ] Ensure all tests pass
- [ ] Update documentation

**Estimated Time:** 4 hours

---

### Day 5 (Friday): Documentation

**Tasks:**
- [ ] Create `docs/Supplier-Spec.md`
- [ ] Document API endpoints
- [ ] Document data model
- [ ] Document validation rules

**Estimated Time:** 4 hours

---

## Week 3-4: Purchase Orders

### Week 3 Tasks:
- [ ] Create PurchaseOrder entity
- [ ] Create OrderLine entity
- [ ] Create Order Repository interface
- [ ] Implement SQLite Repository
- [ ] Create Order Service

### Week 4 Tasks:
- [ ] Create Order Handler
- [ ] Add DB migrations
- [ ] Implement status workflow
- [ ] Wire in main.go
- [ ] E2E testing & code review

---

## Week 5-6: Goods Receipts

### Week 5 Tasks:
- [ ] Create GoodsReceipt entity
- [ ] Create ReceiptLine entity
- [ ] Create Receipt Repository interface
- [ ] Implement SQLite Repository
- [ ] Create Receipt Service

### Week 6 Tasks:
- [ ] Create Receipt Handler
- [ ] Implement PO received quantity tracking
- [ ] Add DB migrations
- [ ] Wire in main.go
- [ ] E2E testing & code review

---

## Week 7-9: Purchase Invoices

### Week 7 Tasks:
- [ ] Create PurchaseInvoice entity
- [ ] Create InvoiceLine entity
- [ ] Create Invoice Repository interface
- [ ] Implement SQLite Repository
- [ ] Create Invoice Service

### Week 8 Tasks:
- [ ] Create Invoice Handler
- [ ] Implement VAT calculation
- [ ] Implement e-invoice status tracking
- [ ] Add DB migrations
- [ ] Wire in main.go

### Week 9 Tasks:
- [ ] Implement GL posting logic
- [ ] Implement journal entries
- [ ] E2E testing
- [ ] Code review
- [ ] Documentation

---

## Week 10-11: Payments

### Week 10 Tasks:
- [ ] Create Payment entity
- [ ] Create Payment Repository interface
- [ ] Implement SQLite Repository
- [ ] Create Payment Service
- [ ] Create Payment Handler

### Week 11 Tasks:
- [ ] Implement payment application to invoices
- [ ] Implement payment GL entries
- [ ] Add DB migrations
- [ ] Wire in main.go
- [ ] E2E testing & code review

---

## Week 12: Integration & Finalization

### Week 12 Tasks:
- [ ] Full integration testing
- [ ] Performance testing
- [ ] Security review
- [ ] Final code review
- [ ] Documentation finalization

---

## Daily Checklist

Before starting work:
- [ ] Check previous day's tests passing
- [ ] Review any blocking issues
- [ ] Update task status

Before ending work:
- [ ] Run all tests
- [ ] Run `go vet`
- [ ] Run `gofmt`
- [ ] Commit changes
- [ ] Update documentation

---

## Blockers & Escalation

| Blocker | Escalation Path |
|---------|-----------------|
| Build fails | Fix immediately, check imports |
| Tests fail | Fix immediately, check mocks |
| Migration fails | Check table names, column types |
| Code review blocked | Address feedback, re-submit |

---

## Success Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Tests passing | 395+ | `go test ./...` |
| Code coverage | 80%+ | `go test -cover` |
| API endpoints | 18 | Manual testing |
| Documentation | Complete | Doc review |
| Code review | APPROVED | Code review |
