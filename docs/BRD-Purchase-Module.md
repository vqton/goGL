# Business Requirements Document: Purchase Module

## Version History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-08-30 | BA Lead + Chief Accountant | Initial BRD |

---

## Executive Summary

### Current State Assessment
**PRODUCTION READINESS: ❌ NOT READY**

The current purchase module is a **stub implementation** with:
- Only `PurchaseInvoice` entity (6 fields vs. 30+ required)
- No service logic (returns `ErrNotImplemented`)
- No repository logic (returns `ErrNotImplemented`)
- No HTTP endpoints (returns 501)

**Verdict:** The module cannot operate in PROD. A complete rewrite is required.

---

## 1. Business Context

### 1.1 Vietnamese Accounting Compliance Requirements
Based on research from official sources (GDT, MOF, VACPA, VAA):

| Regulation | Requirement | Status in Current Module |
|------------|-------------|--------------------------|
| **Thông tư 99/2025/TT-BTC** | Enterprise accounting regime effective 01/01/2026 | ❌ Missing GL accounts (331, 152, 133) |
| **Decree 123/2020/NĐ-CP** | E-invoice mandatory since 01/07/2022 | ❌ No e-invoice tracking |
| **Decree 70/2025/NĐ-CP** | E-invoice amendments effective 01/06/2025 | ❌ Not implemented |
| **Circular 32/2025/TT-BTC** | E-invoice guidance replacing Circular 78/2021 | ❌ Not implemented |
| **Luật Kế toán 88/2015** | Accounting law requirements | ❌ Missing audit trail |
| **Decree 123/2020/NĐ-CP** | Invoice types: VAT, Sales, Export | ❌ No invoice type handling |

### 1.2 Vietnamese Accounting GL Accounts Required

| Account | Name | Purpose |
|---------|------|---------|
| **152** | Nguyên liệu, vật liệu | Raw materials inventory |
| **153** | Công cụ, dụng cụ | Tools and equipment |
| **155** | Sản phẩm dở dang | Work-in-progress |
| **156** | Hàng hóa | Goods for resale |
| **211** | Tài sản cố định hữu hình | Fixed assets |
| **331** | Phải trả người bán | Accounts payable |
| **133** | Thuế GTGT được khấu trừ | Input VAT deductible |
| **632** | Giá vốn hàng bán | Cost of goods sold |
| **511** | Doanh thu bán hàng | Sales revenue |
| **3331** | Thuế GTGT phải nộp | VAT payable |

### 1.3 ERP Comparison (Misa, Fast, Bravo)

| Feature | Misa SME | Fast Accounting | Bravo ERP | Our Module |
|---------|----------|-----------------|-----------|------------|
| Purchase Order | ✅ | ✅ | ✅ | ❌ |
| Goods Receipt | ✅ | ✅ | ✅ | ❌ |
| Purchase Invoice | ✅ | ✅ | ✅ | ❌ (stub) |
| E-Invoice Integration | ✅ | ✅ | ✅ | ❌ |
| VAT Declaration | ✅ | ✅ | ✅ | ❌ |
| Supplier Management | ✅ | ✅ | ✅ | ❌ |
| Multi-currency | ✅ | ✅ | ✅ | ❌ |
| Import/Export | ✅ | ✅ | ✅ | ❌ |
| Customs Declaration | ✅ | ✅ | ✅ | ❌ |
| Payment Processing | ✅ | ✅ | ✅ | ❌ |

---

## 2. Functional Requirements

### 2.1 Purchase Order (Đơn đặt hàng mua)
- **Purpose:** Record purchase intent before goods arrive
- **Status Flow:** Draft → Confirmed → Partial Received → Received → Completed / Cancelled
- **Required Fields:**
  - PO number (auto-generated: PO-00001)
  - Supplier code, name, address
  - Order date, expected delivery date
  - Lines: Item code, quantity, unit price, amount, VAT
  - Payment terms
  - Delivery address
  - Notes

