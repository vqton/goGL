# BRD — Inventory Module
## Business Requirements Document
### Version 1.0 | 2026-09-01

---

## 1. Executive Summary

### 1.1 Purpose
This BRD defines the business requirements for the Inventory Management module of the Go ERP system, ensuring compliance with Vietnamese accounting standards (VAS 02) and Circular 99/2025/TT-BTC.

### 1.2 Current State
The inventory module is currently a **23-line stub** that provides only basic StockMovement entity with 6 fields. It cannot handle any Vietnamese accounting requirements, warehouse management, stock valuation, physical count, or GL posting.

### 1.3 Target State
A fully functional inventory management system that:
- Manages items, warehouses, and stock balances
- Supports FIFO and Weighted Average valuation methods
- Integrates with Purchase and Sales modules
- Posts GL entries for all inventory transactions
- Supports physical count and year-end NRV write-downs
- Complies with VAS 02, Circular 99/2025/TT-BTC, and Decree 123/2020/NĐ-CP

### 1.4 Scope
**In Scope**:
- Item master management
- Warehouse management
- Stock balance tracking (per item per warehouse)
- Goods receipt (from purchase)
- Goods dispatch (for sales)
- Internal stock transfer
- Stock adjustment
- Opening balance import
- Physical count
- NRV write-down and reversal
- Stock reports (balance, movement, valuation, aging)
- GL integration for all transactions

**Out of Scope**:
- Barcode scanning (Phase 2)
- Multi-warehouse picking optimization
- Demand forecasting
- Production planning
- Serial number tracking (Phase 2)
- Batch/lot tracking (Phase 2)

---

## 2. Business Objectives

| # | Objective | Success Measure |
|---|-----------|-----------------|
| BO-01 | Accurate stock valuation per VAS 02 | 100% compliance with VAS 02 |
| BO-02 | Real-time stock visibility | Stock balances updated in real-time |
| BO-03 | GL integration | All movements auto-posted to GL |
| BO-04 | Physical count support | Annual count reconciled within 5 days |
| BO-05 | Integration with Purchase/Sales | Auto-create movements from PO/SI |

---

## 3. Stakeholders

| Role | Interest |
|------|----------|
| Warehouse Manager | Daily stock operations, physical counts |
| Warehouse Staff | Receipt/dispatch entry |
| Chief Accountant | GL posting, year-end valuation, financial statements |
| Finance Manager | Stock reports, valuation, write-downs |
| Procurement Manager | PO tracking, stock levels |
| Sales Manager | Stock availability for sales |
| IT Manager | System integration, maintenance |

---

## 4. Functional Requirements

### FR-01: Item Management
- **FR-01.1**: Create item with code (auto-generated: MH-XXXXX), name, category, unit, GL accounts
- **FR-01.2**: Classify items by type (raw materials, supplies, finished goods, WIP, consignment)
- **FR-01.3**: Assign valuation method per item (FIFO or Weighted Average)
- **FR-01.4**: Set minimum/maximum stock levels and reorder quantities
- **FR-01.5**: Activate/deactivate items
- **FR-01.6**: Search and filter items

### FR-02: Warehouse Management
- **FR-02.1**: Create warehouse with code (auto-generated: KHO-XXX), name, address, type
- **FR-02.2**: Assign warehouse manager
- **FR-02.3**: Activate/deactivate warehouses
- **FR-02.4**: Prevent stock operations on inactive warehouses

### FR-03: Stock Balance Tracking
- **FR-03.1**: Maintain StockCard per item per warehouse
- **FR-03.2**: Track opening quantity and value
- **FR-03.3**: Track total in/out quantities and values
- **FR-03.4**: Maintain current quantity and value
- **FR-03.5**: For Weighted Average: maintain average cost per unit
- **FR-03.6**: Update balances in real-time on every movement

### FR-04: Goods Receipt
- **FR-04.1**: Create receipt from Purchase Order
- **FR-04.2**: Validate received quantity ≤ ordered quantity (± tolerance)
- **FR-04.3**: Calculate unit cost from PO (FIFO) or weighted average
- **FR-04.4**: Create StockMovement(type=receipt)
- **FR-04.5**: Update StockCard
- **FR-04.6**: Post GL: Dr. 152xxx / Dr. 1331 / Cr. 331
- **FR-04.7**: Support partial receipts
- **FR-04.8**: Track cumulative received quantity against PO

### FR-05: Goods Dispatch
- **FR-05.1**: Create dispatch from Sales Invoice
- **FR-05.2**: Validate sufficient stock before dispatch
- **FR-05.3**: Calculate COGS using item's valuation method
- **FR-05.4**: Create StockMovement(type=dispatch)
- **FR-05.5**: Update StockCard
- **FR-05.6**: Post GL: Dr. 632 / Cr. 152xxx
- **FR-05.7**: Support partial dispatches

