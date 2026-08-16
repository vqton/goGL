# Master Data Module (Hệ thống danh mục dữ liệu chính) — PROD-Readiness Verdict

> Status: **NOT PROD-READY** — the module is a 4-layer stub and its domain model
> is fundamentally inadequate for Vietnamese accounting/ERP master data.
> Capabilities below describe the **target state** to be built. Date: 2026-08-11.

## 1. Executive verdict

| Criterion | Assessment | Evidence |
|---|---|---|
| Domain model | 🔴 Inadequate | `CatalogItem{ID, Category, Code, Name, Status}` only — a generic bag. No typed master lists (khách hàng, nhà cung cấp, vật tư-hàng hóa, TSCĐ, nhân viên, sơ đồ tài khoản, đơn vị tính, kho, ngân hàng, ngoại tệ, thuế suất), no hierarchy, no MST/ĐVQHNS/CCCD fields, no account-link fields (TK 131/331/152/632/511…), no validity dates, no audit fields |
| Application layer | 🔴 Stub | `CreateItem`/`GetItem` return `core.ErrNotImplemented`; no list/search/deactivate/merge/import/export |
| Persistence | 🔴 Stub | Repository returns `ErrNotImplemented`. Table `catalog_items` exists in `db.Migrate` but is never read/written |
| HTTP API | 🔴 Stub | `POST /api/v1/master-data/items` and `GET /api/v1/master-data/items/:id` → 501. No list, no search, no pagination |
| Authz | 🔴 Missing | No `danh_muc` / master-data roles or policies in Casbin seed; no role separation for who can create vs deactivate a catalog |
| Migrations | 🟡 Scaffolded | `catalog_items` in table list — but no sequence, no import jobs, no catalog-type registry tables |
| Money/laws mapping | 🔴 None | No tax rates, no MST validation (NĐ 254/2026, Luật QLT 108/2025), no chart-of-accounts seed (Phụ lục 2 TT 99/2025) |
| UI | 🔴 None | No web screens; no import-from-Excel; no multi-language names |
| Tests | 🔴 None | No test files in any masterdata package |
| Integration seams | 🔴 None | No consumer contracts for cash/ledger/invoice/payroll/inventory; code uniqueness & "ngừng sử dụng" rules absent |

**Verdict: the master data module cannot be used in production today — not even
as a demo.** Every call returns 501 / `ErrNotImplemented`, and the data model
would not satisfy a single statutory obligation (e-invoice buyer identity,
chart-of-accounts, customer/AR detail) even if it were implemented as-is.

## 2. What "production-ready" means for this module

Production-ready = the module is the single source of truth for all reference
data that every other module (cash, ledger, invoice, purchase, sales, payroll,
fixedasset, inventory, tax) consumes, and that data is **statutory-compliant
and governed**. Concretely (all verified by tests + demo):

1. **Typed master catalogs**, not a generic bag — at minimum: sơ đồ tài khoản,
   khách hàng, nhà cung cấp, vật tư-hàng hóa, TSCĐ, nhân viên, phòng ban,
   đơn vị tính, kho, quỹ, ngân hàng, ngoại tệ/tỷ giá, thuế suất GTGT,
   đối tượng tập hợp chi phí, lý do thu/chi.
2. **Full lifecycle** — create, update, list/search (code, name, MST, group),
   paginate, deactivate ("ngừng sử dụng") instead of hard delete once
   referenced, merge duplicates, Excel import/export.
3. **Code integrity** — auto-generation or user codes, uniqueness enforced per
   catalog, immutability of `Code` after any transaction references it, atomic
   sequence allocation under concurrency.
4. **Statutory field sets** — e.g. khách hàng/nhà cung cấp carry tên/địa chỉ/
   MST per giấy chứng nhận ĐKDN, mã ĐVQHNS, số định danh cá nhân, hộ chiếu +
   quốc tịch (NĐ 254/2026 Điều 10); sơ đồ tài khoản seeded from Phụ lục 2
   TT 99/2025 with hierarchy + "không hạch toán trực tiếp" parents.
5. **Governance** — audit trail (người tạo/sửa/ngừng, thời điểm), role
   separation (danh mục viên / kế toán trưởng / giám đốc), data-quality rules
   (không trùng MST, không tẩy xóa), reference-integrity guards.
6. **Consumer seams** — stable lookups (`GET /catalog/{type}/{code}`) usable by
   all modules; deactivate blocked while live balances/transactions reference
   the code (e.g. công nợ khách hàng ≠ 0).

## 3. Regulatory basis (verified current, 2026-08-11)

Sources checked: vbpl.vn / thuvienphapluat / congbao.chinhphu.vn /
luatvietnam.vn (cross-referenced with mof.gov.vn and gdt.gov.vn article feeds).

- **Luật Kế toán 2015 (88/2015/QH13)** — in force; consolidated as
  41/VBHN-VPQH (16/03/2026); amended by Luật 56/2024/QH15 (eff 01/01/2025) and
  Luật Quản lý thuế 108/2025/QH15. Principles: ghi sổ kịp thời, đầy đủ, rõ
  ràng, trung thực; sổ kế toán không tẩy xóa — drives "no hard delete, only
  ngừng sử dụng + audit".
