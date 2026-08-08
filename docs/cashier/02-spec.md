# Functional & Technical Specification — Cashier (Thủ quỹ) Module

Version 1.0. Governs the implementation of `internal/domain/cash`, `internal/application/cash`,
`internal/infrastructure/persistence/cash`, `internal/interfaces/http/cash` per this repo's 4-layer pattern.

## 1. Stack (verified)

- Go 1.26.4, Gin v1.12.0, modernc.org/sqlite (driver `"sqlite"`), Casbin v3
- Document-row tables: `(id TEXT PRIMARY KEY, data TEXT NOT NULL)`; migrations in `internal/infrastructure/db/migrate.go`
- Entities carry `bson` tags (legacy) — do **not** add more (repo rule)

## 2. Domain model (target)

```go
// Fund = quỹ tiền mặt, one per currency.
type Fund struct {
    ID             string       // deterministic id
    Code           string       // e.g. "Q-VND", "Q-USD"
    Name           string
    Currency       string       // VND, USD, EUR, GOLD
    OpeningBalance int64        // minor units, at period open
    FXRate         float64      // rate at open date if FX (0 for VND)
    Active         bool
}

type VoucherType string
const (
    VoucherReceipt VoucherType = "receipt" // Phiếu thu
    VoucherPayment VoucherType = "payment" // Phiếu chi
)

type VoucherState string
const (
    VoucherDraft     VoucherState = "draft"
    VoucherApproved  VoucherState = "approved"
    VoucherPosted    VoucherState = "posted"
    VoucherReconciled VoucherState = "reconciled"
    VoucherVoided    VoucherState = "voided"
)

// Voucher = phiếu thu/chi (Mẫu 01-TT / 02-TT).
type Voucher struct {
    ID             string
    RefNo          string        // sequential per fund per period
    RefDate        string        // yyyy-mm-dd (ngày hạch toán)
    Type           VoucherType
    FundID         string
    Currency       string
    AmountMinor    int64         // in fund currency minor units
    AmountWords    string        // tổng tiền bằng chữ (generated)
    FXRate         float64       // 0 for VND
    CounterpartyType string      // customer | supplier | employee | other
    CounterpartyID   string
    CounterpartyName string
    Description    string        // diễn giải
    RefVouchers    []string      // related invoice/contract refs
    Lines          []VoucherLine // double-entry lines
    State          VoucherState
    // Signatures (full name + timestamp) — statutory requirement
    CreatedBy      string
    ApprovedBy     string
    PostedBy       string        // thủ quỹ
    ReceiverName   string        // người nhận tiền (chi) / người nộp (thu)
    ApprovedAt     string
    PostedAt       string
}

type VoucherLine struct {
    Seq        int
    DebitAcc   string   // TK Nợ (e.g. 1111)
    CreditAcc  string   // TK Có (e.g. 331)
    AmountMinor int64
    ObjectID   string   // analytic object (customer/supplier/employee)
}

// CashBookEntry = row of Sổ quỹ tiền mặt (S07-DN).
type CashBookEntry struct {
    ID          string
    FundID      string
    EntryDate   string // column A
    VoucherDate string // column B
    RefNo       string // columns C/D (số phiếu thu or chi)
    Type        VoucherType
    Description string // column E
    Receive     int64  // column 1
    Pay         int64  // column 2
    Balance     int64  // column 3 (tồn) running
    Reconciled  bool
}

// CashCount = biên bản kiểm kê quỹ.
type CashCount struct {
    ID            string
    FundID        string
    CountDate     string
    BookBalance   int64
    CountedAmount int64
    Difference    int64
    Resolution    string   // xử lý chênh lệch
    Participants  []string
    State         string   // open | signed
}
```

## 3. Repository interface (domain)

