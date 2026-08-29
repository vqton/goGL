# Use Cases - Fixed Asset Module

**Version**: 1.0  
**Date**: 2026-08-29

---

## UC-01: Create Fixed Asset

### Actor
Accountant / Asset Manager

### Preconditions
- User authenticated
- Asset data available (invoice, contract)

### Happy Path
1. User navigates to Asset List
2. User clicks "New Asset"
3. System displays asset form
4. User enters asset data:
   - Name, Category, Location
   - Original Cost (>= 30,000,000 VND)
   - Purchase Date, In-Service Date
   - Useful Life (within Circular 45 range)
   - Depreciation Method
5. System validates data
6. System generates code (FA-XXXXX)
7. System creates asset record
8. System posts to Account 211
9. System displays success message
10. System shows asset detail page

### Alternative Paths
- **4a.** Cost < 30M: System rejects, shows "Cost must be >= 30,000,000 VND"
- **4b.** Useful life out of range: System shows valid range from Annex I
- **4c.** Invalid dates: System shows date validation error
- **5a.** Duplicate code: System regenerates code

### Exception Paths
- **E1.** Database error: System shows "System error, please try again"
- **E2.** Session expired: System redirects to login

### Business Rules
- BR-01: Cost must be >= VND 30,000,000
- BR-02: Useful life must be within Circular 45 framework
- BR-03: Depreciation method must be one of three approved methods
- BR-04: Residual value < Original cost

---

## UC-02: Calculate Monthly Depreciation

### Actor
System (automated) / Accountant (manual trigger)

### Preconditions
- Assets exist in "active" state
- Period not yet depreciated

### Happy Path
1. System/Accountant triggers depreciation calculation
2. System selects all active assets
3. System calculates depreciation per asset:
   - **Straight-line**: Original Cost / Useful Life Months
   - **Declining-balance**: Book Value × Rate
   - **Units-of-output**: (Original Cost - Residual) × (Actual Output / Total Output)
4. System generates depreciation entries
5. System displays depreciation schedule for review
6. Accountant reviews and approves
7. System posts depreciation:
   - Debit: Account 623/627/641/642
   - Credit: Account 2141
8. System updates accumulated depreciation
9. System marks period as depreciated

### Alternative Paths
- **3a.** Asset not used full month: Calculate from 1st or 15th
- **3b.** Asset increased in period: Start depreciation from increase date
- **3c.** Asset decreased in period: Stop depreciation from decrease date
- **6a.** Accountant rejects: System clears entries, allows modification

### Exception Paths
- **E1.** Calculation overflow: System rounds to nearest VND
- **E2.** Missing depreciation rate: System uses straight-line default
- **E3.** Asset with zero useful life: Skip depreciation, log warning

### Business Rules
- BR-05: Monthly depreciation = Annual depreciation / 12
- BR-06: Start from day 1 or 15 of month
- BR-07: Assets under construction: No depreciation
- BR-08: Assets unused > 9 months: No tax deduction

---

## UC-03: Transfer Fixed Asset

### Actor
Asset Manager

### Preconditions
- Asset in "active" state
- Target location/department valid

### Happy Path
1. User selects asset to transfer
2. User clicks "Transfer"
3. System displays transfer form
4. User enters:
   - Transfer date
   - Target location
   - Target department
   - Reason
5. System validates data
6. System creates transfer record
7. System updates asset location/department
8. System creates transfer document
9. System shows success message

### Alternative Paths
- **4a.** Same location: System rejects, shows "Asset already at target"
- **5a.** Asset in liquidation: System rejects transfer

### Exception Paths
- **E1.** Target location invalid: System shows validation error
- **E2.** Concurrent modification: System shows conflict, reloads asset

### Business Rules
- BR-09: Only active assets can be transferred
- BR-10: Transfer requires approval for high-value assets

---

## UC-04: Liquidate Fixed Asset

### Actor
Accountant (request) / Finance Manager (approve)

### Preconditions
- Asset exists
- Liquidation reason documented

### Happy Path
1. User selects asset to liquidate
2. User clicks "Request Liquidation"
3. System displays liquidation form
4. User enters:
   - Liquidation reason
   - Selling price (if sold)
   - Liquidation costs
5. System calculates gain/loss
6. System creates liquidation request
7. System sets asset state to "pending_liquidation"
8. Finance Manager reviews request
9. Finance Manager approves
10. System executes liquidation:
    - Posts accounting entries
    - Updates asset state to "scrapped" or "sold"
