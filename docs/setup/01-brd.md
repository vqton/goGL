# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — Business Requirements (BRD)

> Target state. Current implementation is a stub (see `00-verdict.md`). Owner:
> Kế toán trưởng (chief accountant) as business authority; BA lead as author.

## 1. Objective

Deliver the **first-run gateway** of goGL: a single, audited, restart-safe
session that turns an empty database into a legally configured, usable company
— statutory company identity (per giấy chứng nhận ĐKDN), accounting regime
(TT 99/2025 default, TT 133/2016 option), 12-month fiscal year with opened
monthly periods in the ledger, a regime-consistent chart of accounts seeded
through masterdata, and **balanced opening balances** (Tổng Nợ = Tổng Có) with
per-đối tượng detail for customers/suppliers/items/fixed assets — satisfying
Luật Kế toán 88/2015 (Điều 12), TT 99/2025/TT-BTC (Điều 5, 11, 31), TT
133/2016, Luật QLT 108/2025 and NĐ 254/2026.

## 2. Business goals & success criteria

| # | Goal | Success criterion |
|---|---|---|
| G1 | One-session onboarding | A new company goes EMPTY → ACTIVE in one guided wizard; all steps idempotent and restart-safe (re-run after crash never double-seeds) |
| G2 | Statutory company identity | Profile carries tên, MST (10/13-số, validated), mã ĐVQHNS, địa chỉ + người đại diện per ĐKDN, ngành nghề, loại hình — the seller identity NĐ 254/2026 requires on every HĐĐT |
| G3 | Regime correctness | Default TT 99/2025; TT 133/2016 selectable for SME; regime recorded, switchable only at FY boundary, COA seeded through masterdata (single source of truth) |
| G4 | Fiscal-year integrity | Kỳ kế toán năm = 12 tháng (Luật Kế toán Điều 12); ledger monthly periods opened at setup; no posting into un-opened/un-locked periods |
| G5 | Balanced opening balances | Total Nợ == Total Có enforced (with đồng tiền breakdown); per-đối tượng detail required for 131/331/152/211-214; dry-run import before commit |
| G6 | Control & audit | Initialize/regime/lock gated to `ke_toan_truong`/`admin`; every step appended to `audit_logs`; balances locked once any period posts |
| G7 | Productivity | Opening balances importable from an official Excel template (dry-run + per-row error report); ≤ 15 min to complete setup for a mid-size company |

## 3. Users & roles

| Role (VN) | Casbin role | Privileges |
|---|---|---|
| Kế toán trưởng | `ke_toan_truong` | Run/finalize initialization, choose regime, approve opening balances (lock), reopen with reason |
| Kế toán tổng hợp | `ke_toan_tong_hop` | Enter/edit opening balances (draft state), upload import, view status |
| Giám đốc | `giam_doc` | Read-only: profile, status, balance summary |
| Admin | `role:admin` | Override all (existing `* *`); first boot on an empty store is effectively admin-run |
| Kiểm toán (future) | `kiem_toan` | Read-only review of profile + setup audit trail |

Anonymous → fail closed (existing authz dev seam via `X-User-Id`).

## 4. Scope

**In scope (phase order in `06-roadmap.md`):**
- Status machine: `EMPTY → PROFILED → REGIME_SET → ACCOUNTS_SEEDED →
  PERIODS_OPEN → BALANCES_DRAFT → BALANCES_LOCKED → ACTIVE`, idempotent steps.
- Company profile (statutory fields, MST/ĐVQHNS validation, VND default
  currency, ngày bắt đầu hạch toán, regime choice).
- Orchestration: masterdata `SetRegime`/`SeedAccounts`, ledger
  `OpenPeriod` (12 months), audit writes — all in one transaction or a
  resumable step log.
- Opening balances: per-account + per-đối tượng detail, Nợ/Có, currency,
  balance-check (Nợ=Có), dry-run import (CSV/Excel), lock/reopen with reason.
- HTTP API + web wizard + status page; authz; audit.
- Setup-time checks that anchor other modules (fiscal year, regime badge).

