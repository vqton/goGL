# Business Requirements Document (BRD)
## Tools/Equipment Module (Công cụ, dụng cụ - CCDC)

**Document ID**: BRD-TOOLS-001  
**Version**: 1.0  
**Date**: 2026-08-29  
**Author**: BA Lead (20+ years) + Chief Accountant (20+ years, CPA Vietnam)  
**Compliance**: Thông tư 99/2025/TT-BTC, Thông tư 45/2013/TT-BTC, VAS 03

---

## 1. Executive Summary

### 1.1 Purpose
This BRD defines the business requirements for a **Tools/Equipment (Công cụ, dụng cụ - CCDC)** management module that complies with Vietnamese accounting standards (Thông tư 99/2025/TT-BTC). The module tracks tools and equipment that **do not meet Fixed Asset (TSCĐ) criteria** but still require proper accounting and inventory management.

### 1.2 Business Context
According to **Thông tư 99/2025/TT-BTC** (effective from 01/01/2026):
- **Account 153** - Công cụ, dụng cụ: Tracks tools/equipment that don't meet TSCĐ criteria
- **TSCĐ criteria**: Original cost ≥ 30,000,000 VND AND useful life ≥ 12 months
- **CCDC**: Items below TSCĐ threshold, managed like inventory (nguyên liệu, vật liệu)

### 1.3 Scope
| In Scope | Out of Scope |
|----------|--------------|
| Tools/equipment < 30M VND | Fixed Assets (≥ 30M VND) - handled by `fixedasset` module |
| Inventory management (import/export/stock) | Depreciation calculation (not applicable for CCDC) |
| GL integration with Account 153 | Complex approval workflows |
| Basic audit trail | Web UI (Phase 2) |

---

## 2. Business Rules (Regulatory)

### BR-01: Definition of Tools/Equipment (CCDC)
**Source**: Thông tư 99/2025/TT-BTC, Phần B Phụ lục II

Tools/Equipment are labor tools that **do not meet TSCĐ criteria**:
- scaffolding, formwork, specialized assembly tools for construction
- replacement equipment and spare parts
- glass, ceramic, porcelain tools
- management tools, office supplies
- work clothing, work shoes

### BR-02: Accounting Treatment
**Source**: Thông tư 99/2025/TT-BTC, Tài khoản 153

| Aspect | Requirement |
|--------|-------------|
| **Account** | 153 - Công cụ, dụng cụ |
| **Valuation** | At cost (giá gốc) - similar to Account 152 |
| **Tracking** | By warehouse, category, group, individual item |
| **Physical + Value** | Must track both physical and value on detailed ledger |
| **High-value items** | Special preservation required for valuable/rare items |

### BR-03: GL Account Mapping
**Source**: Thông tư 99/2025/TT-BTC

| Account | Name | Purpose |
|---------|------|---------|
| **153** | Công cụ, dụng cụ | Tools/equipment inventory |
| **133** | Thuế GTGT được khấu trừ | Input VAT (if applicable) |
| **111** | Tiền mặt | Cash payment |
| **112** | Tiền gửi ngân hàng | Bank payment |
| **331** | Phải trả người bán | Accounts payable |
| **623** | Chi phí sản xuất | Production costs |
| **627** | Chi phí equipment | Equipment costs |
| **641** | Chi phí quản lý | Management costs |
| **642** | Chi phí bán hàng | Selling costs |

### BR-04: Transaction Types
**Source**: Thông tư 99/2025/TT-BTC

| Transaction | GL Entry |
|-------------|----------|
| **Purchase** | Dr 153, Dr 133 (VAT) / Cr 111/112/331 |
| **Issue to production** | Dr 623/627/641/642 / Cr 153 |
| **Multi-period allocation** | Dr 242 / Cr 153 (initial), then Dr 623/627/641/642 / Cr 242 (periodic) |
| **Return to supplier** | Dr 331 / Cr 153, Cr 133 (VAT) |
| **Disposal/sale** | Dr 632 (cost) / Cr 153; Dr 111/131 / Cr 511 (revenue) |
| **Inventory adjustment** | Dr 153 (surplus) / Cr 3381; Dr 511 / Cr 153 (shortage) |

