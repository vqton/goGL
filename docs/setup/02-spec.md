# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — Technical Specification

> Target state. Current setup packages are stubs. Money model: **integer minor
> units (`int64`, VND)** — reuse `core.Money`/`AmountMinor` exactly as
> cash/ledger do. Dates: `YYYY-MM-DD`; periods `YYYY-MM`.

## 1. Architecture position

```
 web wizard / status page (web/templates/setup/*)
        │
 internal/interfaces/http/setup (Handler /api/v1/setup/...)
        │
 internal/application/setup (Service — ORCHESTRATOR)
        │ validate profile/MST / regime / balance-check / lock/reopen
        ▼
 internal/domain/setup (CompanyProfile, OpeningBalance, SetupStatus, Repository)
        │
 internal/infrastructure/persistence/setup (SqliteRepository — tables
        │   company_profiles, opening_balances, setup_status)
        ▼
 ── cross-module seams (Go interfaces, NOT HTTP) ──
 masterdata.SetRegime / SeedAccounts / GetRegime
 ledger.OpenPeriod / ListPeriods / SeedDefaultAccounts (already in main.go)
 audit append (audit_logs)      authz via Casbin middleware
```

Setup is deliberately **thin**: it owns the company identity + opening balances
+ status machine, and **orchestrates** the already-implemented masterdata and
ledger modules. It must not hold a private copy of the chart of accounts.

## 2. Domain model

### 2.1 CompanyProfile (statutory identity — Điều 5/11/31 TT 99, NĐ 254/2026)
```go
type CompanyProfile struct {
    ID               string   // deterministic "company" row id
    Name             string   // tên (theo ĐKDN) — required
    NameEN           string   // tên tiếng Anh (optional)
    TaxCode          string   // MST 10-số; 13-số "ddd-dddddd-ddd-ddd" đơn vị phụ thuộc
    BudgetUnitCode   string   // mã ĐVQHNS (optional; NĐ 254/2026)
    Address          string   // địa chỉ trụ sở theo ĐKDN — seller identity
    LegalRepresentative string // người đại diện theo pháp luật
    CompanyType      string   // "TNHH" | "CP" | "DNTN" | "HTX" | "LIEN_DOANH" | ...
    Industry         string   // ngành nghề (VSIC-ish, free text v1)
    AccountingCurrency string // default "VND" (Điều 5 TT 99)
    FiscalYearStart  string   // "YYYY-MM-DD" — kỳ kế toán năm = 12 tháng (Điều 12)
    AccountingRegime string   // "TT99-2025" | "TT133-2016"
    BooksStartDate   string   // ngày bắt đầu hạch toán
    Status           SetupStatus
    CreatedBy, CreatedAt, UpdatedBy, UpdatedAt string
}
```

### 2.2 SetupStatus — the state machine
```go
type SetupStatus string
const (
    StatusEmpty          SetupStatus = "empty"            // nothing yet
    StatusProfiled       SetupStatus = "profiled"         // profile saved
    StatusRegimeSet      SetupStatus = "regime_set"       // masterdata regime set
    StatusAccountsSeeded SetupStatus = "accounts_seeded"  // COA seeded
    StatusPeriodsOpen    SetupStatus = "periods_open"     // ledger FY periods opened
    StatusBalancesDraft  SetupStatus = "balances_draft"   // opening balances entered
    StatusBalancesLocked SetupStatus = "balances_locked"  // approved/locked
    StatusActive         SetupStatus = "active"           // go live (period 1 postable)
)
```

### 2.3 OpeningBalance (per TK + per đối tượng)
```go
type OpeningBalance struct {
    ID          string      // deterministic sha256("OB\x00"+account+"\x00"+object) hex 
    AccountCode string      // TK — must exist, ACTIVE, AllowPost (leaf)
    ObjectType  string      // "" | "customer" | "supplier" | "item" | "fixed_asset"
    ObjectCode  string      // mã đối tượng (required when ObjectType set)
    Period      core.Period // From = To = first day of FY (or BooksStartDate)
    Debit       core.Money  // Nợ — exactly one side non-zero (R7)
    Credit      core.Money  // Có
    Currency    string      // VND (v1)
    Status      BalanceStatus // DRAFT | LOCKED
    EnteredBy, EnteredAt, UpdatedBy, UpdatedAt string
}
```

