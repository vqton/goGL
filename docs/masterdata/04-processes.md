# Master Data Module (Hệ thống danh mục dữ liệu chính) — Processes & Data Flows

## 1. Master data governance process (chủ trì: Kế toán trưởng)

```
Đề xuất dữ liệu mới ──► Nhân viên danh mục nhập (khai báo)
     │                         │ R1–R10 validation
     │                         ▼
     │                    ACTIVE (đang sử dụng)
     │                         │
     ├── Có phát sinh? ────────┘      Không phát sinh
     │   │  Không                     │
     │   ▼                            ▼
     │  Ngừng sử dụng           Xóa hẳn (R3) + audit
     │   │ (reason, audit)            │
     │   ▼                            │
     │  INACTIVE (giữ lịch sử)  <─────┘
     │
     ├── Trùng lặp? ──► Gộp (ke_toan_truong, dry-run impact) ──► audit
     └── Nhập khối lượng ──► Import Excel (dry-run → sửa lỗi → commit)
```

**Principle:** no hard delete after any reference; no silent changes; every
state transition audited; Kế toán trưởng owns merges, overrides and regime.

## 2. End-to-end customer → invoice data flow (NĐ 254/2026)

```
Khách hàng masterdata ──► invoice module ──► HĐĐT (NĐ 254/2026)
  tên/địa chỉ theo ĐKDN │   buyer identity   │ nội dung bắt buộc:
  MST | mã ĐVQHNS |     │   from registry    │ tên, địa chỉ, MST/mã
  CCCD | hộ chiếu+QT    └───────────────────┘ ĐVQHNS/số định danh CN
        ▲                                          │
        └──── invoice validates identity set ──────┘
                (missing → 422, cannot issue)
```
Registry is the **only** place buyer identity lives; invoice reads it via
`Lookup(kind=customer, code)` (Go interface, not HTTP). A customer cannot be
deactivated while unpaid invoices/công nợ exist (UC-M3/UC-M16).

## 3. Chart-of-accounts → ledger flow (TT 99/2025)

```
Phụ lục 2 seed (ke_toan_truong) ──► ledger module
  hierarchy / type / AllowPost        │ account must exist, ACTIVE,
  Quy chế hạch toán note ────────────►│ AllowPost to post (ledger R3)
                                      ▼
                          Sổ cái / Sổ chi tiết / BCĐPS
```
Amendments (Điều 11 TT 99/2025) write the Quy chế note into audit; ledger
rejects lines on non-postable parents, exactly as ledger R3 today.

## 4. Item → inventory / sales / purchase flow

```
Vật tư-hàng hóa (TK kho 152/156, giá vốn 632, doanh thu 511,
thuế GTGT, phương pháp giá AVG/FIFO/đích danh, ĐVT, tồn tối thiểu)
        │
        ├──► inventory module: nhập/xuất, tính giá tồn theo phương pháp
        ├──► purchase module:  TK kho mặc định trên chứng từ mua
        └──► sales module:     TK doanh thu + thuế suất trên hóa đơn bán
```

## 5. Master data lifecycle state machine

```
        create (R1,R5)
     ┌───────┴────────┐
     │               │
  ACTIVE ◄────────┐  │
     │  activate  │  │
     │ (R4=0 only)│  │
     ▼            │  │
  INACTIVE ───────┘  │
     ▲  ngừng sử dụng (R4: refs==0, or chief override)
     │
     └── hard delete (R3: refs==0) ──► GONE + audit
```
Versioned records (tax rate, tỷ giá) additionally follow
`ValidFrom..ValidTo` (R10) and never overlap.

## 6. Merge process (gộp đối tượng trùng)

```
Chọn keep + dupes ─► dry-run impact (refs per record)
   ─► commit in ONE tx:
       re-point refs in all consumer tables (by code/id map)
       → dupes INACTIVE + reason "Đã gộp vào {keep}"
       → recompute keep.ReferenceCount
       → audit (người gộp, thời điểm, danh sách gộp)
   Rollback fully on any error (no partial re-pointing).
```

## 7. Import process (Excel)

```
Upload template-v2 ─► parse/validate per row (R1–R10)
   ─► dry-run job: N ok, M errors (row + reason) ─► fix & re-upload
   ─► commit job: upsert by (kind, code), idempotent, batched tx
   ─► report + audit
```
Errors never silently drop rows; a failing batch rolls back that batch only.

## 8. Regime switch process

```
Kế toán trưởng ─► chọn TT 99/2025 | TT 133/2016
   ─► guard: FY boundary only (Luật Kế toán; TT 133 Điều 3)
   ─► dry-run chart diff → confirm → md_regimes history + audit
```

## 9. User journeys

### 9.1 Kế toán viên mới mở công ty (setup)
1. Đăng nhập với vai trò `ke_toan_truong`.
2. Cấu hình mã tự động (KH-, NCC-, VT-, …) và chế độ kế toán (TT 99 mặc định).
3. Seed sơ đồ tài khoản Phụ lục 2 (UC-M6).
4. Thêm đơn vị tính, kho, phòng ban, nhân viên, ngân hàng (UC-M9).
5. Nhập danh sách khách hàng/nhà cung cấp từ Excel (UC-M5).
6. Bắt đầu nghiệp vụ: mọi phân hệ đọc danh mục từ registry.

### 9.2 Nhân viên bán hàng lập hóa đơn cho khách mới
1. Mở hóa đơn bán → gõ mã khách → không có → nhảy nhanh tạo KH (danh mục).
2. Nhập tên + MST; hệ thống kiểm tra NĐ 254 identity (R7).
3. Lưu → quay lại hóa đơn với đối tượng đã chọn → xuất HĐĐT.

### 9.3 Kế toán trưởng xử lý khách hàng có công nợ cần ngừng theo dõi
1. Ngừng sử dụng → 409 "có công nợ" → xem báo cáo công nợ.
2. Xác nhận thu nợ hết → ngừng sử dụng thành công.
3. Hoặc chọn override + lý do (kiểm soát đặc biệt, audit).

### 9.4 Quyết toán tháng — kiểm tra tính nhất quán
1. Mở "Báo cáo chất lượng danh mục": KH không có MST, TK chưa khai báo, mã trùng.
2. Sửa/gộp từ báo cáo (UC-M4, UC-M2).
3. Xuất CSV gửi kế toán trưởng ký xác nhận.

## 10. Supporting artifacts
- Excel templates: `web/static/md/templates/{kind}.xlsx` (versioned, `v2`).
- Import error report: JSON → web table (row, field, VN message, EN message).
- Audit: existing `audit_logs` table (append-only).
- Regime + Quy chế hạch toán: `md_regimes` table + audit rows.
