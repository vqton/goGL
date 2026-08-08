# Implementation Plan: Cashier (Thủ quỹ) Module

## Overview
Build the goGL Cashier capability: statutory-compliant cash receipt/payment lifecycle (phiếu thu/chi → duyệt → Ghi sổ → sổ quỹ → đối chiếu) with role separation, per TT 99/2025/TT-BTC (effective 01/01/2026). Currently `cash` is a 100% stub — see `docs/cashier/00-verdict.md`.

## Architecture decisions
- **Single `cash` module, role-gated** (not two modules) — minimizes churn; revisit if SME licensing demands separate Thủ quỹ.
- **Controls-first**: exact-decimal money, migrations, RBAC policies, audit seam before any feature.
- **Posting before UI**: behavior drives screens; S07-DN/S07a-DN are generated, never edited.
- **Ledger seam**: cash posts TK111 via interface to `ledger` (stub impl OK, documented).
- **Statutory forms per TT99**, not TT200/2014.

## Phases (see tasks/todo.md)
- Phase 0 Foundation: money type, migrations, RBAC, audit seam
- Phase 1 Funds + voucher lifecycle (draft→approved)
- Phase 2 Cashier posting + cash book (core value)
- Phase 3 Reconciliation, void, correction (Điều 30)
- Phase 4 Reports, print templates, UX
- Phase 5 Hardening

## Checkpoints
After each phase: `go build ./...`, `go vet ./...`, `go test ./...` green; authz tests pass.

## Risks
- `core.Money` float64 change ripples across 24 modules → Phase 0 isolation
- Ledger is stub → seam + mock
- SQLite contention on numbering/posting → sequence table + single tx
- Statutory drift → isolated templates + golden tests
- `X-User-Id` dev seam auth → app-level ownership checks; flag in hardening

## Open questions
1. One module vs split → default single role-gated
2. FX difference engine → v2 (multi-currency funds in v1)
3. Numbering per fund/period → confirmed
4. Vàng tiền → defer v2
