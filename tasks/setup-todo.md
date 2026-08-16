# Task List — Setup (Khởi tạo hệ thống / Cấu hình doanh nghiệp) Module

Source: `docs/setup/06-roadmap.md`. Conventions: acceptance criteria +
verification per task (planning-and-task-breakdown). No task touches > 5 files.

Dependencies already implemented: masterdata (`SetRegime`, `SeedAccounts`,
`Lookup`), ledger (`OpenPeriod`, `ListPeriods`), audit, Casbin. Setup is the
orchestrator — no new infra.

## Phase 0 — Foundations: status machine & statutory profile (hard gate)
- [ ] T0.1 Domain: `SetupStatus` enum (EMPTY→ACTIVE), full `CompanyProfile`
  (R1–R5 fields), `OpeningBalance` (R7 + object detail), extended `Repository`.
  Accept: compiles; `core.Money` reused. Verify: `go build ./...`.
- [ ] T0.2 Migration: add `setup_status` to `db.Migrate` (single JSON row);
  use existing `company_profiles` + `opening_balances`. Accept: idempotent.
  Verify: `db.Migrate` twice, no error.
- [ ] T0.3 Authz: seed setup policies per `02-spec §9` (initialize/lock/reopen/
  activate → `ke_toan_truong`+`admin`; balance write/import →
  `ke_toan_tong_hop`; reads open). Accept: enforcement blocks wrong role.
  Verify: authz tests.
- [ ] T0.4 Service skeleton: `Status()`, `GetProfile()`, status transitions R6
  (monotonic + idempotent), audit seam (R13). Accept: transitions enforced.
  Verify: property test (monotonic, no backward except reopen).
- [ ] T0.5 HTTP skeleton: `GET /status`, `GET /profile` real; scaffold rest of
  `02-spec §4`. Accept: 501 gone for these. Verify: httptest.

**Checkpoint:** build/vet/test green.

## Phase 1 — Initialize orchestration
- [ ] T1.1 `POST /initialize`: profile validation (R1–R5: MST format+normalize,
  12-month FY, VND, regime whitelist) → save → PROFILED. Accept: invalid MST/FY
  rejected 422. Verify: unit tests.
- [ ] T1.2 Cross-module seams: `masterdata.SetRegime` → REGIME_SET;
  `masterdata.SeedAccounts` + Quy chế audit note → ACCOUNTS_SEEDED;
  `ledger.OpenPeriod("YYYY-01".."YYYY-12")` → PERIODS_OPEN. Accept: no private
  COA in setup. Verify: seam mock assertions.
- [ ] T1.3 Idempotent resume: each step re-checks "already done?" before
  re-applying; status row is resume point. Accept: crash-resume never
  double-seeds. Verify: property test across all steps (UC-S11).
- [ ] T1.4 `PUT /profile` while Status < BALANCES_LOCKED. Accept: blocked
  later. Verify: state-guard test.
- [ ] T1.5 Concurrent initialize: one wins, others 409. Verify: parallel test
  (UC-S13).

**Checkpoint:** build/vet/test green.

## Phase 2 — Opening balances: entry + check + lock
- [ ] T2.1 `POST/GET/DELETE /opening-balances`: upsert by `OB:{account}:
  {object}`; R7 one-side, R8 draft-only, R10 object-required for 131/331/152/
  155/156/211/214 (validated via masterdata `Lookup` ACTIVE). Accept: each rule
  rejects. Verify: unit tests.
- [ ] T2.2 `POST /opening-balances/check`: Σ Nợ == Σ Có (VND), summary + diff +
  offending TK list; 422 on mismatch (R9). Accept: correct totals.
  Verify: property test (ΣNợ=ΣCó invariant — UC-S12).
- [ ] T2.3 `POST /lock` (ke_toan_truong): re-check → BALANCES_LOCKED + audit;
  `POST /reopen` with reason + posting guard + override (R12); `POST /activate`
  → ACTIVE. Accept: lock blocks edits; reopen needs reason; override audited.
  Verify: state-guard tests.
- [ ] T2.4 Web pages: wizard step 4 (balances table + live balance banner
  `05-ui §2.5-2.6`), lock/reopen modals (`§2.8`). Verify: browser walkthrough.

**Checkpoint:** build/vet/test green.

## Phase 3 — Import + status dashboard
- [ ] T3.1 CSV import (`opening-balances/import`): template v1, dry-run job with
  per-row error report, idempotent upsert, batched commit, template version
  rejection, error CSV export. Accept: errors never silently dropped.
  Verify: import tests.
- [ ] T3.2 Wizard step 3 (accounts preview via masterdata) + status dashboard
  (`05-ui §2.4, §2.9`) with step checklist, regime badge, ΣNợ/ΣCó, audit view.
  Verify: walkthrough.

**Checkpoint:** build/vet/test green.

## Phase 4 — Integration hardening + regression
- [x] T4.1 End-to-end: init → balances → activate → post cash voucher → ledger
  books carry opening balances; cash/ledger consume profile+periods. Verify:
  integration test.
  → **Narrow scope delivered** (`internal/application/setup/integration_test.go`):
  full lifecycle against the real setup+masterdata+ledger+audit stack (EMPTY →
  initialize → seed COA + 12 periods → save 1111/1121/5111 balances → balanced
  check → lock → reopen (reason mandatory) → activate → audit trail). Ledger
  books carrying opening balances + cash/ledger consuming profile/periods is a
  **documented follow-up**, not built.
  → **Follow-up found**: ledger `defaultChart` marks TK 131 as a non-postable
  summary (only 1311 postable) while setup spec R10 names 131 the
  mandatory-object opening-balance TK; masterdata COA seeds 131 as a postable
  leaf. The HTTP fakes mark 131 postable, so R10 passes there but the real
  ledger seam rejects 131 balances (`ErrAccountNotFound`). Decide the source of
  truth (align ledger chart with masterdata/spec, or change R10 to 1311) before
  the "books carry opening balances" wiring lands.
- [ ] T4.2 Concurrency/perf: parallel init + balance upserts; single-writer path
  (BEGIN IMMEDIATE where needed); index tuning if lists slow. Verify: benchmark.
- [ ] T4.3 Security review (authz matrix, audit trail, MST not logged outside
  audit) → `docs/setup/08-security.md`. Verify: review checklist.
- [x] T4.4 Coverage ≥ 80% service+repo; vet clean. Verify: `go test -cover`.
  → service 81.8%, repo 88.1% (both ≥80%), websetup 73.1%; `go vet` clean.

**Checkpoint:** build/vet/test green.

## Phase 5 — Statutory verification + pilot
- [ ] T5.1 Official Phụ lục 2 (TT 99/2025) + TT 133/2016 fixtures; MST rules
  per TT 86/2024 / Luật QLT 108/2025. Accept: seed matches via masterdata.
  Verify: golden fixtures.
- [ ] T5.2 Kế toán trưởng walkthrough: full wizard on fresh DB (real ĐKDN +
  Bảng CĐKT data), post first month, verify BCTC boundary + NĐ 254/2026 seller
  identity. Verify: sign-off checklist.
- [ ] T5.3 Regression suite for pilot + benchmark report. Verify: `go test ./...`.
- [ ] T5.4 `00-verdict.md` re-scored → PROD verdict for pilot. Verify: doc
  updated.