### 2.2 Goods Receipt (Phiếu nhập kho)
- **Purpose:** Record physical receipt of goods
- **Status Flow:** Draft → Approved → Completed
- **Required Fields:**
  - Receipt number (auto-generated: NK-00001)
  - PO reference
  - Supplier code, name
  - Receipt date
  - Warehouse code
  - Lines: Item code, quantity received, quantity accepted, quantity rejected
  - Quality notes
  - Inspector signature reference

### 2.3 Purchase Invoice (Hóa đơn mua hàng)
- **Purpose:** Record supplier's invoice for accounting
- **Status Flow:** Draft → Pending E-Invoice → Validated → Posted → Paid → Reconciled
- **Required Fields:**
  - Invoice number (auto-generated: MH-00001)
  - Supplier invoice number (from supplier)
  - PO reference
  - Goods receipt reference
  - Supplier code, name, tax code, address
  - Invoice date, due date
  - Lines: Item code, quantity, unit price, discount, amount, VAT
  - Sub total, VAT amount, total
  - E-invoice serial, number, date
  - Payment status
  - GL posting status

### 2.4 Supplier Management (Nhà cung cấp)
- **Purpose:** Maintain supplier master data
- **Required Fields:**
  - Supplier code (auto-generated: NCC-00001)
  - Supplier name, tax code
  - Address, phone, email, contact person
  - Payment terms
  - Credit limit
  - Bank account details
  - Status (active/inactive)

### 2.5 Payment Processing (Thanh toán)
- **Purpose:** Track payments to suppliers
- **Status Flow:** Draft → Approved → Processed → Reconciled
- **Required Fields:**
  - Payment number (auto-generated: TT-00001)
  - Supplier code, name
  - Payment date, method (cash/bank/cheque)
  - Bank account, cheque number
  - Amount paid, currency, exchange rate
  - Applied to invoices (which invoices paid)

---

## 3. Non-Functional Requirements

### 3.1 Performance
- Response time < 200ms for CRUD operations
- Support 100+ concurrent users
- List queries with pagination (max 200 per page)

### 3.2 Security
- Role-based access control (purchase clerk, purchase manager, accountant, CFO)
- Audit trail for all transactions
- Approval workflows for payments > threshold

### 3.3 Compliance
- All transactions must comply with Thông tư 99/2025/TT-BTC
- E-invoice integration via GDT portal
- VAT declaration support
- Export to CSV/Excel for tax filing

---

## 4. Use Cases

### UC-001: Create Purchase Order
- **Actor:** Purchase Clerk
- **Precondition:** Supplier exists in system
- **Happy Path:**
  1. Navigate to Purchase Orders
  2. Click "New Order"
  3. Select supplier
  4. Add line items (quantity, unit price)
  5. Save order
  6. System generates PO number
  7. System calculates total and VAT
- **Alternative Path:**
  - 3a. Supplier not found → Create new supplier first
- **Exception Path:**
  - 4a. Insufficient stock for existing items → Warning displayed
  - 5a. Invalid data → Validation error shown

### UC-002: Receive Goods
- **Actor:** Warehouse Staff
- **Precondition:** PO exists and is Confirmed
- **Happy Path:**
  1. Navigate to Goods Receipts
  2. Select PO reference
  3. System loads PO lines
  4. Enter received quantities
  5. Enter accepted/rejected quantities
  6. Save receipt
  7. System updates PO received quantities
  8. System updates inventory
- **Alternative Path:**
  - 4a. Partial delivery → System allows partial receipt
- **Exception Path:**
  - 6a. Quantity exceeds PO → Error "Exceeds ordered quantity"

