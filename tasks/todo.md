# Task List — Cashier (Thủ quỹ) Module

Source: `docs/cashier/06-roadmap.md`. Conventions: acceptance criteria + verification per task (planning-and-task-breakdown). No task touches > 5 files.

## Phase 0 — Foundation
- [ ] T0.1 Exact-decimal money type (`core.Money` → int64 minor units). Accept: no float in cash paths. Verify: `go build ./...`; vet clean.
- [x] T0.2 Migrations: `cash_funds`, `cash_vouchers`, `cash_book`, `cash_counts`, `cash_sequences`. Accept: tables created idempotently. Verify: `db.Migrate` runs twice, no error.
- [x] T0.3 Seed Casbin roles/policies (cashier, cash_accountant, chief_accountant, director). Accept: enforcement blocks accountant from `/post`. Verify: authz tests.
- [x] T0.4 Audit seam for create/update/transition. Accept: every mutation writes audit row. Verify: integration test (wired into cash service in Phase 1).

**Checkpoint:** build/vet/test green.

## Phase 1 — Funds + voucher lifecycle
- [x] T1.1 Domain: Fund, Voucher, VoucherLine, enums. Accept: compiles, types per `02-spec §2`.
- [x] T1.2 Repository: fund+voucher CRUD, NextRefNo seq. Accept: sequence continuous per fund/period/type. Verify: repo test with parallel inserts.
- [x] T1.3 Service: CreateFund/ListFunds/CreateVoucher/UpdateVoucher/ApproveVoucher. Accept: lines balance (BR5); refno assigned; approve≠preparer (R6). Verify: unit tests.
- [x] T1.4 HTTP handlers + wiring in main.go. Accept: create/get/list/approve reachable. Verify: `go run ./cmd/server` smoke + httptest.
- [ ] T1.5 Lifecycle tests. Verify: `go test ./internal/application/cash/... ./internal/interfaces/http/cash/...`.

## Phase 2 — Cashier posting + cash book
- [x] T2.1 Repo: AppendCashBookEntry/ListCashBook/balance lookup. Accept: running balance correct. Verify: golden test.
- [x] T2.2 Service: PostVoucher/GetCashBook. Accept: BR2 (no negative), BR8 (close guard), BR10 (idempotent). Verify: unit tests.
- [x] T2.3 Service: CloseDay/CreateCashCount. Accept: diff → open CashCount + notify chief. Verify: tests.
- [x] T2.4 HTTP: post/close-day/counts + S07-DN view. Accept: cashier can Ghi sổ. Verify: httptest.
- [x] T2.5 Ledger seam interface + mock impl. Accept: posting writes TK111 entry via seam. Verify: mock assertion.
- [x] T2.6 Posting-rule tests (negative, double-post, close). Verify: coverage ≥ 80% service.

## Phase 3 — Reconciliation, void, correction
- [x] T3.1 ReconcileMonth → biên bản → 3-way sign → reconciled. Accept: books must match; open count blocks. Verify: tests.
- [x] T3.2 VoidVoucher (direct + Điều 30 reversal pair, chief approval). Accept: posted void requires reversal. Verify: tests.
- [x] T3.3 HTTP + S07a-DN view + biên bản endpoints. Verify: httptest.
- [x] T3.4 UC-5/UC-6 error-path tests. Verify: `go test ./...`.

## Phase 4 — Reports, templates, UX
- [ ] T4.1 Print templates (01-TT, 02-TT, S07-DN, S07a-DN, biên bản). Accept: render per `05-ui §7`. Verify: golden strings.
- [ ] T4.2 Web pages (dashboard, voucher forms, queue, sổ, khóa sổ, đối chiếu). Accept: walkthrough of full month. Verify: manual + browser-qa.
- [ ] T4.3 Tailwind + VN formatting + amount-in-words + i18n. Verify: visual check; number tests.
- [ ] T4.4 CSV export + UX polish. Verify: export correctness.
- [ ] T4.5 Golden S07-DN fixture test. Verify: `go test` includes fixture.

## Phase 5 — Hardening
- [ ] T5.1 Concurrency/idempotency soak. Accept: no duplicate entries under parallel post. Verify: race test.
- [ ] T5.2 Security review (authz matrix, validation, audit). Verify: review checklist + security-audit.
- [ ] T5.3 Performance targets (book <500ms, post p99<100ms). Verify: benchmark.
- [ ] T5.4 Coverage ≥ 80% cash service; vet clean. Verify: `go test -cover`, `go vet`.
- [ ] T5.5 PROD-readiness sign-off doc update. Verify: `00-verdict.md` updated to ready.
