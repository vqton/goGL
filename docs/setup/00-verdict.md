# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — PROD-Readiness Verdict

> Status: **NOT PROD-READY** — the module is a 4-layer stub. Its dependencies
> (masterdata, ledger, audit, Casbin) are now implemented, so setup is the
> missing **first-run gateway** every other module expects to exist before real
> data is entered. Capabilities below describe the **target state**. Date:
> 2026-08-16.

## 1. Executive verdict

| Criterion | Assessment | Evidence |
|---|---|---|
| Domain model | 🔴 Inadequate | `CompanyProfile{ID, Name, TaxCode, FiscalYearStart, AccountingRegime}` — 5 fields. Missing: địa chỉ/người đại diện theo ĐKDN, mã ĐVQHNS, quy mô/loại hình DN, đơn vị tiền tệ kế toán, ngày bắt đầu hạch toán, trạng thái khởi tạo, MST-format validation, link to fiscal periods. `InitialBalance{AccountCode, Period, Debit, Credit}` — no đối tượng (KH/NCC/vật tư), no trạng thái nháp/khóa, no balance-check result, no actor/audit fields |
| Application layer | 🔴 Stub | `Initialize`/`GetProfile`/`ImportOpeningBalances` return `core.ErrNotImplemented`. No fiscal-period creation, no regime wiring to masterdata, no COA seed orchestration, no status endpoint, no finalize/lock, no dry-run import |
| Persistence | 🔴 Stub | All repo methods return `ErrNotImplemented`. Tables `company_profiles` + `opening_balances` exist in `db.Migrate` (line 34–35) but are never read/written |
| HTTP API | 🔴 Stub | `POST /setup/initialize`, `GET /setup/profile/:id`, `POST /setup/opening-balances` → 501. No `GET /setup/status`, no list/edit balances, no balance-check endpoint, no CSV/Excel import |
| Authz | 🔴 Missing | No setup-specific policies; `initialize`/`finalize` must be gated to `ke_toan_truong`/`admin`; the authz middleware is in place and working, so this is purely un-wired |
| Migrations | 🟡 Scaffolded | `company_profiles`, `opening_balances` in table list — good, no new table strictly needed beyond a status/sequence row |
| Money/laws mapping | 🔴 None | No fiscal-year validation (Luật Kế toán 88/2015 Điều 12 — kỳ kế toán năm = 12 tháng dương lịch), no MST format/checksum (Luật QLT 108/2025), no regime default (TT 99/2025), no currency rules (Điều 5 TT 99), no BCTC-year boundary logic |
| UI | 🔴 None | No web screens; no first-run wizard; no "Khởi tạo dữ liệu" page |
| Tests | 🔴 None | No test files in any setup package |
| Integration seams | 🔴 None | Setup should orchestrate masterdata (`SetRegime`, `SeedAccounts`) + ledger (`SeedDefaultAccounts`, `OpenPeriod`) + audit — none of these seams exist yet |

**Verdict: the setup module cannot be used in production today — not even as a
demo.** Every call returns 501 / `ErrNotImplemented`, and the entity model would
not satisfy a single statutory obligation (fiscal year, MST, regime, balanced
opening balances). Moreover, setup is the **entry point** of the product: the
PROD-pilot cashier and ledger modules have no working company identity, fiscal
period, or opening-balance state to anchor to.

## 2. What "production-ready" means for this module

Production-ready = a new customer can go from an empty database to a legally
configured, locked-for-use company in one audited session, and that
configuration is **statutory-compliant and restart-safe**. Concretely (all
verified by tests + demo):

1. **First-run company configuration** — một bước khởi tạo duy nhất:
   tên công ty, MST (10-số; 13-số for đơn vị phụ thuộc), địa chỉ + người đại
   diện theo giấy chứng nhận ĐKDN, mã ĐVQHNS (optional), loại hình/ngành nghề,
   ngày bắt đầu hạch toán, chế độ kế toán (TT 99/2025 default; TT 133/2016
   option cho SME), đơn vị tiền tệ kế toán (VND default).
2. **Fiscal year + periods** — kỳ kế toán năm = 12 tháng (Luật Kế toán Điều
   12); setup tạo + mở các kỳ kế toán tháng trong ledger (`OpenPeriod`) và ghi
   nhận năm tài chính để BCTC boundary đúng.
3. **Chart-of-accounts orchestration** — sau khi chọn regime, setup gọi
   masterdata `SeedAccounts` (Phụ lục 2 TT 99/2025 hoặc TT 133/2016), KHÔNG
   tự seed riêng một bộ TK — tránh hai nguồn chân lý.
4. **Opening balances có kiểm soát** — nhập/import số dư đầu kỳ theo từng TK
   và (bắt buộc) theo đối tượng (KH 131 / NCC 331 / vật tư 152 / TSCĐ 211-214),
   kiểm tra **cân đối Tổng Nợ = Tổng Có** và tổng ngoại tệ, dry-run trước khi
   commit, khóa số dư sau khi mở kỳ có phát sinh, mọi thay đổi ghi audit.
