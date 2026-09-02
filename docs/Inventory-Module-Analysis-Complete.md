# Inventory Module — Complete Analysis, Requirements & Implementation Roadmap
## As of: 2026-09-01 | BA Lead + Chief Accountant Perspective

---

## Table of Contents
1. Executive Summary
2. PROD-Readiness Assessment
3. Vietnamese Accounting Requirements (VAS 02)
4. Business Rules & Compliance Requirements
5. System Entities (Domain Model)
6. Use Cases (Happy / Alternative / Exception Paths)
7. Processes & Workflows
8. GL Integration (Thông tư 99/2025/TT-BTC)
9. Data Flows
10. User Journeys
11. UI/UX Wireframes
12. Technical Specification
13. Implementation Roadmap
14. Execution Plan
15. Open Questions & Risks

---

## 1. Executive Summary

**What**: The Inventory module is currently a **23-line STUB** — not production-ready. It provides only a basic `StockMovement` entity with 6 fields, a 3-method repository stub, and 2-endpoint handler stub. It cannot handle any Vietnamese accounting requirements, warehouse management, stock valuation, physical count, or GL posting.

**Why it matters**: 
- Inventory is the **largest asset category** on most Vietnamese company balance sheets (typically 30-60% of current assets)
- Incorrect inventory accounting leads to material misstatement of financial statements
- Physical count requirements are mandatory (Circular 99/2025/TT-BTC, VAS 02)
- Without proper stock valuation, COGS and gross profit are misstated
- E-invoicing for inventory transactions is mandatory since July 1, 2022

**Compliance framework**:
- VAS 02 (Inventories) — valuation methods, NRV write-down, disclosure
- Thông tư 99/2025/TT-BTC (replaces Circular 200) — LIFO prohibited, periodic physical count
- Decree 123/2020/NĐ-CP / Decree 70/2025/NĐ-CP — e-invoice requirements
- Nghị định 50/2016/NĐ-CP — warehouse bookkeeping

**Recommended approach**: Build inventory as a **separate module** with entities: Item, Warehouse, StockCard, StockMovement, StockAdjustment, PhysicalCount. NOT just extending the current StockMovement stub.

---

## 2. PROD-Readiness Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Entity completeness | ❌ 6 fields only | Missing: Item (name, category, unit, GL accounts, NRV), Warehouse (name, address, responsible person), StockCard (running balance), StockAdjustment, PhysicalCount, PhysicalCountLine |
| Valuation methods | ❌ None | VAS 02 requires FIFO, Weighted Average; LIFO prohibited |
| GL integration | ❌ None | No posting to 152/153/156/632 accounts |
| Stock balance tracking | ❌ None | No running balance per item per warehouse |
| Physical count | ❌ None | Annual count mandatory per Circular 99/2025/TT-BTC |
| NRV write-down | ❌ None | VAS 02 requires year-end NRV vs original price comparison |
| E-invoice integration | ❌ None | Mandatory since July 1, 2022 |
| Audit trail | ❌ None | No stock movement log |
| Purchase integration | ❌ None | Goods receipt should create stock + GL posting |
| Sales integration | ❌ None | Sales dispatch should reduce stock + GL posting |

**Verdict: NOT production-ready.** Requires complete rewrite with proper domain model.

---

## 3. Vietnamese Accounting Requirements (VAS 02)

### 3.1 What is "Inventory" under VAS 02?

VAS 02 defines inventory as assets held for:
- **(a)** Sale in the normal production and business period
- **(b)** In the on-going process of production and business
- **(c)** Raw materials, materials, tools and instruments for use in production or service provision

Inventory types:
1. **Goods for sale** (Hàng hóa): goods in stock, purchased goods in transit, goods sent for sale, goods sent for processing
2. **Finished products** (Thành phẩm): products completed and in stock, products sent for sale
3. **Unfinished/WIP products** (Bán thành phẩm): partially completed products
4. **Raw materials & supplies** (Nguyên vật liệu): raw materials in stock, materials in stock, tools/instruments in stock, items in transit
5. **Costs of unfinished services** (Chi phí dịch vụ dở dang)

### 3.2 Valuation Methods (VAS 02, Paragraph 13-16)

| Method | VAS 02 Allowed | Circular 99/2025/TT-BTC | Notes |
|--------|---------------|------------------------|-------|
| **FIFO** (First-In, First-Out) | ✅ Allowed | ✅ Encouraged | Assumes oldest items sold first |
| **Weighted Average** (Bình quân gia quyền) | ✅ Allowed | ✅ Allowed | Average cost of all items |
| **LIFO** (Last-In, First-Out) | ✅ Explicitly described but **prohibited** from 2026 | ❌ Prohibited | Circular 99/2025/TT-BTC formally bans |
| **Specific Identification** | ✅ Allowed (for distinguishable items) | ✅ Allowed | For unique, high-value items |

**Our choice**: FIFO (default) + Weighted Average (option). LIFO **must not** be implemented.

### 3.3 Original Price Components (VAS 02, Paragraph 05-12)

Original price includes:
- **Purchasing cost**: buying price + non-refundable taxes + transport + handling + insurance
- **Processing cost**: direct materials + direct labor + variable/allocated production overhead
- **Other directly-related costs**: design costs for specific orders, etc.

Original price does NOT include:
- Abnormally high costs (waste above normal levels)
- Storage costs (except necessary for further production)
- Selling costs
- Enterprise management costs

### 3.4 Net Realizable Value (VAS 02, Paragraph 18-23)

