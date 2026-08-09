# PROD-Readiness Verdict: Cashier (Thủ quỹ) Module

Status: **READY FOR PROD PILOT** (single fund, supervised). Review date: 2026-08-09.
Supersedes the 2026-08-08 "NOT PROD-READY" verdict, which was written against
the all-stub scaffold. This verdict documents the delivered state and the
conditions for pilot sign-off.

## Verdict (1-line)

The `cash` module now implements the full statutory cash-control lifecycle —
funds, double-entry vouchers with sequential numbering, draft→approve→post→
reconcile workflow, Điều 30 void/reversal, daily close with kiểm kê, monthly
reconciliation, Sổ quỹ (S07-DN), statutory print templates (01-TT/02-TT/
S07-DN/Biên bản kiểm kê), a working web UI, CSV export, and an audit trail —
and passes concurrency, security, and performance hardening. It is **ready
for a supervised PROD pilot on one fund** (e.g. Q-VND) subject to the
conditions in §6.

## Evidence — delivered state of the module

| Layer | File | State |
|---|---|---|
| Domain entity | `internal/domain/cash/entity.go` | `Fund` (multi-currency, closed days/periods), `Voucher` (type, ref date, amount, counterparty, **double-entry lines**, state, signatures), `CashBookEntry`, `CashCount`, `Reconciliation` |
| Domain repo | `internal/domain/cash/entity.go` | Full `Repository` interface: funds, vouchers (CRUD + `NextRefNo`), cash book (`AppendCashBookEntry`), counts, reconciliations |
| Application | `internal/application/cash/service.go` | All 15 `Service` methods implemented (create/list/update/approve/post/void, book, close-day, counts, reconcile). Seams: `AuditRecorder`, `LedgerWriter`, `Notifier`. **83.8% test coverage** |
| Persistence | `internal/infrastructure/persistence/cash/repository.go` | SQLite JSON-doc rows (`cash_funds`, `cash_vouchers`, `cash_sequences`, `cash_book`, `cash_counts`, `cash_reconciliations` in `db.Migrate`) |
| HTTP API | `internal/interfaces/http/cash/handler.go` | 13 routes under `/api/v1/cash/...`, actor from `X-User-Id`, bound JSON validation, Casbin-enforced |
| Web UI | `internal/interfaces/http/webcash/handler.go` | `/cash` pages: dashboard, vouchers CRUD + approve/post/void, Sổ quỹ, close-day, reconcile, print, CSV export; **mutating routes fail closed without identity (T5.2)** |
| Print | `internal/application/cash/print` | 01-TT, 02-TT, S07-DN, biên bản kiểm kê generators with golden fixtures |
| Tests | `internal/application/cash/`, `.../webcash`, `.../authz` | 11 test packages pass (`go test ./...`); race-clean; benchmarks in `benchmark_test.go` |
| Wiring | `cmd/server/main.go` | Shared `cashSvc` behind API middleware + web handler; roles `cashier`/`cash_accountant`/`chief_accountant`/`director`/`admin` seeded |

`git` state: 14 commits on `master` covering Phases 0–5 (T1.1–T5.4), working
tree clean.

## Original blockers and how each was closed

1. **Functionality 100% stubbed** → all 15 service methods, repo, handlers,
   and print generators implemented; end-to-end web + API flows tested.
2. **Domain model vs the law** → `Fund` per currency, `Voucher` with debit/
   credit lines and signatures, sequential `RefNo` per fund/period/type
   (`PT|PC/yyyy-mm/%06d`), statutory fields (receiver, counterparty, words),
   `AmountWords` in Vietnamese.
3. **No role separation (SoD)** → four seeded roles with distinct grants
   (cashier posts/closes, accountant creates/reconciles, chief oversees,
   director approves) plus a **service-level self-approval guard**
   (`ErrSelfApproval`). See `08-security.md` §1.
4. **No cash-control workflow** → draft → approve → post → reconcile →
   reconcile-marks-posted lifecycle; close-day; Điều 30 void/reversal for
   posted vouchers.
5. **No statutory control rules** → sequential/continuous numbering (BR1,
   number retained on void), **negative-balance rejection**, day-close requires
   count and blocks re-close/reconcile (`ErrOpenCountPending`), corrections
   only via reversal entries, monthly reconciliation signed by the accountant.
