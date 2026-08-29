# Tools/Equipment Module - Complete Specification
## Công cụ, dụng cụ (CCDC)

**Version**: 1.0  
**Date**: 2026-08-29  
**Module**: Tools/Equipment (Công cụ, dụng cụ)  
**Compliance**: Thông tư 99/2025/TT-BTC, VAS 03

---

## 1. Business Rules (Regulatory)

### BR-01: Definition
Tools/Equipment (CCDC) are labor tools that **do not meet Fixed Asset (TSCĐ) criteria**:
- Original cost < 30,000,000 VND, OR
- Useful life < 12 months

### BR-02: Accounting Treatment
- **Account**: 153 - Công cụ, dụng cụ
- **Valuation**: At cost (giá gốc)
- **Tracking**: By warehouse, category, group, individual item
- **Expense**: Immediate or multi-period allocation (Account 242)

### BR-03: GL Account Mapping
| Account | Name | Purpose |
|---------|------|---------|
| 153 | Công cụ, dụng cụ | Tools/equipment inventory |
| 133 | Thuế GTGT được khấu trừ | Input VAT |
| 111 | Tiền mặt | Cash |
| 112 | Tiền gửi ngân hàng | Bank |
| 331 | Phải trả người bán | Accounts payable |
| 242 | Chi phí chờ phân bổ | Deferred expenses |
| 623 | Chi phí sản xuất | Production costs |
| 627 | Chi phí equipment | Equipment costs |
| 641 | Chi phí quản lý | Management costs |
| 642 | Chi phí bán hàng | Selling costs |
| 632 | Giá vốn hàng bán | Cost of goods sold |
| 511 | Doanh thu bán hàng | Sales revenue |

### BR-04: Transaction Types
| Transaction | GL Entry |
|-------------|----------|
| Purchase | Dr 153, Dr 133 / Cr 111/112/331 |
| Issue to production | Dr 623/627/641/642 / Cr 153 |
| Multi-period allocation | Dr 242 / Cr 153 (initial), Dr 623/627/641/642 / Cr 242 (periodic) |
| Return to supplier | Dr 331 / Cr 153, Cr 133 |
| Disposal/sale | Dr 632 / Cr 153; Dr 111/131 / Cr 511 |
| Inventory adjustment | Dr 153 / Cr 3381 (surplus); Dr 511 / Cr 153 (shortage) |

---

## 2. Data Model

### 2.1 ToolCard Entity

```go
package tools

type ToolCard struct {
    // Core identification
    ID          string `json:"id"`
    Code        string `json:"code"` // Auto-generated: TL-XXXXX
    Name        string `json:"name"`
    
    // Classification
    Category    string `json:"category"` // scaffolding, formwork, tools, office_supplies, clothing
    SubCategory string `json:"sub_category,omitempty"`
    Description string `json:"description,omitempty"`
    
    // Financial (VND)
    OriginalCost    int64 `json:"original_cost"`
    Quantity        int   `json:"quantity"` // Default 1
    Unit            string `json:"unit"` // pcs, set, pair
    
    // Source documents
    PurchaseDate    string `json:"purchase_date"`
    InvoiceNo       string `json:"invoice_no,omitempty"`
    Supplier        string `json:"supplier,omitempty"`
    
    // Location & Assignment
    Warehouse       string `json:"warehouse,omitempty"`
    Department      string `json:"department,omitempty"`
    AssignedTo      string `json:"assigned_to,omitempty"`
    Location        string `json:"location,omitempty"`
    
    // Status
    State           ToolCardState `json:"state"`
    
    // GL Integration
    AccountCode153  string `json:"account_code_153,omitempty"` // Account 153 detail
    AccountCodeExp  string `json:"account_code_exp,omitempty"` // Expense account (623/627/641/642)
    
    // Audit
    CreatedBy       string `json:"created_by,omitempty"`
    CreatedAt       string `json:"created_at"`
    UpdatedBy       string `json:"updated_by,omitempty"`
    UpdatedAt       string `json:"updated_at"`
}

type ToolCardState string

const (
    StateActive     ToolCardState = "active"
    StateInactive   ToolCardState = "inactive"
    StateDisposed   ToolCardState = "disposed"
    StateDamaged    ToolCardState = "damaged"
    StateInStorage  ToolCardState = "in_storage"
)

// ValidateToolCard validates tool card data per Thông tư 99/2025/TT-BTC
func ValidateToolCard(t *ToolCard) error {
    if t.Name == "" {
        return &ValidationError{Field: "name", Message: "name is required"}
    }
    if t.OriginalCost <= 0 {
        return &ValidationError{Field: "original_cost", Message: "original cost must be positive"}
    }
    if t.OriginalCost >= 30_000_000 {
        return &ValidationError{Field: "original_cost", Message: "value must be < 30M VND (use fixedasset module)"}
    }
    if t.Quantity <= 0 {
        return &ValidationError{Field: "quantity", Message: "quantity must be positive"}
    }
    if t.PurchaseDate == "" {
        return &ValidationError{Field: "purchase_date", Message: "purchase date is required"}
    }
    if t.State == "" {
        t.State = StateActive
    }
    if !t.State.IsValid() {
        return &ValidationError{Field: "state", Message: "invalid state"}
    }
    return nil
}
```