- Compare original price vs. NRV at each year-end
- NRV = estimated selling price - estimated completion costs - estimated selling costs
- If NRV < original price → **write down** inventory and recognize loss in P&L
- If NRV recovers in subsequent year → reverse write-down (but only to original price)
- Raw materials: don't write down if finished product will be sold above production cost

### 3.5 Recognition of Cost of Goods Sold (VAS 02, Paragraph 24-26)

- When selling: recognize original price of goods sold as P&L expense **in the same period** as related revenue
- If inventory used for fixed assets → transfer to fixed asset value
- If production costs > NRV → recognize loss immediately

### 3.6 Disclosure Requirements (VAS 02, Paragraph 27-30)

Financial statements must disclose:
- (a) Accounting policies for inventory valuation (method used)
- (b) Original prices of total and each kind of inventory, classified suitably
- (c) Value of inventory price decrease reserve
- (d) Value re-included from reserve
- (e) Cases/events resulting in addition to or re-inclusion from reserve
- (f) Book value of inventory mortgaged/pledged for payable debts

If using LIFO: must disclose difference between LIFO value vs FIFO and vs weighted average (though LIFO is now prohibited).

---

## 4. Business Rules & Compliance Requirements

### BR-01: Stock Valuation Method
- **Rule**: System must support FIFO (default) and Weighted Average methods
- **Compliance**: VAS 02, Paragraph 13-16; Circular 99/2025/TT-BTC
- **Scope**: Applied per item (item-level configuration)

### BR-02: LIFO Prohibition
- **Rule**: LIFO method must NOT be available in the system
- **Compliance**: VAS 02 + Circular 99/2025/TT-BTC (effective Jan 1, 2026)
- **Scope**: Global system constraint

### BR-03: Year-End NRV Write-Down
- **Rule**: At each year-end, system must compare original price vs. NRV per item type and generate write-down journal if NRV < original price
- **Compliance**: VAS 02, Paragraph 18-23
- **GL**: Dr. 632xxx (COGS) / Cr. 152xxx (Inventory) or Dr. 515xxx (Inventory write-down) / Cr. 152xxx

### BR-04: Physical Count
- **Rule**: At least one physical count per year; system must support ad-hoc counts
- **Compliance**: Circular 99/2025/TT-BTC, Article 30; Nghị định 50/2016/NĐ-CP
- **Process**: System count → compare with book → generate adjustment journal

### BR-05: Stock Adjustment Authorization
- **Rule**: Stock adjustments exceeding threshold require management approval
- **Scope**: Configurable threshold per company

### BR-06: Movement Documentation
- **Rule**: Every stock movement must have a reference document (PO, invoice, receipt, transfer, etc.)
- **Compliance**: Nghị định 50/2016/NĐ-CP, Article 7
- **Format**: System enforces reference document ID

### BR-07: GL Posting on Receipt
- **Rule**: Goods receipt from purchase → post GL entry:
  - Dr. 152xxx (Inventory) — amount
  - Dr. 1331 (Input VAT) — VAT amount  
  - Cr. 331 (Accounts Payable) — total
- **Compliance**: Thông tư 99/2025/TT-BTC

### BR-08: GL Posting on Dispatch
- **Rule**: Goods dispatch (sale) → post GL entry:
  - Dr. 632 (COGS) — cost of goods sold
  - Cr. 152xxx (Inventory) — cost of goods sold
- **Compliance**: VAS 02, Paragraph 24

### BR-09: Warehouse Bookkeeping
- **Rule**: System must maintain warehouse book (sổ kho) per warehouse with running balances
- **Compliance**: Nghị định 50/2016/NĐ-CP, Article 7

### BR-10: E-Invoice for Inventory Transactions
- **Rule**: Goods receipt/dispatch must trigger e-invoice where applicable
- **Compliance**: Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP

### BR-11: Item Categories
- **Rule**: Items must be classified by type (raw materials, supplies, finished goods, WIP, etc.)
- **Compliance**: VAS 02, Paragraph 03

### BR-12: Opening Balance Migration
- **Rule**: On module activation, system must allow recording opening stock balances
- **Process**: Opening balance → Dr. 152xxx / Cr. retained earnings

---

## 5. System Entities (Domain Model)

### 5.1 Item (Hàng hóa/Vật tư)
Represents a stock-keeping unit (SKU) that can be inventoried.

```
Item:
  ID              string       // UUID
  Code            string       // Auto: MH-00001
  Name            string       // Item name
  Category        ItemCategory // enum: raw_material, supplies, finished_goods, wip, consignment
  SubCategory     string       // Optional: group classification
  Description     string
  Unit            string       // kg, pcs, m, set, etc.
  
  // Valuation
  ValuationMethod ValuationMethod // FIFO, weighted_average (default: FIFO)
  
  // GL Integration
  GLAccount152    string       // 1521/1522/1523/1524/1526/1527/1528/1529/1531
  GLAccount632    string       // COGS account (6321/6322)
  
  // Physical attributes
  MinStock        float64      // Minimum stock level (reorder point)
  MaxStock        float64      // Maximum stock level
  ReorderQty      float64      // Suggested reorder quantity
  
  // Status
  Status          ItemStatus   // active, inactive, discontinued
  
  // Audit
  CreatedBy       string
  CreatedAt       string
  UpdatedBy       string
  UpdatedAt       string
```

### 5.2 Warehouse (Kho)
```
Warehouse:
  ID              string
  Code            string       // Auto: KHO-001
  Name            string       // e.g., "Kho nguyên vật liệu A"
  Address         string
  WarehouseType   string       // raw_material, finished_goods, general
  Manager         string       // Responsible person
  Phone           string
  Status          WarehouseStatus // active, inactive
  
  // Audit
  CreatedBy       string
  CreatedAt       string
  UpdatedBy       string
  UpdatedAt       string
```

