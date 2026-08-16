# Ledger Module (Kế toán tổng hợp) — Business Requirements (BRD)

> Target state. Current implementation is a stub (see `00-verdict.md`). Owner: Kế
> toán trưởng (chief accountant) as business authority; BA lead as author.

## 1. Objective

Build the general-ledger (kế toán tổng hợp) module that records every economic
event from operating modules (cash, bank, invoice, payroll, fixed assets, …) as
**balanced double-entry journal entries**, maintains the statutory **ledger books**
(Sổ Nhật ký chung, Sổ Cái, Bảng cân đối số phát sinh, Sổ chi tiết) in Vietnamese
accounting form, and safely closes each accounting period — satisfying
TT 99/2025/TT-BTC (in force 01/01/2026) and supporting tax filing.

## 2. Business goals & success criteria

| # | Goal | Success criterion |
|---|---|---|
| G1 | Every money movement from operating modules lands in the GL | Cash `LedgerWriter` posts 100% of cash vouchers; reconciliation cash↔GL shows zero unexplained gaps |
| G2 | Statutory books print-ready | Sổ Nhật ký chung, Sổ Cái, Bảng cân đối số phát sinh, Sổ chi tiết render per Phụ lục III (TT 99/2025) and match SUM-of-lines exactly |
| G3 | Double-entry integrity | Every entry balances; ledger rejects unbalanced/orphan entries with no exceptions |
| G4 | Correct period accounting | Postings to closed periods blocked; opening balances carry forward; kết chuyển cuối kỳ (511→911→421, doanh thu/chi phí) supported |
| G5 | Auditability | Append-only journal; corrections via đảo bút toán/bút toán điều chỉnh; full user trail |
| G6 | Role separation | Kế toán tổng hợp writes; Kế toán trưởng approves/close; Giám đốc read-only; enforced by Casbin |
| G7 | Exact money | Balances never suffer float drift; amounts integer minor units |

## 3. Users & roles

| Role (VN) | Casbin role | Privileges |
|---|---|---|
| Kế toán tổng hợp | `ke_toan_tong_hop` | Create/edit/print entries, journals, period open/close, kết chuyển |
| Kế toán trưởng | `ke_toan_truong` | Approve, correct (đảo bút toán), close period, opening balances, unlock closed period (with reason) |
| Giám đốc | `giam_doc` | Read-only: books, reports, balances |
| Kiểm toán (future) | `kiem_toan` | Read-only external review |

Anonymous → fail closed (existing authz dev seam via `X-User-Id`).

## 4. Scope

**In scope (phase order in `06-roadmap.md`):**
- Chart of accounts (sơ đồ tài khoản) — standard VAS account list (1xx→9xx), editable with structure validation (111, 1111, …).
- Journal entries — manual phiếu kế toán + auto posting from source modules.
- Ledger books — Sổ Nhật ký chung, Sổ Cái, Bảng cân đối số phát sinh, Sổ chi tiết.
- Accounting periods — open/close, opening balances, kết chuyển cuối kỳ templates.
- Statutory print (web) in TT 99/2025 Phụ lục III form.

**Out of scope (now):** BCTC full set (BCĐKT, BCKQKD, BCLCTT), IFRS, tỷ giá tự động, multi-currency engine, AI assistant, Nghị định 254/2026 e-invoice outbound.

## 5. Regulations (the contract this module must meet)

- **TT 99/2025/TT-BTC** (in force 01/01/2026): sổ kế toán forms per Phụ lục III; principles: ghi sổ theo chứng từ, số liệu khớp chứng từ, không tẩy xóa, sửa sai bằng bút toán điều chỉnh/đảo bút toán.
- **Luật Kế toán 2015, Nghị định 174/2016**: sổ kế toán bắt buộc (Sổ Nhật ký chung, Sổ Cái); ghi sổ đúng thời điểm, kịp thời; lưu trữ.
- **TT 133/2016** (SME): optional regime switch later.
- **Luật Quản lý thuế 108/2025/QH15** (eff 01/07/2026; replaces Luật QLT 2019; NĐ 123/2020 detail guidance superseded by NĐ 254/2026): books reconcile to tax declarations; monthly Sổ Cái / Bảng cân đối số phát sinh supports thuế GTGT/ TNDN filing.

## 6. Non-functional requirements

- **Correctness > speed**: posting path must be transactional; an entry either fully posts or fully rolls back.
- **Idempotency**: re-posting a source event (retry) must not duplicate entries.
- **Performance target**: book 12-month, ~50k entries — Sổ Cái/BCĐPS render < 2s (SQLite, indexed); guided by benchmarks like cash module.
- **Concurrency**: safe against concurrent posting; unique voucher sequence per period.
- **Security**: no data tampering; all mutations behind Casbin; close-period requires `ke_toan_truong`.
- **Tests**: ≥ 80% coverage on service + repository; property tests for double-entry balance invariant.

## 7. Assumptions & open questions

- Assumption: single-currency (VND) initially; multi-currency later.
- Assumption: one entity (one company) per install — no multi-entity.
- Q1: Does the pilot company adopt TT 99 (default yes, it replaces TT 200) or stay TT 133? → decides default chart of accounts.
- Q2: Voucher numbering — global sequential (00001/25) vs per-form sequential? Default: per-form per-period.
- Q3: Approve step — mandatory two-step (draft→posted) for manual entries, or post-immediately? Default: draft→posted for manual; auto-post direct for source modules.
- Q4: Fiscal year = calendar year (assume yes, as Vietnamese enterprise standard).