**Out of scope (now):** multi-company/tenancy, "khởi tạo dữ liệu năm mới"
carry-forward across fiscal years, automated MST lookup from gdt.gov.vn, AI
address normalization, tax-declaration configuration (tax module), payroll
periods (payroll module), backup/restore of a partial setup (backup module).

## 5. Regulations (the contract this module must meet)

- **Luật Kế toán 88/2015/QH13** (as amended): **Điều 12** — kỳ kế toán năm = 12
  tháng dương lịch (or 12-month fiscal year); setup enforces the boundary.
- **TT 99/2025/TT-BTC** (eff 01/01/2026, FY from/after 01/01/2026): Điều 5
  (đơn vị tiền tệ kế toán = VND, ngoại tệ only under criteria + BCTC quy đổi);
  Điều 11 (account amendments need Quy chế hạch toán — COA seeded, amendments
  audited); **Điều 31 (regime choice for SME: TT 99 or TT 133, consistency ≥ 1
  FY + notify tax authority)**.
- **TT 133/2016/TT-BTC** — still in force, SME variant; switch at FY boundary.
- **Luật Quản lý thuế 108/2025/QH15** (eff 01/07/2026) + **TT 86/2024/TT-BTC**:
  MST formats (10-số; 13-số with "-" for đơn vị phụ thuộc; số định danh cá
  nhân for individuals) — validate company MST at setup.
- **NĐ 254/2026/NĐ-CP** (eff 01/07/2026) Điều 10: seller identity on HĐĐT
  (tên, địa chỉ, MST/mã ĐVQHNS per ĐKDN) — company profile is the seller
  master record for the invoice module.
- **TT 58/2026/TT-BTC** — replaces TT 132/2018 (siêu nhỏ) eff 01/07/2026; not a
  TT 133 replacement; siêu nhỏ handling lives in tax/masterdata, not setup.
- **NĐ 123/2020 + NĐ 70/2025** — superseded in detail by NĐ 254/2026 — do not
  build to old invoice-form requirements.

## 6. Non-functional requirements

- **Restart-safety**: every setup step idempotent; a crash mid-wizard leaves a
  resumable status (step-log), never a half-seeded COA or half-opened periods.
- **Consistency**: initialize runs in one SQLite transaction where possible;
  cross-module steps (masterdata, ledger, audit) are recorded so a resume
  re-checks before re-running (no double effects).
- **Concurrency**: single writer (SQLite); parallel `initialize` calls → one
  winner, others get 409 "đang khởi tạo"; sequence/period creation atomic.
- **Security**: all mutations behind Casbin; audit append-only; no secrets in
  profile; MST not logged outside audit as free text.
- **Tests**: ≥ 80% coverage on service + repository; property tests for
  balance-check (sum Nợ == sum Có) and idempotent re-run; authz matrix test.
- **Import robustness**: dry-run mandatory before commit; per-row error report;
  template versioning; idempotent by (TK, đối tượng) key.

## 7. Assumptions & open questions

- Assumption: one entity (one company) per install; VND default; regime default
  = TT 99/2025 (replaces TT 200/2014 for enterprises).
- Assumption: opening-balance lock is the gate for live posting — ledger
  already refuses posting into CLOSED periods (ledger R4), setup makes
  `BALANCES_LOCKED → ACTIVE` the moment period 1 opens with postings allowed.
- Q1: Should opening balances require đối tượng detail (KH/NCC/vật tư/TSCĐ) or
  allow lump-sum per TK? Default: **require** for balance-sheet TKs that carry
  đối tượng in the regime (131, 331, 152/155/156, 211/214, 331 …), allow lump
  for others; `ke_toan_truong` override.
- Q2: Import format — CSV first (house pattern: masterdata uses CSV) vs .xlsx
  (MISA/Fast ship .xlsx)? Default: **CSV v1, .xlsx v2** — keep the masterdata
  import pattern.
- Q3: Reopen policy — any reopen after any posting? Default: reopen allowed
  only if no posted vouchers reference the edited TK; otherwise block (chief
  override with reason + audit).
- Q4: When is regime effectively locked? Default: locked at
  `BALANCES_LOCKED/ACTIVE`; switching requires FY-boundary guard + dry-run COA
  diff (mirrors masterdata UC-M13).