### 5.3 StockCard (Sổ kho / Bảng kê tồn kho)
Running balance per item per warehouse. Updated on every movement.

```
StockCard:
  ID              string
  ItemCode        string       // FK → Item.Code
  WarehouseCode   string       // FK → Warehouse.Code
  OpeningQty      float64      // Opening balance quantity
  OpeningValue    int64        // Opening balance value (VND)
  
  // Running totals (updated on each movement)
  TotalInQty      float64      // Total received quantity
  TotalInValue    int64        // Total received value (VND)
  TotalOutQty     float64      // Total dispatched quantity
  TotalOutValue   int64        // Total dispatched value (VND)
  CurrentQty      float64      // Current stock quantity
  CurrentValue    int64        // Current stock value (VND)
  AverageCost     int64        // Weighted average cost per unit (if method = weighted_average)
  
  // Last movement reference
  LastMovementID  string
  LastMovementDate string
  
  // Audit
  UpdatedAt       string
```

### 5.4 StockMovement (Phiếu kho / Chứng từ kho)
Every receipt, dispatch, transfer, or adjustment.

```
StockMovement:
  ID              string
  MovementCode    string       // Auto: PN-00001 (receipt), PX-00001 (dispatch), CC-00001 (transfer), DK-00001 (adjustment)
  MovementType    MovementType // receipt, dispatch, transfer_in, transfer_out, adjustment_plus, adjustment_minus, opening_balance
  MovementDate    string
  ItemCode        string       // FK → Item.Code
  WarehouseCode   string       // FK → Warehouse.Code (destination for transfer_in)
  Quantity        float64      // Always positive
  UnitPrice       int64        // Cost per unit (VND) — calculated by valuation method
  TotalCost       int64        // quantity * unit_price
  
  // Reference document
  RefDocType      string       // purchase_order, goods_receipt, sales_invoice, sales_return, transfer_order, physical_count, manual
  RefDocID        string       // ID of reference document
  RefDocNo        string       // Human-readable reference number
  
  // Transfer-specific
  FromWarehouse   string       // For transfer_out
  ToWarehouse     string       // For transfer_in
  
  // GL posting
  GLPosted        bool
  GLJournalID     string       // FK → ledger.JournalEntry.ID
  
  // Status
  Status          MovementStatus // draft, confirmed, posted, cancelled
  
  // Audit
  CreatedBy       string
  CreatedAt       string
  ConfirmedBy     string
  ConfirmedAt     string
```

### 5.5 StockAdjustment (Điều chỉnh kho)
For manual adjustments outside normal purchase/sales flow.

```
StockAdjustment:
  ID              string
  AdjustmentCode  string       // Auto: DC-00001
  AdjustmentDate  string
  Reason          string       // Physical count adjustment, damage, expiry, error correction
  ItemCode        string
  WarehouseCode   string
  AdjustedQty     float64      // Can be positive or negative
  UnitPrice       int64
  TotalCost       int64
  
  // Authorization
  ApprovedBy      string
  ApprovedAt      string
  
  // GL
  GLPosted        bool
  GLJournalID     string
  
  Status          AdjustmentStatus // pending, approved, posted, rejected
  
  // Audit
  CreatedBy       string
  CreatedAt       string
```

### 5.6 PhysicalCount (Kiểm kê thực tế)
Annual or ad-hoc physical inventory count.

```
PhysicalCount:
  ID              string
  CountCode       string       // Auto: KK-00001
  CountDate       string
  CountType       CountType    // annual, periodic, ad_hoc
  WarehouseCode   string
  
  // Status
  Status          CountStatus  // in_progress, reconciled, posted
  
  // Audit
  CreatedBy       string
  CreatedAt       string
  ReconciledBy    string
  ReconciledAt    string
```

### 5.7 PhysicalCountLine
```
PhysicalCountLine:
  ID              string
  CountID         string       // FK → PhysicalCount.ID
  ItemCode        string
  BookQty         float64      // Quantity per stock card (system)
  ActualQty       float64      // Physical count result
  Difference      float64      // ActualQty - BookQty
  UnitPrice       int64        // Cost per unit
  AdjustmentCost  int64        // Difference * UnitPrice
  Notes           string
```

---

## 6. Use Cases

### UC-01: Record Goods Receipt (Nhập kho hàng mua)
**Actor**: Warehouse staff
**Trigger**: Purchase invoice approved
**Preconditions**: Purchase order exists, goods receipt note exists
**Postconditions**: Stock increased, stock card updated, GL entry posted

**Happy Path**:
1. User selects PO → system loads PO lines
2. User enters received quantities per line
3. System validates: received qty ≤ ordered qty (± tolerance)
4. System creates GoodsReceipt document
5. For each line: system creates StockMovement(type=receipt)
6. System updates StockCard (increment CurrentQty, CurrentValue)
7. System posts GL: Dr. 152xxx / Dr. 1331 / Cr. 331
8. System marks PO line as partially/fully received

**Alternative Paths**:
- AP-01: Received qty > ordered qty → system warns, requires manager approval
- AP-02: Partial receipt → system allows, tracks remaining qty on PO
- AP-03: Multiple deliveries against same PO → system tracks cumulative