### 2.4 Repository (extends current stub)
```go
type Repository interface {
    SaveProfile(ctx context.Context, p *CompanyProfile) error
    GetProfile(ctx context.Context, id string) (*CompanyProfile, error)
    SaveBalance(ctx context.Context, b *OpeningBalance) error
    ListBalances(ctx context.Context, accountCode string) ([]*OpeningBalance, error)
    DeleteBalance(ctx context.Context, id string) error   // draft state only (R8)
    GetStatus(ctx context.Context) (SetupStatus, error)
    SetStatus(ctx context.Context, s SetupStatus) error
}
```

## 3. Invariants (rules enforced in the service layer)

| R# | Rule | Consequence if violated |
|---|---|---|
| R1 | One company per install; profile row id fixed "company" | 409 "đã khởi tạo" |
| R2 | Fiscal year = 12 months (Luật Kế toán Điều 12); `FiscalYearStart` must be 01 of a month, end = start+12mo−1d | 422 |
| R3 | MST format: 10 digits, or 13 with "-" (đơn vị phụ thuộc); normalized (strip spaces) (Luật QLT 108/2025, TT 86/2024) | 422 "MST không hợp lệ" |
| R4 | `AccountingCurrency` = "VND" in v1 (Điều 5 TT 99) | 422 |
| R5 | Regime ∈ {"TT99-2025","TT133-2016"}; recorded; switch only at FY boundary | 422/409 |
| R6 | Steps idempotent; status never goes backwards; re-run re-checks before re-applying | 409 "trạng thái không cho phép" |
| R7 | Each OpeningBalance has exactly one non-zero side (Debit xor Credit); amounts ≥ 0 | 422 |
| R8 | Edit/delete balance only while Status == BALANCES_DRAFT | 409 "đã khóa" |
| R9 | Balance-check on list/import: Σ Nợ == Σ Có (per currency; VND only v1); empty sheet is NOT balanced (Lock blocked until ≥1 row); `offending` lists the TKs carrying the surplus side, `gaps` lists mandatory-object TKs missing an object | 422 with diff |
| R10 | Object detail: **mandatory on 131** (customer); on 331/152/155/156/211/214 the object is validated **when present** (kind: 331→supplier, 152/155/156→item, 211/214→fixed_asset); object must exist + ACTIVE in masterdata | 422 |
| R11 | Lock (`BALANCES_DRAFT → BALANCES_LOCKED`) only after balance-check passes; only `ke_toan_truong`/`admin` | 403/422 |
| R12 | Reopen after posting: blocked unless no posted voucher references the edited TK; chief override + reason + audit | 409 |
| R13 | Every mutation appends audit (actor, at, reason) | audit enforced in service |

## 4. API surface (`/api/v1/setup`)

All mutations 201/200 with resource JSON; validation 422 with VN+EN messages;
dup 409; authz 403 via existing middleware.

| Method | Path | Action | Role |
|---|---|---|---|
| GET | `/api/v1/setup/status` | current SetupStatus + step checklist | read roles |
| GET | `/api/v1/setup/profile` | company profile (single) | read roles |
| POST | `/api/v1/setup/initialize` | run wizard step: `{profile, regime, fiscalYearStart, seedAccounts, openPeriods}` — idempotent, resumes from current status | ke_toan_truong, admin |
| PUT | `/api/v1/setup/profile` | edit profile while Status < BALANCES_LOCKED | ke_toan_truong |
| POST | `/api/v1/setup/opening-balances` | add/edit one balance (draft) | ke_toan_tong_hop |
| GET | `/api/v1/setup/opening-balances` | list (accountCode filter) + balance-check summary | read roles |
| DELETE | `/api/v1/setup/opening-balances/:id` | remove (draft only) | ke_toan_tong_hop |
| POST | `/api/v1/setup/opening-balances/check` | run balance-check, return Σ Nợ/Σ Có/diff + per-đối tượng gaps | read roles |
| POST | `/api/v1/setup/opening-balances/import` | CSV import (dry-run \| commit), template v1 | ke_toan_tong_hop |
| GET | `/api/v1/setup/opening-balances/import/:jobId/report` | per-row error report | ke_toan_tong_hop |
| POST | `/api/v1/setup/opening-balances/lock` | BALANCES_DRAFT → BALANCES_LOCKED (after check) | ke_toan_truong |
| POST | `/api/v1/setup/opening-balances/reopen` | unlock (reason; guards R12) | ke_toan_truong |
| POST | `/api/v1/setup/activate` | BALANCES_LOCKED → ACTIVE (opens posting on period 1) | ke_toan_truong |

## 5. Data flows