### UC-003: Process Purchase Invoice
- **Actor:** Accountant
- **Precondition:** Goods received, supplier invoice received
- **Happy Path:**
  1. Navigate to Purchase Invoices
  2. Click "New Invoice"
  3. Select supplier
  4. Enter supplier invoice number and date
  5. Add line items matching supplier invoice
  6. Save invoice
  7. System generates internal invoice number
  8. System validates VAT amounts
- **Alternative Path:**
  - 5a. Link to PO/GR → System auto-fills from PO/GR
- **Exception Path:**
  - 5a. VAT mismatch → Error "VAT calculation invalid"
  - 7a. Duplicate supplier invoice → Warning

### UC-004: Post to General Ledger
- **Actor:** Accountant
- **Precondition:** Invoice validated
- **Happy Path:**
  1. Select invoice
  2. Click "Post to GL"
  3. System generates journal entries:
     - Debit: 331 (Accounts Payable)
     - Debit: 133 (Input VAT)
     - Credit: 152/156 (Inventory/Goods)
  4. System marks invoice as Posted
- **Exception Path:**
  - 3a. GL period closed → Error "Period closed"

### UC-005: Process Payment
- **Actor:** Finance Manager
- **Precondition:** Invoice Posted, payment approved
- **Happy Path:**
  1. Select invoice(s) to pay
  2. Enter payment details (date, method, amount)
  3. Save payment
  4. System generates payment number
  5. System updates invoice payment status
  6. System generates journal entries:
     - Debit: 331 (Accounts Payable)
     - Credit: 111/112 (Cash/Bank)
- **Exception Path:**
  - 3a. Amount > approved budget → Error
  - 5a. Payment exceeds invoice amount → Error

---

## 5. Data Model

### 5.1 PurchaseOrder
```json
{
  "id": "string (SHA-256 hash)",
  "ref_no": "PO-00001",
  "supplier_code": "NCC-00001",
  "supplier_name": "ABC Company",
  "supplier_address": "123 Le Loi, HCMC",
  "order_date": "2026-08-30",
  "expected_delivery_date": "2026-09-15",
  "status": "draft|confirmed|partial_received|received|completed|cancelled",
  "lines": [
    {
      "line_no": 1,
      "item_code": "SP-001",
      "item_name": "Widget A",
      "unit": "pcs",
      "quantity": 100,
      "received_qty": 0,
      "unit_price": 500000,
      "discount": 0,
      "amount": 50000000,
      "vat_rate": 10,
      "vat_amount": 5000000,
      "total_amount": 55000000
    }
  ],
  "sub_total": {"amount_minor": 50000000, "currency": "VND"},
  "vat_amount": {"amount_minor": 5000000, "currency": "VND"},
  "total_amount": {"amount_minor": 55000000, "currency": "VND"},
  "payment_terms": "Net 30",
  "delivery_address": "456 Warehouse, HCMC",
  "notes": "Urgent order",
  "created_by": "user1",
  "created_at": "2026-08-30T10:00:00Z",
  "updated_by": "user1",
  "updated_at": "2026-08-30T10:00:00Z"
}
```

### 5.2 GoodsReceipt
```json
{
  "id": "string",
  "ref_no": "NK-00001",
  "po_id": "po-123",
  "po_ref_no": "PO-00001",
  "supplier_code": "NCC-00001",
  "supplier_name": "ABC Company",
  "warehouse_code": "KHO-001",
  "receipt_date": "2026-09-10",
  "status": "draft|approved|completed",
  "lines": [
    {
      "line_no": 1,
      "po_line_no": 1,
      "item_code": "SP-001",
      "item_name": "Widget A",
      "unit": "pcs",
      "quantity_ordered": 100,
      "quantity_received": 100,
      "quantity_accepted": 95,
      "quantity_rejected": 5,
      "unit_price": 500000,
      "amount": 47500000,
      "vat_rate": 10,
      "vat_amount": 4750000,
      "total_amount": 52250000
    }
  ],
  "inspection_notes": "5 units damaged",
  "inspector": "warehouse1",
  "created_by": "user1",
  "created_at": "2026-09-10T08:00:00Z"
}
```