```go
type Repository interface {
    CreateFund(ctx, *Fund) error
    ListFunds(ctx) ([]*Fund, error)
    GetFund(ctx, id) (*Fund, error)

    CreateVoucher(ctx, *Voucher) error
    UpdateVoucher(ctx, *Voucher) error
    GetVoucher(ctx, id) (*Voucher, error)
    ListVouchers(ctx, filter VoucherFilter) ([]*Voucher, error)
    NextRefNo(ctx, fundID, period, typ) (string, error) // sequential, locked

    ListCashBook(ctx, fundID, from, to) ([]*CashBookEntry, error)
    AppendCashBookEntry(ctx, *CashBookEntry) error

    CreateCashCount(ctx, *CashCount) error
    ListCashCounts(ctx, fundID) ([]*CashCount, error)
}
```

`VoucherFilter{ FundID, State, From, To, Type }`.

## 4. Service interface (application)

```go
type Service interface {
    CreateFund(ctx, *Fund) error
    ListFunds(ctx) ([]*Fund, error)

    CreateVoucher(ctx, actor, *Voucher) error   // creates draft + assigns RefNo
    UpdateVoucher(ctx, actor, *Voucher) error    // draft only
    ApproveVoucher(ctx, actor, id) error         // draft → approved
    PostVoucher(ctx, actor, id, postDate) error  // approved → posted (cashier, Ghi sổ)
    VoidVoucher(ctx, actor, id, reason) error    // approved|posted → voided (Điều 30)
    GetVoucher(ctx, id) (*Voucher, error)
    ListVouchers(ctx, filter) ([]*Voucher, error)

    GetCashBook(ctx, fundID, from, to) ([]*CashBookEntry, error)
    CloseDay(ctx, actor, fundID, date, countedAmount) error
    ReconcileMonth(ctx, actor, fundID, period, bookAccountantBalance) (CashCount, error)
}
```

## 5. HTTP API (handler) — extend existing registration

Current: `g.POST("/vouchers")`, `g.GET("/vouchers/:id")`. Target surface:

```
POST   /api/v1/cash/funds                     create fund
GET    /api/v1/cash/funds                     list funds
POST   /api/v1/cash/vouchers                  create draft
GET    /api/v1/cash/vouchers/:id              get voucher
PATCH  /api/v1/cash/vouchers/:id              update draft
POST   /api/v1/cash/vouchers/:id/approve      approve
POST   /api/v1/cash/vouchers/:id/post         cashier Ghi sổ
POST   /api/v1/cash/vouchers/:id/void         void (with reason)
GET    /api/v1/cash/vouchers                  list (filter query params)
GET    /api/v1/cash/books                     cash book rows (fund,from,to)
POST   /api/v1/cash/close-day                 daily close + cash count
POST   /api/v1/cash/reconcile                 monthly reconciliation
POST   /api/v1/cash/counts                    biên bản kiểm kê
GET    /api/v1/cash/counts                    list counts
```

Actor = `c.GetHeader(cfg.Authorization.IdentityHeader)` (dev seam) — passed to service for role/ownership checks.

## 6. State machine

```
 draft ──approve──► approved ──post(Ghi sổ)──► posted ──reconcile──► reconciled
   │                    │                         │
   └──void──►voided     └──void──►voided         └──void(Điều30)──►voided
```

Rules:
- `update` only in `draft`
- `approve` only in `draft`; approver ≠ preparer (R6)
- `post` only in `approved`; poster = cashier of the fund; posts into S07-DN with running balance
- `reconcile` only in `posted`
- `void` allowed from draft/approved/posted but **posted void requires an offsetting reversal entry** (Điều 30), never a direct edit

## 7. Business rules (enforced in service layer)

