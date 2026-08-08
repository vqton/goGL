# PROD-Readiness Verdict: Cashier (Thủ quỹ) Module

Status: **NOT PROD-READY** — blocked at every layer. Review date: 2026-08-08.

## Verdict (1-line)

The goGL `cash` module (which is meant to cover both **Quỹ/Tiền mặt** accounting and **Thủ quỹ/Cashier** duties) cannot operate in a production environment in its current state — all four layers are `TODO` stubs that return `core.ErrNotImplemented` (service/repo) or HTTP 501 (handler), there is no entity model that matches Vietnamese statutory cash-accounting, and there is no separation of the Cashier role from the Cash Accountant role.

## Evidence — current state of the module

| Layer | File | State |
|---|---|---|
| Domain entity | `internal/domain/cash/entity.go` | `Voucher{ID, RefNo, RefDate, CashAccount, Counterparty, Amount, Status}` — toy model. No debit/credit, no double-entry, no statutory form fields, no signatures, no per-currency fund, no sequential numbering |
| Domain repo | `internal/domain/cash/entity.go:19-22` | `Create`, `FindByID`, `Update` — interface only |
| Application | `internal/application/cash/service.go` | `CreateVoucher`/`GetVoucher` both `// TODO` returning `core.ErrNotImplemented` |
| Persistence | `internal/infrastructure/persistence/cash/repository.go` | All 3 methods `// TODO: implement sqlite ...` returning `core.ErrNotImplemented` |
| HTTP | `internal/interfaces/http/cash/handler.go` | Registers `POST /api/v1/cash/vouchers`, `GET /api/v1/cash/vouchers/:id` — both return `http.StatusNotImplemented` (501) |
| Tests | — | None. `go test ./...` passes only because there is nothing to run |
| Wiring | `cmd/server/main.go:121` | Registered under `/api/v1/cash` behind the authz middleware when enabled |

`git` state: repo initialized on `master`, **zero commits**, everything untracked.

## Key points — why it cannot ship

### 1. Functionality: 100% stubbed
Every code path returns "not implemented". No voucher can be created, read, listed, posted, or printed. Nothing persists.

### 2. Domain model does not match the law
Vietnamese statutory cash accounting (TT 99/2025/TT-BTC, effective 01/01/2026, which **replaces TT 200/2014**; and TT 133/2016 regime for SMEs) requires:
- **Phiếu thu (Mẫu 01-TT)** / **Phiếu chi (Mẫu 02-TT)** with statutory fields (see `01-brd.md` §5)
- **Sổ quỹ tiền mặt (Mẫu S07-DN)** — cashier's daily book: columns A/B/C/D/E/1/2/3, `Tồn cuối = Tồn đầu + Thu − Chi`, one book per currency
- **Sổ kế toán chi tiết quỹ tiền mặt (Mẫu S07a-DN)** — accountant's parallel book, signed in column G during periodic reconciliation
- **TK 111** (Tiền mặt: 1111 VNĐ, 1112 ngoại tệ, 1113 vàng tiền) with **double-entry** postings — the goGL `Voucher` has a single `Amount` and no journal lines

Current `Voucher` entity has none of this.

### 3. No role separation (segregation of duties) — a legal control
The law is explicit: *"Không được để một người vừa giữ tiền vừa ghi sổ kế toán quỹ"* (one person must not both hold the cash and book the account). Three distinct duties:
1. **Người lập/duyệt chứng từ** — preparer/approver, must not hold cash
2. **Thủ quỹ** — executes receipt/payment against approved vouchers, keeps the cash, writes Sổ quỹ (S07-DN)
3. **Kế toán tiền mặt** — posts to the ledger, monitors TK 111, reconciles with the cashier

goGL has a single `cash` module with a single `Voucher`. Casbin RBAC exists (roles/policies) but no cashier/accountant roles or policies are defined for this domain.

### 4. No cash-control workflow
No draft → approve → post → reconcile lifecycle. MISA SME's Thủ quỹ module is exactly this: the accountant books a "Đề nghị thu, chi" and the cashier **Ghi sổ** (posts into the cash book) from the "Đề nghị thu, chi" tab. goGL has one `CreateVoucher` call that would both create and post in one step — no separation, no approval, no audit trail.