### 5.3 PurchaseInvoice
```json
{
  "id": "string",
  "ref_no": "MH-00001",
  "supplier_invoice_no": "INV-2026-001",
  "po_id": "po-123",
  "po_ref_no": "PO-00001",
  "goods_receipt_id": "gr-456",
  "goods_receipt_ref_no": "NK-00001",
  "supplier_code": "NCC-00001",
  "supplier_name": "ABC Company",
  "supplier_tax_code": "0123456789",
  "supplier_address": "123 Le Loi, HCMC",
  "invoice_date": "2026-09-10",
  "due_date": "2026-10-10",
  "status": "draft|pending_einvoice|validated|posted|paid|reconciled",
  "lines": [
    {
      "line_no": 1,
      "item_code": "SP-001",
      "item_name": "Widget A",
      "unit": "pcs",
      "quantity": 95,
      "unit_price": 500000,
      "discount": 0,
      "amount": 47500000,
      "vat_rate": 10,
      "vat_amount": 4750000,
      "total_amount": 52250000,
      "gl_account": "152"
    }
  ],
  "sub_total": {"amount_minor": 47500000, "currency": "VND"},
  "vat_amount": {"amount_minor": 4750000, "currency": "VND"},
  "total_amount": {"amount_minor": 52250000, "currency": "VND"},
  "e_invoice_serial": "C25SAA",
  "e_invoice_number": "12345",
  "e_invoice_date": "2026-09-10",
  "e_invoice_status": "none|pending|accepted|rejected|cancelled",
  "paid_amount": {"amount_minor": 0, "currency": "VND"},
  "outstanding_amount": {"amount_minor": 52250000, "currency": "VND"},
  "gl_posted": false,
  "gl_reference": "",
  "notes": "",
  "created_by": "user1",
  "created_at": "2026-09-10T09:00:00Z",
  "updated_by": "user1",
  "updated_at": "2026-09-10T09:00:00Z"
}
```

### 5.4 Supplier
```json
{
  "id": "string",
  "ref_no": "NCC-00001",
  "name": "ABC Company",
  "tax_code": "0123456789",
  "address": "123 Le Loi, HCMC",
  "phone": "028-12345678",
  "email": "contact@abc.com",
  "contact_person": "Mr. Smith",
  "payment_terms": "Net 30",
  "credit_limit": {"amount_minor": 1000000000, "currency": "VND"},
  "bank_account": "1234567890",
  "bank_name": "Vietcombank",
  "status": "active|inactive",
  "created_by": "user1",
  "created_at": "2026-08-30T10:00:00Z"
}
```

### 5.5 Payment
```json
{
  "id": "string",
  "ref_no": "TT-00001",
  "supplier_code": "NCC-00001",
  "supplier_name": "ABC Company",
  "payment_date": "2026-10-10",
  "payment_method": "bank_transfer|cash|cheque",
  "bank_account": "1234567890",
  "bank_name": "Vietcombank",
  "cheque_number": "",
  "amount": {"amount_minor": 52250000, "currency": "VND"},
  "applied_invoices": [
    {
      "invoice_id": "inv-789",
      "invoice_ref_no": "MH-00001",
      "amount_applied": 52250000
    }
  ],
  "status": "draft|approved|processed|reconciled",
  "approved_by": "manager1",
  "created_by": "user1",
  "created_at": "2026-10-10T10:00:00Z"
}
```

---

## 6. API Endpoints

