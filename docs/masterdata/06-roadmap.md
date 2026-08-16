# Master Data Module (Hệ thống danh mục dữ liệu chính) — Implementation Roadmap

> Mirrors the cash/ledger phase pattern (P0→P6). **Precondition P0 is a hard
> gate.** Each phase ends with `go build ./... && go vet ./... && go test ./...`
> green and a commit. Test-first (red-green-refactor); ≥ 80% coverage on
> service + repository. Tasks tracked in `tasks/masterdata-todo.md`.

## P0 — Foundations & registry skeleton (hard gate) [1–2 wks]

- M0.1 Replace stub domain: `MasterRecord` + `CatalogKind` registry + typed
  views (Counterparty, Account, Item, TaxRate; simple catalogs as generic
  records). Migrate off `CatalogItem` (keep `catalog_items` table unused,
  document deprecated).
- M0.2 Migrations: add `md_records`, `md_sequences`, `md_import_jobs`,
  `md_regimes` to `db.Migrate`; secondary index column for unique
  `(kind, code)` (benchmark before over-indexing).
- M0.3 Authz: seed `danh_muc` role + policies per `02-spec §9`; gate
  deactivate-override/merge/seed/regime to `ke_toan_truong`; tests.
- M0.4 Scaffold masterdata vertical (domain/app/persistence/http) replacing
  stubs; HTTP list/search/detail on `:kind`.
- M0.5 Audit seam: every mutation writes `audit_logs` (user, at, reason).

**Checkpoint:** build/vet/test green; authz matrix tests pass.

## P1 — Lifecycle, codes, validation [1–2 wks]

- P1.1 Service rules R1–R12 (uniqueness, code-immutability-after-reference,
  no-hard-delete, deactivate guard, group/cycle checks, MST/ĐVQHNS format,
  NĐ 254 identity requirement, account hierarchy, validity overlap).
- P1.2 `md_sequences`: atomic auto-code (KH-, NCC-, VT-, …) with
  BEGIN-IMMEDIATE; company-set prefix config (Q1).
- P1.3 ReferenceCount cache: `references` endpoint + transactional maintenance
  on consumer writes (UC-M16 property test).
- P1.4 HTTP: create/update/deactivate/activate/delete per `02-spec §4`; 409/422
  with VN+EN messages.
- Tests: uniqueness (parallel insert), immutability-after-reference, deactivate
  guard, group cycle, MST checksum, NĐ 254 identity set.

## P2 — Compliance slices: Đối tượng + Sơ đồ tài khoản [2–3 wks]

- P2.1 Customer/supplier full schema (UC-M1/M2/M3) + UI forms
  (`05-ui §2.2`); both-KH-NCC toggle; invoice-consumption identity gate.
- P2.2 Chart of accounts: Phụ lục 2 TT 99/2025 seed fixture + preview/apply
  + Quy chế hạch toán note (Điều 11); hierarchy/AllowPost validation (R8);
  TT 133/2016 variant switch (UC-M6/M7/M13).
- P2.3 Merge (UC-M4): dry-run impact + one-tx re-pointing + INACTIVE reason.
- P2.4 Web pages: catalog list, customer form, accounts tree (`05-ui §2.1-2.3`).
- Tests: seed golden fixture (accounts count/levels), amendment-with-quyche,
  merge re-point + rollback, regime switch guard (FY boundary).

## P3 — Import/export + remaining catalogs [2 wks]

- P3.1 Excel import wizard (UC-M5): template v2, dry-run job, per-row errors,
  idempotent upsert, job report + error CSV; export for all kinds.
- P3.2 Remaining typed catalogs: item (UC-M8, Fast-compatible loại
  21/31/41/51/61), unit/warehouse/department (UC-M9), tax-rate versioned
  (UC-M10), bank/currency/group/cost-object/reason; TSCĐ catalog (nhóm + khung
  khấu hao per TT 45/2013 family).
- P3.3 Data-quality dashboard (`05-ui §2.7`): missing MST, dup codes/MST,
  missing TK defaults; export CSV.
- Tests: import dry-run error report, idempotent re-import, template version
  rejection, tax-rate overlap (R10), quality scan queries.

## P4 — Integration seams + hardening [1–2 wks]

- P4.1 Consumer Go interface (`Lookup/Resolve/ReferenceCount`) wired into
  invoice (buyer identity), ledger (account AllowPost), inventory/purchase/
  sales (item TK defaults), payroll (employee/department), cash (fund).
- P4.2 Deactivation guard end-to-end: invoice/purchase/payroll write paths
  maintain ReferenceCount; deactivate blocked when > 0 (UC-M3/M16).
- P4.3 Concurrency/performance: parallel create + import soak; list/search
  50k items < 500 ms, detail < 50 ms (mirror cash benchmarks); index tuning
  only if failing.
- P4.4 Security review (authz matrix, audit, tamper-proofing) →
  `docs/masterdata/08-security.md`.
- P4.5 Coverage ≥ 80% service+repo; vet clean.

## P5 — Statutory verification + pilot [1–2 wks]

- P5.1 Load official Phụ lục 2 (TT 99/2025) PDF + NĐ 254/2026 Điều 10 field
  list; match seed + identity validation exactly; fix drift.
- P5.2 Kế toán trưởng walkthrough: company setup (UC journey 9.1), customer/
  supplier import from a real registration set, chart seed, one month of
  invoice/purchase data → verify identities render on HĐĐT and accounts post
  in ledger.
- P5.3 Regression suite for the pilot month; benchmark report.
- P5.4 `docs/masterdata/00-verdict.md` re-scored → PROD verdict for pilot.

## Sequencing notes

- P0 blocks everything. P1→P3 strict order (lifecycle before compliance slices
  before import). P4 depends on P1-P3 (consumer seams need stable rules).
- Parallelizable: P2.2 (chart seed) is independent of P3 import work; P4
  integration can start with the ledger account seam once P2.2 lands.
- Dependency on other modules: consumes the existing `core.Money` (int64),
  `audit_logs`, Casbin enforcer — all present. Ledger `Account` seeding must be
  coordinated with ledger P1 (the ledger spec also defines `Account`; masterdata
  is the owner, ledger reads via the seam to avoid two sources of truth).

## Estimates & risks

| Risk | Impact | Mitigation |
|---|---|---|
| Two sources of truth for chart of accounts (md vs ledger) | high | masterdata owns accounts; ledger consumes via seam; seed run from masterdata (P2.2 + P4.1) |
| NĐ 254/2026 identity drift | medium | P5.1 official-text verification; golden tests on invoice rendering |
| Import corrupts data silently | high | dry-run mandatory; batch tx; per-row error report; idempotent (kind, code) |
| Deactivate bypass leaves orphan refs | high | ReferenceCount maintained transactionally + re-scan on deactivate (UC-M16) |
| Code-sequence contention | low | BEGIN IMMEDIATE sequence; parallel-create test (UC-M15) |
| TT 133 ↔ TT 99 regime mixing | medium | regime badge in UI; FY-boundary guard (UC-M13 E1) |
