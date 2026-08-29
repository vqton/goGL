# Business Requirements Document - Fixed Asset Module

**Version**: 1.0  
**Date**: 2026-08-29  
**Module**: Fixed Asset Management (FA)  
**Compliance**: Circular 45/2013/TT-BTC, Circular 99/2025/TT-BTC, VAS 03

---

## 1. Executive Summary

The Fixed Asset (FA) module manages the complete lifecycle of tangible fixed assets for Vietnamese enterprises. The module must comply with Vietnamese accounting standards (VAS 03), Ministry of Finance Circulars, and produce accurate financial statements and tax reports.

**Current Status**: Stub (Not Implemented)  
**Target**: Production-ready FA module with full compliance

---

## 2. Business Objectives

1. Track all tangible fixed assets meeting VND 30M threshold
2. Calculate and post depreciation using approved methods
3. Generate compliant financial statements and tax reports
4. Support asset lifecycle from acquisition to liquidation
5. Provide real-time asset register and depreciation schedule

---

## 3. Scope

### In Scope
- Asset registration and classification
- Depreciation calculation (3 methods)
- Monthly depreciation posting to GL
- Asset transfer between locations/departments
- Asset liquidation with approval workflow
- Asset revaluation
- Reporting and dashboards

### Out of Scope
- Intangible fixed assets (separate module)
- Investment properties (Account 217)
- Construction in progress (Account 241)
- Biological assets (Account 215)

---

## 4. Functional Requirements

### 4.1 Asset Registration

**FR-4.1.1**: System shall allow creation of asset cards with:
- Auto-generated code (FA-XXXXX format)
- Original cost (must be >= VND 30,000,000)
- Estimated residual value (< original cost)
- Useful life (months) within Circular 45 framework
- Depreciation method selection
- Asset classification (6 types per Circular 45)
- Purchase date, in-service date
- Vendor information
- Location, department assignment

**FR-4.1.2**: System shall validate asset data against Circular 45 requirements

**FR-4.1.3**: System shall assign asset to appropriate account (211)

### 4.2 Depreciation Calculation

**FR-4.2.1**: System shall support three depreciation methods:
- Straight-line: Monthly = Original Cost / Useful Life Months
- Declining-balance: Accelerated depreciation per Circular 45 formula
- Units-of-output: Based on production volume

**FR-4.2.2**: System shall enforce depreciation life limits per Annex I of Circular 45

**FR-4.2.3**: System shall calculate depreciation from 1st or 15th of month asset increases

**FR-4.2.4**: System shall stop depreciation when asset decreases

**FR-4.2.5**: System shall not depreciate:
- Assets under construction
- Assets pending liquidation
- Assets unused > 9 months (tax deduction restriction)

### 4.3 Depreciation Posting

**FR-4.3.1**: System shall post monthly depreciation to:
- Account 623 (Production depreciation)
- Account 627 (Equipment depreciation)
- Account 641 (Management depreciation)
- Account 642 (Commercial depreciation)
- Account 2141 (Accumulated depreciation - tangible)

**FR-4.3.2**: System shall support batch posting for all assets

**FR-4.3.3**: System shall generate depreciation journal entries

### 4.4 Asset Transfer

**FR-4.4.1**: System shall record asset transfers with:
- Transfer date
- Source location/department
- Target location/department
- Transfer reason
- Approver authorization

**FR-4.4.2**: System shall update asset location/department

**FR-4.4.3**: System shall create transfer document

### 4.5 Asset Liquidation

**FR-4.5.1**: System shall support liquidation workflow:
- Request submission with reason
- Approval workflow
- Gain/loss calculation
- Accounting entry generation

**FR-4.5.2**: System shall calculate liquidation gain/loss:
- Gain/Loss = Selling Price - (Original Cost - Accumulated Depreciation) - Liquidation Costs

**FR-4.5.3**: System shall post liquidation to:
- Account 336/411 (Remaining value)
- Account 2141 (Reverse accumulated depreciation)
- Account 211 (Reverse original cost)
- Account 515/632 (Gain/loss)

### 4.6 Asset Revaluation

**FR-4.6.1**: System shall support government-mandated revaluation

**FR-4.6.2**: System shall adjust original cost, accumulated depreciation, and residual value

**FR-4.6.3**: System shall record revaluation difference per regulations

### 4.7 Reporting

**FR-4.7.1**: System shall generate:
- Asset register (by category, location, department)
- Depreciation schedule (monthly, yearly)
- Depreciation expense report
- Asset movement report
- Liquidation report

**FR-4.7.2**: System shall support export to Excel/PDF

---

## 5. Non-Functional Requirements

### 5.1 Performance
- Asset list query: < 2 seconds
- Depreciation calculation: < 5 seconds per 1000 assets
- Report generation: < 10 seconds

### 5.2 Data Integrity
- All asset records must have audit trail
- Depreciation calculations must be deterministic
- Financial data must be accurate to VND

### 5.3 Compliance
- All depreciation methods per Circular 45
- Account structure per Circular 99
- Financial disclosures per VAS 03

---

## 6. Data Model

### 6.1 FixedAsset Table