### 6.1 Purchase Orders
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/purchase/orders` | Create purchase order |
| GET | `/api/v1/purchase/orders` | List orders |
| GET | `/api/v1/purchase/orders/:id` | Get order |
| PUT | `/api/v1/purchase/orders/:id` | Update order |
| DELETE | `/api/v1/purchase/orders/:id` | Delete order |
| POST | `/api/v1/purchase/orders/:id/confirm` | Confirm order |
| POST | `/api/v1/purchase/orders/:id/cancel` | Cancel order |

### 6.2 Goods Receipts
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/purchase/receipts` | Create goods receipt |
| GET | `/api/v1/purchase/receipts` | List receipts |
| GET | `/api/v1/purchase/receipts/:id` | Get receipt |
| POST | `/api/v1/purchase/receipts/:id/approve` | Approve receipt |

### 6.3 Purchase Invoices
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/purchase/invoices` | Create purchase invoice |
| GET | `/api/v1/purchase/invoices` | List invoices |
| GET | `/api/v1/purchase/invoices/:id` | Get invoice |
| PUT | `/api/v1/purchase/invoices/:id` | Update invoice |
| DELETE | `/api/v1/purchase/invoices/:id` | Delete invoice |
| POST | `/api/v1/purchase/invoices/:id/post` | Post to GL |
| POST | `/api/v1/purchase/invoices/:id/void` | Void invoice |

### 6.4 Suppliers
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/purchase/suppliers` | Create supplier |
| GET | `/api/v1/purchase/suppliers` | List suppliers |
| GET | `/api/v1/purchase/suppliers/:id` | Get supplier |
| PUT | `/api/v1/purchase/suppliers/:id` | Update supplier |
| DELETE | `/api/v1/purchase/suppliers/:id` | Delete supplier |

### 6.5 Payments
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/purchase/payments` | Create payment |
| GET | `/api/v1/purchase/payments` | List payments |
| GET | `/api/v1/purchase/payments/:id` | Get payment |
| POST | `/api/v1/purchase/payments/:id/approve` | Approve payment |
| POST | `/api/v1/purchase/payments/:id/process` | Process payment |

---

## 7. Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
- [ ] Create Supplier entity with full fields
- [ ] Create Supplier Repository and Service
- [ ] Create Supplier Handler (CRUD)
- [ ] Add DB migrations
- [ ] Write entity tests

### Phase 2: Purchase Orders (Week 3-4)
- [ ] Create PurchaseOrder entity
- [ ] Create PurchaseOrder Repository and Service
- [ ] Create PurchaseOrder Handler
- [ ] Implement status workflow
- [ ] Write tests

### Phase 3: Goods Receipts (Week 5-6)
- [ ] Create GoodsReceipt entity
- [ ] Create GoodsReceipt Repository and Service
- [ ] Create GoodsReceipt Handler
- [ ] Implement PO received quantity tracking
- [ ] Write tests

### Phase 4: Purchase Invoices (Week 7-9)
- [ ] Create PurchaseInvoice entity
- [ ] Create PurchaseInvoice Repository and Service
- [ ] Create PurchaseInvoice Handler
- [ ] Implement GL posting logic
- [ ] Implement VAT calculation
- [ ] Write tests

### Phase 5: Payments (Week 10-11)
- [ ] Create Payment entity
- [ ] Create Payment Repository and Service
- [ ] Create Payment Handler
- [ ] Implement payment application
- [ ] Write tests

### Phase 6: Integration (Week 12)
- [ ] Wire all modules in main.go
- [ ] End-to-end testing
- [ ] Code review
- [ ] Documentation

---

## 8. Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| VAT calculation errors | High - Tax penalties | Implement validation per Thông tư 99/2025 |
| Missing GL accounts | High - Audit findings | Use official account list from TT99 |
| No e-invoice support | High - Non-compliance | Implement GDT integration |
| No audit trail | Medium - Compliance | Add audit logging |
| No multi-currency | Medium - FDI companies | Implement currency support |

---

## 9. Success Criteria

- [ ] All 18 API endpoints implemented
- [ ] 100+ tests passing
- [ ] VAT calculation per Thông tư 99/2025
- [ ] GL posting with correct journal entries
- [ ] E-invoice status tracking
- [ ] Code review approved
- [ ] Documentation complete
