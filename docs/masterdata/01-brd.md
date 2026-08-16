# Master Data Module (Hệ thống danh mục dữ liệu chính) — Business Requirements (BRD)

> Target state. Current implementation is a stub (see `00-verdict.md`). Owner:
> Kế toán trưởng (chief accountant) as business authority; BA lead as author.

## 1. Objective

Build the master-data registry (hệ thống danh mục dữ liệu chính) that every
other module consumes — the **single source of truth** for customers
(khách hàng), suppliers (nhà cung cấp), chart of accounts (sơ đồ tài khoản),
items (vật tư-hàng hóa), fixed-asset categories, employees, departments, units,
warehouses, funds, banks, currencies/tỷ giá, VAT rates and cost objects — with
the lifecycle, identity fields and audit trail Vietnamese law and auditors
require, satisfying TT 99/2025/TT-BTC, Luật Kế toán 2015 (as amended) and
NĐ 254/2026/NĐ-CP.

## 2. Business goals & success criteria

| # | Goal | Success criterion |
|---|---|---|
| G1 | Single source of truth | Every module reads master data via the registry API; no module holds a private copy of a code that the registry also owns |
| G2 | Statutory identity on Đối tượng | 100% of KH/NCC records carry valid MST / mã ĐVQHNS / số định danh cá nhân per NĐ 254/2026; invoice module can always render buyer identity from the registry |
| G3 | TT 99 chart of accounts ready | Sơ đồ tài khoản seeded from Phụ lục 2 TT 99/2025 (TT 133/2016 variant switchable); hierarchy + postable-leaves validated; amendments logged to Quy chế hạch toán |
| G4 | Data integrity | Code unique per catalog and immutable after reference; no orphan references; duplicate/merge tooling; no silent data loss |
| G5 | Governance | Every create/edit/deactivate/merge/import is audited; role separation enforced by Casbin; "ngừng sử dụng" replaces hard delete |
| G6 | Productivity | Excel import (dry-run + error report) and export for every catalog; bulk update; ≤ 30 s to add a customer from a Vietnamese business registration |
| G7 | Correctness over speed | All mutations transactional; sequence allocation atomic under concurrency |

## 3. Users & roles

| Role (VN) | Casbin role | Privileges |
|---|---|---|
| Thủ kho / Nhân viên danh mục | `danh_muc` | Create/edit catalogs, import/export, deactivate (with reason), when no balances block them |
| Kế toán tổng hợp | `ke_toan_tong_hop` | Create/edit chart of accounts, customers, suppliers; run merges; adjust tax rates |
| Kế toán trưởng | `ke_toan_truong` | Merge with re-pointing, deactivate records with live balances (forced, audited), regime switch (TT 99 ↔ TT 133), approve chart amendments |
| Giám đốc | `giam_doc` | Read-only: catalog views, quality reports |
| Kiểm toán (future) | `kiem_toan` | Read-only review |

Anonymous → fail closed (existing authz dev seam via `X-User-Id`).

## 4. Scope

**In scope (phase order in `06-roadmap.md`):**
- Master-data registry: typed catalogs, one API, per-type validation.
- Đối tượng (khách hàng / nhà cung cấp / nhân viên) — full VN statutory fields.
- Sơ đồ tài khoản (chart of accounts) — Phụ lục 2 TT 99/2025 seed + variant switch.
- Vật tư-hàng hóa + đơn vị tính + kho.
- TSCĐ danh mục (nhóm, khung khấu hao per TT 45/2013 family) — catalog only; engine is fixedasset module.
- Thuế suất GTGT/TNDN (versioned), ngân hàng, ngoại tệ/tỷ giá, quỹ, phòng ban, đối tượng tập hợp chi phí, lý do thu/chi.
- Lifecycle: create/edit/list/search/ngừng sử dụng/merge/import/export.
- Excel templates + error report; CSV export; audit trail; authz.

**Out of scope (now):** AI address normalization, group-wide/cloud shared
catalogs (AMIS-style), automated MST lookup API from cơ quan thuế, mobile UI,
full multi-entity (one company per install). Tax-rate *calculation* engines
belong to tax/invoice modules; masterdata only stores versioned rates.

## 5. Regulations (the contract this module must meet)

- **TT 99/2025/TT-BTC** (eff 01/01/2026): Phụ lục 2 hệ thống tài khoản (seeded
  master data); Điều 11 — account amendments permitted but must preserve BCTC
  line items and be covered by a Quy chế hạch toán kế toán.
- **TT 133/2016/TT-BTC** (in force; SMEs may stay): chart-of-account variant
  switch; consistency ≥ 1 FY; notify tax authority on switch.
- **Luật Kế toán 2015 (88/2015/QH13)** as amended by 56/2024 and 108/2025:
  ghi sổ kịp thời, đầy đủ; **không tẩy xóa** — drives no-hard-delete rule.
- **Luật Quản lý thuế 2025 (108/2025/QH15)** (eff 01/07/2026) Điều 11 + TT
  86/2024/TT-BTC: MST structures (10-digit; 13-digit for đơn vị phụ thuộc; số
  định danh cá nhân for individuals).
- **Nghị định 254/2026/NĐ-CP** (eff 01/07/2026) Điều 10 + Phụ lục: buyer
  identity on invoices (tên/địa chỉ/MST|mã ĐVQHNS|số định danh cá nhân|hộ
  chiếu+quốc tịch); identity must match registration; missing identity →
  invoice cannot be used to hạch toán chi phí.
- **TT 45/2013/TT-BTC family** (in force to 31/12/2026; TT 30/2025 latest
  amendment): TSCĐ nhóm + khung thời gian trích khấu hao.
- **Nghị định 123/2020/NĐ-CP + Nghị định 70/2025/NĐ-CP**: superseded in detail
  by NĐ 254/2026 — do not build to old invoice-form requirements.

## 6. Non-functional requirements

- **Consistency > speed**: mutations transactional; registry reads are the hot
  path — list/search 50k items < 500 ms; detail lookups < 50 ms (SQLite,
  JSON-doc + secondary index columns where benchmarks require).
- **Concurrency**: atomic code-sequence allocation (BEGIN IMMEDIATE, mirror
  cash/ledger); import must be resumable/idempotent.
- **Security**: all mutations behind Casbin; audit trail append-only; no
  secret fields in catalogs.
- **Tests**: ≥ 80% coverage on service + repository; property tests for
  uniqueness and for "no orphan references after merge".
- **Import robustness**: dry-run + per-row error report; max row limits;
  template versioning (import to old template blocked).

## 7. Assumptions & open questions

- Assumption: one entity (one company) per install; single-currency VND default.
- Assumption: default regime = TT 99/2025 (replaces TT 200 for enterprises);
  TT 133/2016 available as a switch for SME customers.
- Assumption: code format per catalog configurable at company setup, e.g.
  KH-/NCC-/VT-/NV-/KHO- prefixes; auto-code default on.
- Q1: Per-catalog auto-number prefix — fixed vs company-configurable? Default:
  company-configurable at setup, immutable afterwards.
- Q2: Should deactivation be blocked or just warned when live balances exist?
  Default: **blocked** for khách hàng/nhà cung cấp/vật tư/TSCĐ/tài khoản with
  non-zero balances; chief-accountant override with audit note.
- Q3: Merge granularity — full (codes + references + history) vs simple
  (re-point future references only)? Default: full merge with an audit record
  and a dry-run impact count.
- Q4: Import of MST validation — format-only locally vs live check against
  gdt.gov.vn API (deferred)? Default: format + checksum locally; live check v2.
