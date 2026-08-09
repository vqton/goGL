# Ledger Module (Kế toán tổng hợp) — Technical Specification

> Target state. Current ledger packages are stubs. Money model: **integer minor
> units (`int64`, VND)** — `core.Money` float64 must not be used in ledger math
> (see `00-verdict.md §2.2`).

## 1. Architecture position

```
 source modules ──┐   internal/application/ledger (Service)
 cash/bank/invoice │         │ post/validate
 payroll/fixedasset┘         ▼
         internal/domain/ledger (entities) ── internal/infrastructure/persistence/ledger (SqliteRepository)
                                                      │
                                      internal/infrastructure/db (migrate.go — tables)
                                                      ▼
                     internal/interfaces/http/ledger (Handler /api/v1/ledger/...)
                                                      │
                                            web/templates + web UI (Sổ kế toán)
```

Manual entry path (handler → service → repo) mirrors the cash module reference
implementation (service owns validation+sequence, repo is a thin JSON-doc store).

## 2. Domain entities

### 2.1 Account (Sơ đồ tài khoản)
```go
type Account struct {
    ID          string // deterministic SHA-256 of Code (JSON-doc table pattern)
    Code        string // "1111", "511", "911", ...
    Name        string // "Tiền mặt - VND"
    ParentCode  string // "" if root
    Type        AccountType // ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    Level       int         // 1..6
    Status      core.Status // ACTIVE / INACTIVE / LOCKED
    AllowPost   bool        // false = parent (summary) account, no direct post
}
```

### 2.2 JournalEntry / JournalLine (Sổ Nhật ký chung)
```go
type JournalEntry struct {
    ID          string
    VoucherNo   string        // "PT-00012/25" or "PK-00045/25" — per-form per-period
    VoucherDate time.Time     // ngày hạch toán
    Period      PeriodRef     // YYYYMM derived from VoucherDate
    Source      EntrySource   // MANUAL | CASH | BANK | INVOICE | PAYROLL | FIXEDASSET | CLOSING
    SourceRef   string        // source voucher id (cash voucher id, invoice id, …)
    Description string
    Lines       []JournalLine
    Status      EntryStatus   // DRAFT | POSTED | REVERSED
    CreatedBy   string        // user id
    ReversedBy  string        // user id (if REVERSED)
    ReversedOf  string        // id of original entry (if this is the reversal)
}
type JournalLine struct {
    LineNo      int
    AccountCode string
    Debit       int64 // VND minor units
    Credit      int64
    Note        string
}
```

### 2.3 AccountingPeriod / Closing (Kỳ kế toán, khoá sổ)
```go
type AccountingPeriod struct {
    ID        string // "2026-08"
    Year      int
    Month     int
    Status    PeriodStatus // OPEN | CLOSED
    ClosedAt  time.Time
    ClosedBy  string
    OpenedBy  string
}
type ClosingRecord struct {
    ID         string
    Period     string
    Kind       ClosingKind // KET_CHUYEN_DOANH_THU | KET_CHUYEN_CHI_PHI | KET_CHUYEN_LOI_NHUAN
    EntryIDs   []string    // entries created by the close
    ClosedBy   string
    ClosedAt   time.Time
}
```

### 2.4 OpeningBalance / Template
```go
type OpeningBalance struct {
    Period      string
    AccountCode string
    Debit       int64
    Credit      int64
}
type JournalTemplate struct { // bút toán định kỳ / kết chuyển lặp lại
    ID          string
    Code        string // "KC_DT", "PB_KHAU_HAO", "DK_KHACH_HANG"
    Description string
    Lines       []JournalLine // percentage or fixed amounts (debit/credit)
    Source      EntrySource
}
```

## 3. Invariants (rules enforced in the service layer)

