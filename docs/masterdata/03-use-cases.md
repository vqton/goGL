# Master Data Module (Hệ thống danh mục dữ liệu chính) — Use Cases

> UC numbering: M = master data. Each UC lists happy path, alternative paths,
> exception paths. Roles per `01-brd §3`; rules R1–R12 per `02-spec §3`.

## UC-M1 — Create a customer (khách hàng) with statutory identity

**Actor:** Nhân viên danh mục (`danh_muc`). **Preconditions:** logged in,
authz OK; a group may exist; company regime set.

**Happy path (M1-H)**
1. Actor opens Danh mục → Khách hàng → Thêm.
2. System shows the KH form: Loại (Tổ chức/Cá nhân), Tên, MST, mã ĐVQHNS,
   số định danh cá nhân/CCCD, hộ chiếu + quốc tịch, Địa chỉ (theo giấy chứng
   nhận ĐKDN), Nhóm, Điều khoản TT (số ngày nợ, nợ tối đa), TK công nợ
   (default 131), Nhân viên phụ trách, Người nhận hóa đơn, Ngân hàng.
3. Actor enters data; leaves Code empty → auto-code `KH-00001` offered.
4. Actor saves. System validates R1–R7 (unique code, group exists, MST format,
   identity set per NĐ 254/2026), allocates sequence atomically (R12), inserts,
   appends audit. → 201 with record; list refreshes.

**Alternative A1:** Actor enters a manual code `CUS-ALPHA` → accepted if unique
(R1); auto-numbering skipped.
**Alternative A2:** Record is both KH and NCC (tick "Là nhà cung cấp") → system
links to supplier view with TK công nợ 331 default (R7 applies to both).
**Alternative A3:** Customer is a foreign individual → passport + nationality
required in lieu of MST/CCCD (NĐ 254/2026).

**Exceptions**
- **E1:** Code already exists → 409 "Mã KH-00001 đã tồn tại".
- **E2:** MST malformed (not 10-digit / not 13-with-dash) → 422 with field error
  (VN + EN).
- **E3:** No identity field and invoice consumption enabled → 422 "Cần khai
  MST/mã ĐVQHNS/CCCD/hộ chiếu để xuất hóa đơn (NĐ 254/2026)".
- **E4:** Group not found / group of different kind → 422.
- **E5:** Not authorized → 403 (fail closed).

## UC-M2 — Edit a customer

**Actor:** `danh_muc`/`ke_toan_tong_hop`. **Precondition:** record ACTIVE.

**Happy path (M2-H):** open record → edit Name, Address, Terms, Contact →
save → validated → audit row (người sửa, thời điểm) → 200.
**Alternative A1:** Edit MST/Code on a record with `ReferenceCount == 0` →
allowed.
**Alternative A2:** Edit Code with `ReferenceCount > 0` → blocked (R2); user
offered Merge instead.

**Exceptions:** **E1** 409 on referenced Code edit; **E2** 422 on invalid
values; **E3** 403 unauthorized; **E4** 404 unknown code.

## UC-M3 — Deactivate (ngừng sử dụng) a customer

**Actor:** `danh_muc`; override by `ke_toan_truong`.

**Happy path (M3-H):** select ACTIVE customer → Ngừng sử dụng → reason →
system checks `ReferenceCount` (công nợ, hóa đơn, sổ) == 0 → status INACTIVE →
audit → 200.
**Alternative A1 (override):** ReferenceCount > 0 → 409 → Kế toán trưởng
submits with reason → forced INACTIVE + audit note (R4).

**Exceptions:** **E1** 409 có phát sinh (no override); **E2** 403 role too low
for override; **E3** 404.

## UC-M4 — Merge duplicate customers

**Actor:** `ke_toan_truong`. **Precondition:** same kind, distinct codes.

**Happy path (M4-H):** select keep `CUS-001` + dupes `[CUS-009, CUS-010]` →
"Kiểm tra ảnh hưởng" → dry-run shows impact per record (công nợ n refs, hóa
đơn m refs) → Xác nhận → one tx re-points all references to `CUS-001`, sets
merged INACTIVE with reason "Đã gộp vào CUS-001", recomputes ReferenceCount,
audits → 200.
**Alternative A1:** Dry-run only (no commit).

**Exceptions:** **E1** 422 different kinds in the set; **E2** 409 a dupe already
has a pending audit-locked state; **E3** 403; **E4** 500 mid-tx → full rollback
(tx), no partial re-pointing.

## UC-M5 — Import customers from Excel

**Actor:** `danh_muc`. **Precondition:** template v2 file, ≤ max rows.