### 2.2 Transaction Entity

```go
package tools

type ToolTransaction struct {
    ID              string          `json:"id"`
    ToolCardID      string          `json:"tool_card_id"`
    ToolCardCode    string          `json:"tool_card_code"`
    TransactionType TransactionType `json:"transaction_type"`
    Quantity        int             `json:"quantity"`
    UnitPrice       int64           `json:"unit_price"`
    TotalAmount     int64           `json:"total_amount"`
    FromLocation    string          `json:"from_location,omitempty"`
    ToLocation      string          `json:"to_location,omitempty"`
    FromDepartment  string          `json:"from_department,omitempty"`
    ToDepartment    string          `json:"to_department,omitempty"`
    AssignedTo      string          `json:"assigned_to,omitempty"`
    ReferenceNo     string          `json:"reference_no,omitempty"` // Invoice, voucher
    Notes           string          `json:"notes,omitempty"`
    
    // GL Posting
    GLPosted        bool            `json:"gl_posted"`
    GLReference     string          `json:"gl_reference,omitempty"`
    
    // Audit
    CreatedBy       string          `json:"created_by"`
    CreatedAt       string          `json:"created_at"`
}

type TransactionType string

const (
    TxImport        TransactionType = "import"
    TxExport        TransactionType = "export"
    TxTransfer      TransactionType = "transfer"
    TxReturn        TransactionType = "return"
    TxDisposal      TransactionType = "disposal"
    TxAdjustment    TransactionType = "adjustment"
)
```

### 2.3 Repository Interface

```go
package tools

type Repository interface {
    // ToolCard CRUD
    Create(ctx context.Context, t *ToolCard) error
    FindByID(ctx context.Context, id string) (*ToolCard, error)
    Update(ctx context.Context, t *ToolCard) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, category string, state ToolCardState, limit, offset int) ([]*ToolCard, error)
    NextCode(ctx context.Context) (int64, error)
    
    // Transaction operations
    CreateTransaction(ctx context.Context, tx *ToolTransaction) error
    FindTransactionByID(ctx context.Context, id string) (*ToolTransaction, error)
    ListTransactions(ctx context.Context, toolCardID string, txType TransactionType) ([]*ToolTransaction, error)
    
    // Inventory operations
    GetStock(ctx context.Context, toolCardID string) (int, error)
    AdjustStock(ctx context.Context, toolCardID string, quantity int, reason string) error
}
```

---

## 3. Service Layer

### 3.1 Service Interface

