# Fixed Assets Module — Complete Specification

**Version**: 1.0
**Date**: 2026-08-29
**Module**: Fixed Assets (Tài sản cố định — TSCĐ)
**Compliance**: Circular 45/2013/TT-BTC, Circular 99/2025/TT-BTC, VAS 03, Luật Kế toán 88/2015

---

## 1. Business Rules (Regulatory)

### BR-01: Asset Recognition Threshold
- Original cost ≥ 30,000,000 VND (Circular 45 Art 2)
- Items below threshold → "Dụng cụ" (tools/supplies), Account 153/156

### BR-02: Asset Classification (Circular 45 Art 3)
| Code | Type (Vietnamese) | Type (English) |
|------|-------------------|----------------|
| housing | Nhà cửa, vật kiến trúc | Buildings & structures |
| machinery | Máy móc, thiết bị | Machinery & equipment |
| transport | Phương tiện vận tải | Transport vehicles |
| tools | Dụng cụ, thiết bị | Tools & instruments |
| perennial | Vườn cây lâu năm | Perennial gardens |
| other | Tài sản cố định khác | Other fixed assets |

### BR-03: Depreciation Methods (Circular 45 Art 11-13)
1. **Straight-line**: Monthly = (Original Cost - Residual Value) / Useful Life (months)
2. **Declining-balance**: Book Value × (2 / Useful Life years)
3. **Units-of-output**: (Cost - Residual) × (Actual Output / Total Output)

### BR-04: Useful Life Limits (Circular 45 Annex I)
| Asset Type | Min (months) | Max (months) |
|------------|-------------|-------------|
| Buildings & structures | 60 | 600 |
| Machinery & equipment | 36 | 240 |
| Transport vehicles | 72 | 360 |
| Tools & instruments | 36 | 120 |
| Perennial gardens | 48 | 480 |
| Other | 48 | 300 |

### BR-05: Depreciation Start/Stop (Circular 45 Art 8-9)
- **Start**: From 1st or 15th of month asset increases
- **Stop**: When asset decreases (liquidation, transfer, disposal)
- **No depreciation**: Assets under construction, pending liquidation, unused >9 months

### BR-06: Accelerated Depreciation (Circular 45 Art 10)
- Rate ≤ 2× straight-line rate
- Only: machinery, measuring instruments, transport, management tools, perennial gardens
- Enterprise must be profitable
- Excess over 2×: not tax-deductible

### BR-07: GL Account Mapping (Circular 99/2025)
| Account | Name | Purpose |
|---------|------|---------|
| 211 | TSCĐ hữu hình | Original cost |
| 2141 | Hao mòn TSCĐ hữu hình | Accumulated depreciation |
| 623 | Hao mòn TSCĐ — Sản xuất | Depreciation (production) |
| 627 | Hao mòn TSCĐ — Thiết bị | Depreciation (equipment) |
| 641 | Hao mòn TSCĐ — Quản lý | Depreciation (management) |
| 642 | Hao mòn TSCĐ — Thương mại | Depreciation (commerce) |

### BR-08: Liquidation Gain/Loss (VAS 03 Para 37-41)
- Gain/Loss = Selling Price - (Original Cost - Accumulated Depreciation) - Liquidation Costs
- Gain → Account 515 (Other revenue)
- Loss → Account 632 (Other expenses)

---

## 2. Data Model

### 2.1 FixedAsset (Existing — Enhanced)

```go
type FixedAsset struct {
    // Core
    ID, Code, Name, Description string
    
    // Classification (Circular 45)
    AssetType   AssetType // housing, machinery, transport, tools, perennial, other
    Category    string    // Sub-category
    
    // Financial (VND)
    OriginalCost    int64
    ResidualValue   int64
    AccumulatedDepr int64
    
    // Depreciation
    DepreciationMethod DepreciationMethod // straight_line, declining_balance, units_of_output
    UsefulLifeMonths   int
    
    // Dates
    PurchaseDate   string
    InServiceDate  string
    LastReviewDate string
    
    // Location
    Location   string
    Department string
    
    // Source
    VendorName, PurchaseOrderNo, InvoiceNo, SerialNo string
    
    // Accounting
    AccountCode211 string // Account for original cost
    AccountCode214 string // Account for accumulated depreciation
    
    // State
    State AssetState // active, inactive, scrapped, sold, pending_liquidation
    
    // Audit
    CreatedBy, CreatedAt, UpdatedBy, UpdatedAt string
}
```

