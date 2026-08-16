# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — Processes & Data Flows

## 1. Initialization state machine (điều khiển bởi Kế toán trưởng)

```
EMPTY ──► PROFILED ──► REGIME_SET ──► ACCOUNTS_SEEDED ──► PERIODS_OPEN
  │         │             │               │                  │
  │ save    │ set regime  │ seed COA      │ open 12 periods   │
  │ profile │ (masterdata)│ (masterdata)  │ (ledger)          ▼
  │ R1-R5   │ idempotent  │ + Quy chế     │               BALANCES_DRAFT
  │         │             │ audit note    │                  │
  │         │             │               │   check (ΣNợ=ΣCó)│
  │         │             │               │                  ▼
  │         │             │               │           BALANCES_LOCKED
  │         │             │               │                  │
  │         │             │               │   activate       ▼
  │         │             │               │              ACTIVE
  └─────────┴─────────────┴───────────────┘     (period 1 postable)
```

**Principle:** status is monotonic (R6); every step idempotent and restart-safe;
each transition audited; only `ke_toan_truong`/`admin` moves state forward;
`BALANCES_LOCKED → BALANCES_DRAFT` (reopen) is the only backward edge and needs
a reason (R12).

## 2. First-run wizard process (data flow)

```
POST /setup/initialize {profile, regime, fy, seed, open}
  ├─ resume from SetupStatus (row in setup_status)
  ├─ EMPTY        → validate profile (R1–R5) → save company_profiles → PROFILED
  ├─ PROFILED     → masterdata.SetRegime(regime)                   → REGIME_SET
  ├─ REGIME_SET   → masterdata.SeedAccounts() + Quy chế audit note → ACCOUNTS_SEEDED
  ├─ ACCOUNTS_SEEDED → ledger.OpenPeriod("YYYY-01".."YYYY-12")     → PERIODS_OPEN
  └─ PERIODS_OPEN → report "enter opening balances" → BALANCES_DRAFT (user-driven)
Each step: own tx + audit + status write. Crash-safe resume (UC-S11).
```

## 3. Opening balances → ledger flow

```
nhập/import số dư đầu kỳ (per TK + đối tượng, Nợ/Có)
  │ R7 one-side, R8 draft, R10 object-required for 131/331/152/211-214
  ▼
check: Σ Nợ == Σ Có (VND)  ──mismatch──► 422 {diff, offending TK}
  │ pass
  ▼
lock (ke_toan_truong) ──► BALANCES_LOCKED ──► activate ──► ACTIVE
                                                        │
                                                        ▼
                          ledger period 1 opens → postings (cash/vouchers/books)
```

## 4. Regime selection & switch flow

```
Chế độ kế toán: TT 99/2025 (default) | TT 133/2016 (SME)
  ├─ set at initialize (R5), recorded in profile + masterdata.SetRegime
  ├─ switch (UC-S9): only at FY boundary (Luật Kế toán Điều 12; TT 99 Điều 31)
  │   → dry-run COA diff → confirm → reseed diff + audit
  └─ COA always seeded through masterdata (single source of truth — no private
      chart in setup; ledger already reads accounts via its own layer)
```

## 5. Company identity → invoice flow (NĐ 254/2026)

```
CompanyProfile (setup) ──► invoice module (seller identity on HĐĐT)
  tên / địa chỉ theo ĐKDN    │ tên, địa chỉ, MST|mã ĐVQHNS bắt buộc
  MST | mã ĐVQHNS            │ (Điều 10 NĐ 254/2026)
  người đại diện             ▼
                    hóa đơn điện tử hợp lệ
```

## 6. Opening balances: đối tượng mapping (R10)

| TK | Object type | Notes |
|---|---|---|
| 131 | customer | Phải thu khách hàng — required detail |
| 331 | supplier | Phải trả người bán — required detail |
| 152/155/156 | item | Hàng tồn kho theo vật tư-hàng hóa |
| 211/214 | fixed_asset | TSCĐ (nguyên giá / hao mòn) |
| 111/112/… | — | Tiền/tài sản thuần — lump allowed |

## 7. CSV import process (số dư đầu kỳ)

```
Template v1 .csv ─► upload ─► dry-run (per-row R7/R8/R10 + balance-check)
   ─► job report: N ok, M errors (row + reason + field)
   ─► fix & re-upload ─► commit: upsert by "OB:{account}:{object}", batched tx
   ─► success report + audit
Errors never silently dropped; failing batch rolls back that batch only.
```

## 8. Reopen process (khóa → sửa lại)

```
ke_toan_truong ─► reopen {reason}
   ─► guard: no posted voucher references edited TK? → yes → BALANCES_DRAFT
   ─► no → 409 "có phát sinh" → override + reason → forced reopen + audit (R12)
   ─► edit balances (UC-S3) → re-check (UC-S4) → lock → (activate again)
```

## 9. User journeys

### 9.1 Kế toán trưởng khởi tạo công ty mới
1. Đăng nhập (dev seam `X-User-Id`) với vai trò `ke_toan_truong`.
2. Vào "Khởi tạo" → điền thông tin doanh nghiệp theo giấy chứng nhận ĐKDN.
3. Chọn chế độ kế toán (TT 99/2025 mặc định; TT 133/2016 nếu SME).
4. Xác nhận tải sơ đồ tài khoản + mở 12 kỳ kế toán (nút "Tạo dữ liệu ban đầu").
5. Nhập/import số dư đầu kỳ (nợ tài khoản NH, tài khoản 131/331, kho, TSCĐ).
6. Kiểm tra cân đối → khóa → kích hoạt. Mở quỹ tiền mặt, tạo phiếu thu/chi đầu
   tiên, xem sổ cái — mọi phân hệ hoạt động trên nền đã khởi tạo.

### 9.2 Kế toán tổng hợp nhập số dư đầu kỳ từ dữ liệu Excel cũ
1. Tải template CSV (cột: mã TK, đối tượng, TK đối tượng, Nợ, Có).
2. Điền từ bảng cân đối kế toán năm trước (mã TK cũ → mã TK mới theo TT 99).
3. Upload → dry-run: 2 dòng lỗi (sai định dạng MST khách hàng, thiếu mã kho).
4. Sửa → Commit → kiểm tra cân đối → báo cáo kế toán trưởng khóa.

### 9.3 Kế toán trưởng sửa số dư đầu kỳ sau khi đã khóa
1. Phát hiện sai số dư TK 131 → mở "Số dư đầu kỳ" → Reopen với lý do.
2. Không có phát sinh trên TK 131 → mở khóa → sửa → kiểm tra → khóa lại.
3. (Override nếu có phát sinh — audit ghi nhận lý do.)

### 9.4 Giám đốc xem trạng thái hệ thống
1. Mở "Trạng thái hệ thống": badge "TT 99/2025", các bước khởi tạo đã hoàn
   thành, số TK/khách hàng, Σ Nợ/Σ Có, nhật ký thay đổi.
2. Xuất báo cáo tóm tắt cho kiểm toán.

## 10. Supporting artifacts
- CSV template: `web/static/setup/templates/opening_balances_v1.csv`.
- Import error report: JSON → web table (row, field, VN message, EN message).
- Audit: existing `audit_logs` table (append-only).
- Status: `setup_status` row (single JSON doc).
- Cross-module: `masterdata.SeedAccounts/SetRegime`, `ledger.OpenPeriod`.