**Exception Paths**:
- EX-01: Item not found in system → error, user must create item first
- EX-02: Warehouse inactive → error
- EX-03: Duplicate receipt (same PO, same items, same date) → error
- EX-04: GL posting fails → receipt saved but marked "gl_pending", retry available

---

### UC-02: Record Goods Dispatch (Xuất kho bán hàng)
**Actor**: Warehouse staff
**Trigger**: Sales invoice approved / Sales order confirmed
**Preconditions**: Sales invoice exists, stock available
**Postconditions**: Stock decreased, stock card updated, GL entry posted

**Happy Path**:
1. User selects sales invoice → system loads invoice lines
2. System validates: sufficient stock for each line
3. System creates GoodsDispatch document
4. For each line: system creates StockMovement(type=dispatch)
5. System updates StockCard (decrement CurrentQty, CurrentValue)
6. System posts GL: Dr. 632 (COGS) / Cr. 152xxx (Inventory)
7. COGS = dispatched qty × unit price (from valuation method)

**Alternative Paths**:
- AP-01: Insufficient stock → error, user can backorder or adjust
- AP-02: Partial dispatch → system allows, tracks remaining
- AP-03: Sales return → triggers reverse stock movement (type=dispatch_reverse)

**Exception Paths**:
- EX-01: Stock balance goes negative → error (not allowed under Vietnamese law)
- EX-02: Item discontinued → warning, allow with override

---

### UC-03: Record Stock Transfer (Chuyển kho)
**Actor**: Warehouse manager
**Trigger**: Internal transfer order
**Preconditions**: Source warehouse has sufficient stock
**Postconditions**: Stock moved between warehouses, two stock cards updated

**Happy Path**:
1. User selects source warehouse and destination warehouse
2. User selects item and quantity to transfer
3. System validates: sufficient stock in source warehouse
4. System creates TransferOrder
5. System creates StockMovement(type=transfer_out) in source warehouse
6. System creates StockMovement(type=transfer_in) in destination warehouse
7. System updates both StockCards
8. GL: No entry (same company, no value change)

**Alternative Paths**:
- AP-01: In-transit warehouse (if using multi-step transfers) → intermediate movement
- AP-02: Transfer with value adjustment (if different valuation methods) → GL entry required

**Exception Paths**:
- EX-01: Insufficient stock in source → error
- EX-02: Same source and destination → error
- EX-03: Destination warehouse inactive → error

---

### UC-04: Perform Physical Count (Kiểm kê)
**Actor**: Warehouse staff + Finance
**Trigger**: Annual requirement / management order
**Preconditions**: Count period defined
**Postconditions**: Stock reconciled, adjustment posted

**Happy Path**:
1. User creates PhysicalCount for a warehouse
2. System generates list of all items in warehouse with book quantities
3. User/counter enters actual quantities per item
4. System calculates difference (actual - book) per item
5. User submits for review
6. Finance reviews differences, determines cause
7. System creates StockAdjustment for each difference
8. System posts GL: 
   - Positive adjustment: Dr. 152xxx / Cr. 632 (or 515xxx)
   - Negative adjustment: Dr. 632 (or 515xxx) / Cr. 152xxx
9. PhysicalCount marked as reconciled

**Alternative Paths**:
- AP-01: Count done by external auditor → same flow, different actor
- AP-02: Cycle counting (partial count) → same flow, subset of items

**Exception Paths**:
- EX-01: Large discrepancy (>X%) → requires additional approval
- EX-02: Missing item in warehouse (not in stock card) → create new stock movement
- EX-03: Extra item in warehouse (in stock card but not counted) → investigate

---

### UC-05: Record Opening Balance (Số dư đầu kỳ)
**Actor**: Accountant
**Trigger**: Module activation / new period
**Preconditions**: Items and warehouses created
**Postconditions**: Opening balances recorded, GL opening entry posted

**Happy Path**:
1. User selects warehouse and period (month/year)
2. User enters opening quantity and value per item
3. System validates: values ≥ 0
4. System creates StockMovement(type=opening_balance) per item
5. System creates/updates StockCard with opening values
6. System posts GL: Dr. 152xxx / Cr. retained earnings (111/112 or designated)

**Exception Paths**:
- EX-01: Opening balance already exists for period → error (must reverse first)

---

### UC-06: Write Down Inventory to NRV (Xác định lại giá trị tồn kho)
**Actor**: Accountant (year-end)
**Trigger**: Year-end closing
**Preconditions**: All transactions posted
**Postconditions**: Inventory value adjusted to NRV, GL entry posted

**Happy Path**:
1. System calculates original price per item (from stock card)
2. User enters NRV per item (or system calculates from latest selling price)
3. System compares: if NRV < original price → write-down needed
4. System calculates write-down amount: (original price - NRV) × quantity
5. System creates write-down entry
6. System posts GL: Dr. 515xxx (Inventory write-down) / Cr. 152xxx (Inventory)

**Alternative Paths**:
- AP-01: NRV recovers in subsequent year → reverse write-down (up to original price)

**Exception Paths**:
- EX-01: NRV not determinable → use current replacement price

---

### UC-07: Generate Stock Report (Báo cáo tồn kho)
**Actor**: Finance / Management
**Trigger**: On-demand / periodic
**Preconditions**: Stock data exists

**Reports**:
1. **Stock Balance Report** (Bảng kê tồn kho): item, warehouse, qty, value, avg cost
2. **Stock Movement Report** (Sổ kho): date, in, out, balance per item per warehouse
3. **Stock Valuation Report** (Báo cáo định giá tồn kho): original price, NRV, write-down
4. **Stock Aging Report** (Báo cáo tuổi tồn kho): how long items have been in stock
5. **Slow-Moving/Obsolete Stock Report** (Báo cáo hàng hóa chậm luân chuyển)

