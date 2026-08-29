# Use Cases - Tools/Equipment Module
## Công cụ, dụng cụ (CCDC)

**Version**: 1.0  
**Date**: 2026-08-29  
**Module**: Tools/Equipment

---

## UC-01: Create Tool Card
**Actor**: Inventory Clerk  
**Goal**: Create a new tool card in the system  
**Precondition**: User is logged in with create permission

### Happy Path
1. User navigates to Tool Card List
2. User clicks [+] button
3. System displays Create Tool Card form
4. User fills in required fields:
   - Name (required)
   - Category (required)
   - Original Cost (required, < 30M VND)
   - Quantity (required, > 0)
   - Unit (required)
   - Purchase Date (required)
5. User clicks [Save]
6. System validates data
7. System generates auto code (TL-XXXXX)
8. System saves tool card
9. System displays success message
10. System redirects to Tool Card Detail

### Alternative Paths
- **4a. User selects category from dropdown**: System populates sub-category options
- **4b. User uploads attachment**: System stores file reference

### Exception Paths
- **6a. Validation fails**: System highlights errors, user corrects
- **6b. Name already exists**: System warns, user confirms or changes
- **6c. Cost ≥ 30M VND**: System blocks, suggests Fixed Asset module

---

## UC-02: Import Tool to Warehouse
**Actor**: Warehouse Staff  
**Goal**: Record incoming tools to inventory  
**Precondition**: Tool card exists, user has import permission

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Import]
3. System displays Import form with current stock
4. User enters:
   - Quantity (required, > 0)
   - Unit Price (required, > 0)
   - Reference No (invoice/voucher)
   - Notes (optional)
5. User clicks [Confirm Import]
6. System validates:
   - Quantity > 0
   - Unit Price > 0
7. System creates import transaction
8. System posts GL entry:
   - Dr 153 - Tools
   - Dr 133 - VAT (if applicable)
   - Cr 331 - AP (or Cr 111/112 for cash)
9. System updates tool card quantity
10. System displays success message

### Alternative Paths
- **4a. Multiple items**: User adds line items
- **8a. Cash purchase**: Cr 111/112 instead of 331

### Exception Paths
- **6a. Invalid quantity**: System shows error
- **6b. Invalid price**: System shows error
- **7a. GL posting fails**: System rolls back, shows error

---

## UC-03: Export Tool from Warehouse
**Actor**: Warehouse Staff  
**Goal**: Record outgoing tools from inventory  
**Precondition**: Tool card exists with sufficient stock

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Export]
3. System displays Export form with current stock
4. User enters:
   - Quantity (required, ≤ stock)
   - To Department (required)
   - To Person (required)
   - Notes (optional)
5. User clicks [Confirm Export]
6. System validates:
   - Quantity ≤ available stock
7. System creates export transaction
8. System posts GL entry:
   - Dr 623/627/641/642 (based on department)
   - Cr 153 - Tools
9. System updates tool card quantity
10. System displays success message

### Alternative Paths
- **4a. Export to another warehouse**: System records transfer
- **8a. Multi-period allocation**: Dr 242 instead of expense account

### Exception Paths
- **6a. Insufficient stock**: System shows error, prevents export
- **6b. Tool not in stock**: System shows error

---

## UC-04: Transfer Tool Between Locations
**Actor**: Inventory Clerk  
**Goal**: Move tools between warehouses/departments  
**Precondition**: Tool card exists with sufficient stock

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Transfer]
3. System displays Transfer form
4. User enters:
   - Quantity (required, ≤ stock)
   - From Location (auto-filled)
   - To Location (required)
   - From Department (auto-filled)
   - To Department (required)
   - Notes (optional)
5. User clicks [Confirm Transfer]
6. System validates:
   - Quantity ≤ available stock
   - To Location ≠ From Location
7. System creates transfer transaction
8. System updates location information
9. System displays success message

### Alternative Paths
- **4a. Transfer to person**: User enters assignee
- **8a. Inter-department transfer**: System tracks both locations

### Exception Paths
- **6a. Insufficient stock**: System shows error
- **6b. Same location**: System shows error

---

## UC-05: Return Tool to Supplier
**Actor**: Warehouse Staff  
**Goal**: Return defective/unused tools to supplier  
**Precondition**: Tool card exists, return authorized

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Return]
3. System displays Return form
4. User enters:
   - Quantity (required, ≤ stock)
   - Return Reason (required)
   - Reference No (return authorization)
   - Notes (optional)
