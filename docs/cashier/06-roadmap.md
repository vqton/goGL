# Implementation Roadmap — Cashier (Thủ quỹ) Module

Phased, vertical slices per `planning-and-task-breakdown`. Each phase leaves the build green. Verification checkpoint after each phase. Tasks in `tasks/todo.md`.

## Phase 0 — Foundation (setup, controls-first)

- [ ] T0.1 Replace `core.Money` float64 with exact decimal (int64 minor units + currency). Update cash entities. (`internal/domain/core/types.go`)
- [ ] T0.2 Add tables to `migrate.go`: `cash_funds`, `cash_vouchers`, `cash_book`, `cash_counts`, `cash_sequences`.
- [ ] T0.3 Seed Casbin roles/policies: `role:cashier`, `role:cash_accountant`, `role:chief_accountant`, `role:director` + route policies.
- [ ] T0.4 Audit module seam: log create/update/transition (who/when/before/after).

**Checkpoint:** `go build ./...`, `go vet ./...`, `go test ./...` green; authz tests still pass.

## Phase 1 — Funds + voucher lifecycle (draft→approved)

- [ ] T1.1 Domain model: `Fund`, `Voucher`, `VoucherLine`, enums (`02-spec.md §2`).
- [ ] T1.2 Repository: fund CRUD + voucher CRUD + `NextRefNo` (sequence tx).
- [ ] T1.3 Service: `CreateFund`, `ListFunds`, `CreateVoucher` (validate lines, sums, words, assign RefNo), `UpdateVoucher` (draft only), `ApproveVoucher` (R6 guard).
- [ ] T1.4 HTTP: funds + vouchers create/get/list + approve handlers; wire in `main.go` (extend existing).
- [ ] T1.5 Unit + integration tests for lifecycle + BR1/BR5/BR6 + R6.

**Checkpoint:** create→approve works end-to-end via API; tests green; no posting yet.

## Phase 2 — Cashier posting + cash book (core value)

- [ ] T2.1 Repository: `AppendCashBookEntry`, `ListCashBook`, balance lookup.
- [ ] T2.2 Service: `PostVoucher` (BR2 insufficient balance, BR8 close guard, BR10 idempotency, running balance), `GetCashBook`.
- [ ] T2.3 Service: `CloseDay` (physical count compare → CashCount on diff), `CreateCashCount`.
- [ ] T2.4 HTTP: post, close-day, counts endpoints; S07-DN view.
- [ ] T2.5 Ledger seam: post TK111 entry (interface to `ledger`; stub impl OK, documented).
- [ ] T2.6 Tests: posting rules, negative-balance, double-post, daily close.

**Checkpoint:** cashier can post + view S07-DN; negative balance impossible; audit logged.

## Phase 3 — Reconciliation, void, correction

- [ ] T3.1 Service: `ReconcileMonth` (compare books → biên bản → 3-way sign → `reconciled`).
- [ ] T3.2 Service: `VoidVoucher` (draft/approved direct void; posted → reversal pair, chief approval, Điều 30).
- [ ] T3.3 HTTP + S07a-DN view + biên bản endpoints.
- [ ] T3.4 Tests: reconciliation flow, void/reversal, error paths UC-5/UC-6.

**Checkpoint:** full month cycle works (UC-1..UC-6 user journeys).

## Phase 4 — Reports, print templates, UX

- [ ] T4.1 Print templates (01-TT, 02-TT, S07-DN, S07a-DN, biên bản) per `05-ui.md §7`.
- [ ] T4.2 Web pages: fund dashboard, voucher forms, cashier queue (Ghi sổ), S07-DN/S07a-DN, khóa sổ, đối chiếu.
- [ ] T4.3 Tailwind styling + VN number/amount-in-words helpers + i18n.
- [ ] T4.4 CSV export; UX polish (badges, confirm dialogs, keyboard shortcuts).
- [ ] T4.5 Golden tests: fixture rows → expected S07-DN string.

**Checkpoint:** demo walkthrough of full month; forms print per TT99.

## Phase 5 — Hardening

- [x] T5.1 Concurrency/idempotency soak test; sequence continuity under load.
- [x] T5.2 Security review (authz matrix, input validation, audit completeness). See `08-security.md`.
- [x] T5.3 Load: 12-month cash book query < 500ms; posting p99 < 100ms.
  Measured (i5-5200U, in-memory SQLite, `-race` off): post path 3.7 ms/op,
  12-month book over 1,800 rows 86 ms/op — both well under target.
  Benchmarks in `internal/application/cash/benchmark_test.go`.
- [ ] T5.4 Full test suite ≥ 80% coverage on cash service; go vet clean.
- [ ] T5.5 Production-readiness doc update + sign-off (chief accountant + BA).

**Checkpoint:** green light for PROD pilot on a single fund (e.g. Q-VND).

## Sequencing rationale

- Controls-first (Phase 0) because money correctness is non-negotiable; post before UI so behavior drives screens.
- FX/gold/vàng and cross-module AR/AP/payroll links deferred to v2 (see `01-brd.md` §9).
- Recommend single `cash` module, role-gated; revisit split only if SME licensing demands separate Thủ quỹ.

## Risks & mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `core.Money` float64 change ripples across 24 modules | High | Phase 0 isolation; compile-time breakage is the safety net |
| Ledger module is stub → no real TK111 posting | Med | Ledger seam interface + unit-testable mock; ledger impl parallel workstream |
| SQLite single-writer contention on `NextRefNo`/posting | Med | Sequence counter table + single tx; SQLite WAL; batch Ghi sổ in one tx |
| Statutory form drift (TT99 updates) | Med | Templates isolated; golden tests; monthly legal review (VBPL, MOF) |
| Role bypass via `X-User-Id` dev seam | High (prod) | Flag in hardening; production auth program separate; app-level ownership checks regardless |

## Open questions (from BRD §9)

1. One module vs split — default single role-gated module.
2. FX difference engine in v2 (multi-currency funds yes in v1).
3. Numbering per fund/period — confirmed.
4. Vàng tiền — defer v2.