| Rule | Enforcement |
|---|---|
| BR1 Sequential numbering | `NextRefNo` issues next number in a single SQLite transaction; numbers continuous, no reuse on void (void keeps number) |
| BR2 No negative balance | On `post`, if `Balance − Pay < 0` → error "tồn quỹ không đủ" |
| BR3 No erasure | Posted/reconciled rows immutable; corrections via new reversal voucher (Điều 30) |
| BR4 Amount in words | Generated from AmountMinor; must match statutory VN number-in-words |
| BR5 Double-entry | Voucher must have ≥2 lines, sum(debit) = sum(credit) = AmountMinor; exactly one line on 111x |
| BR6 FX | FX voucher: store FXAmount + rate; book in fund currency; separate fund per currency (no cross-currency fund) |
| BR7 Separation of duties | `post` requires role:cashier AND actor ≠ CreatedBy/ApprovedBy; `approve` requires approver role AND actor ≠ CreatedBy |
| BR8 Daily close | `CloseDay` compares physical counted vs book; diff → creates open CashCount; blocks posting after close for that date |
| BR9 Period open/close | Posting denied outside open accounting period |
| BR10 Idempotent post | Posting a voucher twice returns no-op / conflict — never double-entries |

## 8. Data flow (posting)

```
CreateVoucher (accountant)
  → validate lines, sums
  → assign RefNo (transaction, NextRefNo)
  → persist draft, audit "create"

ApproveVoucher (manager/chief)
  → role check, actor != CreatedBy
  → state → approved, audit

PostVoucher (cashier)  "Ghi sổ"
  → role:cashier, state must be approved
  → check fund balance ≥ amount (payment)
  → append S07-DN row (running balance = prev.Balance + Receive − Pay)
  → state → posted; audit; optionally write ledger seam entry (TK111)
```

## 9. Casbin policies (seed)

Matcher triple: `sub = role`, `obj = route pattern`, `act = method`. Seed examples:

| sub | obj | act |
|---|---|---|
| role:cashier | /api/v1/cash/vouchers/*/post | POST |
| role:cashier | /api/v1/cash/books* | GET |
| role:cashier | /api/v1/cash/close-day | POST |
| role:cash_accountant | /api/v1/cash/vouchers | * |
| role:cash_accountant | /api/v1/cash/vouchers/* | * |
| role:cash_accountant | /api/v1/cash/reconcile | POST |
| role:chief_accountant | /api/v1/cash/* | * |
| role:director | /api/v1/cash/*/approve | POST |

App-level rule R7 is layered on top of Casbin (Casbin is route-level; ownership checks are in service).

## 10. Persistence

- Tables: `cash_funds`, `cash_vouchers`, `cash_book`, `cash_counts` — each `(id TEXT PRIMARY KEY, data TEXT NOT NULL)`, JSON document in `data`.
- Add to `migrate.go` table list.
- `NextRefNo` runs as `INSERT ... RETURNING`-style transaction on a `cash_sequences(fund_id, period, seq)` counter table to guarantee continuity under SQLite single-writer.
- Keep `core.Money`? No — cash module uses `AmountMinor int64` internally; migrate `core.Money` to decimal in a follow-up.

## 11. Security & audit

- All mutations write to `audit` module (to-be-implemented): actor, action, entity, before/after JSON.
- Signed fields (ApprovedAt/PostedAt) derived from server time, never client-supplied.
- Input validation in handler via Gin binding; no client trusts; all money as int64.

## 12. Error taxonomy

| Error | HTTP | Meaning |
|---|---|---|
| cash.ErrFundNotFound | 404 | fund missing |
| cash.ErrVoucherNotFound | 404 | voucher missing |
| cash.ErrInvalidState | 409 | wrong state for transition |
| cash.ErrInsufficientBalance | 422 | tồn quỹ không đủ |
| cash.ErrUnauthorizedActor | 403 | role/ownership violation |
| cash.ErrSequenceConflict | 409 | concurrent numbering |
| cash.ErrPeriodClosed | 422 | posting in closed period |

## 13. Testing strategy

- Service layer: table-driven unit tests for every transition + rule (BR1-BR10), ≥80% coverage
- Repository: sqlite in-memory integration tests (mirror `internal/infrastructure/authorization` test approach)
- Handler: gin httptest covering API surface + authz middleware interplay
- Statutory-form golden tests: fixture rows → expected S07-DN/S07a-DN strings
- Run: `go test ./internal/domain/cash/... ./internal/application/cash/... ./internal/infrastructure/persistence/cash/... ./internal/interfaces/http/cash/...`