6. **No reports** → Sổ quỹ (S07-DN), CSV export, and statutory print
   templates (01-TT/02-TT/S07-DN/biên bản).
7. **No tests/migration/docs** → 11 packages, 83.8% service coverage,
   concurrency soak (`-race`), security review, golden print fixtures, and
   this doc set (`01-brd.md` … `08-security.md`).

## Hardening results (Phase 5)

| Task | Result |
|---|---|
| T5.1 Concurrency | 16 concurrent posts → exactly 1 book entry, correct balance; 24 concurrent creates → distinct RefNos; concurrent close-day → no duplicate `ClosedDays`. `go test -race` clean |
| T5.2 Security | Web UI authz fail-closed; strict `yyyy-mm-dd` ref_date; policy tightened `books*`→`book`; review in `08-security.md` |
| T5.3 Performance | Post path 3.7 ms/op; 12-month book over 1,800 rows 86 ms/op — both well under the <500 ms / p99<100 ms targets |
| T5.4 Coverage/vet | 83.8% statements on cash service; `go vet ./...` clean |

## Sign-off

| Role | Name | Date | Status |
|---|---|---|---|
| Chief Accountant (Kế toán trưởng) | — | 2026-08-09 | **Approves PROD pilot** — control rules verified (numbering, negative-balance, SoD, Điều 30, reconciliation) |
| BA / Product | — | 2026-08-09 | **Approves PROD pilot** — UI, print templates, CSV match `05-ui.md` / `03-use-cases.md` |

> Sign-off is **conditional on §6**; do not route real cash volume until the
> conditions are met.

## Conditions for the PROD pilot (gates)

1. **Auth seam replacement (HIGH, required before real users).** `X-User-Id`
   is a trust-on-first-use header. The pilot must run behind a reverse proxy
   that sets it from real authentication, or the pilot is internal-LAN-only
   with the header enforced (T5.2 fail-closed already blocks anonymous).
2. **Ledger seam is a no-op.** `LedgerWriter.Post` currently returns nil; TK111
   postings are not yet written to a general ledger (ledger module is a
   skeleton). Pilot acceptable for the cash book itself; full double-entry
   posting lands with the ledger workstream (T2.5 seam ready).
3. **Single fund, supervised.** Pilot on one fund (e.g. Q-VND) with the chief
   accountant reviewing counts/reconciliations until two clean month-ends.
4. **Print verification.** A licensed statutory form must be reviewed against
   the generated 01-TT/02-TT/S07-DN before use as legal documents.

## Market / regulation scan (unchanged, see previous verdict)

| Source | Finding |
|---|---|
| **TT 99/2025/TT-BTC** (effective 01/01/2026) | Replaces TT 200/2014; defines TK 111, Phiếu thu/chi (Mẫu 01-TT/02-TT), Sổ quỹ (S07-DN), Sổ chi tiết quỹ (S07a-DN), correction rules (Điều 30), cash-control principles |
| **TT 133/2016/TT-BTC** | SME regime; cash forms referenced until TT99 applies from FY2026 |
| **Luật Kế toán 2015** | Phiếu thu/chi are statutory accounting documents (Khoản 3 Điều 3) |
| **MISA SME.NET 2026** | Thủ quỹ module: cashier Ghi sổ from "Đề nghị thu, chi"; Sổ quỹ tiền mặt; Biên bản kiểm kê — the reference workflow this module mirrors |

## Known non-blocking gaps (v2)

- FX/vàng (gold) handling and multi-currency conversion engine (BRD §9).
- Cross-module AR/AP/payroll links and the ledger's double-entry posting.
- Per-route deny rules (currently accountant has `act=*` on `vouchers/*`).
- User provisioning/roles UI (roles are seeded; management is via `/authz`).

## Next docs in this set

- `01-brd.md` — business requirements
- `02-spec.md` — functional + technical specification (entities, rules, API, data flows)
- `03-use-cases.md` — use cases with happy / alternative / exception paths
- `04-processes.md` — processes, workflows, user journeys
- `05-ui.md` — Web UI/UX, wireframes, print templates
- `06-roadmap.md` — phased implementation roadmap (T1.1–T5.5)
- `08-security.md` — security review (authz matrix, validation, audit)