**Happy path (M5-H):** upload → dry-run → job report: X rows OK, Y errors with
row number + reason → Fix → upload corrected → Commit → upsert by `(kind, code)`
idempotent → success report → audit.
**Alternative A1:** New rows only (skip existing) vs Update existing (upsert)
flag.

**Exceptions:** **E1** old template version → 422 "Dùng template mới nhất";
**E2** malformed file → 422; **E3** row-level errors isolated (never partial
silent commit — per-row OKs commit, errors reported); **E4** 403.

## UC-M6 — Seed chart of accounts (TT 99/2025)

**Actor:** `ke_toan_truong`. **Precondition:** accounts catalog empty or regime
switch approved.

**Happy path (M6-H):** Danh mục → Sơ đồ tài khoản → "Lấy mẫu Phụ lục 2
TT 99/2025" → preview tree (hierarchy, type, AllowPost) → Áp dụng → tx seeds
idempotently + writes Quy chế hạch toán note (Điều 11 TT 99/2025) → audit →
200.
**Alternative A1:** Choose TT 133/2016 variant (SME).
**Alternative A2:** Preview-only without applying.

**Exceptions:** **E1** 409 accounts already seeded (idempotent re-run allowed);
**E2** 422 regime inconsistent with ledger period; **E3** 403 role.

## UC-M7 — Add a chart-of-account amendment (Điều 11)

**Actor:** `ke_toan_truong`. **Precondition:** seed applied.

**Happy path (M7-H):** add `11113` under `1111` (AllowPost) → validation R8
(parent exists, Level ok, type) → saves with mandatory Quy chế hạch toán note
(text) → audit → 200.
**Exceptions:** **E1** 422 parent not postable/not found; **E2** 422 no Quy chế
note; **E3** 409 code exists; **E4** 403.

## UC-M8 — Create item (vật tư-hàng hóa)

**Actor:** `danh_muc`.

**Happy path (M8-H):** form (Loại 21/31/41/51/61, Tên, ĐVT, thuế suất, TK kho/
giá vốn/doanh thu, phương pháp giá, tồn tối thiểu) → auto-code `VT-00001` →
validate → 201.
**Exceptions:** **E1** 409 dup code; **E2** 422 unit/account/tax-rate not found
or of wrong kind; **E3** 422 cost method invalid; **E4** 403.

## UC-M9 — Manage units / warehouses / departments (simple catalogs)

**Actor:** `danh_muc`.
**Happy path (M9-H):** add `Cái`, `Kg`, `Tấn`, `Bộ`, `Lít`, `m2`, `Hộp`, `Giờ`
or a kho → validated group/dup → 201.
**Exceptions:** **E1** 409 dup; **E2** 422 group cycle (R9); **E3** 403.

## UC-M10 — Set tax rates (versioned)

**Actor:** `ke_toan_tong_hop`.
**Happy path (M10-H):** add GTGT 8% valid 01/07/2025 → 31/12/2025 → 201.
**Alternative A1:** Edit rate of an active tax code by starting a new version
(overlap check R10).
**Exceptions:** **E1** 422 overlapping validity; **E2** 409 dup code; **E3** 403.

## UC-M11 — Search & view any catalog

**Actor:** any read role (incl. `giam_doc`).
**Happy path (M11-H):** search by code/name/MST/group, filter status/group →
paginated list → detail.
**Alternative A1:** Export current filter to CSV/Excel.
**Exceptions:** **E1** 404 kind; **E2** 403.

## UC-M12 — Hard-delete an unreferenced record

**Actor:** `danh_muc`.
**Happy path (M12-H):** record with ReferenceCount == 0 → Delete → confirm →
physical delete + audit → 200.
**Exceptions:** **E1** 409 có phát sinh (use deactivate instead — R3);
**E2** 403; **E3** 404.

## UC-M13 — Switch accounting regime (TT 99 ↔ TT 133)

**Actor:** `ke_toan_truong`.
**Happy path (M13-H):** override/Regime → select variant → dry-run diff of
chart of accounts → confirm → md_regimes history + audit → 200.
**Exceptions:** **E1** 409 ledger has posted entries in the affected period
(regime change must start at FY boundary — Luật Kế toán + TT 133 Điều 3);
**E2** 403.

## NFR / cross-cutting use cases

- **UC-M14 Audit trail:** every mutation (create/edit/deactivate/merge/import/
  seed/regime) appends to `audit_logs` with user + timestamp + reason. Verified
  by golden test.
- **UC-M15 Concurrency:** two users create `KH-00001` simultaneously → one
  201, one 409 (atomic sequence R12). Verified by parallel-insert test.
- **UC-M16 Reference guard correctness:** after cash/invoice/payroll writes,
  `ReferenceCount` matches a re-scan of consumer tables. Verified by property
  test.
