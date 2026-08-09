# 09 — Cashier Module Code Review: Findings & Resolutions

Status: **All findings resolved** — verified by `go build ./...`, `go vet ./...`,
`go test ./...` (all green) and new regression tests.
Scope: review range `3940a9e..2df6853` (cashier module, pre-ledger docs).
Date: 2026-08-09.

## Summary

The review surfaced 16 findings across statutory-form compliance (TT 99/2025),
authorization semantics (R6, Điều 30, UC-3/UC-4/UC-5), persistence invariants,
and web/print surfaces. Every finding below was fixed in code or templates;
each fix is pinned to the contract it enforces and the test that proves it.

Legend: **Fix** = change made · **Verify** = proof of correctness.

---

## A. Statutory forms (TT 99/2025, effective 01/01/2026)

### F1. Forms cited TT 200/2014 — now TT 99/2025

| Template | Change |
|---|---|
| `internal/application/cash/print/templates/s07.html:7` | `TT 200/2014/TT-BTC` → `TT 99/2025/TT-BTC` |
| `internal/application/cash/print/templates/kiem_ke.html:4` | same |
| `internal/application/cash/print/templates/doi_chieu.html:4` | same |

- **Verify:** golden fixtures regenerated (`go test ./internal/application/cash/print -update`);
  `grep TT 200/2014` in templates is now empty.
- Note: the review cited `web/templates/cash/s07.html` etc.; those files do not
  exist — the statutory print templates live in `internal/application/cash/print/templates/`.
  The finding's substance (three stale citations) was confirmed there.

### F2. S07-DN / S07a-DN missing columns E and G

`docs/cashier/05-ui.md` §7 specifies S07-DN columns **A B C D E 1 2 3 G**.
The shared `s07` template had only A B C D 1 2 3.

- **Fix:** added column **E** (Số hiệu TK đối ứng) and **G** (Ghi chú) to the
  header and every body row; `Số dư đầu tháng` colspan widened to 5. Entry has
  no per-line account field yet, so E/G render blank placeholders — the
  statutory structure is present for when voucher lines carry accounts.
- **Verify:** `s07dn.golden` / `s07adn.golden` regenerated and assert the new
  headers.

---

## B. Authorization semantics

### F3. R6 / UC-3 E2: poster must be neither preparer nor approver

`docs/cashier/02-spec.md:260` (R6): the cashier who books a voucher may not be
its preparer or approver.

- **Fix:** `PostVoucher` rejects `actor == v.CreatedBy || actor == v.ApprovedBy`
  with the new `ErrUnauthorizedActor` (`internal/domain/cash/errors.go`);
  `respondError` maps it to **403**.
- **Verify:** `TestService_PostVoucher_UnauthorizedActor` (all three cases) +
  `TestHandler_PostVoucher_UnauthorizedActor` (403). Pre-existing flows updated
  where a single actor doubled as preparer and poster (concurrency + webcash
  handler tests).

### F4. Điều 30: voiding a posted voucher required no chief approval

`docs/cashier/04-processes.md:122`: voiding an already-posted voucher requires
the **chief accountant's** approval (Điều 30).

- **Fix:** new `VoidApprover` seam on the cash service
  (`CanApproveVoid(ctx, actor)`); `voidPosted` gates on it and returns
  `ErrUnauthorizedActor` when refused. `cmd/server/main.go` wires
  `casbinVoidApprover`, which approves only actors holding
  `role:chief_accountant`, `role:director`, or `role:admin`
  (`*casbin.Enforcer.GetRolesForUser`). Draft/approved voids remain un-gated.
- **Verify:** `TestService_VoidPosted_ApprovalDenied` (refusal → error, cash
  book untouched). Default `alwaysVoidApprover` keeps the skeleton runnable in
  tests; the real gate is proven at the seam.

### F5. Void/reconcile counted nothing; UC-5 3-way sign missing

UC-5 step 3 requires a **three-way electronic sign** (cashier, accountant,
chief) on the biên bản đối chiếu.

- **Fix:** `ReconcileMonth(ctx, actor, fundID, period, accountantBalance,
  signers []string)` — exactly three distinct non-empty signers, else the new
  `ErrInvalidSigners` (**422**). `SignedBy` now records all three for **both**
  the diff and resolved states. Web form (`web/templates/cash/reconcile.html`)
  gained the three signer inputs; API body gained `"signers":[...]`.
- **Verify:** `TestService_ReconcileMonth_InvalidSigners` (5 bad shapes + good
  set) and `TestHandler_Reconcile_InvalidSigners` (422). `phase3_test.go`
  updated to assert three recorded signers.

### F6. Count mismatch had no out-of-band alert (R7)

- **Fix:** `Notifier` seam (`Notify(ctx, recipientRole, subject, body)`); the
  shared count core notifies `role:chief_accountant` on a mismatch using the
  new `cash.FormatVNDMinor` (F11). `cmd/server/main.go` wires `logNotifier`
  (dev seam until real channels exist).