5. **Status machine** — trạng thái khởi tạo rõ ràng:
   `EMPTY → PROFILED → REGIME_SET → ACCOUNTS_SEEDED → PERIODS_OPEN →
   BALANCES_DRAFT → BALANCES_LOCKED → ACTIVE`; mọi bước idempotent và
   restart-safe (re-run không nhân đôi seed).
6. **Authz + audit** — chỉ `ke_toan_truong`/`admin` khởi tạo, đổi regime,
   khóa số dư; mọi bước ghi `audit_logs`.
7. **Idempotency & rollback** — initialize dùng transaction; chạy lại sau
   crash an toàn; không bao giờ seed 2 lần COA hay mở 2 lần cùng kỳ.

## 3. Regulatory basis (verified current, 2026-08-16)

Sources checked: vbpl.vn / thuvienphapluat / congbao.chinhphu.vn /
luatvietnam.vn / mof.gov.vn feeds (cross-referenced with MISA/Fast/Bravo
release notes).

- **Luật Kế toán 2015 (88/2015/QH13)** — in force; consolidated
  41/VBHN-VPQH (16/03/2026); amended by Luật 56/2024/QH15 (eff 01/01/2025) and
  Luật QLT 108/2025/QH15. **Điều 12: kỳ kế toán năm là 12 tháng dương lịch**;
  đơn vị có thể chọn năm tài chính khác (năm tài chính) nhưng vẫn phải là 12
  tháng liên tục. → setup phải ép 12-tháng, ghi nhận ngày bắt đầu hạch toán,
  và dùng làm boundary cho mở/khóa kỳ.
- **Thông tư 99/2025/TT-BTC** (27/10/2025; **eff 01/01/2026**, FY from/after
  01/01/2026) — **replaces** TT 200/2014, TT 75/2015, TT 53/2016, TT 195/2012.
  Điều 5: đơn vị tiền tệ kế toán = VND (doanh nghiệp có thể chọn ngoại tệ nếu
  doanh thu/chi phí chủ yếu bằng ngoại tệ, phải thông báo và quy đổi VND khi
  lập BCTC). Điều 6: hoạt động liên tục. Điều 11: doanh nghiệp được bổ sung/
  sửa tài khoản nếu không làm thay đổi các chỉ tiêu BCTC và có Quy chế hạch
  toán. Điều 31: **đối tượng áp dụng** — mọi doanh nghiệp; SME có thể tiếp tục
  TT 133/2016 hoặc chọn áp dụng TT 99 (nhất quán ≥ 1 năm tài chính + thông báo
  cơ quan thuế). → setup phải có lựa chọn regime + ghi nhận quyết định áp dụng.
- **Thông tư 133/2016/TT-BTC** — **still in force** (SME option). Reviewed
  2026–2027 (Quyết định 3389/QĐ-BTC 2025); **TT 58/2026/TT-BTC** replaces
  TT 132/2018 (siêu nhỏ) eff 01/07/2026 but not TT 133. → regime switch must
  allow TT 99 ↔ TT 133; siêu nhỏ handled by the tax module, not setup.
- **Luật Quản lý thuế 2025 (108/2025/QH15)** (eff **01/07/2026**) + TT
  86/2024/TT-BTC: MST doanh nghiệp = mã số theo giấy chứng nhận ĐKDN (10-số;
  13-số with "-" cho đơn vị phụ thuộc); MST cá nhân = số định danh cá nhân. →
  setup validates company MST format at initialization.
- **Nghị định 254/2026/NĐ-CP** (eff **01/07/2026**) — HĐĐT/chứng từ điện tử
  replacing NĐ 123/2020 detail guidance. Điều 10: invoice must show tên, địa
  chỉ, MST/mã ĐVQHNS của **người bán** per ĐKDN. → company identity captured at
  setup is the seller identity the invoice module needs; validate at entry.
- **Luật Thuế GTGT 2024 (48/2024/QH15)** (eff 01/07/2025) + Luật 149/2025/QH15
  amendments (eff 01/01/2026, ngưỡng 500 tr cho HKD); **Luật Thuế TNDN 2025
  (67/2025/QH15)**. → tax-rate versioning handled by masterdata; setup only
  selects regime + opens fiscal year.
- **NĐ 123/2020 + NĐ 70/2025** — prior e-invoice regime; superseded for detail
  guidance by NĐ 254/2026 — do not build to old form numbers.

> ⚠️ During implementation, load the exact Phụ lục 2 (TT 99/2025) / Phụ lục 1
> (TT 133/2016) chart fixtures from the official PDFs; do not hardcode mã hiệu
> from memory. Reuse masterdata's `SeedAccounts` (already TT 99-ready) as the
> single seed path.

## 4. Competitor scan (all active and TT 99-ready, 2026)