### BR-05: Inventory Management
**Source**: Thông tư 99/2025/TT-BTC

- Detailed accounting by warehouse, category, group, individual item
- Track both physical quantity and value
- For tools issued for production/business/rental: track by location, rental object, responsible person
- High-value/rare items: special preservation methods required

### BR-06: Cost Allocation
**Source**: Thông tư 99/2025/TT-BTC

- **Single-period items**: Expense immediately to production costs
- **Multi-period items**: Record to Account 242 (deferred expenses), allocate periodically
- **Low-value items**: Can expense immediately to production costs

---

## 3. Functional Requirements

### 3.1 Core Entities

#### 3.1.1 Tool Card (Thẻ công cụ)
```go
type ToolCard struct {
    // Identification
    ID          string  // Unique identifier
    Code        string  // Auto-generated: TL-XXXXX
    Name        string  // Required
    
    // Classification
    Category    string  // scaffolding, formwork, tools, office_supplies, clothing, etc.
    SubCategory string  // Optional sub-category
    Description string  // Optional
    
    // Financial (VND)
    OriginalCost    int64  // Purchase price
    Quantity        int    // Default 1
    Unit            string // pcs, set, pair, etc.
    
    // Source documents
    PurchaseDate    string // YYYY-MM-DD
    InvoiceNo       string
    Supplier        string
    
    // Location & Assignment
    Warehouse       string // Warehouse location
    Department      string // Department code
    AssignedTo      string // Person responsible
    Location        string // Physical location
    
    // Status
    State           ToolCardState // active, inactive, disposed, damaged
    
    // GL Integration
    AccountCode153  string // Account 153 detail
    AccountCodeExp  string // Expense account (623/627/641/642)
    
    // Audit
    CreatedBy       string
    CreatedAt       string
    UpdatedBy       string
    UpdatedAt       string
}
```

#### 3.1.2 Tool Card States
```go
type ToolCardState string

const (
    StateActive     ToolCardState = "active"      // In use
    StateInactive   ToolCardState = "inactive"    // Not in use
    StateDisposed   ToolCardState = "disposed"    // Disposed/sold
    StateDamaged    ToolCardState = "damaged"     // Damaged
    StateInStorage  ToolCardState = "in_storage"  // In warehouse
)
```

#### 3.1.3 Transaction (Giao dịch)
```go
type ToolTransaction struct {
    ID              string
    ToolCardID      string
    TransactionType TransactionType
    Quantity        int
    UnitPrice       int64
    TotalAmount     int64
    FromLocation    string
    ToLocation      string
    FromDepartment  string
    ToDepartment    string
    AssignedTo      string
    ReferenceNo     string // Invoice, voucher number
    Notes           string
    CreatedBy       string
    CreatedAt       string
}

type TransactionType string

const (
    TxImport        TransactionType = "import"        // Nhập kho
    TxExport        TransactionType = "export"        // Xuất kho
    TxTransfer      TransactionType = "transfer"      // Điều chuyển
    TxReturn        TransactionType = "return"        // Trả lại NCC
    TxDisposal      TransactionType = "disposal"      // Thanh lý
    TxAdjustment    TransactionType = "adjustment"    // Kiểm kê điều chỉnh
)
```