---

## 7. Processes & Workflows

### Process 1: Purchase → Inventory → GL
```
Purchase Order
    ↓
Goods Receipt Note (Phiếu nhập kho)
    ↓
Stock Movement (type=receipt) → StockCard updated
    ↓
GL Entry: Dr. 152xxx / Dr. 1331 / Cr. 331
    ↓
Purchase Invoice matched
```

### Process 2: Sales → Inventory → GL
```
Sales Invoice
    ↓
Goods Dispatch Note (Phiếu xuất kho)
    ↓
Stock Movement (type=dispatch) → StockCard updated
    ↓
GL Entry: Dr. 632 / Cr. 152xxx
    ↓
Revenue recognized
```

### Process 3: Internal Transfer
```
Transfer Order
    ↓
Stock Movement (type=transfer_out) → Source StockCard updated
    ↓
Stock Movement (type=transfer_in) → Destination StockCard updated
    ↓
No GL entry (same entity)
```

### Process 4: Physical Count → Adjustment
```
Physical Count Created
    ↓
Actual count entered
    ↓
Difference calculated
    ↓
Adjustment approved
    ↓
Stock Adjustment created → StockCard updated
    ↓
GL Entry: Dr. 632 or 152xxx / Cr. 152xxx or 632
```

### Process 5: Year-End NRV Write-Down
```
Year-End Trigger
    ↓
Original price per item calculated (from stock cards)
    ↓
NRV entered/calculated
    ↓
Write-down needed? (NRV < original price)
    ↓ Yes
Write-down journal created
    ↓
GL Entry: Dr. 515xxx / Cr. 152xxx
    ↓
Stock value updated
```

---

## 8. GL Integration

### Chart of Accounts for Inventory

| Account | Name | Type | When Used |
|---------|------|------|-----------|
| 1521 | Raw materials (Nguyên liệu chính) | Asset | Raw materials |
| 1522 | Spare parts (Phụ tùng thay thế) | Asset | Spare parts |
| 1523 | Tools (Công cụ dụng cụ) | Asset | Low-value tools (<30M VND) |
| 1524 | Unfinished products (Bán thành phẩm) | Asset | WIP |
| 1526 | Goods in transit (Hàng hóa trên đường đi) | Asset | In-transit goods |
| 1527 | Goods for sale (Hàng hóa) | Asset | Trading goods |
| 1528 | Goods sent for processing (Hàng hóa gửi đi加工) | Asset | Outsource processing |
| 1529 | Goods at consignment (Hàng hóa ký gửi) | Asset | Consignment |
| 1531 | Supplies (Vật tư) | Asset | Consumable supplies |
| 6321 | COGS - materials (Giá vốn hàng bán) | Expense | COGS |
| 6322 | COGS - services | Expense | COGS |
| 1331 | Input VAT (Thuế GTGT được khấu trừ) | Asset | VAT on purchase |
| 331 | Accounts Payable (Phải trả người bán) | Liability | Purchase on credit |
| 3331 | VAT payable (Thuế GTGT phải nộp) | Liability | VAT on sales |
| 515x | Inventory write-down (Chênh lệch định giá) | Expense | NRV write-down |

### Journal Entries

**Goods Receipt (Nhập kho hàng mua)**:
```
Dr. 152xxx (Inventory)           XXX
Dr. 1331 (Input VAT)             XXX
    Cr. 331 (Accounts Payable)       XXX
```

**Goods Dispatch (Xuất kho bán hàng)**:
```
Dr. 632 (COGS)                   XXX
    Cr. 152xxx (Inventory)           XXX
```

**Stock Adjustment - Increase**:
```
Dr. 152xxx (Inventory)           XXX
    Cr. 632 (COGS)                   XXX
```

**Stock Adjustment - Decrease**:
```
Dr. 632 (COGS)                   XXX
    Cr. 152xxx (Inventory)           XXX
```

**NRV Write-Down**:
```
Dr. 515xxx (Inventory Write-down) XXX
    Cr. 152xxx (Inventory)           XXX
```

**NRV Write-Down Reversal** (next year, up to original price):
```
Dr. 152xxx (Inventory)           XXX
    Cr. 515xxx (Inventory Write-down) XXX
```

---

## 9. Data Flows

### Flow 1: Purchase Receipt
```
User Input
    ↓
Validation (PO exists, item exists, warehouse active, qty ≤ ordered)
    ↓
Create GoodsReceipt document
    ↓
For each line:
    Create StockMovement(type=receipt, item, warehouse, qty, unit_price)
    Update StockCard:
        CurrentQty += qty
        CurrentValue += qty * unit_price
        TotalInQty += qty
        TotalInValue += qty * unit_price
    Calculate unit_price (FIFO: from PO; Weighted Average: recalculate average)
    ↓
Post GL:
    Dr. 152xxx
    Dr. 1331
    Cr. 331
    ↓
Return receipt with stock movement IDs
```

### Flow 2: Sales Dispatch
```
User Input (from Sales Invoice)
    ↓
Validation (item exists, warehouse active, stock sufficient)
    ↓
Create GoodsDispatch document
    ↓
For each line:
    Create StockMovement(type=dispatch, item, warehouse, qty, unit_price)
    Update StockCard:
        CurrentQty -= qty
        CurrentValue -= qty * unit_price
        TotalOutQty += qty
        TotalOutValue += qty * unit_price
    Calculate unit_price (FIFO: oldest layer; Weighted Average: current avg)
    ↓
Post GL:
    Dr. 632
    Cr. 152xxx
    ↓
Return dispatch with stock movement IDs
```