| R# | Rule | Consequence if violated |
|---|---|---|
| R1 | Σ Debit = Σ Credit on every POSTED entry (incl. reversal) | Entry rejected |
| R2 | A line must have exactly one side > 0 (Debit XOR Credit, never both, never neither) | Entry rejected |
| R3 | Every line's `AccountCode` exists, is ACTIVE, and `AllowPost` | Entry rejected |
| R4 | `VoucherDate` must fall in an OPEN period | Entry rejected (period locked) |
| R5 | `SourceRef` unique per (Source, VoucherNo) — idempotency | Duplicate posting rejected; retry returns the existing entry id |
| R6 | Entry is append-only once POSTED; edits only by full reversal (REVERSED + new entry) | Mutation blocked |
| R7 | Deletion allowed only for DRAFT entries | Mutation blocked |
| R8 | Close-period requires `ke_toan_truong` or `giam_doc`; blocks R4 for that period | Authz denied / postings fail |
| R9 | Reversal must negate the original entry exactly (same lines, opposite sides), matching VoucherNo suffix "R" | Entry rejected |
| R10 | Voucher sequence increments atomically; never reuses numbers after delete/reversal | Sequence exception |

## 4. API surface (`/api/v1/ledger`)

| Method | Path | Action | Role |
|---|---|---|---|
| GET | `/api/v1/ledger/accounts` | list chart of accounts (filter: type, parent, q) | giam_doc, ke_toan_tong_hop, ke_toan_truong |
| POST | `/api/v1/ledger/accounts` | create account | ke_toan_tong_hop |
| POST | `/api/v1/ledger/entries` | create manual entry (draft or post) | ke_toan_tong_hop |
| POST | `/api/v1/ledger/entries/:id/post` | approve + post (draft→POSTED) | ke_toan_truong (or ke_toan_tong_hop per config) |
| GET | `/api/v1/ledger/entries/:id` | fetch entry + lines | read roles |
| POST | `/api/v1/ledger/entries/:id/reverse` | reversal | ke_toan_truong |
| DELETE | `/api/v1/ledger/entries/:id` | delete draft only | ke_toan_tong_hop |
| GET | `/api/v1/ledger/books/general-journal` | Sổ Nhật ký chung (period range) | read roles |
| GET | `/api/v1/ledger/books/ledger` | Sổ Cái per account | read roles |
| GET | `/api/v1/ledger/books/trial-balance` | Bảng cân đối số phát sinh | read roles |
| GET | `/api/v1/ledger/books/detail` | Sổ chi tiết per account | read roles |
| GET | `/api/v1/ledger/periods` | list periods | read roles |
| POST | `/api/v1/ledger/periods/:id/open` | open period | ke_toan_truong |
| POST | `/api/v1/ledger/periods/:id/close` | close period (khoá sổ) | ke_toan_truong |
| POST | `/api/v1/ledger/periods/:id/reopen` | unlock (requires reason, audited) | ke_toan_truong |
| POST | `/api/v1/ledger/periods/:id/close/run` | execute kết chuyển cuối kỳ templates | ke_toan_truong |
| POST | `/api/v1/ledger/opening-balances` | set opening balances for a period | ke_toan_truong |
| GET | `/api/v1/ledger/templates` / POST `/api/v1/ledger/templates` | manage bút toán định kỳ | ke_toan_tong_hop |
| POST | `/api/v1/ledger/postings/:source/:sourceRef` | idempotent re-posting of a source event (retry path) | ke_toan_tong_hop |

All mutations 201/200 with full resource JSON; validation errors 422 with
per-rule messages (VN + EN); period-lock 409; authz 403 via existing middleware.

## 5. Data flows

### 5.1 Source module → GL posting (cash `LedgerWriter` seam, the critical one)

