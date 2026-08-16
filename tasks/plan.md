# Implementation Plan: goGL module portfolio

## Overview
Build goGL — Vietnamese accounting/ERP (module goGL, Go 1.26.4, Gin + SQLite,
JSON-doc tables, Casbin RBAC). Each feature module is a 4-layer vertical
(domain / application / persistence / http) wired in `cmd/server/main.go`.
Deliverables per module follow the same doc set in `docs/<m>/` (`00-verdict`
… `06-roadmap`, plus `08-security`/`09-review` for hardened modules) and task
lists in `tasks/<m>-todo.md`.

## Status (2026-08-16)
- **PROD-pilot, done:** cash (docs/cashier, verdict READY) and ledger
  (docs/ledger) — TT 99/2025 statutory forms, sequences, periods, audit.
- **Implemented, docs as target-state:** masterdata (docs/masterdata,
  vertical in 51eec6b — `SeedAccounts` Phụ lục 2 TT 99/2025, `SetRegime`
  TT 99↔TT 133, lifecycle, merge, CSV import) and **setup** (docs/setup,
  vertical in 05d6b4c — company profile + regime + 12-month FY, COA seed +
  period open via masterdata/ledger seams, balanced opening balances with
  per-đối tượng detail, status machine + lock/reopen/activate, CSV import,
  web wizard). Phase 0–2 done, T4.1/T4.4 done; remaining in
  `tasks/setup-todo.md`: T3.1/T3.2 partials (async import job, accounts
  preview + audit view), T4.2 perf, T4.3 security doc, Phase 5 pilot.
- **Still stubs (24 modules total, 20 remain):** audit, backup, bank, budget,
  contract, costing, document, fixedasset, inventory, invoice, options,
  payroll, purchase, reporting, sales, system, task, tax, tools, user.

## Architecture decisions
- **Controls-first**: exact-decimal money (`core.Money` int64 VND), migrations,
  RBAC policies, audit seam before any feature.
- **No two sources of truth**: masterdata owns sơ đồ tài khoản + regime;
  ledger/cash/setup orchestrate via Go seams, never seed a private chart.
- **Setup is the orchestrator** of masterdata/ledger/audit, not a data owner.
- **Statutory forms per TT 99/2025**, not TT 200/2014 (superseded 01/01/2026);
  TT 133/2016 kept as SME variant; NĐ 254/2026 for HĐĐT identity.
- **Posting before UI**; stat forms generated, never edited.
- **`X-User-Id` dev seam** for authz subject; fail closed when missing.

## Phases (per-module roadmap; setup next)
- Phase 0 Foundation: money/migrations/RBAC/audit + status machine
- Phase 1 Core lifecycle + orchestration seams
- Phase 2 Compliance slices + balances/books
- Phase 3 Import/export + reports + UX
- Phase 4 Integration hardening + regression
- Phase 5 Statutory verification + PROD pilot

## Checkpoints
After each phase: `go build ./...`, `go vet ./...`, `go test ./...` green;
authz tests pass; coverage ≥ 80% on service+repo.

## Risks
- Second source of truth for COA/regime (setup vs masterdata) → setup only
  orchestrates masterdata seams.
- Non-idempotent init double-seeds on crash-resume → status row + idempotent
  upstreams + property tests.
- Unbalanced opening balances slip into ledger → ΣNợ=ΣCó enforced at entry,
  check, lock (R9).
- `core.Money` float64 regression → int64 minor units only.
- SQLite contention on numbering/posting → sequence tables + single tx.
- Statutory drift → isolated templates + golden fixtures from official PDFs.