### Flow 3: Physical Count
```
Physical Count Created
    ↓
Load all items in warehouse with book quantities from StockCards
    ↓
User enters actual quantities
    ↓
Calculate differences:
    For each item:
        Difference = ActualQty - BookQty
        AdjustmentCost = Difference * UnitPrice (from StockCard)
    ↓
If Difference > 0:
    Create StockAdjustment(type=increase)
    StockMovement(type=adjustment_plus)
    GL: Dr. 152xxx / Cr. 632
If Difference < 0:
    Create StockAdjustment(type=decrease)
    StockMovement(type=adjustment_minus)
    GL: Dr. 632 / Cr. 152xxx
    ↓
Update StockCards
    ↓
PhysicalCount marked reconciled
```

---

## 10. User Journeys

### Journey 1: Warehouse Staff — Daily Receipt
```
Login → Dashboard → "Pending Receipts" 
→ Select PO → View items → Enter received quantities 
→ Confirm → Receipt created, stock updated
→ Print receipt (optional)
```

### Journey 2: Warehouse Staff — Daily Dispatch
```
Login → Dashboard → "Pending Dispatches"
→ Select Sales Invoice → View items → Pick from stock
→ Confirm → Dispatch created, stock reduced
→ Print dispatch note (optional)
```

### Journey 3: Finance — Month-End
```
Login → Reports → Stock Balance
→ Review balances → Check for anomalies
→ Run Stock Valuation → Check NRV
→ If write-down needed → Create adjustment
→ Approve adjustments → Post GL
→ Export for financial statements
```

### Journey 4: Warehouse Manager — Physical Count
```
Login → Physical Count → Create new count
→ Select warehouse → System generates item list
→ Print count sheet → Staff counts manually
→ Enter actual quantities → System shows differences
→ Review large differences → Investigate
→ Submit for approval → Finance reviews
→ Approve → Adjustments posted
→ Mark count as complete
```

---

## 11. UI/UX Wireframes

### Screen 1: Item Master
```
┌─────────────────────────────────────────────────┐
│ Item Master                                     │
├─────────────────────────────────────────────────┤
│ [+ New Item] [Import] [Export]                  │
├─────────────────────────────────────────────────┤
│ Filter: [Category ▼] [Status ▼] [Search...  ] │
├─────────────────────────────────────────────────┤
│ Code   │ Name          │ Category │ Unit │ Status│
│ MH-001 │ Steel Rods    │ Raw Mat. │ kg   │ Active│
│ MH-002 │ Packaging     │ Supplies │ pcs  │ Active│
│ MH-003 │ Widget A      │ Finished │ pcs  │ Active│
└─────────────────────────────────────────────────┘
```

### Screen 2: Stock Balance (Bảng kê tồn kho)
```
┌─────────────────────────────────────────────────────┐
│ Stock Balance Report                    [Print] [Export]
├─────────────────────────────────────────────────────┤
│ Warehouse: [All warehouses ▼]  Date: [2026-09-01]  │
├─────────────────────────────────────────────────────┤
│ Item Code │ Item Name    │ Warehouse │ Qty │ Value   │
│ MH-001    │ Steel Rods   │ KHO-A     │ 500 │ 250,000 │
│ MH-001    │ Steel Rods   │ KHO-B     │ 200 │ 100,000 │
│ MH-002    │ Packaging    │ KHO-A     │ 1000│ 50,000  │
│ MH-003    │ Widget A     │ KHO-FG    │ 300 │ 450,000 │
├─────────────────────────────────────────────────────┤
│ TOTAL                              │ 2,000│ 850,000 │
└─────────────────────────────────────────────────────┘
```

### Screen 3: Goods Receipt Form
```
┌─────────────────────────────────────────────────────┐
│ Goods Receipt Note                                  │
├─────────────────────────────────────────────────────┤
│ Receipt No: [PN-00045]  Date: [2026-09-01]         │
│ PO Reference: [PO-00012]  Supplier: [NCC-001]     │
│ Warehouse: [Kho nguyên vật liệu ▼]                 │
├─────────────────────────────────────────────────────┤
│ Line │ Item     │ PO Qty │ Received │ Price │ Total │
│ 1    │ MH-001   │ 100    │ [100]    │ 500   │ 50,000│
│ 2    │ MH-002   │ 500    │ [480]    │ 50    │ 24,000│
├─────────────────────────────────────────────────────┤
│ [+ Add Line]                                       │
├─────────────────────────────────────────────────────┤
│ Subtotal: 74,000  VAT (10%): 7,400  Total: 81,400 │
│                                                     │
│         [Save Draft]  [Confirm & Post GL]           │
└─────────────────────────────────────────────────────┘
```

### Screen 4: Physical Count Entry
```
┌─────────────────────────────────────────────────────┐
│ Physical Count: KK-0001                             │
│ Warehouse: Kho nguyên vật liệu A                    │
│ Date: 2026-09-01                                    │
├─────────────────────────────────────────────────────┤
│ Item Code │ Item Name  │ Book Qty │ Actual │ Diff   │
│ MH-001    │ Steel Rods │ 500      │ [498]  │ -2     │
│ MH-002    │ Packaging  │ 1000     │ [1005] │ +5     │
├─────────────────────────────────────────────────────┤
│ Total Discrepancies: -2 items                       │
│ Estimated Adjustment: -50,000 VND                   │
│                                                     │
│         [Save]  [Submit for Review]                 │
└─────────────────────────────────────────────────────┘
```