### 2.2 DepreciationEntry (NEW)

```go
type DepreciationEntry struct {
    ID                string            `json:"id"`
    AssetID           string            `json:"asset_id"`
    AssetCode         string            `json:"asset_code"`
    Period            string            `json:"period"` // YYYY-MM
    DepreciationMethod DepreciationMethod `json:"depreciation_method"`
    Amount            int64             `json:"amount"`
    AccumulatedDepr   int64             `json:"accumulated_depr"`
    BookValue         int64             `json:"book_value"`
    AccountDebit      string            `json:"account_debit"`  // 623/627/641/642
    AccountCredit     string            `json:"account_credit"` // 2141
    Posted            bool              `json:"posted"`
    PostedAt          string            `json:"posted_at,omitempty"`
    CreatedBy         string            `json:"created_by"`
    CreatedAt         string            `json:"created_at"`
}
```

### 2.3 AssetTransfer (NEW)

```go
type AssetTransfer struct {
    ID            string `json:"id"`
    AssetID       string `json:"asset_id"`
    TransferDate  string `json:"transfer_date"`
    FromLocation  string `json:"from_location"`
    ToLocation    string `json:"to_location"`
    FromDepartment string `json:"from_department"`
    ToDepartment  string `json:"to_department"`
    Reason        string `json:"reason"`
    ApprovedBy    string `json:"approved_by,omitempty"`
    CreatedBy     string `json:"created_by"`
    CreatedAt     string `json:"created_at"`
}
```

### 2.4 AssetLiquidation (NEW)

```go
type AssetLiquidation struct {
    ID               string `json:"id"`
    AssetID          string `json:"asset_id"`
    RequestDate      string `json:"request_date"`
    Reason           string `json:"reason"`
    SellingPrice     int64  `json:"selling_price"`
    LiquidationCosts int64  `json:"liquidation_costs"`
    GainLoss         int64  `json:"gain_loss"`
    Status           string `json:"status"` // requested, approved, executed, rejected
    RequestedBy      string `json:"requested_by"`
    ApprovedBy       string `json:"approved_by,omitempty"`
    ExecutedAt       string `json:"executed_at,omitempty"`
    CreatedAt        string `json:"created_at"`
}
```

---

## 3. API Endpoints (Target)

```
POST   /api/v1/fixed-assets                        - Create asset
GET    /api/v1/fixed-assets                        - List assets
GET    /api/v1/fixed-assets/:id                    - Get asset
PUT    /api/v1/fixed-assets/:id                    - Update asset
DELETE /api/v1/fixed-assets/:id                    - Delete asset

POST   /api/v1/fixed-assets/:id/transfer           - Transfer asset
POST   /api/v1/fixed-assets/:id/liquidate          - Request liquidation
POST   /api/v1/fixed-assets/:id/liquidate/approve  - Approve liquidation
POST   /api/v1/fixed-assets/:id/liquidate/execute  - Execute liquidation

POST   /api/v1/fixed-assets/depreciation/calculate - Calculate monthly depreciation
POST   /api/v1/fixed-assets/depreciation/post      - Post depreciation to GL

GET    /api/v1/fixed-assets/:id/depreciation-schedule - Get schedule

GET    /api/v1/reports/asset-register              - Asset register
GET    /api/v1/reports/depreciation                - Depreciation report
GET    /api/v1/reports/depreciation-schedule/:id   - Individual schedule
```

---

## 4. Process Flows

### 4.1 Monthly Depreciation Cycle

```
[Start of Month]
    │
    ▼
[1. Close Previous Month in Ledger]
    │
    ▼
[2. Accountant: "Calculate Depreciation" for YYYY-MM]
    │
    ▼
[3. System: Select All Active Assets]
    │
    ├──► [For Each Asset]
    │        │
    │        ▼
    │    [Check: In-Service Date ≤ Period End?]
    │        │
    │        ├── NO → Skip
    │        │
    │        └── YES → [Check: Unused > 9 months?]
    │                      │
    │                      ├── YES → Skip (log warning)
    │                      │
    │                      └── NO → [Calculate Depreciation]
    │                                  │
    │                                  ├── Straight-Line
    │                                  ├── Declining-Balance
    │                                  └── Units-of-Output
    │                                  │
    │                                  ▼
    │                              [Create Draft Entry]
    │
    ▼
[4. System: Display Depreciation Summary]
    │
    ├──► [Accountant: APPROVE]
    │        │
    │        ▼
    │    [5. Post to GL]
    │        │
    │        ├── Debit: 623/627/641/642 (by department)
    │        └── Credit: 2141
    │        │
    │        ▼
    │    [6. Update Accumulated Depreciation on each asset]
    │        │
    │        ▼
    │    [7. Mark Entries as Posted]
    │
    └──► [Accountant: REJECT]
             │
             ▼
         [Clear Draft Entries]
```

