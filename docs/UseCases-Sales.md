# Use Cases - Sales Module
## Bán hàng (Sales)

**Version**: 1.0
**Date**: 2026-08-29
**Module**: Sales

---

## UC-01: Create Sales Quotation
**Actor**: Sales Staff
**Goal**: Create a new sales quotation for a customer
**Precondition**: User is logged in with create permission

### Happy Path
1. User navigates to Sales Quotations
2. User clicks [+] button
3. System displays Create Quotation form
4. User selects customer from dropdown
5. System auto-fills customer details (address, tax code)
6. User adds line items:
   - Item code (searchable dropdown)
   - Quantity
   - Unit price (auto-filled from price list)
   - Discount % (optional)
7. System calculates line totals, subtotal, VAT, total
8. User sets validity period
9. User adds notes (optional)
10. User clicks [Save]
11. System validates data
12. System generates auto code (BC-XXXXX)
13. System saves quotation (status: draft)
14. System displays success message

### Alternative Paths
- **5a. Customer not in system**: User creates new customer
- **7a. Price list not found**: User enters price manually

### Exception Paths
- **6a. Invalid quantity**: System shows error
- **6b. Invalid price**: System shows error
- **11a. Validation fails**: System highlights errors, user corrects

---

## UC-02: Send Quotation to Customer
**Actor**: Sales Staff
**Goal**: Send quotation to customer via email
**Precondition**: Quotation exists in draft status

### Happy Path
1. User opens Quotation Detail
2. User clicks [Send]
3. System displays Send Quotation dialog
4. User enters recipient email
5. User clicks [Confirm Send]
6. System generates quotation PDF
7. System sends email to customer
8. System updates quotation status to "sent"
9. System displays success message

### Alternative Paths
- **4a. Multiple recipients**: User enters multiple emails
- **6a. PDF preview**: User previews PDF before sending

### Exception Paths
- **7a. Email delivery fails**: System shows error, user retries

---

## UC-03: Accept Quotation & Convert to Order
**Actor**: Sales Manager
**Goal**: Accept quotation and create sales order
**Precondition**: Quotation exists in "sent" status

### Happy Path
1. User opens Quotation Detail
2. User clicks [Accept]
3. System displays Accept Quotation dialog
4. User confirms acceptance
5. System creates Sales Order from quotation
6. System copies quotation lines to order
7. System updates quotation status to "accepted"
8. System sets quotation.convertedToOrder = true
9. System links quotation to order
10. System displays success message with order link

### Alternative Paths
- **4a. Modify before accept**: User modifies order before confirming
- **5a. Partial acceptance**: User accepts selected lines only

### Exception Paths
- **5a. Order creation fails**: System rolls back, shows error

---

## UC-04: Create Sales Order
**Actor**: Sales Staff
**Goal**: Create a new sales order
**Precondition**: User is logged in with create permission

### Happy Path
1. User navigates to Sales Orders
2. User clicks [+] button
3. System displays Create Order form
4. User selects customer from dropdown
5. User adds line items
6. User sets delivery date
7. User enters delivery address
8. User enters payment terms
9. User clicks [Save]
10. System validates data
11. System generates auto code (DH-XXXXX)
12. System saves order (status: draft)
13. System displays success message

### Alternative Paths
- **4a. From quotation**: User selects "Convert from Quotation"
- **8a. Default payment terms**: System applies default terms

### Exception Paths
- **10a. Validation fails**: System highlights errors

---

## UC-05: Confirm Sales Order
**Actor**: Sales Manager
**Goal**: Confirm sales order for fulfillment
**Precondition**: Order exists in draft status

### Happy Path
1. User opens Order Detail
2. User clicks [Confirm]
3. System validates order:
   - All required fields present
   - Delivery date valid
   - Customer credit limit OK
4. System updates order status to "confirmed"
5. System notifies warehouse for fulfillment
6. System displays success message

### Alternative Paths
- **3a. Credit limit exceeded**: System warns, user overrides or cancels

### Exception Paths
- **3a. Validation fails**: System shows error

---

## UC-06: Record Delivery
**Actor**: Warehouse Staff
**Goal**: Record delivery of goods to customer
**Precondition**: Order is confirmed, goods are picked

### Happy Path
1. User opens Order Detail
2. User clicks [Deliver]
3. System displays Delivery form
4. User selects line items to deliver
5. User enters quantity per item
6. User clicks [Confirm Delivery]
7. System validates:
   - Quantity ≤ remaining to deliver
8. System creates delivery record
9. System updates order line delivered quantities
10. System updates order delivery status
11. System updates delivered amount
12. If all lines fully delivered: status = "delivered"
13. If partial: status = "partial_delivery"
14. System displays success message

### Alternative Paths
- **5a. Partial delivery**: Only some items delivered
- **12a. Full delivery**: All items delivered

### Exception Paths
- **7a. Insufficient stock**: System shows error
- **7b. Quantity exceeds remaining**: System shows error

---

## UC-07: Create Sales Invoice
**Actor**: Sales Staff
**Goal**: Create invoice for delivered goods
**Precondition**: Order has been delivered (fully or partially)