5. User clicks [Confirm Return]
6. System validates:
   - Quantity ≤ available stock
7. System creates return transaction
8. System posts GL entry:
   - Dr 331 - AP (or Dr 111/112 for cash refund)
   - Cr 153 - Tools
   - Cr 133 - VAT (if applicable)
9. System updates tool card quantity
10. System displays success message

### Alternative Paths
- **8a. Cash refund**: Dr 111/112 instead of 331
- **8b. Credit note**: System records credit note reference

### Exception Paths
- **6a. Insufficient stock**: System shows error
- **7a. Return not authorized**: System blocks

---

## UC-06: Dispose/Sell Tool
**Actor**: Accountant  
**Goal**: Dispose or sell tools from inventory  
**Precondition**: Tool card exists, disposal authorized

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Dispose]
3. System displays Disposal form
4. User enters:
   - Quantity (required, ≤ stock)
   - Disposal Type (sell/donate/scrap)
   - Sale Price (if selling)
   - Buyer (if selling)
   - Reason (required)
   - Notes (optional)
5. User clicks [Confirm Disposal]
6. System validates:
   - Quantity ≤ available stock
7. System creates disposal transaction
8. System posts GL entry:
   - Dr 632 - Cost of Goods Sold
   - Cr 153 - Tools
   - (If sold) Dr 111/131 / Cr 511 - Revenue
9. System updates tool card state to "disposed"
10. System displays success message

### Alternative Paths
- **8a. Donation**: Only cost entry, no revenue
- **8b. Scrap**: No revenue entry

### Exception Paths
- **6a. Insufficient stock**: System shows error
- **7a. Disposal not authorized**: System blocks

---

## UC-07: Adjust Inventory (Physical Count)
**Actor**: Accountant  
**Goal**: Adjust inventory after physical count  
**Precondition**: Physical count completed

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Adjust Inventory]
3. System displays current book quantity
4. User enters:
   - Physical Count (required, ≥ 0)
   - Adjustment Reason (required)
5. System calculates difference:
   - Surplus: Physical > Book
   - Shortage: Physical < Book
6. User clicks [Confirm Adjustment]
7. System creates adjustment transaction
8. System posts GL entry:
   - (Surplus) Dr 153 / Cr 3381
   - (Shortage) Dr 511 / Cr 153
9. System updates tool card quantity
10. System displays success message

### Alternative Paths
- **8a. Write-off**: Dr 632 / Cr 153 for damaged items
- **8b. Investigation pending**: Dr 1381 / Cr 153

### Exception Paths
- **6a. Negative quantity**: System shows error
- **7a. Large variance**: System requires approval

---

## UC-08: View Transaction History
**Actor**: Any User  
**Goal**: View transaction history for a tool  
**Precondition**: Tool card exists

### Happy Path
1. User opens Tool Card Detail
2. User clicks [Transactions]
3. System displays transaction list with filters:
   - Date range
   - Transaction type
4. User applies filters (optional)
5. System displays filtered results
6. User clicks on transaction to view details

### Alternative Paths
- **4a. Export to Excel**: System generates file
- **5a. Print**: System generates printable view

### Exception Paths
- **3a. No transactions**: System shows empty state

---

## UC-09: Generate Reports
**Actor**: Accountant/Manager  
**Goal**: Generate inventory reports  
**Precondition**: None

### Happy Path
1. User navigates to Reports
2. User selects report type:
   - Inventory List
   - Transaction Log
   - GL Summary
   - Stock Balance
3. User applies filters (date range, category, etc.)
4. System generates report
5. User views/prints/exports report

### Alternative Paths
- **4a. Export to Excel**: System generates file
- **4b. Export to PDF**: System generates PDF

### Exception Paths
- **4a. No data**: System shows empty report

---

## Appendix: GL Account Reference

| Transaction | Debit | Credit |
|-------------|-------|--------|
| Import (on credit) | 153, 133 | 331 |
| Import (cash) | 153, 133 | 111/112 |
| Export to production | 623/627 | 153 |
| Export to management | 641 | 153 |
| Export to sales | 642 | 153 |
| Multi-period (initial) | 242 | 153 |
| Multi-period (allocation) | 623/627/641/642 | 242 |
| Return to supplier (credit) | 331 | 153, 133 |
| Return to supplier (cash) | 111/112 | 153, 133 |
| Disposal cost | 632 | 153 |
| Sale revenue | 111/131 | 511 |
| Inventory surplus | 153 | 3381 |
| Inventory shortage | 511 | 153 |

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
