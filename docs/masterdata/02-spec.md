# Master Data Module (Hệ thống danh mục dữ liệu chính) — Technical Specification

> Target state. Current masterdata packages are stubs. Money model: **integer
> minor units (`int64`, VND)** — reuse `core.Money`/`AmountMinor` for any
> monetary field (e.g. nợ tối đa, nguyên giá) exactly as cash/ledger do.

## 1. Architecture position

```
 all consumer modules ──┐   internal/application/masterdata (Service)
 cash/ledger/invoice    │         │ validate / lifecycle / merge
 purchase/sales/payroll │         ▼
 fixedasset/inventory┘  └─  internal/domain/masterdata (entities + Registry)
                                         │
                      internal/infrastructure/persistence/masterdata (SqliteRepository)
                                         │
                       internal/infrastructure/db (migrate.go — tables)
                                         ▼
                     internal/interfaces/http/masterdata (Handler /api/v1/master-data/...)
                                         │
                             web/templates + web UI (danh mục screens)
```

Manual entry path (handler → service → repo) mirrors the cash module reference
implementation. The service owns validation + sequence + lifecycle; the repo is
a thin JSON-doc store. **No module bypasses the registry** — a consumer may only
*read* catalogs or ask "is this code referenced?".

## 2. Domain model

### 2.1 Catalog type registry (built-in, seeded)
```go
type CatalogKind string
const (
    CatalogAccount        CatalogKind = "account"        // sơ đồ tài khoản
    CatalogCustomer       CatalogKind = "customer"       // khách hàng
    CatalogSupplier       CatalogKind = "supplier"       // nhà cung cấp
    CatalogItem           CatalogKind = "item"           // vật tư - hàng hóa
    CatalogUnit           CatalogKind = "unit"           // đơn vị tính
    CatalogWarehouse      CatalogKind = "warehouse"      // kho
    CatalogDepartment     CatalogKind = "department"     // phòng ban
    CatalogEmployee       CatalogKind = "employee"       // nhân viên
    CatalogFixedAssetCat  CatalogKind = "fixed_asset_cat"// nhóm TSCĐ
    CatalogBank           CatalogKind = "bank"           // ngân hàng
    CatalogCurrency       CatalogKind = "currency"       // ngoại tệ
    CatalogTaxRate        CatalogKind = "tax_rate"       // thuế suất (GTGT/TNDN)
    CatalogFund           CatalogKind = "fund"           // quỹ (links cash)
    CatalogCostObject     CatalogKind = "cost_object"    // đối tượng tập hợp chi phí
    CatalogReason         CatalogKind = "reason"         // lý do thu/chi
    CatalogCustomerGroup  CatalogKind = "customer_group" // nhóm khách hàng
    CatalogItemGroup      CatalogKind = "item_group"     // nhóm vật tư
    CatalogSupplierGroup  CatalogKind = "supplier_group" // nhóm nhà cung cấp
)
```

### 2.2 Common master record (base of every typed entity)
```go
type MasterRecord struct {
    ID          string        // deterministic SHA-256 of (kind, code)
    Kind        CatalogKind
    Code        string        // unique per kind, immutable after reference
    Name        string        // VN name (required)
    NameEN      string        // optional EN name
    GroupCode   string        // parent/group code (hierarchy), "" if root
    Status      core.Status   // ACTIVE | INACTIVE ("đang sử dụng"/"ngừng sử dụng")
    ValidFrom   time.Time     // effective-from (tax rates, tỷ giá, accounts)
    ValidTo     time.Time     // zero = open-ended
    ReferenceCount int64      // cache of referencing transactions (guard flag)
    CreatedBy   string
    CreatedAt   time.Time
    UpdatedBy   string
    UpdatedAt   time.Time
    DeactivatedBy string     // set on ngừng sử dụng
    DeactivatedAt time.Time
    DeactivateReason string  // required for override-deactivate
    Extras      map[string]string // per-kind schema (see §2.3), JSON
}
```

### 2.3 Typed schemas (the "smart" surface — validated per kind)
```go
// Khách hàng / Nhà cung cấp (Đối tượng) — NĐ 254/2026, Luật QLT 108/2025
type Counterparty struct {
    MasterRecord
    IsOrganization   bool   // Tổ chức vs Cá nhân
    IsSupplier       bool   // vừa khách hàng vừa nhà cung cấp
    TaxCode          string // MST: 10-digit, or 13-digit (đơn vị phụ thuộc)
    BudgetUnitCode   string // mã ĐVQHNS (NĐ 254/2026 Điều 10)
    IdNumber         string // số định danh cá nhân / CCCD (individuals)
    PassportNo       string // hộ chiếu + quốc tịch (foreign buyers)
    Nationality      string
    Address          string // per giấy chứng nhận ĐKDN (must match)
    GroupCode        string // nhóm khách hàng / nhà cung cấp
    TermsDays        int    // số ngày được nợ (điều khoản thanh toán)
    CreditLimitMinor int64  // nợ tối đa (VND minor)
    ARAccountCode    string // TK công nợ phải thu, default 131
    APAccountCode    string // TK công nợ phải trả, default 331
    EmployeeCode     string // nhân viên phụ trách
    InvoiceRecipient string // người nhận hóa đơn điện tử
    BankCode         string // tài khoản ngân hàng (thanh toán)
    Phone, Email, Website string
}
```

