# Tools / Equipments Module — PROD-Readiness Verdict

> **Author**: BA Lead (20+ yrs) + Chief Accountant (20+ yrs, CPA Vietnam)
> **Date**: 2026-08-29
> **Modules Analyzed**: `tools` (Tool Cards) + `fixedasset` (TSCĐ)
> **Status**: **NOT PRODUCTION-READY** (both modules)

---

## 1. Executive Summary

The goGL codebase contains **two distinct modules** related to physical assets:

### Module A: `tools` — Tool Card Tracker
- **Purpose**: Track physical tools (drills, hammers, measuring instruments)
- **Location**: `/api/v1/tools/cards`
- **Status**: Functional CRUD — but **NOT accounting-compliant**
- **Verdict**: **NOT PROD-READY** for any accounting use

### Module B: `fixedasset` — Fixed Assets (TSCĐ)
- **Purpose**: Manage tangible fixed assets per Vietnamese accounting standards
- **Location**: `/api/v1/fixed-assets`
- **Status**: Partially implemented — entity + depreciation engine + basic CRUD
- **Verdict**: **NOT PROD-READY** — missing GL integration, reports, UI, approval workflows

---

## 2. Module A: Tools (Tool Card Tracker) — Analysis

### 2.1 Current Implementation

| Layer | Status |
|-------|--------|
| Entity | ✅ Simple ToolCard (name, brand, model, serial, price, location, department, state) |
| Service | ✅ CRUD + Scrap |
| Repository | ✅ SQLite JSON-doc |
| HTTP | ✅ REST API |

### 2.2 GAP Analysis

| Requirement | Status |
|-------------|--------|
| Vietnamese accounting compliance | ❌ NONE |
| GL account mapping | ❌ NONE |
| Depreciation | ❌ NONE |
| Cost threshold (≥30M VND) | ❌ NONE |
| Asset classification (Circular 45) | ❌ NONE |
| Approval workflows | ❌ NONE |
| Audit trail | ❌ NONE |
| Financial reporting | ❌ NONE |
| Web UI | ❌ NONE |

### 2.3 Verdict for `tools`

**This module cannot operate in PROD for accounting purposes.** It is a facility management tracker, not an accounting module. For Vietnamese enterprises, physical tools above 30M VND must be tracked as Fixed Assets (TSCĐ) per Circular 45. Tools below 30M VND are tracked as "Dụng cụ" (supplies) in Account 153/156 — which this module also doesn't support.

**Recommendation**: Either:
1. Deprecate this module entirely and merge its use case into `fixedasset`, OR
2. Keep it as a non-accounting facility management tool (clearly separated from accounting modules)

---

## 3. Module B: Fixed Assets (TSCĐ) — Analysis

### 3.1 Current Implementation

| Layer | File | Status |
|-------|------|--------|
| Entity | `internal/domain/fixedasset/entity.go` | ✅ **Good** — Full entity model |
| Depreciation | `internal/application/fixedasset/depreciation.go` | ✅ **Partial** — SL + DB engines |
| Service | `internal/application/fixedasset/service.go` | ✅ **Partial** — CRUD + Transfer + Liquidation |
| Repository | `internal/infrastructure/persistence/fixedasset/repository.go` | ✅ Basic CRUD |
| HTTP | `internal/interfaces/http/fixedasset/handler.go` | ✅ Basic REST |

### 3.2 What's Implemented Well

**Entity Model** (entity.go):
- ✅ 6 Asset Types per Circular 45 Art 6 (housing, machinery, transport, tools, perennial, other)
- ✅ 3 Depreciation Methods (straight_line, declining_balance, units_of_output)
- ✅ 5 Asset States (active, inactive, scrapped, sold, pending_liquidation)
- ✅ Original Cost, Residual Value, Accumulated Depreciation
- ✅ Useful Life (months)
- ✅ GL Account fields (AccountCode211, AccountCode214)
- ✅ Purchase Date, In-Service Date, Last Review Date
- ✅ Vendor, PO, Invoice, Serial tracking
- ✅ Cost threshold validation (≥30M VND)
- ✅ Useful life range validation (Annex I, Circular 45)
- ✅ CurrentValue() and IsFullyDepreciated() methods