---

## 12. Technical Specification

### 12.1 Module Structure
```
internal/
├── domain/inventory/
│   ├── entity.go           // Item, Warehouse, StockCard, StockMovement, StockAdjustment, PhysicalCount, PhysicalCountLine
│   ├── entity_test.go      // Entity validation tests
│   └── valuation.go        // FIFO and Weighted Average valuation logic
├── application/inventory/
│   └── service.go          // Business logic for all operations
├── infrastructure/persistence/inventory/
│   └── repository.go       // SQLite JSON-doc storage
└── interfaces/http/inventory/
    └── handler.go          // HTTP handlers
```

### 12.2 HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST   | /api/v1/inventory/items | Create item |
| GET    | /api/v1/inventory/items | List items |
| GET    | /api/v1/inventory/items/:code | Get item by code |
| PUT    | /api/v1/inventory/items/:code | Update item |
| DELETE | /api/v1/inventory/items/:code | Deactivate item |
| POST   | /api/v1/inventory/warehouses | Create warehouse |
| GET    | /api/v1/inventory/warehouses | List warehouses |
| GET    | /api/v1/inventory/warehouses/:code | Get warehouse |
| PUT    | /api/v1/inventory/warehouses/:code | Update warehouse |
| POST   | /api/v1/inventory/stock-cards | Get stock balances |
| GET    | /api/v1/inventory/stock-cards/:itemCode/:warehouseCode | Get specific balance |
| POST   | /api/v1/inventory/receipts | Create goods receipt |
| GET    | /api/v1/inventory/receipts/:id | Get receipt |
| GET    | /api/v1/inventory/receipts | List receipts |
| POST   | /api/v1/inventory/dispatches | Create goods dispatch |
| GET    | /api/v1/inventory/dispatches/:id | Get dispatch |
| GET    | /api/v1/inventory/dispatches | List dispatches |
| POST   | /api/v1/inventory/transfers | Create transfer |
| GET    | /api/v1/inventory/transfers/:id | Get transfer |
| POST   | /api/v1/inventory/adjustments | Create adjustment |
| PUT    | /api/v1/inventory/adjustments/:id/approve | Approve adjustment |
| POST   | /api/v1/inventory/physical-counts | Create physical count |
| PUT    | /api/v1/inventory/physical-counts/:id/submit | Submit count |
| PUT    | /api/v1/inventory/physical-counts/:id/reconcile | Reconcile count |
| GET    | /api/v1/inventory/physical-counts/:id | Get count |
| POST   | /api/v1/inventory/reports/stock-balance | Stock balance report |
| POST   | /api/v1/inventory/reports/stock-movement | Stock movement report |
| POST   | /api/v1/inventory/reports/valuation | Stock valuation report |
| POST   | /api/v1/inventory/nrv/write-down | NRV write-down |
| POST   | /api/v1/inventory/nrv/reverse | NRV reversal |

### 12.3 Service Methods

```go
type Service interface {
    // Item management
    CreateItem(ctx, item) (*Item, error)
    GetItem(ctx, code) (*Item, error)
    UpdateItem(ctx, item) (*Item, error)
    DeactivateItem(ctx, code) error
    ListItems(ctx, category, status, search, limit, offset) ([]*Item, int, error)
    
    // Warehouse management
    CreateWarehouse(ctx, wh) (*Warehouse, error)
    GetWarehouse(ctx, code) (*Warehouse, error)
    UpdateWarehouse(ctx, wh) (*Warehouse, error)
    ListWarehouses(ctx, status, limit, offset) ([]*Warehouse, int, error)
    
    // Stock operations
    GetStockBalance(ctx, itemCode, warehouseCode) (*StockCard, error)
    ListStockBalances(ctx, warehouseCode, limit, offset) ([]*StockCard, int, error)
    
    // Goods receipt
    CreateReceipt(ctx, receipt) (*StockMovement, error)
    GetReceipt(ctx, id) (*StockMovement, error)
    ListReceipts(ctx, warehouseCode, fromDate, toDate, limit, offset) ([]*StockMovement, int, error)
    
    // Goods dispatch
    CreateDispatch(ctx, dispatch) (*StockMovement, error)
    GetDispatch(ctx, id) (*StockMovement, error)
    ListDispatches(ctx, warehouseCode, fromDate, toDate, limit, offset) ([]*StockMovement, int, error)
    
    // Transfer
    CreateTransfer(ctx, transfer) (*StockMovement, *StockMovement, error)
    
    // Adjustment
    CreateAdjustment(ctx, adj) (*StockAdjustment, error)
    ApproveAdjustment(ctx, id, approvedBy) error
    
    // Physical count
    CreatePhysicalCount(ctx, count) (*PhysicalCount, error)
    SubmitPhysicalCount(ctx, id, lines) error
    ReconcilePhysicalCount(ctx, id, reconciledBy) error
    
    // Valuation
    CalculateUnitPrice(ctx, itemCode, method) (int64, error)
    CalculateCOGS(ctx, itemCode, warehouseCode, quantity, method) (int64, error)
    
    // NRV
    RunNRVWriteDown(ctx, date, writeDowns) error
    ReverseNRVWriteDown(ctx, date) error
    
    // Reports
    StockBalanceReport(ctx, warehouseCode, date) ([]StockBalanceRow, error)
    StockMovementReport(ctx, itemCode, warehouseCode, fromDate, toDate) ([]StockMovementRow, error)
    StockValuationReport(ctx, date) ([]StockValuationRow, error)
}
```