```go
// Sơ đồ tài khoản — Phụ lục 2 TT 99/2025 (or TT 133/2016 variant)
type Account struct {
    MasterRecord
    Regime      string      // "TT99-2025" | "TT133-2016"
    Type        AccountType // ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    Level       int         // 1..4+
    ParentCode  string
    AllowPost   bool        // false = parent/summary (không hạch toán trực tiếp)
    ForeignOnly bool        // only for ngoại tệ accounts (optional)
    ClosedAt    time.Time   // set when locked for the year
}
```

```go
// Vật tư - hàng hóa
type Item struct {
    MasterRecord
    ItemType     string // "21" Vật tư, "31" CCLĐ, "41" Bán thành phẩm,
                        // "51" Thành phẩm, "61" Hàng hóa (Fast-compatible)
    UnitCode     string // đơn vị tính gốc
    VatRateCode  string // thuế suất GTGT mặc định
    InvAccount   string // TK kho (152/156/153)
    CostAccount  string // TK giá vốn (632)
    RevenueAccount string // TK doanh thu (511)
    CostMethod   string // "AVG" bình quân | "FIFO" | "SPECIFIC" đích danh
    MinStockQty  float64
    WarrantyMonths int
    Barcode, Spec string
}
```

```go
// Thuế suất (versioned by ValidFrom/ValidTo)
type TaxRate struct {
    MasterRecord
    TaxKind string // "GTGT" | "TNDN"
    Rate    int32  // basis points: 0, 500, 800, 1000 → 0%, 5%, 8%, 10%
}
```

Others (Employee, Unit, Warehouse, Department, Bank, Currency, FixedAssetCat,
Fund, CostObject, Reason, groups) use `MasterRecord` + a small typed view
(e.g. Employee carries IdNumber + DepartmentCode + WageRateBasis; Bank carries
AccountNo + SwiftCode). TSCĐ catalog only — depreciation engine lives in the
fixedasset module.

## 3. Invariants (rules enforced in the service layer)

| R# | Rule | Consequence if violated |
|---|---|---|
| R1 | `Code` unique per `Kind` | Rejected 409 (dup code) |
| R2 | `Code` immutable once `ReferenceCount > 0` (or once used in any transaction) | Rejected 409 |
| R3 | Hard delete only when `ReferenceCount == 0`; otherwise soft deactivate | Rejected 409 "có phát sinh" |
| R4 | Deactivate (ngừng sử dụng) requires `ReferenceCount == 0` unless `ke_toan_truong` override + reason | Rejected 403/422 |
| R5 | Name required; Code required; group must exist and be same kind; no cycles in group tree | Rejected 422 |
| R6 | TaxCode format: 10-digit, or 13 with "-" separator; individuals may use số định danh cá nhân instead | Rejected 422 |
| R7 | For KH/NCC: if invoice consumption enabled, one of TaxCode | BudgetUnitCode | IdNumber | PassportNo required (NĐ 254/2026) | Rejected 422 |
| R8 | Account: parent exists, Level = parent.Level+1, only leaves AllowPost, type consistent, regime consistent | Rejected 422 |
| R9 | Group-tree: no self-parent, no cycles, max depth (e.g. 5) | Rejected 422 |
| R10 | Versioned records (tax rates, currency FX) must not overlap `ValidFrom..ValidTo` for same (Kind, Code) | Rejected 422 |
| R11 | Merge allowed only same kind; dry-run impact count shown; references re-pointed in one transaction | Rejected/confirmed via audit |
| R12 | Sequence allocation atomic; auto-code numbers never reused after deactivate | Sequence exception |

## 4. API surface (`/api/v1/master-data`)

All mutations 201/200 with full resource JSON; validation 422 with VN+EN
messages; dup 409; authz 403 via existing middleware.

| Method | Path | Action | Role |
|---|---|---|---|
| GET | `/api/v1/master-data/catalogs` | list catalog kinds (registry) | read roles |
| GET | `/api/v1/master-data/:kind` | list records (q, group, status, page, size) | read roles |
| GET | `/api/v1/master-data/:kind/:code` | detail | read roles |
| POST | `/api/v1/master-data/:kind` | create | danh_muc, ke_toan_tong_hop |
| PUT | `/api/v1/master-data/:kind/:code` | update (name/fields; not Code once referenced) | danh_muc, ke_toan_tong_hop |
| POST | `/api/v1/master-data/:kind/:code/deactivate` | ngừng sử dụng (+reason; override flag for chief) | danh_muc; ke_toan_truong override |
| POST | `/api/v1/master-data/:kind/:code/activate` | re-enable | danh_muc |
| DELETE | `/api/v1/master-data/:kind/:code` | hard delete (only if ReferenceCount==0) | danh_muc |
| POST | `/api/v1/master-data/:kind/merge` | merge dupes `{keep, merge}` with dry-run option | ke_toan_truong |
| POST | `/api/v1/master-data/import` | Excel import (dry-run | commit); returns job + error report | danh_muc |
| GET | `/api/v1/master-data/import/:jobId/report` | import error report / status | danh_muc |
| GET | `/api/v1/master-data/:kind/export` | Excel/CSV export (filters) | read roles |
| GET | `/api/v1/master-data/:kind/:code/references` | reference count + sample refs (guard) | read roles |
| GET | `/api/v1/master-data/accounts/seed` | TT 99 default chart (preview) | ke_toan_truong |
| POST | `/api/v1/master-data/accounts/seed` | apply seed + Quy chế hạch toán note | ke_toan_truong |
| POST | `/api/v1/master-data/regime` | switch TT 99 ↔ TT 133 variant | ke_toan_truong |