Existing contract in `internal/application/cash/service.go`:
```go
type LedgerEntry struct {
    Date      time.Time
    Account   string   // debit account
    Debit     int64    // AmountMinor (cash module already uses int64)
    Credit    int64
    RefNo     string   // cash voucher no
    FundID    string
    VoucherID string
}
type LedgerWriter interface { Post(ctx, e LedgerEntry) error }
```
Cash currently calls `noopLedger{}`. The ledger module provides a real writer that
internally maps each cash `LedgerEntry` to a `JournalEntry` (Source=CASH,
SourceRef=VoucherID, double-entry against the fund/bank account), enforcing R1–R6.
Flow:
```
CashService.Post(context) ──▶ LedgerWriter.Post(ctx, e)
       ──▶ LedgerService.PostFromSource(ctx, CASH, ref, lines)
       ──▶ tx { validate(R1–R6) → assign VoucherNo (R10) → insert entry (append-only) } commit
```
Retry: `POST /postings/:source/:sourceRef` re-runs the same mapping; R5 makes it
idempotent. Failure anywhere in the tx rolls back the cash voucher too (caller
propagates error) — no silent drops.

### 5.2 Manual entry flow
```
web/UI form → POST /entries (DRAFT) → list/verify → POST /entries/:id/post
  → R1–R6, R10 → status POSTED → appears in all books
```

### 5.3 Reversal flow
```
POST /entries/:id/reverse (ke_toan_truong)
  → creates ReversedOf entry with negated lines (R9), VoucherNo "…-R"
  → original marked REVERSED (its amounts still appear in books, as law requires)
```

### 5.4 Month-end close
```
close/run → for each enabled template (kết chuyển doanh thu 511→911, chi phí 632/641/642→911,
  lợi nhuận 911→421) → create POSTED closing entries → mark ClosingRecord
  → Kế toán trưởng confirms → period CLOSED (R8) → postings blocked (R4)
```

## 6. Persistence (db.Migrate additions, JSON-doc pattern)

Add to the fixed table list in `internal/infrastructure/db/migrate.go`:
`ledger_accounts`, `ledger_journals`, `ledger_sequences`, `ledger_periods`,
`ledger_templates`. All `(id TEXT PRIMARY KEY, data TEXT NOT NULL)`.

- `ledger_journals` rows = serialized `JournalEntry` (lines inline). Index on
  `id` only (existing pattern); read-path filters by period/source in Go with a
  `ledger_periods` scan, or add a per-module secondary index column when volumes
  demand (see NFR performance target — profile before over-indexing).
- `ledger_sequences` rows `(form, period)` → next voucher number; increment under
  a `database/sql` transaction with `BEGIN IMMEDIATE` (R10).

## 7. Money & drift safety

- Store `Debit`/`Credit` as `int64` VND minor units in `JournalLine` (mirrors
  cash `AmountMinor`). No floats in ledger code paths.
- Migration task M0 (roadmap): convert `core.Money` to a decimal-backed type or
  add `Minor() int64` helper; audit all cash/invoice/payroll sites. Ledger must
  consume only integer amounts.
- Report math (Sổ Cái, BCĐPS) sums `int64`; comparisons exact.

## 8. Sequencing, idempotency & concurrency

- Voucher format `{form}-{5-digit}/{YY}` e.g. `PK-00045/26`; per-form-per-period
  sequence (Q2 default). Reversal suffix `-R`.
- Concurrent posts: single SQLite writer; `ledger_sequences` increment and entry
  insert inside one tx → unique VoucherNo guaranteed (R10).
- Idempotency key = `(Source, SourceRef)`: check-before-insert returns existing
  entry (R5) so webhooks/retries never double-post.

## 9. Authz wiring

Extend `SeedDefaultPolicies` (internal/infrastructure/authorization/enforcer.go)
with the role→route map in §4 (matcher uses `c.FullPath()` + method already).
Roles: `ke_toan_tong_hop`, `ke_toan_truong`, `giam_doc`, `kiem_toan`. Admin keeps
`* *`. Gate `/api/v1/ledger/books/*` read for `giam_doc`+ `kiem_toan`, mutations
for `ke_toan_*`, close/reopen/reverse only for `ke_toan_truong`.