### 12.4 Valuation Methods

**FIFO Implementation**:
```
For dispatch:
  1. Get all receipt layers (ordered by date, oldest first)
  2. Allocate dispatch qty from oldest layer first
  3. If layer exhausted, move to next
  4. Unit price = cost of allocated layer(s)
  5. Update layers (reduce consumed quantities)
```

**Weighted Average Implementation**:
```
For receipt:
  1. Get current stock card (qty, value)
  2. New avg cost = (current_value + receipt_value) / (current_qty + receipt_qty)
  3. Update stock card

For dispatch:
  1. Use current average cost from stock card
  2. Dispatch cost = qty × average_cost
```

### 12.5 Database Tables (new in migrate.go)

```sql
-- Items
CREATE TABLE IF NOT EXISTS inventory_items (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Warehouses  
CREATE TABLE IF NOT EXISTS inventory_warehouses (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Stock Cards (one per item per warehouse)
CREATE TABLE IF NOT EXISTS inventory_stock_cards (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Stock Movements (receipts, dispatches, transfers, adjustments)
CREATE TABLE IF NOT EXISTS inventory_stock_movements (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- FIFO Layers (for FIFO valuation)
CREATE TABLE IF NOT EXISTS inventory_fifo_layers (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Stock Adjustments
CREATE TABLE IF NOT EXISTS inventory_stock_adjustments (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Physical Counts
CREATE TABLE IF NOT EXISTS inventory_physical_counts (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

-- Physical Count Lines
CREATE TABLE IF NOT EXISTS inventory_physical_count_lines (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL
);
```

---

## 13. Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
- [ ] Item entity + validation + Repository interface
- [ ] Warehouse entity + validation + Repository interface
- [ ] Item and Warehouse HTTP handlers (CRUD)
- [ ] Basic tests for entities and handlers

### Phase 2: Stock Tracking (Week 3-4)
- [ ] StockCard entity + operations
- [ ] FIFO valuation logic (layers)
- [ ] Weighted Average valuation logic
- [ ] Goods Receipt flow (with GL posting)
- [ ] Goods Dispatch flow (with GL posting)
- [ ] Stock balance reports

### Phase 3: Transfers & Adjustments (Week 5-6)
- [ ] Stock Transfer flow
- [ ] Stock Adjustment flow
- [ ] Opening Balance import
- [ ] Stock movement reports

### Phase 4: Physical Count & NRV (Week 7-8)
- [ ] Physical Count entity + flow
- [ ] Physical count reconciliation
- [ ] NRV write-down and reversal
- [ ] Year-end inventory valuation

### Phase 5: Integration & Polish (Week 9-10)
- [ ] Purchase integration (auto-create receipt from PO)
- [ ] Sales integration (auto-create dispatch from SI)
- [ ] E-invoice integration
- [ ] Final testing and documentation

---

## 14. Execution Plan

### Week 1: Item + Warehouse
- Day 1-2: Item entity, validation, Repository, tests
- Day 3-4: Warehouse entity, validation, Repository, tests
- Day 5: HTTP handlers for both, wiring in main.go

### Week 2: Stock Card + FIFO
- Day 1-2: StockCard entity, FIFO layer entity
- Day 3-4: FIFO valuation logic
- Day 5: Weighted Average logic

### Week 3: Goods Receipt
- Day 1-2: Receipt entity, service logic
- Day 3-4: GL posting integration
- Day 5: HTTP handler + tests

### Week 4: Goods Dispatch
- Day 1-2: Dispatch entity, service logic
- Day 3-4: GL posting integration
- Day 5: HTTP handler + tests

### Week 5: Transfer + Adjustment
- Day 1-2: Transfer flow
- Day 3-4: Adjustment flow
- Day 5: Tests

### Week 6: Opening Balance + Reports
- Day 1-2: Opening balance import
- Day 3-4: Stock reports
- Day 5: Tests

### Week 7: Physical Count
- Day 1-2: Physical count entity + flow
- Day 3-4: Reconciliation
- Day 5: Tests

### Week 8: NRV + Year-End
- Day 1-2: NRV write-down
- Day 3-4: NRV reversal
- Day 5: Year-end process

### Week 9: Integration
- Day 1-2: Purchase integration
- Day 3-4: Sales integration
- Day 5: E-invoice integration

### Week 10: Final
- Day 1-2: Final testing
- Day 3-4: Documentation
- Day 5: Code review

---

## 15. Open Questions & Risks

### Open Questions
1. **Barcode support**: Does the system need barcode scanning for warehouse operations?
2. **Multi-currency**: Do we need to handle foreign currency for inventory purchases?
3. **Batch/lot tracking**: Is batch or lot tracking required?
4. **Expiry tracking**: Do we need to track expiry dates for perishable items?
5. **Serial number tracking**: For high-value items, do we need serial number tracking?
6. **Minimum stock alerts**: Auto-reorder when stock falls below minimum?
7. **Warehouse zones/locations**: Do we need sub-location tracking within warehouses?

### Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| FIFO implementation complexity | High | Build layer-based FIFO first, simplify if needed |
| Performance on large stock cards | Medium | Index on (item_code, warehouse_code) |
| Concurrent stock updates | High | Use database transactions with row-level locking |
| GL posting failures | Medium | Queue-based retry mechanism |
| Data migration from existing systems | Medium | Import API for opening balances |

---

*Document prepared by: BA Lead + Vietnamese Chief Accountant*
*Date: 2026-09-01*
*Version: 1.0*