### 5. No statutory control rules
Missing checks that any Vietnamese accounting package enforces:
- Voucher numbers **sequential and continuous within the accounting period**
- **No negative cash balance** (Tồn quỹ âm — illegal/wrong; "chi trước, thu sau" is a common source of error)
- **No same-day-overdraw** and daily `Tồn quỹ` must equal physical cash in the safe
- **No erasing/correcting directly on the book** — corrections only via adjustment entries per **Điều 30 TT 99/2025/TT-BTC**
- Per-currency books (VNĐ, USD, EUR…) with exchange-rate conversion at posting-date rate for FX
- Monthly reconciliation between cashier book and accountant book, signed, with biên bản

### 6. No reports
Sổ quỹ tiền mặt, Sổ chi tiết quỹ, biên bản kiểm kê quỹ, daily cash position — none exist.

### 7. No tests, no migration, no docs
Zero coverage. DB migration list (in `internal/infrastructure/db/migrate.go`) has no cash table (the module is document-row JSON anyway, but no table is registered). No BRD/spec/use-cases exist (this doc set is the first).

## Market / regulation scan (sources cited)

| Source | Finding |
|---|---|
| **TT 99/2025/TT-BTC** (MOF, 27/10/2025, effective 01/01/2026) | Replaces TT 200/2014 entirely. Defines TK 111, Phiếu thu/chi (Mẫu 01-TT/02-TT), Sổ quỹ (S07-DN), Sổ chi tiết quỹ (S07a-DN), correction rules (Điều 30), cash-control principles. Source: ketoanleanh.edu.vn, thuvienphapluat.vn, taca.edu.vn |
| **TT 133/2016/TT-BTC** | SME regime — cash forms remain referenced until TT99 applies from FY2026 |
| **Luật Kế toán 2015** | Defines phiếu thu/chi as statutory accounting documents (Khoản 3 Điều 3) |
| **MISA SME.NET 2026** (active) | 18 phân hệ incl. **Thủ quỹ**: cashier posts "Đề nghị thu, chi" → **Ghi sổ**; views Sổ quỹ tiền mặt; checks Biên bản kiểm kê. Module optional (Hệ thống → Tùy chọn → Ẩn/hiện nghiệp vụ) when roles not split. Source: helpsme.misa.vn, sme.misa.vn |
| **Fast Accounting 11 R09** (active, 24k+ customers) | Phân hệ "Kế toán tiền mặt, tiền gửi, tiền vay"; auto-create phiếu thu from sales invoices. Source: fast.com.vn |
| **Bravo ERP** (active) | Full ERP; cash/treasury module integrated with purchasing/sales/AR/AP |
| **BIG-4 / professional bodies** | E&Y, PwC, Deloitte, KPMG Vietnam, VACPA, VAA all publish TT99 adoption guidance — consistent with the above |
| **GDT (gdt.gov.vn)** | Electronic filing (thuedientu) targets tax declarations/invoices, not internal cash books; cashier module stays internal |

## What "PROD-ready" would mean (target gap-closure)

1. Split responsibilities: **cash (Quỹ/Tiền mặt accounting)** + **cashier (Thủ quỹ custody/book)** — either two modules or one module with role-gated actions
2. Statutory entity model (funds, vouchers with debit/credit, signatures, sequential numbering)
3. Posting engine that writes TK 111 + counter-account with double-entry, rejects negative balance
4. Cash book (S07-DN) + detail book (S07a-DN) generators, monthly reconciliation flow
5. Print templates for Phiếu thu (01-TT), Phiếu chi (02-TT), Sổ quỹ, Biên bản kiểm kê
6. Casbin roles: `role:cashier`, `role:cash_accountant`, `role:chief_accountant`, `role:director` + policies
7. Audit trail + unit/integration tests + migration entry

## Recommended immediate actions

- Do **not** route any real cash volume through `cash` in prod
- Treat this doc set as the approved spec baseline (spec-driven development)
- Implement per `06-roadmap.md`, module split first, controls before convenience

## Next docs in this set

- `01-brd.md` — business requirements
- `02-spec.md` — functional + technical specification (entities, rules, API, data flows)
- `03-use-cases.md` — use cases with happy / alternative / exception paths
- `04-processes.md` — processes, workflows, user journeys
- `05-ui.md` — Web UI/UX, wireframes, print templates
- `06-roadmap.md` — phased implementation roadmap