### 4.2 Asset Lifecycle State Machine

```
[Asset Created]
    │
    ▼
State: ACTIVE ◄──────────────────────────────┐
    │                                         │
    ├──► [Monthly Depreciation] ──► CONTINUE  │
    │                                         │
    ├──► [Transfer] ──► [Update Location] ──► │
    │                                         │
    └──► [Request Liquidation]                │
              │                               │
              ▼                               │
         State: PENDING_LIQUIDATION           │
              │                               │
              ├──► [APPROVE] ──► [Execute]    │
              │        │                      │
              │        ▼                      │
              │    State: SCRAPPED / SOLD     │
              │                               │
              └──► [REJECT] ──────────────────┘
```

### 4.3 Data Flow

```
[Asset Acquisition]
    │
    ▼
[Create Asset Card]
    │
    ├──► [Account 211 — Debit: Original Cost]
    │
    ▼
[Monthly Depreciation]
    │
    ├──► [Account 623/627/641/642 — Debit]
    ├──► [Account 2141 — Credit]
    │
    ▼
[Asset Liquidation]
    │
    ├──► [Account 2141 — Debit: Reverse Accumulated Depreciation]
    ├──► [Account 211 — Credit: Reverse Original Cost]
    ├──► [Account 515/632 — Gain/Loss]
```

---

## 5. UI/UX Wireframes

### 5.1 Asset List View
```
+----------------------------------------------------------------------+
| Tài sản cố định (Fixed Assets)                        [+ Thêm mới] |
+----------------------------------------------------------------------+
| Tìm kiếm: [________________] Phân loại: [Tất cả ▼] Trạng thái: [▼] |
+----------------------------------------------------------------------+
| Mã     | Tên           | Giá gốc   | Hao mòn  | Giá trị | Trạng thái |
|--------|---------------|------------|----------|---------|------------|
| FA-001 | Máy ép plastic| 500,000,000| 200,000K | 300,000 | Hoạt động |
| FA-002 | Xe tải 1.5T   | 800,000,000| 320,000K | 480,000 | Hoạt động |
| FA-003 | Laptop Dell    | 25,000,000 | 25,000K  | 0       | Đã thanh lý|
+----------------------------------------------------------------------+
| < 1 2 3 ... >                                                      |
+----------------------------------------------------------------------+
```

### 5.2 Asset Detail View
```
+----------------------------------------------------------------------+
| FA-001 — Máy ép plastic                    [Sửa] [Xóa] [Thanh lý] |
+----------------------------------------------------------------------+
| Thông tin cơ bản          | Khấu hao                                |
|---------------------------|-----------------------------------------|
| Phân loại: Máy móc, TB   | Phương pháp: Đường thẳng               |
| Địa điểm: Nhà máy A     | Thời gian: 10 năm (120 tháng)          |
| Bộ phận: Sản xuất        | Tỷ lệ: 0.83%/tháng                     |
| Trạng thái: Hoạt động    | Hao mòn tháng: 4,166,667 VND           |
|                           |                                        |
| Thông tin tài chính       |                                        |
|---------------------------|-----------------------------------------|
| Giá gốc:     500,000,000 | Ngày mua: 15/03/2024                   |
| Hao mòn lũy: 200,000,000 | Ngày sử dụng: 01/04/2024               |
| Giá trị còn:  300,000,000 | Nhà cung cấp: CTY TNHH ABC            |
|                           | Số PO: PO-2024-001                     |
+----------------------------------------------------------------------+
| [ChuyểnLocation] [Tính khấu hao] [Xem lịch] [Xuất Excel]          |
+----------------------------------------------------------------------+
```