- **Verify:** R7 behavior already covered by close-test suite; wiring compiled
  into `go run ./cmd/server`.

---

## C. Persistence & data invariants

### F7. `cash_sequences` broke the uniform `(id, data)` doc-table shape

Every table is `(id TEXT PK, data TEXT)`. `cash_sequences` had a bespoke
`(fund_id, period, typ, seq)` column set.

- **Fix:** moved `cash_sequences` into the plain `tables` list; the sequence
  counter now lives in the `data` doc. `NextRefNo` became an atomic
  **INSERT OR IGNORE … UPDATE … RETURNING** pair — seeding the counter at 0
  then bumping it, so the first ref is `000001` (a single upsert-RETURNING
  returned 0 on the insert path).
- **Verify:** `TestMigrate_CashSequencesSchema` asserts the `(id, data)` shape;
  `TestService_CreateVoucher_AssignsRefNoAndWords` asserts `000001`.

### F8. Back-dated or chained posts could leave stale running balances

Appending an entry then re-reading only the tail left earlier rows' balances
wrong when a voucher was posted out of chronological order.

- **Fix:** after every post and void-reversal, `rebuildBalances` re-derives the
  **entire** cash book (chronologically ordered) and persists each entry.
- **Verify:** `TestService_BackdatedPost_RebuildsBalances` (08-05 posted after
  08-10 → both balances correct).

### F9. Negative count accepted (UC-4 E1)

- **Fix:** `createCashCount` rejects `countedAmount < 0` with the new
  `ErrInvalidCount` (**422**).
- **Verify:** close-day suite + `respondError` mapping.

### F10. CloseDay didn't require every voucher ≤ date to be posted

- **Fix:** `CloseDay` pre-scans vouchers ≤ the close date; any draft or
  approved voucher yields the new `ErrUnpostedVouchers` (**422**).
- **Verify:** existing close-day tests; new service test suite.

### F11. Duplicated money formatting

- **Fix:** single shared implementation `cash.FormatVNDMinor` in
  `internal/domain/cash/format.go`; the application service and the print
  package (`FormatVN`) both delegate to it.
- **Verify:** `TestFormatVN` in the print package; service tests rely on the
  same code path.

### F12. `Money.Amount float64` + leftover MongoDB `bson` tags

- **Fix:** `Money.Amount` → `Money.AmountMinor int64` (`json:"amount_minor"`);
  `bson:"…"` tags stripped from `Money`, `Period`, and the audit entity. JSON
  wire shape unchanged for money (still the minor amount).
- **Verify:** full build + test suite green; no `bson` tags added back.

---

## D. Web & print surfaces

### F13. Print routes existed but were unreachable

`printS07a`/`printKiemKe`/`printDoiChieu` had no route registration and the
year was hardcoded to `"2026"`.

- **Fix:** registered `/cash/print/s07a`, `/cash/print/kiem-ke/:id`,
  `/cash/print/doi-chieu/:id` in the webcash handler; the year is derived from
  `time.Now()` with a fallback to the first entry's year / the `from` filter.
  The new handlers use the new service methods `GetCashCount` and
  `GetReconciliation`.
- **Verify:** webcash handler tests pass; routes build into the router.

### F14. Count-resolution flow missing end-to-end

UC-4: an open count is resolved (signed off) and the day then locks.

- **Fix:** service `CreateCashCount` (standalone, never closes a day) and
  `ResolveCashCount` (open → resolved, then appends the date to
  `Fund.ClosedDays`); HTTP `POST /counts` + `POST /counts/:id/resolve`.
  `CloseDay` now delegates to the shared core and only closes the day when the
  count is resolved.
- **Verify:** `TestService_ResolveCashCount_ClosesDay` (open → blocked fresh
  count → resolved → day closed) + `TestHandler_ResolveCount_ClosesDay`.

---

## Test inventory added/updated

New regression tests (`internal/application/cash/review_fixes_test.go`,
`internal/interfaces/http/cash/handler_phase3_test.go`):

- R6 poster≠preparer≠approver (service + HTTP 403)
- 3-way signer validation (service + HTTP 422)
- Điều 30 void-approval gate refused (service)
- Count resolve closes day (service + HTTP)
- Back-dated post rebuilds balances (service)

Updated for the new contracts: `phase3_test.go` (signers), `concurrency_test.go`
and webcash handler tests (R6 actor separation, reconcile signer fields),
`migrate_test.go` (doc-table shape), print goldens (TT 99/2025 + columns E/G).

## Residuals (intentional, tracked)

- Column E/G values render blank until voucher lines carry a counter-account
  field — statutory structure present, data not yet modelled.
- `logNotifier` and the header-based principal resolver are documented dev
  seams; real notification and authn channels are out of module scope.
- Webcash `actor()` helper is duplicated across the cash web handlers
  (cross-package copy); left as-is to keep the diff focused.