11. System generates liquidation document

### Alternative Paths
- **9a.** Manager rejects: System cancels request, restores asset state
- **5a.** Gain: System posts to Account 515
- **5b.** Loss: System posts to Account 632

### Exception Paths
- **E1.** Asset fully depreciated but still in use: Warning, allow with approval
- **E2.** Outstanding loans on asset: System blocks, shows "Clear liabilities first"

### Business Rules
- BR-11: Liquidation requires management approval
- BR-12: Gain/loss = Selling Price - (Cost - Accumulated Depreciation) - Costs
- BR-13: Fully depreciated assets: No gain/loss (residual value only)

---

## UC-05: Revalue Fixed Asset

### Actor
Finance Manager / Government Authority

### Preconditions
- Government revaluation mandate issued
- Asset subject to revaluation

### Happy Path
1. Authority issues revaluation decision
2. Finance Manager enters revaluation:
   - Asset ID
   - New value
   - Revaluation date
   - Authority reference
3. System calculates adjustment
4. System updates:
   - Original cost
   - Accumulated depreciation
   - Residual value
5. System records revaluation difference
6. System creates audit trail

### Alternative Paths
- **3a.** Decrease in value: System creates impairment entry
- **4a.** Partial revaluation: System handles proportional adjustment

### Exception Paths
- **E1.** Revaluation not authorized: System rejects
- **E2.** Asset in liquidation: System blocks revaluation

### Business Rules
- BR-14: Revaluation only per government mandate
- BR-15: Adjustments per VAS 03 and regulations

---

## UC-06: Generate Asset Register Report

### Actor
Accountant / Auditor

### Preconditions
- Assets exist in system

### Happy Path
1. User navigates to Reports
2. User selects "Asset Register"
3. User sets filters:
   - Period
   - Category
   - Location
   - Department
4. System generates report:
   - Asset code, name
   - Original cost
   - Accumulated depreciation
   - Current value
   - Depreciation method
   - Useful life
5. User views report on screen
6. User exports to Excel/PDF

### Alternative Paths
- **4a.** No assets match filters: System shows "No data found"
- **6a.** Export fails: System shows error, allows retry

### Exception Paths
- **E1.** Report timeout: System shows "Report too large, refine filters"

### Business Rules
- BR-16: Report must comply with VAS 03 disclosure requirements

---

## UC-07: View Depreciation Schedule

### Actor
Accountant / Auditor

### Preconditions
- Asset exists with depreciation history

### Happy Path
1. User selects asset
2. User clicks "View Schedule"
3. System displays schedule:
   - Period
   - Depreciation amount
   - Accumulated depreciation
   - Book value
4. User views schedule
5. User exports to Excel

### Alternative Paths
- **3a.** No depreciation yet: System shows "No depreciation history"

### Exception Paths
- **E1.** Schedule calculation error: System shows error message

### Business Rules
- BR-17: Schedule must show monthly breakdown

---

## Process Flow: Monthly Depreciation Cycle

```
[Start of Month]
    │
    ▼
[1. Close Previous Month]
    │
    ▼
[2. Calculate Depreciation] ──► [For Each Active Asset]
    │                            │
    │                            ▼
    │                        [Get Asset Data]
    │                            │
    │                            ▼
    │                        [Calculate Amount]
    │                            │
    │                            ▼
    │                        [Create Entry]
    │
    ▼
[3. Review Entries]
    │
    ├──► [Approve] ──► [Post to GL]
    │                    │
    │                    ▼
    │                [Update Accumulated Depreciation]
    │                    │
    │                    ▼
    │                [Update Book Value]
    │
    └──► [Reject] ──► [Clear Entries]
                       │
                       ▼
                   [Recalculate]
```

---

## Process Flow: Asset Lifecycle

```
[Asset Acquisition]
    │
    ▼
[Create Asset Card] ──► [State: ACTIVE]
    │
    ├──► [Monthly Depreciation] ──► [Continuing]
    │
    ├──► [Transfer] ──► [Update Location]
    │
    ├──► [Upgrade/Enhancement] ──► [Adjust Cost]
    │
    ├──► [Revaluation] ──► [Adjust Values]
    │
    └──► [Liquidation Request]
              │
              ▼
         [State: PENDING_LIQUIDATION]
              │
              ├──► [Approve] ──► [Execute] ──► [State: SCRAPPED/SOLD]
              │
              └──► [Reject] ──► [State: ACTIVE]
```

---

*Use cases complete.*