### 3.2 Functional Requirements Matrix

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| FR-01 | Create tool card with auto-generated code | P0 | ✅ Implemented |
| FR-02 | Update tool card information | P0 | ✅ Implemented |
| FR-03 | Delete tool card (only if active) | P0 | ✅ Implemented |
| FR-04 | List/search tool cards by category, state | P0 | ✅ Implemented |
| FR-05 | Scrap/dispose tool card | P0 | ✅ Implemented |
| FR-06 | Import tool to warehouse | P0 | ❌ Missing |
| FR-07 | Export tool from warehouse | P0 | ❌ Missing |
| FR-08 | Transfer tool between locations/departments | P1 | ❌ Missing |
| FR-09 | Track tool assignment to persons | P1 | ❌ Missing |
| FR-10 | GL integration with Account 153 | P0 | ❌ Missing |
| FR-11 | Inventory adjustment (surplus/shortage) | P1 | ❌ Missing |
| FR-12 | Multi-period cost allocation | P2 | ❌ Missing |
| FR-13 | Audit trail for all transactions | P1 | ❌ Missing |
| FR-14 | Reports (inventory list, transaction log) | P1 | ❌ Missing |
| FR-15 | Web UI | P1 | ❌ Missing |

---

## 4. Non-Functional Requirements

### 4.1 Compliance
- **NFR-01**: Must comply with Thông tư 99/2025/TT-BTC
- **NFR-02**: Must support Account 153 with sub-accounts as needed
- **NFR-03**: Must maintain audit trail for regulatory compliance
- **NFR-04**: Must support Vietnamese language interfaces

### 4.2 Performance
- **NFR-05**: List queries < 500ms for 10,000 records
- **NFR-06**: Support concurrent users (minimum 50)

### 4.3 Data Integrity
- **NFR-07**: All transactions must be atomic
- **NFR-08**: Inventory counts must reconcile with GL balances
- **NFR-09**: Prevent deletion of tools with pending transactions

---

## 5. Integration Requirements

### 5.1 GL Integration
- Post purchase transactions to Account 153
- Post expense allocations to Accounts 623/627/641/642
- Support VAT posting to Account 133
- Monthly reconciliation with GL

### 5.2 Inventory Integration
- Sync with warehouse management
- Support barcode/QR for physical tracking
- Integration with procurement module

---

## 6. Assumptions & Constraints

### 6.1 Assumptions
1. The `fixedasset` module handles items ≥ 30M VND
2. This module handles items < 30M VND (CCDC)
3. GL integration is mandatory for accounting compliance
4. Users have basic accounting knowledge

### 6.2 Constraints
1. Must use existing SQLite database infrastructure
2. Must follow existing 4-layer architecture pattern
3. Must integrate with existing Casbin authorization
4. Must comply with Thông tư 99/2025/TT-BTC (effective 01/01/2026)

---

## 7. Success Criteria

| Criterion | Metric |
|-----------|--------|
| Accounting compliance | 100% compliant with Thông tư 99/2025/TT-BTC |
| GL reconciliation | Account 153 balance matches inventory records |
| Transaction accuracy | Zero posting errors |
| User adoption | All inventory staff use the system |
| Audit readiness | All required reports available |

---

## 8. Recommendations

### 8.1 Immediate Actions (P0)
1. **Enhance ToolCard entity** with GL account fields (AccountCode153, AccountCodeExp)
2. **Implement Transaction tracking** (import, export, transfer, disposal)
3. **Add GL integration** for Account 153 posting
4. **Add audit trail** for all mutations

### 8.2 Short-term Actions (P1)
1. **Implement Web UI** for tool card management
2. **Add Reports** (inventory list, transaction log, GL summary)
3. **Add approval workflow** for disposal

### 8.3 Long-term Actions (P2)
1. **Barcode/QR support** for physical tracking
2. **Multi-period cost allocation** (Account 242)
3. **Integration with procurement module**

---

## Appendix A: Reference Documents

1. **Thông tư 99/2025/TT-BTC** - Hệ thống tài khoản kế toán doanh nghiệp
2. **Thông tư 45/2013/TT-BTC** - Quản lý, sử dụng và trích khấu hao TSCĐ
3. **VAS 03** - Tài sản cố định
4. **Luật Kế toán 88/2015** - Luật kế toán Việt Nam

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
