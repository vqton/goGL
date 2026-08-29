# FA Module Analysis - Vietnamese ERP Production Readiness

## VERDICT: NOT PRODUCTION READY

Current FA module is **STUB**. Returns `501 Not Implemented`. Zero business logic. Zero compliance.

---

## CRITICAL GAPS vs VIETNAMESE STANDARDS

### 1. Missing Core Accounting Compliance

| Requirement | Source | Status |
|------------|--------|--------|
| Asset recognition (VND 30M threshold) | Circular 45/2013/TT-BTC | MISSING |
| 3 depreciation methods (straight-line, declining-balance, units-of-output) | VAS 03, Circular 45 | Entity defined, logic MISSING |
| Depreciation life framework (Annex I) | Circular 45/2013 | MISSING |
| Account 214 (Accumulated Depreciation) | Circular 99/2025 | MISSING |
| Asset classification (6 types) | Circular 45 | MISSING |
| Asset card per object | Circular 45 Art 5 | MISSING |
| Transfer/deposit vouchers | Circular 45 | MISSING |
| Liquidation/sale accounting | VAS 03 Para 37-41 | MISSING |
| Periodic useful life review | VAS 03 Para 33-34 | MISSING |
| Depreciation method change rules | Circular 45 Art 13 | MISSING |

### 2. Missing Account Structure (Circular 99/2025)

Required Chart of Accounts:
- **111** - Cash on hand
- **211** - Tangible Fixed Assets (original cost)
- **214** - Accumulated Depreciation (2141 for tangible)
- **336** - Payable to employees
- **411** - Owner's equity
- **623** - Depreciation of production assets
- **627** - Depreciation of production equipment
- **641** - Depreciation of management assets
- **642** - Depreciation of commercial assets

### 3. Missing Business Rules

#### Depreciation Rules (Circular 45/2013)
1. **Monthly depreciation** = Annual depreciation / 12
2. **Start depreciation**: From day asset increases (1st or 15th of month)
3. **Stop depreciation**: When asset decreases
4. **Exceptions** (no depreciation):
   - Assets under construction
   - Assets pending liquidation (fully depreciated but still in use)
   - Assets temporarily unused (if >9 months, no tax deduction)
   - Biological assets (use Account 215 per Circular 99)

#### Accelerated Depreciation Conditions
Only applies to: machinery, measuring instruments, transport, management tools, perennial gardens
- Max 2x straight-line rate
- Enterprise must be profitable
- Excess over 2x: not tax-deductible

#### Declining Balance Method Conditions
Both conditions required:
1. New (unused) asset
2. Machinery or measuring instruments only
3. Technology requires rapid change

#### Units-of-Output Method Conditions
All three conditions required:
1. Directly related to production
2. Total output units determinable by design capacity
3. Average monthly capacity >= 100% of design

### 4. Missing Document Requirements

Per Circular 45 Art 5:
- Transfer/receiving minutes
- Purchase contracts
- Purchase invoices
- Individual asset card
- Asset code numbering
- Separate tracking per asset

### 5. Missing Financial Statement Disclosures (VAS 03 Para 39)

Per asset type:
- Depreciation method, useful life, depreciation rate
- Historical cost, accumulated depreciation, residual value (beginning and end)
- Increases/decreases in period
- Depreciated amount in period
- Residual value of temporarily unused assets
- Historical cost of fully-depreciated assets still in use

---

## COMPETITOR ANALYSIS

### Fast Accounting (Version 6.x)
- Full FA module with depreciation schedule
- Auto-depreciation posting
- Asset card management
- Transfer/liquidation support
- Vietnamese account mapping

### MISA (Version 2024+)
- FA sub-ledger integrated with GL
- 3 depreciation methods
- Asset classification per Circular 45
- Depreciation schedule report
- Tax declaration support

### Bravo ERP
- Complete FA lifecycle
- Barcode/QR for asset tracking
- Depreciation calculation engine
- Asset revaluation support
- Integration with fixed asset register

---

## REQUIRED SPECS FOR PROD

### 1. Entity Fields (Enhanced)

