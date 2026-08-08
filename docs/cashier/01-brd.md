# Business Requirements Document (BRD) — Cashier (Thủ quỹ) Module

Document owner: BA Lead / Chief Accountant (20+ yrs VN accounting). Version 1.0. Date 2026-08-08.
Status: Draft for review. Governed by: Luật Kế toán 2015, TT 99/2025/TT-BTC (eff. 01/01/2026, replaces TT 200/2014), TT 133/2016/TT-BTC (SME).

## 1. Purpose & Scope

**Purpose:** Build the goGL **Cashier** capability so a VN SME can run statutory-compliant cash receipt/payment (thu/chi tiền mặt) day to day: vouchers with approval, cashier posting, daily cash book (Sổ quỹ tiền mặt), accountant's parallel detail book, monthly reconciliation, and cash counts (kiểm kê quỹ).

**In scope:**
- Cash funds (quỹ) per currency: VNĐ, FX, gold (vàng tiền)
- Phiếu thu (01-TT) / Phiếu chi (02-TT) lifecycle: draft → approved → posted → reconciled/voided
- Cashier workspace: post approved receipts/payments into cash book; daily closing; cash counts
- Cash book (S07-DN) + detail book (S07a-DN) generation and printing
- Monthly reconciliation between cashier and accountant with biên bản
- Role separation: preparer ≠ approver ≠ cashier ≠ cash accountant
- Integration seam to Ledger (TK 111 double-entry), AR/AP (customer/supplier settle), Payroll (lương), Tax (thuế nộp tiền mặt)

**Out of scope (future/separate):**
- Electronic cashierless payments (POS/e-wallet) — only manual cash at this stage
- Direct GDT/HTKK e-filing (tax module owns it)
- Bank module integration (Nộp/Rút tiền gửi is a cross-module flow, defined as interface only)

## 2. Background & Problem

- MISA SME splits **Quỹ** (cash accounting) from **Thủ quỹ** (cashier). Fast keeps a combined "Kế toán tiền mặt, tiền gửi, tiền vay". goGL has a single stub `cash` module that is 100% `TODO`.
- No separation of duties exists; without it the business cannot defend cash records to auditors or tax authorities.
- Statutory forms changed: **TT 99/2025/TT-BTC** replaces TT 200/2014 from FY2026 → new S07-DN/S07a-DN structures must be implemented now, not the 2014 forms.
- Failures to prevent: negative cash balance, out-of-sequence voucher numbers, one person both holding cash and posting the ledger, unreconciled cashier-vs-accountant books, direct erasures on the book.

## 3. Goals & Success Metrics

| Goal | Metric (definition of done) |
|---|---|
| Statutory-compliant voucher lifecycle | Phiếu thu/chi with full signature fields; sequential numbers per fund per period; no direct edits after posting |
| Role separation | Same user cannot be both cashier and cash accountant for same fund; enforced by Casbin + app rule |
| Daily cash book correctness | Tồn cuối = Tồn đầu + Thu − Chi; never negative; equals physical count at daily close |
| Monthly reconciliation | Cashier book vs accountant book equal after reconciliation; biên bản produced |
| Accounting accuracy | Every posted voucher generates valid double-entry on TK 111 + counter account; books balance |
| Testability | ≥ 80% coverage on service layer; handler integration tests; all paths incl. exception covered |

## 4. Stakeholders & Roles

| Role | Who | Responsibilities |
|---|---|---|
| Người đề nghị | Employee / Sales / AP clerk | Raises receipt/payment proposal (Đề nghị thu, chi) |
| Người duyệt | Manager / Chief Accountant / Director | Approves proposal + voucher per limit |
| Thủ quỹ (Cashier) | Cash custodian | Posts approved vouchers, holds cash, keeps S07-DN, daily close, counts |
| Kế toán tiền mặt (Cash Accountant) | Accountant | Posts to ledger, TK 111, keeps S07a-DN, reconciles |
| Kế toán trưởng (Chief Accountant) | Head accountant | Approves, enforces controls, signs reconciliation |
| Quản trị viên (Admin) | System admin | Configures funds, roles, policies, limits |

## 5. Regulatory Requirements (traceability)

| Reg ID | Requirement | Source |
|---|---|---|
| R1 | Every cash receipt/payment must have phiếu thu/chi with: unit name+address, number + seq-in-quyển, date, amount in number **and words**, FX rate if FX, signature+name of preparer, approver, receiver/giver, cashier | TT99 Art.4-6; Luật Kế toán 2015 Khoản 3 Điều 3; Thông tư số 02-TT form |
| R2 | Voucher numbers continuous & sequential within accounting period, per fund | TT99; MISA/Fast behavior |
| R3 | Cash book S07-DN: columns A(ghi sổ), B(chứng từ), C/D(số phiếu thu/chi), E(diễn giải), 1(thu), 2(chi), 3(tồn); Tồn = đầu + thu − chi; one book per currency; written daily in occurrence order | TT99; ketoanleanh.edu.vn guide |
| R4 | Detail book S07a-DN kept by cash accountant in parallel; periodic check + signature in column G | TT99 |
| R5 | No erasure/correction on books; corrections via adjustment entries per Điều 30 TT99 | TT99 |
| R6 | Segregation of duties: preparer/approver do not hold cash; cashier executes + writes S07-DN; cash accountant posts ledger; **nobody both holds cash and posts the ledger** | TT99 (internal control); VACPA guidance |
| R7 | Daily: cashier reconciles physical cash vs book; discrepancy → biên bản + report to chief accountant | TT99 |
| R8 | Monthly: cashier vs accountant books reconciled, signed; biên bản đối chiếu quỹ | TT99 |
| R9 | FX: separate books per currency; convert at bookkeeping rate on transaction date | TT99 |
| R10 | Cash count (kiểm kê quỹ) periodic + surprise; biên bản kiểm kê quỹ | TT99; MISA Thủ quỹ feature |

