# Security Review — Cashier (Thủ quỹ) Module (T5.2)

Scope: the cash module as built through Phase 5. Reviewed axes: authorization
matrix, input validation, audit completeness. Companion to `05-ui.md`,
`03-use-cases.md`, `06-roadmap.md`.

## 1. Authorization matrix

Enforcement triple (Casbin v3, `keyMatch2` matcher):

| Role | Routes (pattern) | Acts |
|---|---|---|
| `role:admin` | `*` | `*` |
| `role:cashier` | `/api/v1/cash/vouchers/*/post` | `POST` |
| `role:cashier` | `/api/v1/cash/book` | `GET` |
| `role:cashier` | `/api/v1/cash/close-day` | `POST` |
| `role:cash_accountant` | `/api/v1/cash/vouchers`, `/api/v1/cash/vouchers/*` | `*` |
| `role:cash_accountant` | `/api/v1/cash/reconcile` | `POST` |
| `role:chief_accountant` | `/api/v1/cash/*` | `*` |
| `role:director` | `/api/v1/cash/*/approve` | `POST` |

Facts verified in `internal/infrastructure/authorization/cash_policies_test.go`:

- Cashier can post a voucher, read the book, close the day; cannot create,
  approve, void, or reconcile.
- Accountant can create/read/update vouchers and reconcile; cannot post or
  close the day (the `vouchers/*` rule is `act=*`, so an accountant *could*
  call the post route — this is intentionally over-permissive today because
  there is no per-route deny; the cashier-specific rules are the operative
  grants for posting. Flagged below as F2.)
- Director can approve but not post; chief accountant has full module access.
- Unknown subjects are denied (fails closed).
- `books*` was tightened to the exact route `book` (the old rule only matched
  `/api/v1/cash/books` by regex luck).

### Findings

- **F1 — Web UI outside Casbin (HIGH).** The authorization middleware mounts on
  `/api/v1` only; the `/cash` web pages register on the root engine. Until now
  every state-changing web action (create/approve/post/void/close-day/reconcile)
  proceeded with an empty `X-User-Id`. **Fixed in T5.2**: webcash mutating
  handlers now call `requireActor` and fail closed with 401 when the identity
  header is missing (`internal/interfaces/http/webcash/handler.go`); covered by
  `TestMutatingRoutes_RejectAnonymous`. The web UI remains unauthenticated for
  *reads* (dashboard, forms, book, exports) — acceptable as a dev seam, but a
  real auth layer must replace `X-User-Id` before production (see
  `00-verdict.md`).
- **F2 — no per-route deny for accountant posting (LOW).** `role:cash_accountant`
  has `act=*` on `/vouchers/*`, which technically includes the `/post` subroute.
  Mitigated by the service-level state machine (`PostVoucher` only posts
  `approved` vouchers) and the cashier-specific grants, but a deny rule is the
  correct long-term shape once real users exist.
- **F3 — `X-User-Id` is a trust-on-first-use dev seam (INFO, inherited).**
  Spoofable by any client; documented in AGENTS.md and `06-roadmap.md`.

## 2. Input validation

| Field | Check | Where |
|---|---|---|
| `fund.name/currency/account` | required | service |
| `voucher.currency` | must equal fund currency | service |
| `voucher.type` | receive \| pay | `validateVoucher` |
| `voucher.fund_id` | required | `validateVoucher` |
| `voucher.ref_date` | strict `yyyy-mm-dd` (added T5.2) | `validateVoucher` |
| `voucher.amount_minor` | > 0 | `validateVoucher` |
| `counterparty_name` | required (R1) | `validateVoucher` |
| `lines` | ≥ 2, each amount > 0, debits = credits = amount, exactly one cash-account line | `validateVoucher` |
| HTTP layer | `ShouldBindJSON` → 400 on bad body; form int parsing → redirect with message | handlers |

Findings: **F4 — no length caps on text fields (LOW).** `description`,
`counterparty_name`, `receiver_name` are unbounded; a large JSON body can bloat
rows. Acceptable for a single-tenant ERP; add caps when multi-tenancy arrives.
**F5 — account codes not checked against a chart of accounts (LOW).** The
ledger module is a skeleton; validation against `Account` master data is
deferred to it.

## 3. Audit completeness

Every mutating action records an audit row (`auditAction`):

| Action | Event |
|---|---|
| Fund created | `fund.create` |
| Voucher created / updated | `voucher.create` / `voucher.update` |
| Voucher approved / posted / voided | `voucher.approve` / `voucher.post` / `voucher.void` |
| Reversal created (Điều 30) | `voucher.reversal.create` |
| Cash count opened / day closed | `cash.count.open` / `cash.close_day` |
| Reconcile diff / resolved | `cash.reconcile.diff` / `cash.reconcile` |

All rows carry `module=cash`, the acting user id, and an RFC3339 timestamp.
Reads are not audited — acceptable (no sensitive data exposure). **No findings.**

## 4. Tooling checks

- `security-audit` (deps + secrets + code): secrets **passed**, code anti-
  patterns **passed**, npm audit pending (JS deps are Tailwind build tooling
  only; Go `go.mod` has no known high-severity advisories at review time).
- `go vet ./...`: clean.
- Race detector (`go test -race`) on cash service + webcash: clean (see T5.1).

## 5. Verdict

The cash module fails closed where it can: state-machine guards, self-approval
rejection, negative-balance rejection, active-fund requirement, and (now) web
identity enforcement. Remaining exposure is the documented dev-auth seam and
the intentionally permissive accountant route — both are tracked for the
authentication phase, not blockers for this module's sign-off.