| Field | Type | Description |
|-------|------|-------------|
| id | TEXT PK | SHA-256 hash |
| code | TEXT | Auto-generated (FA-XXXXX) |
| name | TEXT | Asset name |
| category | TEXT | Asset type (6 types) |
| description | TEXT | Description |
| original_cost | INTEGER | Original cost (VND cents) |
| residual_value | INTEGER | Estimated liquidation value |
| accumulated_depr | INTEGER | Accumulated depreciation |
| depreciation_method | TEXT | straight_line/declining_balance/units_of_output |
| useful_life_months | INTEGER | Useful life in months |
| depreciation_rate | INTEGER | Rate in basis points |
| monthly_depr_amount | INTEGER | Calculated monthly amount |
| asset_type | TEXT | Classification per Circular 45 |
| asset_group | TEXT | Group within type |
| state | TEXT | active/inactive/scrapped/sold/pending_liquidation |
| purchase_date | TEXT | Purchase date |
| in_service_date | TEXT | In-service date |
| last_review_date | TEXT | Last useful life review |
| vendor_name | TEXT | Vendor name |
| purchase_order_no | TEXT | PO number |
| invoice_no | TEXT | Invoice number |
| serial_no | TEXT | Serial number |
| barcode | TEXT | Barcode/QR |
| account_code_211 | TEXT | GL account for cost |
| account_code_214 | TEXT | GL account for depreciation |
| location | TEXT | Physical location |
| department | TEXT | Department |
| created_by | TEXT | Created by user |
| created_at | TEXT | Creation timestamp |
| updated_by | TEXT | Updated by user |
| updated_at | TEXT | Update timestamp |

### 6.2 DepreciationEntry Table

| Field | Type | Description |
|-------|------|-------------|
| id | TEXT PK | Entry ID |
| asset_id | TEXT FK | Asset reference |
| period | TEXT | Period (YYYY-MM) |
| depreciation_method | TEXT | Method used |
| amount | INTEGER | Depreciation amount |
| accumulated_depr | INTEGER | Running total |
| account_debit | TEXT | Debit account |
| account_credit | TEXT | Credit account |
| posted | BOOLEAN | Posted to GL |
| created_at | TEXT | Creation timestamp |

### 6.3 AssetTransfer Table

| Field | Type | Description |
|-------|------|-------------|
| id | TEXT PK | Transfer ID |
| asset_id | TEXT FK | Asset reference |
| transfer_date | TEXT | Transfer date |
| from_location | TEXT | Source location |
| to_location | TEXT | Target location |
| from_department | TEXT | Source department |
| to_department | TEXT | Target department |
| reason | TEXT | Transfer reason |
| approved_by | TEXT | Approver |
| created_by | TEXT | Requester |
| created_at | TEXT | Creation timestamp |

### 6.4 AssetLiquidation Table

| Field | Type | Description |
|-------|------|-------------|
| id | TEXT PK | Liquidation ID |
| asset_id | TEXT FK | Asset reference |
| request_date | TEXT | Request date |
| reason | TEXT | Liquidation reason |
| selling_price | INTEGER | Selling price |
| liquidation_costs | INTEGER | Costs |
| gain_loss | INTEGER | Calculated gain/loss |
| status | TEXT | requested/approved/executed/rejected |
| requested_by | TEXT | Requester |
| approved_by | TEXT | Approver |
| executed_at | TEXT | Execution timestamp |
| created_at | TEXT | Creation timestamp |

---

## 7. Integration Points

### 7.1 General Ledger
- Account 211: Original cost
- Account 2141: Accumulated depreciation
- Account 623/627/641/642: Depreciation expense
- Account 336/411: Remaining value on disposal

### 7.2 Tax Reporting
- Depreciation for tax deduction
- Asset register for tax inspection
- Liquidation gain/loss for CIT

### 7.3 Authorization (Casbin)
- Role-based access control
- Approval workflows for liquidation/revaluation
- Audit trail for all changes

---

## 8. Acceptance Criteria

### 8.1 Asset Creation
- [ ] Asset created with valid data
- [ ] Code auto-generated
- [ ] Validation errors returned for invalid data
- [ ] Account 211 posted

### 8.2 Depreciation
- [ ] Monthly depreciation calculated correctly
- [ ] Three methods produce correct results
- [ ] Depreciation posted to correct accounts
- [ ] Schedule generated accurately

### 8.3 Transfer
- [ ] Asset location updated
- [ ] Transfer record created
- [ ] Audit trail maintained

### 8.4 Liquidation
- [ ] Workflow executed correctly
- [ ] Gain/loss calculated accurately
- [ ] Accounting entries correct
- [ ] Asset state updated

---

## 9. Dependencies

- Circular 45/2013/TT-BTC (depreciation framework)
- Circular 30/2025/TT-BTC (updates)
- Circular 99/2025/TT-BTC (account structure)
- VAS 03 (accounting standards)
- VBHN 12/VBHN-BTC 2025 (consolidated)

---

## 10. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Non-compliance with Circular 45 | Legal/Financial | Full implementation per requirements |
| Incorrect depreciation calculation | Financial | Extensive testing with known values |
| Missing disclosures in financial statements | Audit finding | VAS 03 checklist verification |
| Integration errors with GL | Financial | Reconciliation reports |

---

## 11. Approval

| Role | Name | Date |
|------|------|------|
| BA Lead | _____________ | _____________ |
| Chief Accountant | _____________ | _____________ |
| Project Manager | _____________ | _____________ |

---

*Document ends.*