## 6. Functional Requirements

### FR1 Fund management
- Create cash funds keyed by code+currency (VNĐ/USD/EUR/Vàng). Fields: code, name, currency, opening balance (số dư đầu kỳ), active flag, assigned cashier(s).
- Reject opening balance for non-VNĐ unless FX rate provided at open date.

### FR2 Receipt/payment voucher lifecycle
States: `draft` → `approved` → `posted` → `reconciled` | `voided`. Transitions audited.
- Voucher header: fund, type (receipt/payment), date (ngày hạch toán), counterparty (customer/supplier/employee/other), amount number+words, FX amount + rate (if FX), reason, related documents (invoice no, contract no), signatures.
- Lines (double-entry): at minimum TK Nợ/TK Có derived from transaction type; editable by cash accountant only.
- Auto-generate voucher number on first save; sequential per fund/period.

### FR3 Cashier workspace
- List "Đề nghị thu, chi" (approved-but-unposted vouchers) → **Ghi sổ** → writes S07-DN row (date, running balance).
- Ghi sổ by voucher date or chosen date (MISA parity).
- Daily close: force reconciliation of S07-DN balance vs counted cash; block further posting until resolved or noted.

### FR4 Cash book (S07-DN) & detail book (S07a-DN)
- Generate per fund + period; print per template; running balance; per-currency.
- Column G signature for accountant on S07a-DN.

### FR5 Reconciliation & cash counts
- Monthly: reconcile S07-DN vs S07a-DN; produce biên bản đối chiếu; sign-off by cashier + cash accountant + chief accountant.
- Cash count: create biên bản kiểm kê quỹ (date, counted, book balance, difference, resolution); supports surprise counts.

### FR6 Reports
Sổ quỹ tiền mặt (S07-DN), Sổ chi tiết quỹ (S07a-DN), Báo cáo tồn quỹ theo ngày, Biên bản kiểm kê quỹ, Sổ cái TK 111 (via ledger seam).

### FR7 Role separation (Casbin)
Roles: `role:cashier`, `role:cash_accountant`, `role:chief_accountant`, `role:director`. Policies per action (see 02-spec §9). Enforced by AuthorizationMiddleware + app-level fund checks.

## 7. Non-functional Requirements

| NFR | Requirement |
|---|---|
| N1 Correctness | Money stored as exact decimal (fix `core.Money` float64 → int64 minor units or decimal lib) |
| N2 Audit | Every state transition logged (who, when, before/after) in `audit` module |
| N3 Performance | Cash book for 12 months < 500ms query; posting < 100ms p99 |
| N4 Security | All actions under `/api/v1/cash*` protected by Casbin; principal from `X-User-Id` dev seam |
| N5 Concurrency | No two posters may double-post same voucher (idempotency key / row lock on state) |
| N6 Availability | SQLite single-writer, WAL; no data loss on single write |
| N7 Localization | VN language UI; amounts with VN number-in-words formatting |

## 8. Assumptions

1. First deployment targets VN SME applying TT 99/2025/TT-BTC (general) — TT 133 forms as configurable templates later.
2. Cash is manual (két tiền) at launch; no POS integration in v1.
3. Authentication remains the dev seam (`X-User-Id`); production auth is a separate program of work. Cashier duties assume identities can be trusted at the app layer.
4. Ledger, AR/AP, payroll modules remain stubs; Cashier module defines interfaces and posts into ledger seam (ledger impl may lag).
5. Existing `core.Money` is float64 — will be replaced with an exact-decimal type; breaking change contained to cash module first.

## 9. Open Questions (for review)

1. Two modules (`cash` + `cashier`) or one module with role-gated actions? → Recommend **one module `cash`, role-gated**, minimizing churn; revisit if SME wants separate Thủ quỹ licensing like MISA.
2. Should VNĐ-only funds be the v1 scope with FX deferred to v2? → Recommend v1 supports multi-currency funds but FX difference engine (lãi/lỗ tỷ giá) in v2.
3. Numbering: per fund per period vs global per period? → Recommend per fund per period (statutory + MISA behavior).
4. Vàng tiền (gold) support: include or defer? → Defer to v2.

## 10. Acceptance Criteria (overall)

- [ ] Cashier can receive/approve/post/reconcile a full month of cash operations with role separation enforced
- [ ] S07-DN and S07a-DN print correctly per TT99 with running balance and signature fields
- [ ] Negative balance and duplicate posting are impossible
- [ ] Monthly biên bản đối chiếu generated and signed in-system
- [ ] Casbin denies cross-role actions (cashier cannot post ledger, accountant cannot post cash book)
- [ ] All service methods unit-tested (≥80% coverage); handlers integration-tested; `go test ./...` green