- **Thông tư 99/2025/TT-BTC** (27/10/2025; **eff 01/01/2026**, FY from/after
  01/01/2026) — **replaces** TT 200/2014, TT 75/2015, TT 53/2016, TT 195/2012.
  Phụ lục 1 chứng từ, **Phụ lục 2 hệ thống tài khoản kế toán (chart of
  accounts = core master data)**, Phụ lục 3 sổ kế toán (42 mẫu). Điều 11:
  enterprise may add/amend accounts without MoF approval but must (a) not
  change BCTC line items, (b) issue a Quy chế hạch toán kế toán taking legal
  responsibility. → masterdata must support account hierarchy, type, and a
  "quy chế" audit note on amendments.
- **Thông tư 133/2016/TT-BTC** — **still in force**; not replaced by TT 99.
  SMEs may keep TT 133 or optionally adopt TT 99 (must apply consistently ≥ 1
  FY and notify the tax authority). Being reviewed 2026-2027 (Quyết định
  3389/QĐ-BTC 2025); **TT 58/2026/TT-BTC** replaces TT 132/2018 (siêu nhỏ) eff
  01/07/2026 but does **not** replace TT 133. → masterdata needs a regime
  switch (chart-of-accounts variant + form numbers), not a hardcode.
- **Luật Quản lý thuế 2025 (108/2025/QH15)** (10/12/2025; eff **01/07/2026**;
  HKD/cá nhân kinh doanh parts eff 01/01/2026). Điều 11: MST of cá nhân = số
  định danh cá nhân; MST of doanh nghiệp/HTX/đơn vị phụ thuộc per pháp luật
  (10-digit; 13-digit with "-" for đơn vị phụ thuộc). TT 86/2024/TT-BTC (đăng
  ký thuế) replaced TT 105/2020 from 06/02/2025; from 01/07/2025 cá nhân use số
  định danh cá nhân in place of MST. → customer/supplier/employee master data
  must validate and store these identifiers.
- **Nghị định 254/2026/NĐ-CP** (30/06/2026; eff **01/07/2026**) — implements
  Luật QLT 108/2025 on HĐĐT/chứng từ điện tử, replacing NĐ 123/2020 detail
  guidance (NĐ 70/2025 amendments superseded). Điều 10 + Phụ lục: invoice must
  show tên, địa chỉ, **MST / mã ĐVQHNS / số định danh cá nhân** của người mua;
  buyer identity must match giấy chứng nhận ĐKDN/ĐKDKTT; foreign buyers show hộ
  chiếu + quốc tịch; **invoice lacking buyer identity cannot be used to hạch
  toán chi phí / quyết toán thuế**. → khách hàng master data is a hard
  compliance gate for the invoice/sales modules.
- **Thông tư 45/2013/TT-BTC (TSCĐ)** — **continues in force until 31/12/2026**
  (Quyết định 1760/QĐ-BTC 01/07/2026), amended by TT 147/2016, TT 28/2017,
  **TT 30/2025/TT-BTC** (eff 15/07/2025). → TSCĐ master data (nhóm TSCĐ, khung
  thời gian trích khấu hao, TK khấu hao) must follow this family.
- **Nghị định 123/2020/NĐ-CP + Nghị định 70/2025/NĐ-CP** — prior e-invoice
  regime; NĐ 70 introduced mã ĐVQHNS on invoices; superseded for detail
  guidance by NĐ 254/2026 (do not build to the old form numbers).
- **Luật Thuế TNDN 2025 (67/2025/QH15)**, **Luật Thuế GTGT 2024 (48/2024/QH15)**
  — tax-rate master data must stay configurable (e.g. GTGT 0%/5%/10%, TNDN
  marginal rates) and versioned by effective date.

> ⚠️ During implementation, load the exact Phụ lục 2 (chart of accounts) and
> Phụ lục 3 (sổ) lists from the official TT 99/2025 PDF + the NĐ 254/2026
> invoice-field list; do not hardcode mã hiệu from memory.

## 4. Competitor scan (all active and TT 99-ready, 2026)

