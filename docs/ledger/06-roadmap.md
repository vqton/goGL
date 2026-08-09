# Ledger Module — Implementation Roadmap

> Mirrors the cash module phases (P0→P6). **Precondition P0 is a hard gate.** Each
> phase ends with `go build ./... && go vet ./... && go test ./...` green and a
> commit. Test-first (red-green-refactor) throughout; ≥ 80% coverage on service +
> repository.

## P0 — Foundations & money correctness (hard gate) [1–2 wks]

- M0.1 Convert `core.Money` to integer minor units (`AmountMinor int64` /
  decimal-backed) or add `Minor()`; remove float path from cash/invoice/payroll
  that feeds the GL. **Blocker — do not proceed without this.**
- M0.2 Add ledger tables to `db.Migrate`: `ledger_accounts`, `ledger_journals`,
  `ledger_sequences`, `ledger_periods`, `ledger_templates`.
- M0.3 Authz: seed roles `ke_toan_tong_hop`, `ke_toan_truong`, `kiem_toan`;
  policies per `02-spec §9`; tests.
- M0.4 Scaffold ledger vertical (domain/app/persistence/http) replacing stubs.

## P1 — Chart of accounts + periods [1–2 wks]

- P1.1 `Account` entity + CRUD service + repo (JSON-doc store) + HTTP.
- P1.2 Validate hierarchy: Code length/structure, parent exists, Level, postable
  leaves only (R3 backend).
- P1.3 `AccountingPeriod`: open/close/reopen with reason + audit; R4 lock.
- P1.4 Seed default VAS chart (list per TT 99/2025) as startup fixture.
- Tests: hierarchy rules, period lock, authz matrix.

## P2 — Journal engine (double-entry core) [2–3 wks]

- P2.1 `JournalEntry`/`JournalLine` service: create draft, post, get, delete
  draft; R1–R3, R5, R6, R7, R10.
- P2.2 `ledger_sequences`: atomic per-form-per-period voucher numbering (BEGIN
  IMMEDIATE tx).
- P2.3 Idempotency key `(Source, SourceRef)`; repost returns existing entry.
- P2.4 HTTP: `POST/GET entries`, `POST entries/:id/post`, `DELETE entries/:id`.
- P2.5 Web pages: entry create/edit/list (05-ui §2.1).
- Tests: balance invariant property tests (R1), one-side rule, duplicate posting.

## P3 — Ledger books + source posting seam [2–3 wks]

- P3.1 Books read-model: Sổ Nhật ký chung, Sổ Cái, Sổ chi tiết, BCĐPS with exact
  int64 sums, Số dư đầu kỳ/đi kỳ.
- P3.2 HTTP: `/books/general-journal`, `/books/ledger`, `/books/detail`,
  `/books/trial-balance`; filters period/account; paging.
- P3.3 Web print templates (05-ui §3, A4) — first statutory pass.
- P3.4 **LedgerWriter implementation**: cash seam → POSTED entry; map fund/bank
  accounts; atomic rollback; cash voucher `LedgerPosted`/`LedgerError` flags.
- P3.5 Repost endpoint `/postings/:source/:sourceRef` + error queue view.
- Tests: end-to-end cash voucher → GL → Sổ Cái 111; idempotent retry; rollback.

## P4 — Reversals + month-end close [2 wks]

- P4.1 Reversal (R9) + REVERSED status + audit note link (UC-L8).
- P4.2 Kết chuyển templates engine: 511→911, 632/641/642→911, 911→421; batch run.
- P4.3 Close wizard (05-ui §2.4): readiness check, run templates, ClosingRecord,
  confirm lock, opening-balance rollover (UC-L6).
- P4.4 Reopen with reason (audited) (exception E9).
- Tests: reversal correctness, close blocks postings, rollover math, templates.

## P5 — Opening balances, templates UI, hardening [1–2 wks]

- P5.1 Opening balances service + validation (Σ Nợ = Σ Có) + UI (UC-L7).
- P5.2 Bút toán định kỳ template CRUD UI (UC-L2 alt A2).
- P5.3 Concurrency + performance benchmarks (mirror cash: post p95, 12-month
  book < 2 s); index/query tuning only if benchmarks fail.
- P5.4 Security review (authz matrix, tamper-proofing, audit trail) → `08-security.md`.
- P5.5 Coverage report ≥ 80%; lint/vet gates.

## P6 — Statutory print verification + pilot [1–2 wks]

- P6.1 Load official Phụ lục III (TT 99/2025) PDF; match mã hiệu + layout of
  every print template exactly; fix drift.
- P6.2 Kế toán trưởng walkthrough: real month of cashier-pilot data → books +
  BCĐPS + close; sign-off checklist.
- P6.3 Regression suite for pilot month; benchmark report.
- P6.4 `docs/ledger/00-verdict.md` re-scored → PROD verdict for pilot.

## Sequencing notes

- P0 blocks everything (money). P1→P4 strict order (books depend on journal;
  close depends on books+reversal). P5/P6 overlap allowed.
- Parallel with cashier pilot from P3.4: ledger becomes the **required**
  `LedgerWriter` for cash; pilot books flow to GL as of that merge.
- Tasks tracked in `tasks/ledger-todo.md` (mirror format of `tasks/todo.md`).

## Estimates & risks

| Risk | Impact | Mitigation |
|---|---|---|
| float64 money corruption | critical | P0 gate; property tests; ban float in ledger paths |
| TT 99 Phụ lục III form drift | medium | P6.1 official-PDF verification; don't hand-type mã hiệu |
| SQLite write contention during close | low | batch templates; BEGIN IMMEDIATE; benchmark P5.3 |
| authz misconfiguration exposes books | high | Casbin matrix tests per role (P0.3) |
| source modules post stale events | medium | period lock + source-ref idempotency + error queue |
