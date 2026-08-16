# Task List — Master Data (Hệ thống danh mục dữ liệu chính) Module

Source: `docs/masterdata/06-roadmap.md`. Conventions: acceptance criteria +
verification per task (planning-and-task-breakdown). No task touches > 5 files.

## Phase 0 — Foundations & registry skeleton (hard gate)
- [ ] T0.1 Domain: `MasterRecord` + `CatalogKind` registry + typed views (Counterparty, Account, Item, TaxRate). Accept: compiles; `CatalogItem` deprecated (table kept, unused). Verify: `go build ./...`.
- [ ] T0.2 Migrations: `md_records`, `md_sequences`, `md_import_jobs`, `md_regimes`. Accept: created idempotently. Verify: `db.Migrate` twice, no error.
- [ ] T0.3 Seed Casbin role `danh_muc` + policies per `02-spec §9`; override/merge/seed/regime → `ke_toan_truong`. Accept: enforcement blocks `danh_muc` from merge/override. Verify: authz tests.
- [ ] T0.4 Scaffold masterdata vertical; HTTP list/search/detail on `:kind`. Accept: GET/POST reachable, 501 gone. Verify: httptest.
- [ ] T0.5 Audit seam for all mutations. Accept: every mutation writes audit row. Verify: integration test.

**Checkpoint:** build/vet/test green.

## Phase 1 — Lifecycle, codes, validation
- [ ] T1.1 Service rules R1–R12 (unique code, immutable-after-ref, no-hard-delete, deactivate guard, group/cycle, MST format, NĐ 254 identity, account hierarchy, validity overlap). Accept: each rule rejects. Verify: unit tests.
- [ ] T1.2 `md_sequences` auto-code (KH-, NCC-, VT-, …) + company prefix config. Accept: continuous per kind; no reuse. Verify: parallel-insert test.
- [ ] T1.3 ReferenceCount cache + `references` endpoint; maintained on consumer writes. Accept: matches re-scan. Verify: property test (UC-M16).
- [ ] T1.4 HTTP: create/update/deactivate/activate/delete. Accept: 201/200/409/422 with VN+EN. Verify: httptest.

**Checkpoint:** build/vet/test green.

## Phase 2 — Compliance slices: Đối tượng + Sơ đồ tài khoản
- [ ] T2.1 Customer/supplier schema + UI forms (UC-M1/M2/M3); KH-NCC toggle; invoice identity gate (R7). Accept: NĐ 254 identity required when invoice-enabled. Verify: tests + browser.
- [ ] T2.2 Chart of accounts: Phụ lục 2 TT 99/2025 seed + preview/apply + Quy chế note (Điều 11); R8 validation. Accept: seed golden match. Verify: golden fixture test.
- [ ] T2.3 Merge (UC-M4): dry-run impact + one-tx re-point + INACTIVE reason. Accept: no partial re-point on failure. Verify: rollback test.
- [ ] T2.4 Web pages: catalog list, customer form, accounts tree (`05-ui §2.1-2.3`). Verify: walkthrough.
- [ ] T2.5 TT 133/2016 variant switch (UC-M13) + FY-boundary guard. Verify: guard test.

**Checkpoint:** build/vet/test green.

## Phase 3 — Import/export + remaining catalogs
- [ ] T3.1 Excel import wizard (UC-M5): template v2, dry-run, per-row errors, idempotent upsert, error CSV; export all kinds. Accept: errors never silently dropped. Verify: import tests.
- [ ] T3.2 Item catalog (UC-M8, loại 21/31/41/51/61); unit/warehouse/department (UC-M9); tax-rate versioned (UC-M10); bank/currency/group/cost-object/reason; TSCĐ catalog (TT 45 family). Verify: unit tests.
- [ ] T3.3 Data-quality dashboard (`05-ui §2.7`): missing MST, dup codes/MST, missing TK defaults; CSV export. Verify: scan query tests.

**Checkpoint:** build/vet/test green.

## Phase 4 — Integration seams + hardening
- [ ] T4.1 Consumer Go interface (Lookup/Resolve/ReferenceCount) → invoice (identity), ledger (AllowPost), inventory/purchase/sales (TK defaults), payroll (employee/department), cash (fund). Verify: seam mock assertions.
- [ ] T4.2 Deactivate guard end-to-end (UC-M3/M16). Verify: cash/invoice write paths maintain count; deactivate blocked.
- [ ] T4.3 Concurrency/perf: parallel create+import soak; 50k list < 500 ms, detail < 50 ms. Verify: benchmark.
- [ ] T4.4 Security review → `08-security.md`; coverage ≥ 80% service+repo. Verify: `go test -cover`, `go vet`.

**Checkpoint:** build/vet/test green.

## Phase 5 — Statutory verification + pilot
- [ ] T5.1 Official Phụ lục 2 (TT 99/2025) + NĐ 254/2026 Điều 10 verification; fix drift. Verify: golden fixtures.
- [ ] T5.2 Kế toán trưởng walkthrough: setup, import, seed, one month of invoice/purchase → identities on HĐĐT, accounts post in ledger. Verify: sign-off checklist.
- [ ] T5.3 Regression suite + benchmark report. Verify: `go test ./...`.
- [ ] T5.4 `00-verdict.md` re-scored → PROD verdict for pilot. Verify: doc updated.
