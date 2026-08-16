# Ledger Module (Kế toán tổng hợp) — PROD-Readiness Verdict

> Status: **NOT PROD-READY** — the module is a 4-layer stub. All capabilities below
> describe the **target state** to be built. Date: 2026-08-09.

## 1. Executive verdict

| Criterion | Assessment | Evidence |
|---|---|---|
| Domain model | 🔴 Stub | `JournalLine{AccountCode, Debit, Credit core.Money}` + `JournalEntry{ID, VoucherNo, VoucherDate, Lines, Status}` only — no account/period/template/opening models, no closing logic |
| Application layer | 🔴 Stub | `CreateEntry` / `GetEntry` return `core.ErrNotImplemented` |
| Persistence | 🔴 Stub | Repository returns `ErrNotImplemented`; **no ledger tables** in `db.Migrate` |
| HTTP API | 🔴 Stub | `POST /api/v1/ledger/entries` and `GET /api/v1/ledger/entries/:id` → 501 |
| Money correctness | 🔴 Blocker | `core.Money` is still **float64** (float rounding corrupts ledger balances) |
| Ledger posting seam | 🔴 Missing | Cash module's `LedgerWriter` (`internal/application/cash/service.go`) is still `noopLedger{}` — cash journal is **never posted to the general ledger** |
| Print / reporting | 🔴 None | No Sổ Nhật ký chung, Sổ Cái, Bảng cân đối số phát sinh, Sổ chi tiết |
| UI | 🔴 None | No web screens under `/api/v1/ledger/...` or web UI |
| Authz | 🔴 Missing | No ledger roles/policies in Casbin seed |
| Tests | 🔴 None | No test files in any ledger package |

**Verdict: the ledger module cannot be used in production today.** It is a contract
placeholder. It does not yet satisfy the most basic legal obligation (double-entry
journaling with balanced entries) and it silently breaks the cashier pilot: every
cash transaction is posted to a `noopLedger`, so **no GL record exists for any
money movement in production.**

## 2. What "production-ready" means for this module

Production-ready = the module enables the company to satisfy its statutory
accounting obligations under current law, and cash/bank/invoice/payroll events
reliably flow into the GL. Concretely (all verified by tests + demo):

1. **Double-entry journaling** — every entry balances (`Σ Debit = Σ Credit`), lines
   reference real chart-of-account codes, and no line is ever orphaned.
2. **Exact money** — amounts stored as integer minor units (`AmountMinor int64`);
   `core.Money` float64 is retired or the ledger never touches it.
3. **Posting from source modules** — cash `LedgerWriter` seam is implemented; posting
   is atomic and idempotent (retry-safe); a failed posting surfaces, never silently
   drops.
4. **Sổ kế toán (ledger books)** — Sổ Nhật ký chung, Sổ Cái, Bảng cân đối số phát
   sinh, Sổ chi tiết render per Phụ lục III của TT 99/2025/TT-BTC.
5. **Period lifecycle** — open/close period, reject postings to closed periods,
   opening balances carried forward, kết chuyển cuối kỳ.
6. **Auditability** — every entry is append-only; corrections use đảo bút toán /
   bút toán điều chỉnh; full create/modify/delete trail by user.
7. **Authz** — `ke_toan_tong_hop` (Kế toán tổng hợp), `ke_toan_truong` (Kế toán
   trưởng), `giam_doc` (Giám đốc) roles with read/write/close privileges.

## 3. Regulatory basis (confirmed current)

- **Thông tư 99/2025/TT-BTC** (27/10/2025, effective **01/01/2026**) — replaces
  TT 200/2014/TT-BTC for enterprises; the operative regulation for this module.
  Phụ lục III prescribes sổ kế toán forms and mã hiệu (e.g. **S01-DN Sổ Nhật ký –
  Sổ Cái**, **S02a-DN Chứng từ ghi sổ**, **S02c1-DN / S02c2-DN Sổ Cái**, Bảng cân
  đối số phát sinh, Sổ chi tiết). ⚠️ Exact mã hiệu/mẫu list must be loaded from the
  official Phụ lục III PDF during implementation — do not hardcode from memory.
- **Thông tư 133/2016/TT-BTC** — remains valid for SMEs that do not adopt TT 99;
  the product may need an SME regime switch later.
- **Nghị định 254/2026/NĐ-CP** (e-invoice, effective 01/07/2026) — reporting
  integration context; not in scope for the ledger core.
- **Luật Kế toán 2015** + **Nghị định 174/2016/NĐ-CP** — principles: ghi sổ kịp
  thời, đầy đủ, rõ ràng, trung thực; sổ kế toán không tẩy xóa.
- **Luật Quản lý thuế 108/2025/QH15** (eff 01/07/2026; replaces Luật QLT 2019)
  — sổ sách làm cơ sở kê khai thuế; books must reconcile with tax declarations.
  (Nghị định 123/2020 detail guidance superseded by Nghị định 254/2026/NĐ-CP —
  do not build to the old invoice/book form numbers.)

## 4. Competitor scan (all active and TT 99-ready in 2026)

| Product | Offering | Ledger-relevant strengths goGL must match |
|---|---|---|
| **MISA SME 2026 / AMIS Kế toán** | SMB/SME, desktop + web | 200+ reports, phân hệ kế toán tổng hợp, khoá sổ / bỏ khoá sổ kỳ, chứng từ ghi sổ, bút toán kết chuyển, in sổ Nhật ký chung / Sổ Cái / S06; AVA AI assistant |
| **Fast Accounting 11 (R09)** + Fast Accounting Online | SMB + enterprise | 11/14 phân hệ; Kế toán tổng hợp = phiếu kế toán, bút toán điều chỉnh, bút toán định kỳ, phân bổ cuối kỳ, chênh lệch tỷ giá, khoá sổ, kết chuyển, in sổ cái / chi tiết / bảng cân đối số phát sinh |
| **BRAVO 10 ERP** | Mid/mid-large, ISO/IEC 27001:2022 | 12 modules; Quản lý Tài chính-Kế toán: sổ cái, sổ chi tiết tài khoản, bảng cân đối tài khoản, BCTC theo VAS và IFRS |

**Differentiation gap:** goGL must at minimum deliver — for its planned size —
the core statutory outputs (balanced double-entry, Sổ Nhật ký chung, Sổ Cái, Bảng
cân đối số phát sinh, chi tiết, khoá sổ, kết chuyển). These are table stakes, not
differentiators. do not build AI/IFRS until the statutory core is provably correct.

## 5. Recommendation

1. **Fix money first.** Retire `core.Money` float64 for all financial modules or
   add `AmountMinor` discipline; ledger correctness is impossible otherwise.
2. **Implement the posting seam.** Cash → GL posting is the pilot's missing half;
   without it the cashier pilot's accounting data is fictional.
3. **Build the GL vertical** per `06-roadmap.md` (phases P0→P6), tests-first.
4. **Re-review after P3** (posting + balanced double-entry + period lock) — that is
   the earliest honest "can this run alongside the cashier pilot" checkpoint.
5. **Then** iterate statutory prints + kết chuyển cuối kỳ to pass a chief
   accountant walkthrough with the exact Phụ lục III mẫu biểu.