```go
package tools

type Service interface {
    // ToolCard operations
    Create(ctx context.Context, c *ToolCard, actor string) (*ToolCard, error)
    Get(ctx context.Context, id string) (*ToolCard, error)
    Update(ctx context.Context, id string, patch *ToolCard, actor string) (*ToolCard, error)
    List(ctx context.Context, category string, state ToolCardState, limit, offset int) ([]*ToolCard, error)
    Delete(ctx context.Context, id string) error
    
    // Transaction operations
    Import(ctx context.Context, toolCardID string, quantity int, unitPrice int64, ref string, actor string) (*ToolTransaction, error)
    Export(ctx context.Context, toolCardID string, quantity int, toLocation, toDepartment, toPerson string, actor string) (*ToolTransaction, error)
    Transfer(ctx context.Context, toolCardID string, quantity int, toLocation, toDepartment string, actor string) (*ToolTransaction, error)
    Return(ctx context.Context, toolCardID string, quantity int, ref string, actor string) (*ToolTransaction, error)
    Dispose(ctx context.Context, toolCardID string, reason string, actor string) (*ToolTransaction, error)
    
    // Inventory operations
    GetStock(ctx context.Context, toolCardID string) (int, error)
    AdjustInventory(ctx context.Context, toolCardID string, newQuantity int, reason string, actor string) (*ToolTransaction, error)
    
    // Scrap (legacy compatibility)
    Scrap(ctx context.Context, id, actor string) (*ToolCard, error)
}
```

### 3.2 Service Implementation

```go
package tools

type service struct {
    repo Repository
}

func (s *service) Import(ctx context.Context, toolCardID string, quantity int, unitPrice int64, ref string, actor string) (*ToolTransaction, error) {
    // 1. Get tool card
    card, err := s.repo.FindByID(ctx, toolCardID)
    if err != nil {
        return nil, err
    }
    
    // 2. Create transaction
    tx := &ToolTransaction{
        ID:              generateID(),
        ToolCardID:      toolCardID,
        ToolCardCode:    card.Code,
        TransactionType: TxImport,
        Quantity:        quantity,
        UnitPrice:       unitPrice,
        TotalAmount:     int64(quantity) * unitPrice,
        ReferenceNo:     ref,
        CreatedBy:       actor,
        CreatedAt:       NowRFC3339(),
    }
    
    // 3. Save transaction
    if err := s.repo.CreateTransaction(ctx, tx); err != nil {
        return nil, err
    }
    
    // 4. Update tool card quantity
    card.Quantity += quantity
    card.UpdatedBy = actor
    card.UpdatedAt = NowRFC3339()
    if err := s.repo.Update(ctx, card); err != nil {
        return nil, err
    }
    
    return tx, nil
}
```

---

## 4. HTTP Handler

### 4.1 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tools/cards` | Create tool card |
| GET | `/api/v1/tools/cards` | List tool cards |
| GET | `/api/v1/tools/cards/:id` | Get tool card by ID |
| PUT | `/api/v1/tools/cards/:id` | Update tool card |
| DELETE | `/api/v1/tools/cards/:id` | Delete tool card |
| POST | `/api/v1/tools/cards/:id/scrap` | Scrap/dispose tool |
| POST | `/api/v1/tools/cards/:id/import` | Import tool to warehouse |
| POST | `/api/v1/tools/cards/:id/export` | Export tool from warehouse |
| POST | `/api/v1/tools/cards/:id/transfer` | Transfer tool |
| POST | `/api/v1/tools/cards/:id/return` | Return to supplier |
| GET | `/api/v1/tools/cards/:id/transactions` | List transactions |
| GET | `/api/v1/tools/cards/:id/stock` | Get current stock |
| POST | `/api/v1/tools/transactions/:id/adjust` | Adjust inventory |

### 4.2 Request/Response Examples