**Depreciation Engine** (depreciation.go):
- ✅ Straight-line calculation
- ✅ Declining-balance calculation
- ✅ MonthlyDepreciation() method
- ✅ CalculatePeriodDepreciation() for multi-month runs
- ✅ Fully-depreciated check
- ✅ Max depreciation cap (depreciable value - accumulated)

**Service** (service.go):
- ✅ Create with code generation (FA-XXXXX)
- ✅ Get, Update, Delete, List
- ✅ Transfer (location/department update)
- ✅ Liquidate → PendingLiquidation
- ✅ ConfirmLiquidation → Scrapped/Sold
- ✅ Deactivate/Reactivate

### 3.3 GAP Analysis — Missing for PROD

| # | Gap | Impact | Standard |
|---|-----|--------|----------|
| G1 | **No GL Integration** — Depreciation not posted to accounts 623-642 → 2141 | CRITICAL | Circular 99/2025 |
| G2 | **No Depreciation Schedule Repository** — Can't store/query depreciation history | CRITICAL | VAS 03 |
| G3 | **No Monthly Batch Processor** — No way to run depreciation for all assets in a period | CRITICAL | Circular 45 |
| G4 | **No Approval Workflows** — Liquidation has no approval chain | HIGH | Enterprise governance |
| G5 | **No Audit Trail** — No audit logging for mutations | HIGH | Luật Kế toán 88/2015 |
| G6 | **No Asset Revaluation** — Government-mandated revaluations not supported | HIGH | VAS 03 Para 35-36 |
| G7 | **No Transfer Documents** — No "biên bản điều chuyển" generation | MEDIUM | Circular 45 |
| G8 | **No Reporting** — No asset register, depreciation report, schedule report | CRITICAL | VAS 03 Para 39 |
| G9 | **No Web UI** — Zero web interface | HIGH | UX requirement |
| G10 | **No Excel/PDF Export** — Reports can't be exported | MEDIUM | Business requirement |
| G11 | **No Units-of-Output Logic** — Method defined but falls back to straight-line | MEDIUM | Circular 45 Art 12 |
| G12 | **No Mid-Month Proration** — Depreciation doesn't handle 1st/15th start rules | MEDIUM | Circular 45 Art 8 |
| G13 | **No 9-Month Unused Rule** — No exclusion for assets unused >9 months | MEDIUM | Circular 45 Art 9 |
| G14 | **No Barcode/QR Support** — Physical inventory tracking missing | LOW | Operational |
| G15 | **No Authorization (Casbin)** — No role-based access control | HIGH | Security |

### 3.4 Can `fixedasset` Module Operate in PROD?

**NO — Not yet.** The entity and depreciation engine are solid foundations, but the module lacks:

1. **GL Integration** — Without posting depreciation to accounts, the module is disconnected from the accounting system. A Chief Accountant cannot use it.
2. **Depreciation Schedule** — No way to store or query historical depreciation runs.
3. **Monthly Batch Processing** — No ability to run depreciation for all assets in a period.
4. **Reports** — No statutory reports (asset register, depreciation schedule, etc.)
5. **Web UI** — No interface for end users.

**Risk Level: HIGH** — The module has good bones but is incomplete for production use.

---

## 4. Competitor Comparison ( Vietnamese Market)

| Feature | MISA AMIS | Fast Accounting | Bravo ERP | goGL `fixedasset` |
|---------|-----------|-----------------|-----------|-------------------|
| Asset card management | ✅ | ✅ | ✅ | ✅ |
| 3 depreciation methods | ✅ | ✅ | ✅ | ⚠️ Partial (UOO not implemented) |
| GL integration | ✅ | ✅ | ✅ | ❌ |
| Depreciation schedule | ✅ | ✅ | ✅ | ❌ |
| Monthly batch processing | ✅ | ✅ | ✅ | ❌ |
| Approval workflows | ✅ | ✅ | ✅ | ❌ |
| Asset register report | ✅ | ✅ | ✅ | ❌ |
| Depreciation report | ✅ | ✅ | ✅ | ❌ |
| Excel export | ✅ | ✅ | ✅ | ❌ |
| Barcode/QR | ✅ | ✅ | ✅ | ❌ |
| Asset revaluation | ✅ | ✅ | ✅ | ❌ |
| Web UI | ✅ | ✅ | ✅ | ❌ |

---

## 5. Key Points — What Must Change

### 5.1 For BA Lead (20+ years)

