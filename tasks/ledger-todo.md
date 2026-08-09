# Task List — Ledger (Kế toán tổng hợp) Module

Source: `docs/ledger/06-roadmap.md`. Conventions: acceptance criteria + verification
per task (planning-and-task-breakdown). No task touches > 5 files. Test-first
(red-green-refactor); `go build ./... && go vet ./... && go test ./...` green after
every phase. **P0 is a hard gate — do not start P1 until it is done.**

## Phase 0 — Foundations & money correctness (HARD GATE)
- [ ] M0.1 Retire `core.Money` float64 → int64 minor units (or decimal-backed + `Minor()`); remove float paths in cash/invoice/payroll that feed the GL. Accept: no float in ledger math. Verify: `go build ./...`; vet clean.
- [ ] M0.2 `db.Migrate`: add `ledger_accounts`, `ledger_journals`, `ledger_sequences`, `ledger_periods`, `ledger_templates`. Accept: idempotent. Verify: migrate runs twice, no error.
- [ ] M0.3 Seed Casbin roles/policies: `ke_toan_tong_hop`, `ke_toan_truong`, `kiem_toan`; matrix per `docs/ledger/02-spec §9`. Accept: enforcement blocks wrong-role actions. Verify: authz tests.
- [ ] M0.4 Scaffold ledger vertical (domain/app/persistence/http) replacing stubs. Accept: compiles; handlers no longer return 501 for core routes. Verify: httptest.

**Checkpoint:** build/vet/test green.

## Phase 1 — Chart of accounts + periods
- [ ] P1.1 `Account` entity + CRUD (service + repo + HTTP). Accept: list/create/get/update/inactivate. Verify: unit + httptest.
- [ ] P1.2 Hierarchy validation: code structure, parent exists, Level, only leaves postable (R3). Accept: parent accounts reject direct post. Verify: tests.
- [ ] P1.3 `AccountingPeriod` open/close/reopen with reason + audit; R4 lock. Accept: posting to closed period rejected 409. Verify: tests.
- [ ] P1.4 Seed default VAS chart (per TT 99/2025) as startup fixture. Accept: fresh DB has standard accounts. Verify: migrate + seed twice.

## Phase 2 — Journal engine (double-entry core)
- [ ] P2.1 Service: CreateEntry (draft/post), GetEntry, DeleteDraft; R1–R3, R6, R7. Accept: Σ Nợ = Σ Có; append-only after POSTED. Verify: unit + property tests.
- [ ] P2.2 `ledger_sequences`: atomic per-form-per-period VoucherNo (BEGIN IMMEDIATE). Accept: no duplicate/sequence-gap reuse. Verify: parallel-insert test.
- [ ] P2.3 Idempotency key `(Source, SourceRef)`; repost returns existing entry (R5). Accept: double-post impossible. Verify: retry test.
- [ ] P2.4 HTTP: POST/GET entries, POST `:id/post`, DELETE `:id`; main.go wiring. Accept: routes live, 422/403/409 correct. Verify: httptest.
- [ ] P2.5 Web pages: entry create/edit/list (05-ui §2.1). Accept: live balance chip; Ghi sổ locked when unbalanced. Verify: browser-qa.

## Phase 3 — Ledger books + source posting seam
- [ ] P3.1 Books read-model: Sổ Nhật ký chung, Sổ Cái, Sổ chi tiết, BCĐPS; exact int64 sums; Số dư đầu/đi kỳ. Accept: totals always balance. Verify: golden fixture.
- [ ] P3.2 HTTP book endpoints + filters (period/account/paging). Accept: renders under 2 s for 12mo/50k entries. Verify: httptest + benchmark.
- [ ] P3.3 Web print templates A4 (05-ui §3, first pass). Accept: prints per template. Verify: golden strings.
- [ ] P3.4 Implement cash `LedgerWriter` seam → POSTED entry; fund/bank mapping; atomic rollback; voucher `LedgerPosted`/`LedgerError`. Accept: every cash voucher has GL entry. Verify: e2e cash→GL→Sổ Cái 111.
- [ ] P3.5 Repost endpoint `/postings/:source/:sourceRef` + error queue view. Accept: retry no-dup. Verify: tests.

## Phase 4 — Reversals + month-end close
- [ ] P4.1 Reversal (R9): negated entry, REVERSED status, audit link (UC-L8). Accept: exact mirror; double-reverse 409. Verify: tests.
- [ ] P4.2 Kết chuyển templates: 511→911, 632/641/642→911, 911→421; batch run. Accept: closing entries POSTED with KC- sequence. Verify: golden close.
- [ ] P4.3 Close wizard (05-ui §2.4): readiness → run templates → ClosingRecord → confirm lock → rollover. Accept: closed blocks postings; opening rolls to M+1. Verify: e2e.
- [ ] P4.4 Reopen with reason (audited) (E9). Accept: auditable unlock. Verify: tests.

## Phase 5 — Opening balances, templates UI, hardening
- [ ] P5.1 Opening balances + validation (Σ Nợ = Σ Có) + UI (UC-L7). Accept: unbalanced set rejected 422. Verify: tests.
- [ ] P5.2 Bút toán định kỳ template CRUD UI (A2). Accept: template→entry round-trip. Verify: tests.
- [ ] P5.3 Concurrency + performance benchmarks (post p95; 12mo book < 2 s). Accept: targets met or tuning task added. Verify: benchmark report.
- [ ] P5.4 Security review → `docs/ledger/08-security.md`. Accept: authz matrix, tamper-proof, audit trail. Verify: security-reviewer pass.
- [ ] P5.5 Coverage ≥ 80% service+repo; lint/vet gates. Verify: `go test -cover`.

## Phase 6 — Statutory print verification + pilot
- [ ] P6.1 Load official Phụ lục III (TT 99/2025) PDF; match mã hiệu + layout exactly. Accept: no form drift. Verify: walkthrough vs official form.
- [ ] P6.2 Kế toán trưởng walkthrough on one real cashier-pilot month: books + BCĐPS + close + sign-off. Accept: sign-off checklist done. Verify: checklist artifact.
- [ ] P6.3 Regression suite for pilot month + benchmark report. Verify: `go test ./...`.
- [ ] P6.4 Re-score `docs/ledger/00-verdict.md` → PROD verdict. Accept: verdict + next-step signed. Verify: doc updated.