### FR-06: Stock Transfer
- **FR-06.1**: Create transfer between warehouses
- **FR-06.2**: Validate sufficient stock in source warehouse
- **FR-06.3**: Create two movements (transfer_out + transfer_in)
- **FR-06.4**: Update both StockCards
- **FR-06.5**: No GL entry (same entity)

### FR-07: Stock Adjustment
- **FR-07.1**: Create adjustment for count discrepancies or errors
- **FR-07.2**: Support positive and negative adjustments
- **FR-07.3**: Require approval for adjustments above threshold
- **FR-07.4**: Create StockMovement(type=adjustment)
- **FR-07.5**: Post GL: Dr./Cr. 152xxx ↔ Cr./Dr. 632

### FR-08: Opening Balance
- **FR-08.1**: Import opening balances per item per warehouse
- **FR-08.2**: Validate no duplicate opening for same period
- **FR-08.3**: Create StockMovement(type=opening_balance)
- **FR-08.4**: Post GL opening entry

### FR-09: Physical Count
- **FR-09.1**: Create physical count for a warehouse
- **FR-09.2**: Generate item list with book quantities
- **FR-09.3**: Enter actual quantities
- **FR-09.4**: Calculate differences and adjustment amounts
- **FR-09.5**: Submit for review
- **FR-09.6**: Reconcile and post adjustments
- **FR-09.7**: Support annual, periodic, and ad-hoc counts

### FR-10: NRV Write-Down
- **FR-10.1**: Calculate original price per item (from stock cards)
- **FR-10.2**: Accept NRV input (or calculate from latest selling price)
- **FR-10.3**: Identify items needing write-down (NRV < original price)
- **FR-10.4**: Create write-down journal
- **FR-10.5**: Post GL: Dr. 515xxx / Cr. 152xxx
- **FR-10.6**: Support reversal in subsequent year (up to original price)

### FR-11: Reports
- **FR-11.1**: Stock Balance Report (per warehouse, per item)
- **FR-11.2**: Stock Movement Report (in/out/balance per item)
- **FR-11.3**: Stock Valuation Report (original price, NRV, write-down)
- **FR-11.4**: Stock Aging Report
- **FR-11.5**: Export to Excel/PDF

---

## 5. Non-Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| NFR-01 | Stock balances update in real-time (<100ms) | High |
| NFR-02 | Support 10,000+ items | Medium |
| NFR-03 | Support 100+ warehouses | Medium |
| NFR-04 | Concurrent stock operations with locking | High |
| NFR-05 | Full audit trail for all movements | High |
| NFR-06 | GL posting with retry on failure | Medium |
| NFR-07 | Export reports for 1M+ rows | Low |

---

## 6. Data Requirements

### 6.1 Required Data
- Item master (code, name, category, unit, GL accounts, valuation method)
- Warehouse master (code, name, address, type, manager)
- Stock cards (per item per warehouse)
- Stock movements (receipts, dispatches, transfers, adjustments)
- FIFO layers (for FIFO valuation)
- Physical count records
- NRV write-down records

### 6.2 Data Retention
- Stock movements: 10 years (per Vietnamese accounting law)
- Stock cards: 10 years
- Physical counts: 10 years
- GL entries: 10 years

---

## 7. Compliance Requirements

| Regulation | Requirement | Implementation |
|------------|-------------|----------------|
| VAS 02 | Inventory valuation at original price or NRV | FIFO/Weighted Average + NRV write-down |
| VAS 02 | LIFO prohibited | System does not offer LIFO |
| VAS 02 | Disclosure in financial statements | Reports for all required disclosures |
| Circular 99/2025/TT-BTC | Periodic physical count | Physical count module |
| Circular 99/2025/TT-BTC | Warehouse bookkeeping | Stock card = warehouse book |
| Decree 123/2020/NĐ-CP | E-invoice for inventory | Integration with e-invoice system |
| Decree 70/2025/NĐ-CP | E-invoice amendments | Updated e-invoice format |

---

## 8. Assumptions

1. Purchase module is already implemented and can be integrated
2. Sales module is already implemented and can be integrated
3. Ledger module exists for GL posting
4. Master data module exists for items and warehouses
5. SQLite database is sufficient for inventory data
6. Single-currency (VND) for inventory valuation

---

## 9. Dependencies

| Dependency | Type | Impact |
|------------|------|--------|
| Purchase Module | Internal | Receipts auto-created from PO |
| Sales Module | Internal | Dispatches auto-created from SI |
| Ledger Module | Internal | GL posting for all movements |
| Master Data Module | Internal | Item and warehouse definitions |
| Config Module | Internal | Valuation method, thresholds |

---

## 10. Approval

| Role | Name | Date | Sign |
|------|------|------|------|
| Product Owner | | | |
| Chief Accountant | | | |
| Warehouse Manager | | | |
| IT Manager | | | |

---

*Document prepared by: BA Lead + Vietnamese Chief Accountant*
*Date: 2026-09-01*
*Version: 1.0*