Internal (used by consumers via a Go interface, not HTTP): `Lookup(kind, code)`,
`Resolve(kind, q)`, `ReferenceCount(kind, code)`.

## 5. Data flows

### 5.1 Create customer (khách hàng)
```
web form → POST /master-data/customer
  → validate R1–R9 (code unique, group exists, MST format, NĐ 254 identity set)
  → assign auto-code KH-00001 (R12, BEGIN IMMEDIATE sequence)
  → insert row (JSON-doc) + audit_logs append
  → 201 with full record
```

### 5.2 Deactivate with guard (ngừng sử dụng)
```
POST /master-data/customer/CUS-001/deactivate
  → ReferenceCount==0? yes → status INACTIVE + audit
  → no → 409 "có công nợ/phát sinh"
  → ke_toan_truong override + reason → INACTIVE + audit (forced)
```

### 5.3 Merge duplicates (gộp đối tượng)
```
POST /master-data/customer/merge {keep: CUS-001, merge: [CUS-009, CUS-010]}
  → dry-run? return impact counts (references per record)
  → commit: one tx { re-point refs in all tables → set merged records INACTIVE
    with reason "Đã gộp vào CUS-001" → audit } 
  → re-compute ReferenceCount of keep
```

### 5.4 Excel import
```
upload .xlsx (template v2) → POST /master-data/import {dryRun:true}
  → validate each row (R1–R10); collect per-row errors → job report
  → dryRun:false → one tx per batch, idempotent by (kind, code) upsert
  → job report: success count, error rows with reason + row number
```

### 5.5 Chart-of-accounts seed (TT 99/2025)
```
POST /master-data/accounts/seed (ke_toan_truong)
  → tx { insert Phụ lục 2 accounts (hierarchy, type, AllowPost) idempotently
    → write Quy chế hạch toán audit note (Điều 11) }
```

## 6. Persistence (db.Migrate additions, JSON-doc pattern)

Add to the fixed table list in `internal/infrastructure/db/migrate.go`:
`md_records`, `md_sequences`, `md_import_jobs`, `md_regimes`. `catalog_items`
may be dropped or kept as a v1-only table (recommend: leave in place, unused,
and document deprecation — do not ALTER/DROP in a migration-tool-free repo).

- `md_records`: `(id, data)` — row = serialized `MasterRecord` + typed view.
  Secondary index column `kind_code` = `{kind}:{code}` for unique enforcement
  and fast lookups (the repo allows per-module index columns; benchmark first).
- `md_sequences`: rows `(kind, prefix)` → next auto-code number (R12), atomically
  incremented under `BEGIN IMMEDIATE`.
- `md_import_jobs`: job status, counters, error report rows (JSON).
- `md_regimes`: current accounting regime (TT 99/2025 vs TT 133/2016) + history.

## 7. Money & data quality

- Monetary fields (CreditLimitMinor, MaxBalance, nguyên giá) are `int64` VND
  minor units — no floats in masterdata paths.
- `ReferenceCount` is a **cache** maintained transactionally on every consumer
  write; authoritative check on deactivate re-scans the consumer tables.
- No free-text "MST" — always validate format (R6) and normalize (strip spaces).

## 8. Sequencing, idempotency & concurrency

- Auto-code format `{prefix}{5-digit}` e.g. `KH-00001`, `NCC-00001`,
  `VT-00001`; prefix configured at company setup (Q1), immutable afterwards.
- Concurrent creates: single SQLite writer; `md_sequences` increment + insert in
  one tx → unique code guaranteed (R12).
- Import idempotency key = `(kind, code)`: upsert semantics with a dry-run pass.

## 9. Authz wiring

Extend `SeedDefaultPolicies` (internal/infrastructure/authorization/enforcer.go)
with the role→route map in §4 (matcher uses `c.FullPath()` + method already).
Roles: `danh_muc`, plus reuse `ke_toan_tong_hop`, `ke_toan_truong`,
`giam_doc`, `kiem_toan`. Admin keeps `* *`. `:kind/deactivate` override and
`/merge` and `/accounts/seed` and `/regime` gated to `ke_toan_truong`; hard
delete and import gated to `danh_muc`+`ke_toan_tong_hop`; reads open to all
authenticated roles.