1. **Consolidate**: The `tools` module is redundant. Its use case (facility tool tracking) should be absorbed by `fixedasset` for items ≥30M VND, or kept as a separate non-accounting module with clear separation.

2. **Priority order for `fixedasset`**:
   - P0: GL Integration (depreciation posting)
   - P0: Depreciation Schedule Repository
   - P0: Monthly Batch Processor
   - P1: Reports (asset register, depreciation schedule, depreciation expense)
   - P1: Web UI
   - P2: Approval workflows
   - P2: Audit trail
   - P3: Revaluation, barcode, export

3. **Existing BRD is good**: `docs/BRD-FixedAsset-Module.md` and `docs/UseCases-FixedAsset.md` already define the target state. Implementation should follow these.

### 5.2 For Chief Accountant (20+ years, CPA Vietnam)

1. **GL integration is non-negotiable**: Without Account 211/2141/623-642 posting, the module cannot produce financial statements. This is the #1 blocker.

2. **Depreciation schedule is mandatory**: Tax auditors require a complete depreciation history. The current engine calculates but doesn't store results.

3. **Missing approvals**: Liquidation of assets worth >500M VND requires Director approval per enterprise governance. The current code has no approval chain.

4. **Missing reports**: The Statutory Asset Register (Bảng kê TSCĐ) and Depreciation Schedule (Lịch khấu hao) are required for annual financial statements per VAS 03 Para 39.

---

## 6. Implementation Roadmap

### Phase 1: Depreciation Schedule Repository (Week 1)
- [ ] Create `depreciation_entries` table (migration)
- [ ] Implement `DepreciationEntry` entity (id, asset_id, period, method, amount, accumulated, account_debit, account_credit, posted, created_at)
- [ ] Implement repository (CRUD + list by asset/period)
- [ ] Unit tests

### Phase 2: Monthly Batch Processor (Week 2)
- [ ] Implement `RunMonthlyDepreciation(period)` service method
- [ ] Iterate all active assets
- [ ] Calculate depreciation per asset
- [ ] Store entries in depreciation_entries table
- [ ] Return summary for review
- [ ] Unit tests with known values

### Phase 3: GL Integration (Week 2-3)
- [ ] Implement depreciation posting to GL:
  - Debit: Account 623/627/641/642 (based on department)
  - Credit: Account 2141
- [ ] Implement `PostDepreciation(period, actor)` method
- [ ] Update accumulated_depr on each asset
- [ ] Integration tests with ledger module

### Phase 4: Reports (Week 3-4)
- [ ] Asset Register Report (Bảng kê TSCĐ)
- [ ] Depreciation Schedule Report (Lịch khấu hao)
- [ ] Depreciation Expense Report (Báo cáo chi phí khấu hao)
- [ ] Monthly Depreciation Summary

### Phase 5: Web UI (Week 4-5)
- [ ] Asset List page (with filters)
- [ ] Asset Detail page
- [ ] Asset Creation form
- [ ] Depreciation Schedule view
- [ ] Monthly Depreciation Run page
- [ ] Transfer form
- [ ] Liquidation form (with approval workflow)

### Phase 6: Approval & Audit (Week 5-6)
- [ ] Liquidation approval workflow (request → approve → execute)
- [ ] Casbin authorization policies
- [ ] Audit trail for all mutations
- [ ] Asset revaluation support

### Phase 7: Export & Polish (Week 6-7)
- [ ] Excel export for all reports
- [ ] PDF export
- [ ] Barcode/QR generation
- [ ] Performance optimization

---

## 7. Recommendation

1. **Merge or separate `tools`**: Decide whether `tools` stays as a non-accounting facility tracker or gets absorbed into `fixedasset`. For a Vietnamese ERP, I recommend **keeping them separate** — `tools` for non-accounting asset tracking, `fixedasset` for accounting-grade TSCĐ management.

2. **Focus on `fixedasset`**: The entity and depreciation engine are solid. Prioritize GL integration and depreciation schedule storage — these are the critical path to PROD readiness.

3. **Follow existing BRD**: `docs/BRD-FixedAsset-Module.md` already defines the target state. Implementation should follow the phased roadmap above.

4. **Target PROD-ready**: 7 weeks from start (vs current 0% PROD readiness).

---

*Analysis complete. Both modules need significant work before PROD deployment.*