### 5.1 Initialize (idempotent, resumable)
```
POST /setup/initialize {profile, regime, fy, seedAccounts:true, openPeriods:true}
  → resume from SetupStatus:
    empty        → save profile (R1–R5)                        → PROFILED
    profiled     → masterdata.SetRegime(regime) (idempotent)   → REGIME_SET
    regime_set   → masterdata.SeedAccounts() (idempotent upsert)
                   + audit Quy chế note                        → ACCOUNTS_SEEDED
    accounts_seeded → ledger.OpenPeriod("YYYY-01..12")         → PERIODS_OPEN
    periods_open → (balances entered separately)               → BALANCES_DRAFT
  Each step in its own tx; re-run re-checks "already done?" before re-applying.
  Failures leave status at last completed step (resumable).
```

### 5.2 Opening balance entry
```
POST /setup/opening-balances {account:1111, object?:KH-0001, debit:500_000_000}
  → R7 (one side), R8 (draft state), R10 (object required for 131/331/...)
  → R5 object ACTIVE in masterdata → upsert by the deterministic BalanceID → audit
```

### 5.3 Balance check & lock
```
POST /setup/opening-balances/check
  → Σ Nợ == Σ Có ? → summary {nợ, có, diff:0, gaps:[]} 
  → mismatch → 422 {diff, accounts-offending}
POST /setup/opening-balances/lock (ke_toan_truong)
  → re-check → BALANCES_LOCKED + audit
POST /setup/activate → Status ACTIVE → ledger period 1 becomes postable
```

### 5.4 CSV import (opening balances)
```
upload template-v1 .csv → POST /setup/opening-balances/import {dryRun:true}
  → per-row R7/R8/R10 + balance-check → job report (N ok, M errors row+reason)
  → dryRun:false → one tx per batch, idempotent by the deterministic BalanceID
  → report + audit
```

## 6. Persistence (db.Migrate additions, JSON-doc pattern)

Tables `company_profiles` and `opening_balances` already exist in
`internal/infrastructure/db/migrate.go` (lines 34–35). **Add**:
- `setup_status`: `(id, data)` — single row `{"status":"...", "steps":[...]}`,
  mirroring the status-machine persistence so a resume reads only one row.
  (Alternative: store status inside `company_profiles`; a dedicated row is
  clearer for pre-profile resume.)

No other new tables. `opening_balances.data` = serialized `OpeningBalance`
(JSON). Deterministic BalanceID (sha256 hash of account+object) gives idempotent upsert.

## 7. Money & data quality

- All monetary fields `int64` VND minor units via `core.Money`; no floats.
- `Debit`/`Credit` one-side-zero enforced (R7); balance-check totals are int64
  (no rounding drift).
- MST normalized (strip spaces) before validation; checksum optional (v2 —
  keep format validation v1, mirror masterdata R6).
- Import rows never silently dropped; failing batch rolls back that batch only.

## 8. Sequencing, idempotency & concurrency

- Single SQLite writer; `initialize` steps run serially with status transitions;
  parallel initialize → second caller sees 409 "đang khởi tạo" until first
  commits, then 409 "đã khởi tạo" (R1).
- Idempotency key for opening balances = `BalanceID = sha256("OB\x00"+account+"\x00"+object)` hex (R7 upsert).
- Cross-module idempotency: masterdata `SeedAccounts` and ledger `OpenPeriod`
  are already idempotent upserts — safe to re-invoke on resume.

## 9. Authz wiring

Extend `SeedDefaultPolicies` (internal/infrastructure/authorization/enforcer.go)
with the role→route map in §4. Reads (`status`, `profile`, `opening-balances`
GET, `check`) open to all authenticated roles; `initialize`/`profile` PUT/
`lock`/`reopen`/`activate` → `ke_toan_truong` (+ `role:admin` via `* *`);
`opening-balances` write/import → `ke_toan_tong_hop` + `ke_toan_truong`.
Anonymous → fail closed (existing middleware).

## 10. Integration seams (Go interfaces, not HTTP)

- `masterdata.SetRegime(regime, actor)` — set/switch accounting regime.
- `masterdata.SeedAccounts(actor)` — Phụ lục 2 TT 99/2025 (or TT 133) seed.
- `masterdata.Lookup(kind, code)` — validate đối tượng exists + ACTIVE (R10).
- `ledger.OpenPeriod(actor, "YYYY-MM")` — open FY periods; reuse existing.
- `audit` — append rows on every mutation (R13).