```go
type FixedAsset struct {
    // Core
    ID, Code, Name, Description string
    Category, Location, Department string
    
    // Financial (VND cents)
    OriginalCost      int64   // Original cost
    ResidualValue     int64   // Estimated liquidation value
    AccumulatedDepr   int64   // Accumulated depreciation
    CurrentValue      int64   // OriginalCost - AccumulatedDepr
    
    // Depreciation
    DepreciationMethod string // straight_line, declining_balance, units_of_output
    UsefulLifeMonths   int    // In months
    DepreciationRate   int64  // Basis points (100 = 1%)
    MonthlyDeprAmount  int64  // Calculated monthly amount
    
    // Classification (Circular 45)
    AssetType          string // housing, machinery, transport, etc.
    AssetGroup         string // Group within type
    
    // Status
    State              string // active, inactive, scrapped, sold, pending_liquidation
    
    // Dates
    PurchaseDate       string
    InServiceDate      string
    LastReviewDate     string
    
    // Source
    VendorName         string
    PurchaseOrderNo    string
    InvoiceNo          string
    
    // Tracking
    SerialNo           string
    Barcode            string
    
    // Accounting
    AccountCode211     string // Account for original cost
    AccountCode214     string // Account for accumulated depreciation
    
    // Audit
    CreatedBy, CreatedAt string
    UpdatedBy, UpdatedAt string
}
```

### 2. Repository Interface

```go
type Repository interface {
    // CRUD
    Create(ctx, asset) error
    FindByID(ctx, id) (*FixedAsset, error)
    Update(ctx, asset) error
    Delete(ctx, id) error
    List(ctx, filter) ([]*FixedAsset, error)
    
    // Accounting
    CalculateDepreciation(ctx, id, period) (*DepreciationEntry, error)
    GetDepreciationSchedule(ctx, id) ([]DepreciationEntry, error)
    
    // Reporting
    GetAssetRegister(ctx, filter) (*AssetRegister, error)
    GetDepreciationReport(ctx, period) (*DepreciationReport, error)
    
    // Sequence
    NextCode(ctx) (int64, error)
}
```

### 3. Service Methods

```go
type Service interface {
    // Lifecycle
    CreateAsset(ctx, input, actor) (*FixedAsset, error)
    GetAsset(ctx, id) (*FixedAsset, error)
    UpdateAsset(ctx, id, patch, actor) (*FixedAsset, error)
    DeleteAsset(ctx, id) error
    
    // Depreciation
    CalculateMonthlyDepreciation(ctx, period) ([]DepreciationEntry, error)
    PostDepreciation(ctx, period, actor) error
    
    // Transfer
    TransferAsset(ctx, id, target, actor) error
    
    // Liquidation
    RequestLiquidation(ctx, id, reason, actor) (*LiquidationRequest, error)
    ApproveLiquidation(ctx, id, approver) error
    ExecuteLiquidation(ctx, id, actor) error
    
    // Reporting
    GetAssetRegister(ctx, filter) (*AssetRegister, error)
    GetDepreciationSchedule(ctx, id) (*DepreciationSchedule, error)
    
    // Revaluation
    RevalueAsset(ctx, id, newValue, approver) error
}
```

### 4. API Endpoints

```
POST   /api/v1/fixed-assets              - Create asset
GET    /api/v1/fixed-assets              - List assets
GET    /api/v1/fixed-assets/:id          - Get asset
PUT    /api/v1/fixed-assets/:id          - Update asset
DELETE /api/v1/fixed-assets/:id          - Delete asset

POST   /api/v1/fixed-assets/:id/depreciation/calculate - Calculate depreciation
POST   /api/v1/fixed-assets/:id/depreciation/post      - Post depreciation

POST   /api/v1/fixed-assets/:id/transfer   - Transfer asset
POST   /api/v1/fixed-assets/:id/liquidate  - Request liquidation
POST   /api/v1/fixed-assets/:id/liquidate/approve - Approve liquidation

GET    /api/v1/reports/asset-register      - Asset register report
GET    /api/v1/reports/depreciation        - Depreciation report
GET    /api/v1/reports/depreciation-schedule/:id - Individual schedule
```

---

## IMPLEMENTATION ROADMAP

### Phase 1: Core Entity (Week 1)
- [ ] Enhance entity with all required fields
- [ ] Add account code fields (211, 214)
- [ ] Add classification fields per Circular 45
- [ ] Validation rules per VAS 03 and Circular 45
- [ ] Unit tests

### Phase 2: Repository + DB (Week 1-2)
- [ ] SQLite repository implementation
- [ ] Migration tables
- [ ] CRUD operations
- [ ] Basic queries

### Phase 3: Service - Lifecycle (Week 2)
- [ ] Create asset with code generation
- [ ] Update asset (with state checks)
- [ ] Delete asset (with state checks)
- [ ] List with filters

### Phase 4: Depreciation Engine (Week 2-3)
- [ ] Straight-line calculation
- [ ] Declining-balance calculation
- [ ] Units-of-output calculation
- [ ] Monthly depreciation posting
- [ ] Depreciation schedule generation