### Happy Path
1. User navigates to Sales Invoices
2. User clicks [+] button
3. System displays Create Invoice form
4. User selects customer
5. User selects source order (optional)
6. System auto-fills undelivered items
7. User adds/modifies line items
8. System calculates totals
9. User reviews GL entry preview
10. User clicks [Save Draft]
11. System validates data
12. System generates auto code (HD-XXXXX)
13. System saves invoice (status: draft)
14. System displays success message

### Alternative Paths
- **5a. Direct invoice**: No source order, manual entry
- **6a. From order**: System copies delivered items

### Exception Paths
- **11a. Validation fails**: System highlights errors

---

## UC-08: Issue E-Invoice
**Actor**: Sales Staff
**Goal**: Issue e-invoice via GDT system
**Precondition**: Invoice exists in draft/pending status

### Happy Path
1. User opens Invoice Detail
2. User clicks [Issue E-Invoice]
3. System validates invoice:
   - All required fields present
   - Customer tax code valid
   - Line items complete
4. System submits invoice to GDT
5. System receives acceptance from GDT
6. System receives serial + number from GDT
7. System updates invoice:
   - status = "issued"
   - e_invoice_serial = "1C25TCH"
   - e_invoice_number = "000001"
   - e_invoice_date = now
   - e_invoice_status = "accepted"
8. System posts GL entry:
   - Dr 131 - Receivable
   - Cr 511 - Revenue
   - Cr 333 - VAT Payable
9. System updates invoice GL fields
10. System displays success message

### Alternative Paths
- **5a. GDT rejection**: System shows error, user corrects
- **8a. GL posting fails**: System rolls back e-invoice

### Exception Paths
- **4a. GDT connection fails**: System shows error, user retries
- **5a. GDT rejects invoice**: System shows rejection reason

---

## UC-09: Void Invoice
**Actor**: Sales Manager
**Goal**: Void an issued invoice
**Precondition**: Invoice exists, requires chief accountant approval

### Happy Path
1. User opens Invoice Detail
2. User clicks [Void]
3. System displays Void Invoice dialog
4. User enters void reason
5. System requires chief accountant approval
6. Chief accountant approves
7. System issues credit note (e-invoice)
8. System posts GL reversal:
   - Dr 511 - Revenue (reversal)
   - Dr 333 - VAT Payable (reversal)
   - Cr 131 - Receivable (reversal)
9. System updates invoice status to "voided"
10. System displays success message

### Alternative Paths
- **5a. Approval rejected**: System cancels void request

### Exception Paths
- **7a. Credit note issuance fails**: System shows error

---

## UC-10: Process Sales Return
**Actor**: Sales Staff
**Goal**: Process return of goods from customer
**Precondition**: Invoice exists and is issued

### Happy Path
1. User opens Invoice Detail
2. User clicks [Return]
3. System displays Return form
4. User selects items to return
5. User enters return quantity (≤ original)
6. User selects return reason
7. User clicks [Create Return]
8. System validates:
   - Return quantity ≤ invoice quantity
   - Items are returnable
9. System creates return (status: draft)
10. System generates auto code (PH-XXXXX)
11. Manager approves return
12. Warehouse receives returned goods
13. System issues credit note (e-invoice)
14. System posts GL reversal:
    - Dr 511 - Revenue (reversal)
    - Dr 333 - VAT Payable (reversal)
    - Cr 131 - Receivable (reversal)
15. System re-enters goods to inventory:
    - Dr 156 - Finished Goods
    - Cr 632 - COGS
16. System displays success message

### Alternative Paths
- **5a. Partial return**: Only some items returned
- **14a. Cash refund**: Cr 111/112 instead of Cr 131

### Exception Paths
- **8a. Return quantity exceeds original**: System shows error
- **11a. Approval rejected**: System cancels return

---

## UC-11: View Customer Balance
**Actor**: Sales Staff
**Goal**: View customer outstanding balance
**Precondition**: None

### Happy Path
1. User navigates to Customer Balance
2. User enters customer code
3. System queries customer invoices
4. System calculates:
   - Total invoiced amount
   - Total paid amount
   - Outstanding balance
5. System displays balance details:
   - Invoice list with amounts
   - Payment history
   - Outstanding total

### Alternative Paths
- **2a. Customer not found**: System shows error

### Exception Paths
- **3a. No invoices**: System shows zero balance

---

## UC-12: Generate Sales Reports
**Actor**: Sales Manager/Accountant
**Goal**: Generate sales analytics reports
**Precondition**: None

### Happy Path
1. User navigates to Reports
2. User selects report type:
   - Revenue by Customer
   - Revenue by Item
   - Revenue by Period
   - Outstanding Receivables
   - Sales Performance
3. User applies filters (date range, customer, item)
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
| Sale (credit) | 131 | 511, 333 |
| Sale (cash) | 111/112 | 511, 333 |
| COGS | 632 | 156, 152 |
| Return (credit note) | 511, 333 | 131 |
| Return (cash refund) | 511, 333 | 111/112 |
| Discount | 511 | 131 |
| Advance received | 111/112 | 1389 |
| Advance offset | 1389 | 131 |
| Void (reversal) | 511, 333 | 131 |

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