#### Create Tool Card
```json
// Request
POST /api/v1/tools/cards
{
    "name": "Drill Machine Makita",
    "category": "tools",
    "original_cost": 2500000,
    "quantity": 2,
    "unit": "pcs",
    "purchase_date": "2026-08-29",
    "warehouse": "WH-01",
    "department": "production"
}

// Response
{
    "data": {
        "id": "tools_abc123",
        "code": "TL-00001",
        "name": "Drill Machine Makita",
        "category": "tools",
        "original_cost": 2500000,
        "quantity": 2,
        "state": "active",
        "created_at": "2026-08-29T10:30:00Z"
    }
}
```

#### Import Tool
```json
// Request
POST /api/v1/tools/cards/tools_abc123/import
{
    "quantity": 5,
    "unit_price": 2500000,
    "reference_no": "INV-2026-001"
}

// Response
{
    "data": {
        "id": "tx_xyz789",
        "tool_card_id": "tools_abc123",
        "transaction_type": "import",
        "quantity": 5,
        "unit_price": 2500000,
        "total_amount": 12500000,
        "reference_no": "INV-2026-001",
        "gl_posted": true,
        "created_at": "2026-08-29T10:35:00Z"
    }
}
```

---

## 5. UI/UX Wireframes

### 5.1 Tool Card List Page
```
┌─────────────────────────────────────────────────────────────────┐
│ Tools/Equipment Inventory                                    [+]│
├─────────────────────────────────────────────────────────────────┤
│ Filters:                                                        │
│ Category: [All ▼]  State: [All ▼]  Search: [____________] [🔍] │
├─────────────────────────────────────────────────────────────────┤
│ Code    │ Name              │ Category │ Qty │ State    │ Actions│
│---------│-------------------│----------│-----│----------│--------│
│ TL-00001│ Drill Machine     │ Tools    │  5  │ Active   │ ✏️ 🗑️ │
│ TL-00002│ Safety Helmet     │ Clothing │ 20  │ Active   │ ✏️ 🗑️ │
│ TL-00003│ Office Chair      │ Office   │  3  │ Inactive │ ✏️ 🗑️ │
├─────────────────────────────────────────────────────────────────┤
│ Showing 1-3 of 3 | < 1 >                                       │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Tool Card Detail Page
```
┌─────────────────────────────────────────────────────────────────┐
│ Tool Card: TL-00001 - Drill Machine Makita                 [✏️]│
├─────────────────────────────────────────────────────────────────┤
│ General Information:                                            │
│ Code: TL-00001          Category: Tools                        │
│ Name: Drill Machine Makita  State: Active                      │
│ Original Cost: 2,500,000 VND  Quantity: 5 pcs                  │
│ Purchase Date: 2026-08-29   Supplier: ABC Corp                 │
├─────────────────────────────────────────────────────────────────┤
│ Location & Assignment:                                          │
│ Warehouse: WH-01           Department: Production              │
│ Location: Workshop A       Assigned To: Nguyen Van A           │
├─────────────────────────────────────────────────────────────────┤
│ GL Accounts:                                                    │
│ Account 153: 1531-Tools    Expense: 627-Equipment              │
├─────────────────────────────────────────────────────────────────┤
│ Recent Transactions:                                            │
│ Date       │ Type   │ Qty │ Amount    │ Reference              │
│------------│--------│-----│-----------│------------------------│
│ 2026-08-29 │ Import │  5  │ 12,500,000│ INV-2026-001           │
├─────────────────────────────────────────────────────────────────┤
│ [Import] [Export] [Transfer] [Return] [Dispose] [Transactions] │
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 Import Form
```
┌─────────────────────────────────────────────────────────────────┐
│ Import Tool to Warehouse                                   [X]  │
├─────────────────────────────────────────────────────────────────┤
│ Tool Card: TL-00001 - Drill Machine Makita                     │
│ Current Stock: 5 pcs                                           │
├─────────────────────────────────────────────────────────────────┤
│ Quantity:     [________] pcs                                   │
│ Unit Price:   [________] VND                                   │
│ Reference No: [________] (Invoice/Voucher)                     │
│ Notes:         [________________________________]              │
├─────────────────────────────────────────────────────────────────┤
│ GL Entry Preview:                                               │
│ Dr 153 - Tools:        12,500,000 VND                         │
│ Dr 133 - VAT (10%):     1,250,000 VND                         │
│ Cr 331 - AP:           13,750,000 VND                         │
├─────────────────────────────────────────────────────────────────┤
│                              [Cancel]  [Confirm Import]         │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Process Flows

### 6.1 Import Process
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Start        │────▶│ Create Tool  │────▶│ Create Import│────▶│ Update Stock │
│              │     │ Card         │     │ Transaction  │     │              │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                           │                     │                     │
                           ▼                     ▼                     ▼
                     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
                     │ Validate     │     │ Post to GL   │     │ Complete     │
                     │ Data         │     │ Account 153  │     │              │
                     └──────────────┘     └──────────────┘     └──────────────┘
```