### 5.3 Depreciation Schedule View
```
+----------------------------------------------------------------------+
| Lịch khấu hao — FA-001 Máy ép plastic             [Xuất Excel]    |
+----------------------------------------------------------------------+
| Năm 2026                                                             |
+----------------------------------------------------------------------+
| Tháng | Hao mòn     | Hao mòn lũy | Giá trị còn | Ghi chú          |
|-------|-------------|-------------|-------------|------------------|
| 01/26 | 4,166,667   | 175,000,000 | 325,000,000 |                  |
| 02/26 | 4,166,667   | 179,166,667 | 320,833,333 |                  |
| 03/26 | 4,166,667   | 183,333,333 | 316,666,667 |                  |
| ...   | ...         | ...         | ...         |                  |
| 12/26 | 4,166,667   | 225,000,000 | 275,000,000 |                  |
+----------------------------------------------------------------------+
| Tổng năm: 50,000,000 VND                                             |
+----------------------------------------------------------------------+
```

### 5.4 Monthly Depreciation Run
```
+----------------------------------------------------------------------+
| Chạy khấu hao tháng — Tháng 08/2026                                |
+----------------------------------------------------------------------+
| Tổng tài sản: 45 | Đang khấu hao: 42 | Bỏ qua: 3                   |
+----------------------------------------------------------------------+
| Mã     | Tên           | Phương pháp | Số tiền   | TK Nợ | TK Có    |
|--------|---------------|-------------|-----------|-------|----------|
| FA-001 | Máy ép plastic| Đường thẳng| 4,166,667 | 627   | 2141     |
| FA-002 | Xe tải 1.5T   | Đường thẳng| 4,444,444 | 641   | 2141     |
| FA-004 | Máy CNC       | Số dư giảm | 8,333,333 | 627   | 2141     |
+----------------------------------------------------------------------+
| Tổng hao mòn tháng: 500,000,000 VND                                 |
+----------------------------------------------------------------------+
| [Duyệt & Ghi sổ]  [Từ chối]                                        |
+----------------------------------------------------------------------+
```

### 5.5 Liquidation Form
```
+----------------------------------------------------------------------+
| Đề nghị thanh lý tài sản — FA-003 Laptop Dell                       |
+----------------------------------------------------------------------+
| Giá gốc:         25,000,000 VND                                     |
| Hao mòn lũy:     25,000,000 VND                                     |
| Giá trị còn lại:  0 VND                                              |
+----------------------------------------------------------------------+
| Lý do thanh lý: [Hỏng, không thể sửa chữa                    ]     |
| Giá bán:         [0               ] VND                              |
| Chi phí thanh lý: [500,000        ] VND                              |
+----------------------------------------------------------------------+
| Kết quả: Lỗ thanh lý: -500,000 VND (Ghi nhận vào TK 632)           |
+----------------------------------------------------------------------+
| [Gửi đề nghị]                                                       |
+----------------------------------------------------------------------+
```

---

## 6. Implementation Roadmap

### Phase 1: Depreciation Schedule Repository (Week 1)
- [ ] Create `depreciation_entries` + `asset_transfers` + `asset_liquidations` tables
- [ ] Implement `DepreciationEntry` entity + repository
- [ ] Unit tests

### Phase 2: Monthly Batch Processor (Week 2)
- [ ] Implement `RunMonthlyDepreciation(period)` service method
- [ ] Iterate active assets, calculate, store draft entries
- [ ] Unit tests with known values

### Phase 3: GL Integration (Week 2-3)
- [ ] Implement `PostDepreciation(period, actor)` — post to GL
- [ ] Update accumulated_depr on assets
- [ ] Integration tests with ledger module

### Phase 4: Reports (Week 3-4)
- [ ] Asset Register Report
- [ ] Depreciation Schedule Report
- [ ] Depreciation Expense Report

### Phase 5: Web UI (Week 4-5)
- [ ] Asset List, Detail, Create pages
- [ ] Depreciation Schedule view
- [ ] Monthly Depreciation Run page
- [ ] Transfer and Liquidation forms

### Phase 6: Approval & Audit (Week 5-6)
- [ ] Liquidation approval workflow
- [ ] Casbin authorization
- [ ] Audit trail

### Phase 7: Export & Polish (Week 6-7)
- [ ] Excel/PDF export
- [ ] Barcode/QR support
- [ ] Performance optimization

---

*Specification complete. Ready for implementation.*