| Product | Offering | Setup capabilities goGL must match |
|---|---|---|
| **MISA AMIS Kế toán** (web, R90 06/02/2025 → 2026 R-series) | SME/SMB cloud | Wizard "Khởi tạo dữ liệu": bước 1 Thông tin doanh nghiệp (tên, MST, địa chỉ, người đại diện, ngành nghề, chế độ kế toán TT 99/TT 133, kỳ kế toán, đơn vị tiền tệ) → bước 2 Tải sơ đồ tài khoản theo chế độ → bước 3 Nhập số dư đầu kỳ (Excel hoặc tay, **kiểm tra cân đối Nợ=Có ngay khi nhập**, cho phép nhập theo đối tượng/ngoại tệ, số dư chi tiết cho KH/NCC/VTHH/TSCĐ) → bước 4 Hoàn tất. "Khởi tạo dữ liệu năm mới" / "Cập nhật số dư năm trước": kế thừa danh mục + số dư + chứng từ sang năm mới, tự chuyển đổi TK theo TT 99. Chỉ cho phép 1 công ty/đợt khởi tạo; nếu đã phát sinh sẽ chặn hoặc yêu cầu khóa niên độ |
| **Fast Accounting 12 / Online** | SMB + enterprise | TT 99/2025 update (Dec-2025): công cụ chuyển đổi tài khoản + bút toán điều chỉnh số dư đầu năm 2026. Khởi tạo: "Khai báo thông tin doanh nghiệp", "Năm tài chính", "Số dư đầu kỳ" nhập theo TK + đối tượng, kiểm tra cân đối, import Excel; mở khóa chỉ khi niên độ chưa phát sinh |
| **BRAVO 10 ERP** | Mid/mid-large, ISO 27001:2022 | "Khởi tạo hệ thống" + "Số dư ban đầu": khai báo thông tin công ty, năm tài chính, đồng tiền, mở kỳ theo niên độ, nhập số dư đầu kỳ từ Excel với báo cáo lỗi, khóa số dư sau khi nhập liệu |

**Differentiation gap:** for goGL, the table stakes are: one-wizard
initialization, regime-aware COA seed **through masterdata**, fiscal-year +
period creation **through ledger**, balanced opening balances with đối tượng
detail and Excel import, and a status machine with audit. Do **not** build
multi-company/tenancy, AI-based MST lookup, or full "khởi tạo năm mới"
carry-forward until the single-company core is provably correct.

## 5. Key points — what must change (review analysis)

1. **Replace the 5-field `CompanyProfile` with a statutory company entity** —
   tên, tên tiếng Anh, MST (10/13-số, validated + normalized), mã ĐVQHNS
   (optional), địa chỉ + người đại diện per ĐKDN, loại hình, ngành nghề, đơn vị
   tiền tệ kế toán (VND default), ngày bắt đầu hạch toán, regime, năm tài chính.
2. **Status machine + idempotent steps** — track
   `EMPTY→…→ACTIVE` in a row; every step re-runnable; initialize inside a
   transaction; never double-seed COA/periods (mirror `SeedDefaultAccounts`
   idempotency in ledger).
3. **Wire setup to the implemented modules** — setup is the **orchestrator**,
   not a data owner: calls masterdata `SetRegime` + `SeedAccounts`, ledger
   `OpenPeriod`, audit `audit_logs`. Do not seed a private COA.
4. **Opening balances with guards** — per TK + per đối tượng (KH/NCC/vật tư/
   TSCĐ); bắt buộc cân đối Tổng Nợ = Tổng Có (and đồng tiền); dry-run import;
   lock after any period with postings; reopen only with reason + audit.
5. **Fiscal-year boundary rules** — kỳ kế toán năm = 12 tháng (Luật Kế toán
   Điều 12); regime switch only at FY start; closing/opening niên độ gated to
   `ke_toan_truong`.
6. **Authz + audit from day one** — initialize/regime/lock → `ke_toan_truong`/
   `admin`; edit balances → `ke_toan_tong_hop`; every mutation audited.
7. **Excel import as first-class** — import số dư đầu kỳ from an official
   template with dry-run + per-row error report (MISA/Fast/Bravo all ship it).
8. **Tests** — golden COA-seed counts, balance-check property tests, idempotent
   re-run, parallel initialize, lock/reopen, authz matrix, audit assertions.

## 6. Recommendation

1. **Treat setup as the next vertical after masterdata/ledger.** Cashier +
   ledger are PROD-pilot; both need a company identity, fiscal periods and
   opening balances to be demonstrable as a statutory whole.
2. **Ship P0 gate first:** status machine + statutory profile + authz + audit +
   migrations seam — mirroring the controls-first pattern used by cash.
3. **Orchestration before data entry:** implement `Initialize` as the
   masterdata/ledger/audit orchestrator (P1) before opening-balance editing UI
   (P2) and import (P3).
4. **Re-review after P3** — the earliest honest "can a new company go live"
   checkpoint. Full PROD pilot = cash + ledger + masterdata + setup on one
   TT 99/2025 fiscal year with opening balances.
