# Implementation Roadmap: Purchase Module

## Version History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-08-30 | BA Lead + Chief Accountant | Initial roadmap |

---

## Executive Summary

| Metric | Value |
|--------|-------|
| **Total Duration** | 12 weeks |
| **Total Files** | 25+ files |
| **Total Tests** | 150+ tests |
| **API Endpoints** | 18 endpoints |
| **Entities** | 5 entities |

---

## Phase 1: Foundation (Week 1-2)

### Week 1: Supplier Entity & Repository

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create Supplier entity with full fields | `domain/purchase/supplier.go` | 15 tests |
| Day 2 | Create Supplier Repository interface | `domain/purchase/entity.go` | - |
| Day 3 | Implement SQLite Repository | `infrastructure/persistence/purchase/repository.go` | 10 tests |
| Day 4 | Create Supplier Service | `application/purchase/service.go` | 12 tests |
| Day 5 | Create Supplier Handler | `interfaces/http/purchase/handler.go` | 8 tests |

### Week 2: Supplier Integration & Migrations

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Add DB migrations for suppliers | `infrastructure/db/migrate.go` | - |
| Day 2 | Wire Supplier in main.go | `cmd/server/main.go` | - |
| Day 3 | E2E testing for Supplier | - | 5 tests |
| Day 4 | Code review & fixes | - | - |
| Day 5 | Documentation | `docs/Supplier-Spec.md` | - |

**Phase 1 Deliverables:**
- [ ] Supplier CRUD (Create, Read, Update, Delete, List)
- [ ] 50+ tests passing
- [ ] DB migrations
- [ ] Documentation

---

## Phase 2: Purchase Orders (Week 3-4)

### Week 3: Purchase Order Entity & Repository

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create PurchaseOrder entity | `domain/purchase/order.go` | 20 tests |
| Day 2 | Create OrderLine entity | `domain/purchase/order.go` | 10 tests |
| Day 3 | Create Order Repository interface | `domain/purchase/entity.go` | - |
| Day 4 | Implement SQLite Repository | `infrastructure/persistence/purchase/repository.go` | 15 tests |
| Day 5 | Create Order Service | `application/purchase/service.go` | 18 tests |

### Week 4: Purchase Order Integration

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create Order Handler | `interfaces/http/purchase/handler.go` | 12 tests |
| Day 2 | Add DB migrations for orders | `infrastructure/db/migrate.go` | - |
| Day 3 | Implement status workflow (Draft→Confirmed→Received) | `application/purchase/service.go` | 8 tests |
| Day 4 | Wire in main.go | `cmd/server/main.go` | - |
| Day 5 | E2E testing & code review | - | 10 tests |

**Phase 2 Deliverables:**
- [ ] Purchase Order CRUD
- [ ] Status workflow (Draft→Confirmed→Received→Completed/Cancelled)
- [ ] 75+ tests passing
- [ ] Documentation

---

## Phase 3: Goods Receipts (Week 5-6)

### Week 5: Goods Receipt Entity & Repository

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create GoodsReceipt entity | `domain/purchase/receipt.go` | 20 tests |
| Day 2 | Create ReceiptLine entity | `domain/purchase/receipt.go` | 10 tests |
| Day 3 | Create Receipt Repository interface | `domain/purchase/entity.go` | - |
| Day 4 | Implement SQLite Repository | `infrastructure/persistence/purchase/repository.go` | 15 tests |
| Day 5 | Create Receipt Service | `application/purchase/service.go` | 18 tests |

### Week 6: Goods Receipt Integration

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create Receipt Handler | `interfaces/http/purchase/handler.go` | 12 tests |
| Day 2 | Implement PO received quantity tracking | `application/purchase/service.go` | 10 tests |
| Day 3 | Add DB migrations for receipts | `infrastructure/db/migrate.go` | - |
| Day 4 | Wire in main.go | `cmd/server/main.go` | - |
| Day 5 | E2E testing & code review | - | 10 tests |

**Phase 3 Deliverables:**
- [ ] Goods Receipt CRUD
- [ ] PO received quantity tracking
- [ ] Quality inspection fields
- [ ] 75+ tests passing
- [ ] Documentation

---

## Phase 4: Purchase Invoices (Week 7-9)

### Week 7: Purchase Invoice Entity & Repository

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create PurchaseInvoice entity | `domain/purchase/invoice.go` | 25 tests |
| Day 2 | Create InvoiceLine entity | `domain/purchase/invoice.go` | 12 tests |
| Day 3 | Create Invoice Repository interface | `domain/purchase/entity.go` | - |
| Day 4 | Implement SQLite Repository | `infrastructure/persistence/purchase/repository.go` | 18 tests |
| Day 5 | Create Invoice Service | `application/purchase/service.go` | 20 tests |

### Week 8: Purchase Invoice Integration

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create Invoice Handler | `interfaces/http/purchase/handler.go` | 15 tests |
| Day 2 | Implement VAT calculation (10%, 8%, 0%) | `application/purchase/service.go` | 12 tests |
| Day 3 | Implement e-invoice status tracking | `application/purchase/service.go` | 8 tests |
| Day 4 | Add DB migrations for invoices | `infrastructure/db/migrate.go` | - |
| Day 5 | Wire in main.go | `cmd/server/main.go` | - |