### 6.2 Export Process
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Start        │────▶│ Check Stock  │────▶│ Create Export│────▶│ Update Stock │
│              │     │ Availability │     │ Transaction  │     │              │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                           │                     │                     │
                           ▼                     ▼                     ▼
                     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
                     │ Validate     │     │ Post to GL   │     │ Complete     │
                     │ Quantity     │     │ Accounts 62x │     │              │
                     └──────────────┘     └──────────────┘     └──────────────┘
```

---

## 7. User Journeys

### 7.1 Journey 1: Import New Tools
```
1. User navigates to Tool Card List
2. User clicks [+] to create new tool card
3. User fills in tool information (name, category, cost, etc.)
4. User saves tool card
5. User clicks [Import] on tool card detail
6. User enters quantity, unit price, reference number
7. User confirms import
8. System updates stock and posts to GL
9. User sees updated stock balance
```

### 7.2 Journey 2: Export Tools for Use
```
1. User navigates to Tool Card Detail
2. User clicks [Export]
3. User selects destination (department, person)
4. User enters quantity
5. System validates stock availability
6. User confirms export
7. System updates stock and posts to expense accounts
8. User sees updated stock balance
```

---

## 8. Implementation Roadmap

### Phase 1: Core Entity (Week 1)
- [ ] Update ToolCard entity with GL account fields
- [ ] Add validation for < 30M VND threshold
- [ ] Update Repository interface
- [ ] Unit tests

### Phase 2: Transaction Tracking (Week 2)
- [ ] Implement ToolTransaction entity
- [ ] Implement Import/Export/Transfer services
- [ ] Add inventory adjustment
- [ ] Unit tests

### Phase 3: GL Integration (Week 3)
- [ ] Implement GL posting for Account 153
- [ ] Implement expense posting (623/627/641/642)
- [ ] Add VAT handling (Account 133)
- [ ] Integration tests

### Phase 4: HTTP Handlers (Week 4)
- [ ] Implement all API endpoints
- [ ] Add request validation
- [ ] Add error handling
- [ ] API tests

### Phase 5: Web UI (Week 5-6)
- [ ] Tool Card List page
- [ ] Tool Card Detail page
- [ ] Import/Export forms
- [ ] Transaction history

### Phase 6: Reports & Polish (Week 7)
- [ ] Inventory report
- [ ] Transaction log
- [ ] GL summary report
- [ ] Audit trail

---

## 9. Verification Checklist

- [ ] All transactions comply with Thông tư 99/2025/TT-BTC
- [ ] GL Account 153 postings are correct
- [ ] Inventory counts reconcile with GL balances
- [ ] All mutations have audit trail
- [ ] Unit tests pass (coverage > 80%)
- [ ] Integration tests pass
- [ ] API documentation complete
- [ ] User documentation complete

---

*Document prepared by:*
- **BA Lead** (20+ years experience)
- **Chief Accountant** (20+ years, CPA Vietnam)

*Date: 2026-08-29*