### Phase 5: Business Operations (Week 3)
- [ ] Asset transfer
- [ ] Asset revaluation
- [ ] Liquidation workflow
- [ ] State transitions

### Phase 6: Reports + UI (Week 3-4)
- [ ] Asset register report
- [ ] Depreciation report
- [ ] Depreciation schedule report
- [ ] Web UI templates
- [ ] Export to Excel/PDF

### Phase 7: Integration (Week 4)
- [ ] GL posting (Account 211, 214)
- [ ] Tax reporting integration
- [ ] Authorization (Casbin)

---

## USER JOURNEYS

### Journey 1: Asset Acquisition
1. User creates asset card
2. System validates (cost >= 30M, valid dates, category)
3. System generates code (FA-XXXXX)
4. System posts to Account 211
5. User uploads supporting documents
6. Asset created in "active" state

### Journey 2: Monthly Depreciation
1. System calculates depreciation per method
2. System generates depreciation entries
3. User reviews depreciation schedule
4. User approves and posts
5. System posts to Account 623/627/641/642 and Account 214

### Journey 3: Asset Transfer
1. User selects asset to transfer
2. User enters target location/department
3. System creates transfer record
4. System updates asset location
5. System creates accounting entry

### Journey 4: Asset Liquidation
1. User requests liquidation
2. System validates (fully depreciated or manual approval)
3. System calculates gain/loss
4. Approver reviews
5. System posts liquidation entry
6. Asset state changes to "scrapped" or "sold"

---

## WIREFRAME SKETCHES

### Asset List View
```
+------------------------------------------+
| Fixed Assets                    [+ New]   |
+------------------------------------------+
| Search: [____________] Filter: [All ▼]    |
+------------------------------------------+
| Code    | Name      | Cost    | Status   |
| FA-00001| Machine A | 500M    | Active   |
| FA-00002| Truck     | 800M    | Active   |
| FA-00003| Laptop    | 25M     | Scrapped |
+------------------------------------------+
| < 1 2 3 ... >                            |
+------------------------------------------+
```

### Asset Detail View
```
+------------------------------------------+
| FA-00001 - Machine A            [Edit]   |
+------------------------------------------+
| Basic Info              | Depreciation    |
| Category: Machinery     | Method: SL      |
| Location: Factory A     | Life: 10 years  |
| Status: Active          | Rate: 10%/year  |
|                         | Monthly: 4.17M  |
+------------------------------------------+
| Financial Summary                       |
| Original Cost:    500,000,000 VND        |
| Accum. Depr:      200,000,000 VND        |
| Current Value:    300,000,000 VND        |
+------------------------------------------+
| [Transfer] [Liquidate] [View Schedule]   |
+------------------------------------------+
```

### Depreciation Schedule
```
+------------------------------------------+
| Depreciation Schedule - FA-00001         |
+------------------------------------------+
| Month | Depr Amount | Accum Depr | Value |
|-------|-------------|------------|-------|
| 01/26 | 4,166,667   | 4,166,667  | 496M  |
| 02/26 | 4,166,667   | 8,333,333  | 492M  |
| ...   | ...         | ...        | ...   |
| 12/26 | 4,166,667   | 50,000,000 | 450M  |
+------------------------------------------+
```

---

## COMPLIANCE CHECKLIST

- [ ] Circular 45/2013/TT-BTC compliance
- [ ] Circular 30/2025/TT-BTC updates
- [ ] Circular 99/2025/TT-BTC account structure
- [ ] VAS 03 - Tangible Fixed Assets
- [ ] VBHN 12/VBHN-BTC 2025 consolidated
- [ ] Depreciation life framework (Annex I)
- [ ] Asset classification (6 types)
- [ ] Three depreciation methods
- [ ] Accelerated depreciation conditions
- [ ] Asset card per object
- [ ] Transfer documentation
- [ ] Liquidation procedures
- [ ] Financial statement disclosures
- [ ] Tax deduction rules
- [ ] Monthly depreciation posting

---

## DATA FLOW

```
[Asset Acquisition]
    │
    ▼
[Asset Card Created] ──► [Account 211 - Debit]
    │
    ▼
[Monthly Depreciation] ──► [Account 623/627/641/642 - Debit]
    │                     ──► [Account 214 - Credit]
    ▼
[Depreciation Schedule Updated]
    │
    ▼
[Asset Transfer/Liquidation] ──► [Account 336/411 - Credit]
                               ──► [Account 214 - Debit]
                               ──► [Account 211 - Credit]
```

---

*Analysis complete. Module needs full implementation before PROD use.*
