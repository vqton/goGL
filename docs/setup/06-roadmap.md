# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — Implementation Roadmap

> Mirrors the cash/ledger/masterdata phase pattern (P0→P5). **Precondition P0
> is a hard gate.** Each phase ends with `go build ./... && go vet ./... &&
> go test ./...` green and a commit. Test-first (red-green-refactor); ≥ 80%
> coverage on service + repository. Tasks tracked in `tasks/setup-todo.md`.

Dependencies are **already implemented**: masterdata (`SetRegime`,
`SeedAccounts`, `Lookup`), ledger (`OpenPeriod`, `ListPeriods`,
`SeedDefaultAccounts`), audit, Casbin authz. Setup is the orchestrator that
wires them — the risk is in the state machine and balance rules, not in
building new infra.

## P0 — Foundations: status machine + statutory profile (hard gate) [1 wk]

- S0.1 Replace stub domain: `SetupStatus` enum (EMPTY→ACTIVE), full
  `CompanyProfile` (R1–R5 fields), `OpeningBalance` (R7 fields + object detail),
  extended `Repository` interface. Keep `core.Money` (int64 VND).
- S0.2 Migration: add `setup_status` table to `db.Migrate` (single JSON row);
  `company_profiles` + `opening_balances` already exist — start using them.
- S0.3 Authz: seed setup policies per `02-spec §9` (reads open; initialize/
  lock/reopen/activate → `ke_toan_truong`+`admin`; balance write/import →
  `ke_toan_tong_hop`); authz matrix test.
- S0.4 Service skeleton: `Status()`, `GetProfile()`, status transitions R6
  (monotonic + idempotent), audit seam (R13) on every mutation.
- S0.5 HTTP skeleton: `GET /status`, `GET /profile` — real responses; the rest
  of `02-spec §4` scaffolded.

**Checkpoint:** build/vet/test green; status machine property test (monotonic,
no backward except reopen); authz matrix tests pass.

## P1 — Initialize orchestration [1–2 wks]

- P1.1 `POST /initialize`: profile validation (R1–R5: MST format+normalize,
  12-month FY, VND, regime whitelist) → save → PROFILED.
- P1.2 Wire cross-module seams (Go interfaces): `masterdata.SetRegime` →
  REGIME_SET; `masterdata.SeedAccounts` + Quy chế audit note → ACCOUNTS_SEEDED;
  `ledger.OpenPeriod("YYYY-01".."YYYY-12")` → PERIODS_OPEN.
- P1.3 Idempotent resume: each step re-checks "already done?" before re-applying
  (rely on masterdata/ledger idempotency); status row is the resume point.
- P1.4 `PUT /profile` edit while Status < BALANCES_LOCKED.
- Tests: happy-path init, crash-resume at each step (no double seed/periods —
  UC-S11), MST validation, concurrent init (one 409), authz.

## P2 — Opening balances (entry + check + lock) [1–2 wks]

- P2.1 `POST/GET/DELETE /opening-balances`: per-TK + per-đối tượng upsert by
  `OB:{account}:{object}`; R7 one-side, R8 draft-only, R10 object-required for
  131/331/152/155/156/211/214 (validated against masterdata `Lookup` ACTIVE).
- P2.2 `POST /opening-balances/check`: Σ Nợ == Σ Có (VND), return summary +
  diff + offending TK list; 422 on mismatch (R9).
- P2.3 `POST /opening-balances/lock` (ke_toan_truong): re-check → BALANCES_LOCKED
  + audit; `POST /reopen` with reason + posting guard + override (R12);
  `POST /activate` → ACTIVE.
- P2.4 Web pages: wizard step 4 (balances table + live balance banner,
  `05-ui §2.5–2.6`), lock/reopen modals (`§2.8`).
- Tests: balance invariant property test (ΣNợ=ΣCó), object-required cases,
  lock blocks edits, reopen guard, override audit.

## P3 — Import + status dashboard [1 wk]

- P3.1 CSV import (`opening-balances/import`): template v1, dry-run job with
  per-row error report, idempotent upsert, commit in batched tx; template
  version rejection; error CSV export.
- P3.2 Wizard step 3 (accounts preview via masterdata) + status dashboard
  (`05-ui §2.4, §2.9`) with step checklist, regime badge, ΣNợ/ΣCó, audit view.
- Tests: import dry-run/error isolation, re-import idempotency, template
  version, dashboard counts.

## P4 — Integration hardening + regression [1 wk]

- P4.1 End-to-end: init → open balances → activate → post a cash voucher →
  ledger books show opening balances carried forward; cash/ledger consume
  setup profile + periods correctly.
- P4.2 Concurrency/perf: parallel init + parallel balance upserts; verify
  single-writer SQLite path (BEGIN IMMEDIATE where needed); index tuning if
  lists slow.
- P4.3 Security review (authz matrix, audit trail, MST not logged outside
  audit) → `docs/setup/08-security.md`.
- P4.4 Coverage ≥ 80% service+repo; vet clean.

## P5 — Statutory verification + pilot [1 wk]

- P5.1 Load official Phụ lục 2 (TT 99/2025) + TT 133/2016 fixtures; confirm
  seed matches via masterdata; MST rules match TT 86/2024/Luật QLT 108/2025.
- P5.2 Kế toán trưởng walkthrough: full wizard on a fresh DB (real ĐKDN data,
  opening balances from a real Bảng CĐKT), post first month, verify BCTC
  boundary + seller identity fields for NĐ 254/2026.
- P5.3 Regression suite for the pilot; benchmark report.
- P5.4 `docs/setup/00-verdict.md` re-scored → PROD verdict for pilot.

## Sequencing notes

- P0 blocks everything (status + profile + authz are prerequisites).
- P1 (orchestration) strictly before P2 (balances need open periods + regime).
- P2.2/2.3 (check/lock) before P3 import (import writes into a locked-safe
  draft state).
- Parallelizable: P3.2 dashboard can start once P0 status page exists;
  P4.1 needs P1–P3.
- **No new infrastructure** — reuse masterdata/ledger/audit/authz seams. Setup
  must NOT create a second chart of accounts (the leading risk).

## Estimates & risks

| Risk | Impact | Mitigation |
|---|---|---|
| Two sources of truth for COA/regime (setup vs masterdata) | high | setup only orchestrates: `SetRegime`+`SeedAccounts` in masterdata; no private chart |
| Non-idempotent init double-seeds on crash-resume | high | status row is resume point; rely on masterdata/ledger idempotent upserts; UC-S11 property test |
| Unbalanced opening balances slip into live ledger | high | R9 enforced at check AND lock; 422 with diff; property test |
| Regime/currency mismatch against FY boundary | medium | R5/R4 at init; FY-boundary guard for switch (mirror masterdata UC-M13) |
| Object detail missing for 131/331/152/211-214 | medium | R10 hard validation at entry + import; UI highlight |
| Reopen bypass leaves wrong books | medium | R12 guard + chief override + mandatory reason + audit |
| Concurrent init races | low | single writer; one winner, 409 losers (UC-S13) |