| Product | Offering | Master-data capabilities goGL must match |
|---|---|---|
| **MISA SME 2026 / AMIS Kế toán** | SME/SMB desktop+web | Danh mục Đối tượng (khách hàng / nhà cung cấp / nhân viên) with ~30 fields each: MST/CCCD chủ hộ, mã ĐVQHNS, hộ chiếu, điều khoản thanh toán, số ngày nợ, nợ tối đa, TK công nợ (131/331), ngân hàng, người nhận HĐĐT, người đại diện PL, tổ chức/cá nhân, vừa KH-vừa-NCC, đối tượng nội bộ, trường mở rộng 1-5, trạng thái Sử dụng/Ngừng sử dụng. Batch: xóa, gộp trùng (tự động theo tên/MST/SĐT/địa chỉ), phân nhóm, nhập/xuất Excel, cập nhật địa chỉ từ cơ quan thuế, AI chuẩn hóa địa chỉ. Danh mục VTHH: thông tin chung, ngầm định (TK kho/giá vốn/doanh thu, thuế suất), đơn vị chuyển đổi, định mức NVL, mã quy cách, bảo hành, lô. Kho. Danh mục dùng chung đồng bộ qua AMIS Hệ thống (Kế toán ↔ CRM ↔ Kho ↔ Mua hàng) |
| **Fast Accounting 12 / Fast Accounting Online** | SMB + enterprise | TT 99/2025 update shipped Dec-2025: Danh mục hệ thống tài khoản (đổi tên/xóa/bổ sung theo Phụ lục 2) + công cụ chuyển đổi tài khoản + bút toán điều chỉnh số dư đầu năm 2026. Danh mục hàng hóa vật tư: nhóm 1/2/3, loại cố định (21 Vật tư, 31 CCLĐ, 41 Bán thành phẩm, 51 Thành phẩm, 61 Hàng hóa), tabs Thông tin chung / Ngầm định / Lô, đổi đơn vị, khai báo hàng loạt, cây phân nhóm |
| **BRAVO 10 ERP** | Mid/mid-large, ISO 27001:2022 | Danh mục dữ liệu gốc centralized, phân quyền, kiểm soát sửa/xóa khi đã phát sinh, đồng bộ danh mục giữa các phân hệ |

**Differentiation gap:** for goGL's size, the table stakes are typed catalogs,
code uniqueness, ngừng sử dụng lifecycle, MST/ĐVQHNS fields, Excel import and
TT 99 chart-of-accounts seeding. Do **not** build AI address normalization,
group-wide shared catalogs, or API-marketplace sync until the statutory core
(customers/AR, suppliers/AP, chart of accounts, items, TSCĐ) is provably
correct.

## 5. Key points — what must change (review analysis)

1. **Replace the generic `CatalogItem` bag with a typed master-data registry.**
   One service module, per-type entities/schemas, one lookup API. A generic
   bag produces duplicated, unvalidated, unreportable data.
2. **Add the lifecycle the law and auditors require:** only soft-deactivate
   ("Ngừng sử dụng"); no hard delete once any transaction/chứng từ references
   the code (Luật Kế toán: không tẩy xóa). Block deactivation while live
   balances exist (công nợ, tồn kho, số dư sổ cái).
3. **Statutory identity fields for Đối tượng (KH/NCC):** MST 10-digit + mã
   ĐVQHNS + số định danh cá nhân/CCCD + hộ chiếu/quốc tịch (NĐ 254/2026); Tổ
   chức/Cá nhân flag; vừa KH-vừa-NCC; TK công nợ 131/331 default; điều khoản
   thanh toán; người nhận hóa đơn. These are *mandatory* inputs to the
   invoice module from 01/07/2026.
4. **Seed sơ đồ tài khoản from Phụ lục 2 TT 99/2025** (with TT 133/2016 variant
   switch), hierarchy + Level + postable-leaf-only + "không hạch toán trực
   tiếp" parents; amendments logged to a Quy chế hạch toán audit note (Điều 11).
5. **Code discipline:** unique per catalog; immutable after reference; atomic
   sequence for auto-codes (e.g. KH-00001, NCC-00001, VT-00001) — same
   BEGIN-IMMEDIATE pattern as cash/ledger sequences.
6. **Merge & dedup** (gộp đối tượng trùng) with full re-pointing of
   references, mirroring MISA/Fast; plus duplicate detection on create.
7. **Excel import/export** as first-class (MISA/Fast ship it; VN users live in
   Excel) with a dry-run/error report, not silent partial imports.
8. **Authz + audit:** role `danh_muc` (create/edit), `ke_toan_truong` (merge,
   deactivate-with-reason, regime switch), `giam_doc` read; every mutation to
   audit_logs.
9. **Consumer seams + test contract:** stable `GET /catalog/{type}` lookups,
   and a "referenced?" query so cash/ledger/invoice guard deactivation.
10. **Regime/period awareness:** tax-rate and chart-of-accounts variants
    versioned by effective date; avoid hardcoding form numbers (NĐ 254/2026
    superseded NĐ 123/2020 detail; TT 99/2025 superseded TT 200/2014).

## 6. Recommendation

1. **Treat master data as the foundation phase.** Cashier and ledger are already
   PROD-pilot; both *consume* customer/supplier/account/unit data. Build the
   registry vertical (P0–P2 in `06-roadmap.md`) before invoice/purchase.
2. **Ship P0 gate first:** migrations + typed catalog model + code uniqueness +
   authz + audit seam — mirroring the cash module's controls-first approach.
3. **Customers/suppliers + chart of accounts are the compliance-critical
   slices** (NĐ 254/2026, Phụ lục 2 TT 99/2025); items + units + TSCĐ next;
   employees/banks/currency/tax-rate as generic catalogs.
4. **Re-review after P2** (typed catalogs + lifecycle + import) — the earliest
   honest "can the invoice module rely on this" checkpoint.