### Week 9: GL Posting & Testing

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Implement GL posting logic | `application/purchase/service.go` | 15 tests |
| Day 2 | Implement journal entries (331, 133, 152) | `application/purchase/service.go` | 10 tests |
| Day 3 | E2E testing for invoice workflow | - | 10 tests |
| Day 4 | Code review & fixes | - | - |
| Day 5 | Documentation | `docs/PurchaseInvoice-Spec.md` | - |

**Phase 4 Deliverables:**
- [ ] Purchase Invoice CRUD
- [ ] VAT calculation per Thông tư 99/2025
- [ ] E-invoice status tracking
- [ ] GL posting with journal entries
- [ ] 100+ tests passing
- [ ] Documentation

---

## Phase 5: Payments (Week 10-11)

### Week 10: Payment Entity & Repository

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Create Payment entity | `domain/purchase/payment.go` | 20 tests |
| Day 2 | Create Payment Repository interface | `domain/purchase/entity.go` | - |
| Day 3 | Implement SQLite Repository | `infrastructure/persistence/purchase/repository.go` | 15 tests |
| Day 4 | Create Payment Service | `application/purchase/service.go` | 18 tests |
| Day 5 | Create Payment Handler | `interfaces/http/purchase/handler.go` | 12 tests |

### Week 11: Payment Integration

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Implement payment application to invoices | `application/purchase/service.go` | 12 tests |
| Day 2 | Implement payment GL entries (331, 111/112) | `application/purchase/service.go` | 8 tests |
| Day 3 | Add DB migrations for payments | `infrastructure/db/migrate.go` | - |
| Day 4 | Wire in main.go | `cmd/server/main.go` | - |
| Day 5 | E2E testing & code review | - | 10 tests |

**Phase 5 Deliverables:**
- [ ] Payment CRUD
- [ ] Payment application to invoices
- [ ] Payment GL entries
- [ ] 65+ tests passing
- [ ] Documentation

---

## Phase 6: Integration & Finalization (Week 12)

### Week 12: Integration & Deployment

| Day | Task | Files | Tests |
|-----|------|-------|-------|
| Day 1 | Full integration testing | - | 30 tests |
| Day 2 | Performance testing | - | 10 tests |
| Day 3 | Security review | - | - |
| Day 4 | Final code review | - | - |
| Day 5 | Documentation finalization | `docs/Purchase-Module-Spec.md` | - |

**Phase 6 Deliverables:**
- [ ] Full integration testing
- [ ] Performance benchmarks
- [ ] Security review passed
- [ ] Code review approved
- [ ] Complete documentation

---

## File Structure

```
internal/
├── domain/purchase/
│   ├── entity.go          # Repository interface + shared types
│   ├── supplier.go        # Supplier entity
│   ├── order.go           # PurchaseOrder entity
│   ├── receipt.go         # GoodsReceipt entity
│   ├── invoice.go         # PurchaseInvoice entity
│   ├── payment.go         # Payment entity
│   └── entity_test.go     # Entity tests
├── application/purchase/
│   ├── service.go         # Service interface + implementation
│   └── service_test.go    # Service tests
├── infrastructure/persistence/purchase/
│   ├── repository.go      # SQLite repository
│   └── repository_test.go # Repository tests
├── interfaces/http/purchase/
│   ├── handler.go         # HTTP handlers
│   └── handler_test.go    # Handler tests
└── infrastructure/db/
    └── migrate.go         # DB migrations
```

---

## Test Coverage Targets

| Phase | Target | Actual |
|-------|--------|--------|
| Phase 1: Supplier | 50+ | - |
| Phase 2: Purchase Orders | 75+ | - |
| Phase 3: Goods Receipts | 75+ | - |
| Phase 4: Purchase Invoices | 100+ | - |
| Phase 5: Payments | 65+ | - |
| Phase 6: Integration | 30+ | - |
| **Total** | **395+** | - |

---

## Risk Mitigation

| Risk | Phase | Mitigation |
|------|-------|------------|
| VAT calculation errors | Phase 4 | Validate per Thông tư 99/2025 |
| GL posting errors | Phase 4 | Use official account list |
| Missing migrations | Phase 1-5 | Add migrations per phase |
| Code review delays | Phase 6 | Review after each phase |
| Integration issues | Phase 6 | E2E testing per phase |

---

## Dependencies

| Dependency | Required By | Notes |
|------------|-------------|-------|
| SQLite driver | All phases | Already in project |
| Gin framework | All phases | Already in project |
| JSON doc storage | All phases | Already in project |
| Core types (Money, etc.) | All phases | Already in project |

---

## Success Criteria

| Criterion | Target | How to Verify |
|-----------|--------|---------------|
| All 18 API endpoints | 18 | Manual testing |
| Test coverage | 395+ tests | `go test ./...` |
| VAT calculation | Per TT99 | Unit tests |
| GL posting | Correct entries | Unit tests |
| E-invoice tracking | Status flow | Unit tests |
| Code review | APPROVED | Code review |
| Documentation | Complete | Doc review |

---

## Daily Standup Template

```
Daily Standup - [Date]
Phase: [X] - [Phase Name]
Week: [X] of 12

Yesterday:
- [Completed tasks]

Today:
- [Planned tasks]

Blockers:
- [Any blockers]
```

---

## Go/No-Go Checklist

Before each phase starts:
- [ ] Previous phase tests passing
- [ ] Previous phase code reviewed
- [ ] Previous phase documentation complete
- [ ] No blocking issues

Before Phase 6 (Final):
- [ ] All 5 entities implemented
- [ ] All 18 API endpoints working
- [ ] All 395+ tests passing
- [ ] No critical bugs
- [ ] No security vulnerabilities
