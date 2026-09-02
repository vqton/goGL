# Implementation Roadmap — Inventory Module
## Phased Delivery Plan
### Version 1.0 | 2026-09-01

---

## Overview

The Inventory module will be implemented in **5 phases** over **10 weeks**, with each phase building on the previous. The current stub will be **completely rewritten**.

**Prerequisites**: Purchase Module and Sales Module are already implemented.

---

## Phase 1: Foundation (Weeks 1-2)
### Goal: Item + Warehouse Master Data + Basic Operations

**Week 1: Domain Entities**
| Day | Task | Files |
|-----|------|-------|
| 1 | Item entity + validation + error vars | `internal/domain/inventory/entity.go` |
| 2 | Warehouse entity + validation | `internal/domain/inventory/entity.go` |
| 3 | StockCard entity | `internal/domain/inventory/entity.go` |
| 4 | Repository interfaces (Item, Warehouse, StockCard) | `internal/domain/inventory/entity.go` |
| 5 | Entity tests (40+ cases) | `internal/domain/inventory/entity_test.go` |

**Week 2: Infrastructure + HTTP**
| Day | Task | Files |
|-----|------|-------|
| 1 | Item repository (SQLite) | `internal/infrastructure/persistence/inventory/repository.go` |
| 2 | Warehouse repository (SQLite) | `internal/infrastructure/persistence/inventory/repository.go` |
| 3 | StockCard repository (SQLite) | `internal/infrastructure/persistence/inventory/repository.go` |
| 4 | Item + Warehouse HTTP handlers | `internal/interfaces/http/inventory/handler.go` |
| 5 | Repository + Handler tests | `*_test.go` |

**Deliverables**:
- Item CRUD (create, read, update, deactivate)
- Warehouse CRUD
- Stock balance query (read-only)
- Auto-generated codes: MH-XXXXX (items), KHO-XXX (warehouses)
- DB migrations for 3 tables

**Acceptance Criteria**:
- [ ] Can create item with auto-code
- [ ] Can create warehouse
- [ ] Can query stock balance (initially empty)
- [ ] 40+ entity tests pass
- [ ] 20+ repository tests pass
- [ ] 16+ handler tests pass

---

## Phase 2: Stock Tracking + Valuation (Weeks 3-4)
### Goal: FIFO/Weighted Average Valuation + Goods Receipt

**Week 3: Valuation Logic**
| Day | Task | Files |
|-----|------|-------|
| 1 | FIFO layer entity + storage | `internal/domain/inventory/entity.go` |
| 2 | FIFO valuation logic | `internal/domain/inventory/valuation.go` |
| 3 | Weighted Average logic | `internal/domain/inventory/valuation.go` |
| 4 | Unit price calculation tests | `internal/domain/inventory/valuation_test.go` |
| 5 | Integration tests (FIFO vs Weighted Average) | `valuation_test.go` |

**Week 4: Goods Receipt**
| Day | Task | Files |
|-----|------|-------|
| 1 | Receipt service logic | `internal/application/inventory/service.go` |
| 2 | GL posting integration (Dr. 152xxx / Cr. 331) | `service.go` |
| 3 | Receipt HTTP handler | `handler.go` |
| 4 | DB migrations (stock_movements, fifo_layers) | `migrate.go` |
| 5 | Integration tests (receipt + GL) | `service_test.go`, `handler_test.go` |

**Deliverables**:
- FIFO valuation (layer-based allocation)
- Weighted Average valuation
- Goods Receipt from Purchase Order
- GL auto-posting on receipt
- Stock card auto-update

**Acceptance Criteria**:
- [ ] Can receive goods against PO
- [ ] Stock card updated correctly
- [ ] FIFO layers created and consumed correctly
- [ ] Weighted average recalculated correctly
- [ ] GL entry posted: Dr. 152xxx / Cr. 331
- [ ] 30+ valuation tests pass

---

## Phase 3: Goods Dispatch + Transfers (Weeks 5-6)
### Goal: Complete Stock Movement Flows

**Week 5: Goods Dispatch**
| Day | Task | Files |
|-----|------|-------|
| 1 | Dispatch service logic | `service.go` |
| 2 | COGS calculation (FIFO/WA) | `valuation.go` |
| 3 | GL posting (Dr. 632 / Cr. 152xxx) | `service.go` |
| 4 | Dispatch HTTP handler | `handler.go` |
| 5 | Integration tests | `service_test.go`, `handler_test.go` |

**Week 6: Transfer + Adjustment**
| Day | Task | Files |
|-----|------|-------|
| 1 | Stock Transfer logic (2 movements) | `service.go` |
| 2 | Stock Adjustment logic | `service.go` |
| 3 | GL posting for adjustments | `service.go` |
| 4 | HTTP handlers for both | `handler.go` |
| 5 | Integration tests | `service_test.go`, `handler_test.go` |

**Deliverables**:
- Goods Dispatch (from Sales Invoice)
- Stock Transfer between warehouses
- Stock Adjustment (manual corrections)
- GL posting for all movements
- Stock card updates for all movement types

**Acceptance Criteria**:
- [ ] Can dispatch goods against Sales Invoice
- [ ] Stock reduced, COGS calculated correctly
- [ ] GL entry: Dr. 632 / Cr. 152xxx
- [ ] Can transfer between warehouses
- [ ] Both stock cards updated
- [ ] Can create adjustment with approval workflow
- [ ] 40+ new tests pass

---

## Phase 4: Physical Count + NRV (Weeks 7-8)
### Goal: Compliance Features

**Week 7: Physical Count**
| Day | Task | Files |
|-----|------|-------|
| 1 | PhysicalCount entity + lines | `entity.go` |
| 2 | Count service logic (create, submit, reconcile) | `service.go` |
| 3 | Auto-generate adjustment from count | `service.go` |
| 4 | HTTP handler | `handler.go` |
| 5 | Integration tests | `service_test.go` |

**Week 8: NRV Write-Down**
| Day | Task | Files |
|-----|------|-------|
| 1 | NRV write-down service logic | `service.go` |
| 2 | NRV reversal logic (next year) | `service.go` |
| 3 | GL posting for write-down/reversal | `service.go` |
| 4 | HTTP handler | `handler.go` |
| 5 | Integration tests | `service_test.go` |

**Deliverables**:
- Physical Count workflow (create → enter → review → reconcile)
- Auto-adjustment generation from count differences
- NRV write-down (year-end)
- NRV reversal (subsequent year)
- GL posting for all

**Acceptance Criteria**:
- [ ] Can create and complete physical count
- [ ] Adjustments auto-generated from differences
- [ ] NRV write-down posts GL: Dr. 515xxx / Cr. 152xxx
- [ ] NRV reversal posts GL: Dr. 152xxx / Cr. 515xxx
- [ ] 30+ new tests pass

---

## Phase 5: Integration + Reports (Weeks 9-10)
### Goal: Full Integration + Reporting

**Week 9: Integration**
| Day | Task | Files |
|-----|------|-------|
| 1 | Purchase → Receipt auto-creation | `service.go` |
| 2 | Sales → Dispatch auto-creation | `service.go` |
| 3 | E-invoice integration | `handler.go` |
| 4 | Opening balance import | `service.go` |
| 5 | Integration tests (end-to-end) | `handler_test.go` |

**Week 10: Reports + Final**
| Day | Task | Files |
|-----|------|-------|
| 1 | Stock Balance Report | `service.go` |
| 2 | Stock Movement Report | `service.go` |
| 3 | Stock Valuation Report | `service.go` |
| 4 | Final testing | `*_test.go` |
| 5 | Code review + documentation | `docs/` |

**Deliverables**:
- Auto-create receipt from Purchase Order
- Auto-create dispatch from Sales Invoice
- Opening balance import
- Stock Balance Report
- Stock Movement Report
- Stock Valuation Report
- Full test coverage

**Acceptance Criteria**:
- [ ] Purchase order → auto-receipt on confirmation
- [ ] Sales invoice → auto-dispatch on confirmation
- [ ] Can import opening balances
- [ ] All 3 reports generate correctly
- [ ] End-to-end workflow tested
- [ ] 99+ total tests pass
- [ ] Code review approved

---

## Test Coverage Targets

| Phase | Unit Tests | Integration Tests | Total |
|-------|------------|-------------------|-------|
| Phase 1 | 40 | 20 | 60 |
| Phase 2 | 30 | 20 | 50 |
| Phase 3 | 40 | 20 | 60 |
| Phase 4 | 30 | 15 | 45 |
| Phase 5 | 20 | 20 | 40 |
| **Total** | **160** | **95** | **255** |

---

## Risk Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| FIFO layer complexity | High | High | Start with simple FIFO, iterate |
| Concurrent stock updates | Medium | High | Use SQLite transactions + row locking |
| GL posting failures | Medium | Medium | Queue-based retry mechanism |
| Performance with large datasets | Low | Medium | Index optimization |
| Integration with existing modules | Low | Low | Well-defined interfaces |

---

## Dependencies on Existing Modules

| Module | Integration Point | Status |
|--------|-------------------|--------|
| Purchase | PO → Receipt auto-creation | ✅ Ready |
| Sales | SI → Dispatch auto-creation | ✅ Ready |
| Ledger | GL posting | ✅ Ready |
| Master Data | Item/Warehouse definitions | ✅ Ready |
| Config | Valuation method settings | ✅ Ready |

---

*Document prepared by: BA Lead + Vietnamese Chief Accountant*
*Date: 2026-09-01*
*Version: 1.0*
