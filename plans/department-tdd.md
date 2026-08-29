# Department Module - TDD Implementation Plan

**Approach:** Test-Driven Development (Red → Green → Refactor)  
**Order:** Simple → Complex, Core → Edge  
**Scope:** No third-party integration in this version  
**Standard:** Every step must pass `go build ./...`, `go vet ./...`, `go test ./...`

---

## Implementation Order

### Wave 1: Domain Layer (Core)
1. DepartmentType enum and constants
2. Department-specific fields in Record struct
3. Department validation rules
4. Department tree construction logic

### Wave 2: Application Layer (Business Logic)
5. Create department with validation
6. Update department with validation
7. Get department by code
8. List departments with filtering
9. Deactivate department with checks
10. Get department tree

### Wave 3: Infrastructure Layer (Persistence)
11. Budget record entity and repository
12. Set budget per department/year
13. Get budget per department/year
14. Calculate employee count

### Wave 4: Interface Layer (HTTP)
15. Create department endpoint
16. Update department endpoint
17. Get department endpoint
18. List departments endpoint
19. Deactivate department endpoint
20. Get department tree endpoint
21. Set budget endpoint
22. Get budget endpoint

---

## Wave 1: Domain Layer

### Step 1: DepartmentType Enum

**Red:** Write test first
**Green:** Implement enum
**Refactor:** Clean up

**File:** `internal/domain/masterdata/entity.go`

**Test:**
```go
func TestDepartmentType_IsValid(t *testing.T) {
    tests := []struct {
        input    DepartmentType
        expected bool
    }{
        {DepartmentTypeExecutive, true},
        {DepartmentTypeOperational, true},
        {DepartmentTypeSupport, true},
        {DepartmentType("invalid"), false},
        {DepartmentType(""), false},
    }
    for _, tt := range tests {
        if got := tt.input.IsValid(); got != tt.expected {
            t.Errorf("DepartmentType(%q).IsValid() = %v, want %v", tt.input, got, tt.expected)
        }
    }
}
```

**Implementation:**
```go
type DepartmentType string

const (
    DepartmentTypeExecutive   DepartmentType = "executive"
    DepartmentTypeOperational DepartmentType = "operational"
    DepartmentTypeSupport     DepartmentType = "support"
)

func (dt DepartmentType) IsValid() bool {
    switch dt {
    case DepartmentTypeExecutive, DepartmentTypeOperational, DepartmentTypeSupport:
        return true
    default:
        return false
    }
}
```

---

### Step 2: Department-Specific Fields

**Red:** Write test for new fields
**Green:** Add fields to Record struct
**Refactor:** Ensure backward compatibility

**File:** `internal/domain/masterdata/entity.go`

**Test:**
```go
func TestRecord_DepartmentFields(t *testing.T) {
    rec := &Record{
        Kind:           KindDepartment,
        Code:           "BP-00001",
        Name:           "Phòng Kế toán",
        Extra:          map[string]string{},
    }
    
    // Test setting department-specific fields via Extra
    rec.Extra["cost_center_code"] = "CC-001"
    rec.Extra["manager_code"] = "NV-002"
    rec.Extra["budget_annual"] = "1500000000"
    rec.Extra["department_type"] = "support"
    rec.Extra["phone"] = "024-12345678"
    rec.Extra["email"] = "ketoan@company.vn"
    rec.Extra["address"] = "Tầng 5, Hà Nội"
    
    if rec.Extra["cost_center_code"] != "CC-001" {
        t.Errorf("cost_center_code = %q, want CC-001", rec.Extra["cost_center_code"])
    }
}
```

**Note:** We use Extra map for now to maintain backward compatibility. In a future version, we can extract these into typed fields.

---

### Step 3: Department Validation Rules

**Red:** Write validation tests
**Green:** Implement validation
**Refactor:** Extract validation helpers

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestValidateDepartment_CodeFormat(t *testing.T) {
    svc := &service{}
    
    tests := []struct {
        name    string
        code    string
        wantErr bool
    }{
        {"valid code", "BP-00001", false},
        {"valid code long", "BP-12345", false},
        {"invalid prefix", "DEP-00001", true},
        {"invalid format", "BP-000", true},
        {"empty code", "", false}, // Auto-generated
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rec := &masterdata.Record{
                Kind: masterdata.KindDepartment,
                Code: tt.code,
                Name: "Test Department",
                Extra: map[string]string{
                    "department_type": "support",
                },
            }
            err := svc.validateDepartment(rec)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateDepartment() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Implementation:**
```go
func (s *service) validateDepartment(rec *masterdata.Record) error {
    // BR-D001: Code format (BP-XXXXX)
    if rec.Code != "" {
        if !strings.HasPrefix(rec.Code, "BP-") || len(rec.Code) != 8 {
            return &masterdata.ValidationError{
                Kind: rec.Kind, Code: rec.Code,
                MessageVn: "Mã phòng ban phải đúng định dạng BP-XXXXX",
                MessageEn: "department code must be in format BP-XXXXX",
            }
        }
    }
    
    // BR-D002: Name required, max 200 chars
    if strings.TrimSpace(rec.Name) == "" {
        return &masterdata.ValidationError{
            Kind: rec.Kind, Code: rec.Code,
            MessageVn: "Tên phòng ban là bắt buộc",
            MessageEn: "department name is required",
        }
    }
    if len(rec.Name) > 200 {
        return &masterdata.ValidationError{
            Kind: rec.Kind, Code: rec.Code,
            MessageVn: "Tên phòng ban tối đa 200 ký tự",
            MessageEn: "department name max 200 characters",
        }
    }
    
    // BR-D009: Department type validation
    if dt := masterdata.DepartmentType(rec.Extra["department_type"]); dt != "" && !dt.IsValid() {
        return &masterdata.ValidationError{
            Kind: rec.Kind, Code: rec.Code,
            MessageVn: "Loại phòng ban không hợp lệ",
            MessageEn: "invalid department type",
        }
    }
    
    return nil
}
```

---

### Step 4: Department Tree Construction

**Red:** Write tree construction test
**Green:** Implement tree logic
**Refactor:** Optimize

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestBuildDepartmentTree(t *testing.T) {
    departments := []*masterdata.Record{
        {Code: "BP-00001", Name: "Ban Giám đốc", GroupCode: "", Level: 0},
        {Code: "BP-00002", Name: "Phòng Kế toán", GroupCode: "BP-00001", Level: 1},
        {Code: "BP-00003", Name: "Phòng Kinh doanh", GroupCode: "BP-00001", Level: 1},
        {Code: "BP-00004", Name: "Tổ Kế toán", GroupCode: "BP-00002", Level: 2},
    }
    
    tree := buildDepartmentTree(departments)
    
    if len(tree) != 1 {
        t.Errorf("root nodes = %d, want 1", len(tree))
    }
    if tree[0].Code != "BP-00001" {
        t.Errorf("root code = %q, want BP-00001", tree[0].Code)
    }
    if len(tree[0].Children) != 2 {
        t.Errorf("root children = %d, want 2", len(tree[0].Children))
    }
}
```

**Implementation:**
```go
type DepartmentNode struct {
    *Record
    Children []*DepartmentNode
}

func buildDepartmentTree(departments []*Record) []*DepartmentNode {
    nodeMap := make(map[string]*DepartmentNode)
    var roots []*DepartmentNode
    
    // Create nodes
    for _, dept := range departments {
        nodeMap[dept.Code] = &DepartmentNode{Record: dept}
    }
    
    // Build tree
    for _, dept := range departments {
        node := nodeMap[dept.Code]
        if dept.GroupCode == "" {
            roots = append(roots, node)
        } else if parent, ok := nodeMap[dept.GroupCode]; ok {
            parent.Children = append(parent.Children, node)
        }
    }
    
    return roots
}
```

---

## Wave 2: Application Layer

### Step 5: Create Department with Validation

**Red:** Write creation test
**Green:** Implement CreateDepartment
**Refactor:** Extract common logic

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestCreateDepartment_Success(t *testing.T) {
    // Setup
    repo := &mockRepository{}
    svc := NewService(repo)
    
    input := &masterdata.Record{
        Name: "Phòng Kế toán",
        Extra: map[string]string{
            "department_type": "support",
            "cost_center_code": "CC-001",
        },
    }
    
    // Mock NextCode
    repo.nextCode = 1
    repo.GetByCodeFunc = func(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
        return nil, masterdata.ErrNotFound
    }
    
    // Act
    result, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Code != "BP-00001" {
        t.Errorf("code = %q, want BP-00001", result.Code)
    }
    if result.State != masterdata.StateActive {
        t.Errorf("state = %q, want active", result.State)
    }
}
```

---

### Step 6: Update Department with Validation

**Red:** Write update test
**Green:** Implement UpdateDepartment
**Refactor:** Extract validation

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestUpdateDepartment_Success(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    existing := &masterdata.Record{
        Code: "BP-00001",
        Name: "Phòng Kế toán",
        State: masterdata.StateActive,
        Extra: map[string]string{"department_type": "support"},
    }
    
    repo.GetByCodeFunc = func(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
        return existing, nil
    }
    
    patch := &masterdata.Record{
        Name: "Phòng Kế toán Tổng hợp",
    }
    
    result, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-00001", patch, "admin")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Name != "Phòng Kế toán Tổng hợp" {
        t.Errorf("name = %q, want Phòng Kế toán Tổng hợp", result.Name)
    }
}
```

---

### Step 7: Get Department by Code

**Red:** Write get test
**Green:** Implement GetDepartment
**Refactor:** Clean up

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestGetDepartment_Success(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    expected := &masterdata.Record{
        Code: "BP-00001",
        Name: "Phòng Kế toán",
    }
    
    repo.GetByCodeFunc = func(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
        return expected, nil
    }
    
    result, err := svc.Get(context.Background(), masterdata.KindDepartment, "BP-00001")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Code != expected.Code {
        t.Errorf("code = %q, want %q", result.Code, expected.Code)
    }
}
```

---

### Step 8: List Departments with Filtering

**Red:** Write list test
**Green:** Implement ListDepartments
**Refactor:** Optimize filtering

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestListDepartments_FilterByState(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    departments := []*masterdata.Record{
        {Code: "BP-00001", Name: "Active Dept", State: masterdata.StateActive},
        {Code: "BP-00002", Name: "Inactive Dept", State: masterdata.StateInactive},
        {Code: "BP-00003", Name: "Another Active", State: masterdata.StateActive},
    }
    
    repo.ListFunc = func(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
        return departments, nil
    }
    
    result, _, err := svc.List(context.Background(), masterdata.KindDepartment, "", "", "active", 0, 0)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(result) != 2 {
        t.Errorf("result count = %d, want 2", len(result))
    }
}
```

---

### Step 9: Deactivate Department with Checks

**Red:** Write deactivation test
**Green:** Implement DeactivateDepartment
**Refactor:** Extract checks

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestDeactivateDepartment_Success(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    dept := &masterdata.Record{
        Code: "BP-00001",
        Name: "Phòng Kế toán",
        State: masterdata.StateActive,
    }
    
    repo.GetByCodeFunc = func(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
        return dept, nil
    }
    
    repo.ListFunc = func(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
        return []*masterdata.Record{}, nil // No employees
    }
    
    result, err := svc.Deactivate(context.Background(), masterdata.KindDepartment, "BP-00001", "Restructuring", "admin")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.State != masterdata.StateInactive {
        t.Errorf("state = %q, want inactive", result.State)
    }
    if result.DeactivateReason != "Restructuring" {
        t.Errorf("reason = %q, want Restructuring", result.DeactivateReason)
    }
}
```

---

### Step 10: Get Department Tree

**Red:** Write tree test
**Green:** Implement GetDepartmentTree
**Refactor:** Optimize

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestGetDepartmentTree(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    departments := []*masterdata.Record{
        {Code: "BP-00001", Name: "Ban Giám đốc", GroupCode: ""},
        {Code: "BP-00002", Name: "Phòng Kế toán", GroupCode: "BP-00001"},
        {Code: "BP-00003", Name: "Phòng Kinh doanh", GroupCode: "BP-00001"},
    }
    
    repo.ListFunc = func(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
        return departments, nil
    }
    
    tree, err := svc.GetDepartmentTree(context.Background())
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(tree) != 1 {
        t.Errorf("root count = %d, want 1", len(tree))
    }
    if len(tree[0].Children) != 2 {
        t.Errorf("children count = %d, want 2", len(tree[0].Children))
    }
}
```

---

## Wave 3: Infrastructure Layer

### Step 11: Budget Record Entity

**Red:** Write budget entity test
**Green:** Implement budget struct
**Refactor:** Clean up

**File:** `internal/domain/masterdata/entity.go`

**Test:**
```go
func TestBudgetRecord(t *testing.T) {
    budget := &BudgetRecord{
        ID:           "budget-001",
        DepartmentCode: "BP-00001",
        FiscalYear:   2026,
        Amount:       1500000000,
        Notes:        "Ngân sách phòng kế toán",
        CreatedAt:    "2026-01-01T00:00:00Z",
    }
    
    if budget.DepartmentCode != "BP-00001" {
        t.Errorf("department_code = %q, want BP-00001", budget.DepartmentCode)
    }
    if budget.FiscalYear != 2026 {
        t.Errorf("fiscal_year = %d, want 2026", budget.FiscalYear)
    }
}
```

**Implementation:**
```go
type BudgetRecord struct {
    ID              string `json:"id"`
    DepartmentCode  string `json:"department_code"`
    FiscalYear      int    `json:"fiscal_year"`
    Amount          int64  `json:"amount"`
    Notes           string `json:"notes,omitempty"`
    CreatedBy       string `json:"created_by,omitempty"`
    CreatedAt       string `json:"created_at"`
    UpdatedBy       string `json:"updated_by,omitempty"`
    UpdatedAt       string `json:"updated_at"`
}
```

---

### Step 12: Set Budget Per Department/Year

**Red:** Write set budget test
**Green:** Implement SetBudget
**Refactor:** Extract validation

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestSetBudget_Success(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    dept := &masterdata.Record{
        Code:  "BP-00001",
        State: masterdata.StateActive,
    }
    
    repo.GetByCodeFunc = func(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
        return dept, nil
    }
    
    repo.GetBudgetFunc = func(ctx context.Context, deptCode string, year int) (*masterdata.BudgetRecord, error) {
        return nil, masterdata.ErrNotFound
    }
    
    budget, err := svc.SetBudget(context.Background(), "BP-00001", 2026, 1500000000, "Initial budget", "admin")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if budget.Amount != 1500000000 {
        t.Errorf("amount = %d, want 1500000000", budget.Amount)
    }
}
```

---

### Step 13: Get Budget Per Department/Year

**Red:** Write get budget test
**Green:** Implement GetBudget
**Refactor:** Clean up

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestGetBudget_Success(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    expected := &masterdata.BudgetRecord{
        DepartmentCode: "BP-00001",
        FiscalYear:    2026,
        Amount:        1500000000,
    }
    
    repo.GetBudgetFunc = func(ctx context.Context, deptCode string, year int) (*masterdata.BudgetRecord, error) {
        return expected, nil
    }
    
    result, err := svc.GetBudget(context.Background(), "BP-00001", 2026)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Amount != expected.Amount {
        t.Errorf("amount = %d, want %d", result.Amount, expected.Amount)
    }
}
```

---

### Step 14: Calculate Employee Count

**Red:** Write employee count test
**Green:** Implement CalculateEmployeeCount
**Refactor:** Optimize

**File:** `internal/application/masterdata/service_test.go`

**Test:**
```go
func TestCalculateEmployeeCount(t *testing.T) {
    repo := &mockRepository{}
    svc := NewService(repo)
    
    employees := []*masterdata.Record{
        {Code: "NV-001", Extra: map[string]string{"department_code": "BP-00001"}},
        {Code: "NV-002", Extra: map[string]string{"department_code": "BP-00001"}},
        {Code: "NV-003", Extra: map[string]string{"department_code": "BP-00002"}},
    }
    
    repo.ListFunc = func(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
        return employees, nil
    }
    
    count, err := svc.CalculateEmployeeCount(context.Background(), "BP-00001")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if count != 2 {
        t.Errorf("count = %d, want 2", count)
    }
}
```

---

## Wave 4: Interface Layer

### Step 15: Create Department Endpoint

**Red:** Write HTTP test
**Green:** Implement handler
**Refactor:** Extract common logic

**File:** `internal/interfaces/http/masterdata/handler_test.go`

**Test:**
```go
func TestCreateDepartment_Success(t *testing.T) {
    svc := &mockService{}
    handler := NewHandler(svc)
    
    body := `{
        "name": "Phòng Kế toán",
        "extra": {
            "department_type": "support",
            "cost_center_code": "CC-001"
        }
    }`
    
    req := httptest.NewRequest("POST", "/master-data/department", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-User-Id", "admin")
    
    w := httptest.NewRecorder()
    router := gin.New()
    router.POST("/master-data/department", handler.createRecord)
    router.ServeHTTP(w, req)
    
    if w.Code != http.StatusCreated {
        t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
    }
}
```

---

### Step 16-22: Remaining Endpoints

Follow the same pattern for:
- Update department endpoint
- Get department endpoint
- List departments endpoint
- Deactivate department endpoint
- Get department tree endpoint
- Set budget endpoint
- Get budget endpoint

---

## Verification

After each wave, run:
```bash
go build ./...
go vet ./...
go test ./internal/domain/masterdata/...
go test ./internal/application/masterdata/...
go test ./internal/infrastructure/persistence/masterdata/...
go test ./internal/interfaces/http/masterdata/...
```

After all waves, run:
```bash
go test -v ./...
go test -cover ./...
go test -race ./...
```

---

## File Structure

```
internal/
├── domain/
│   └── masterdata/
│       └── entity.go          # Add DepartmentType, BudgetRecord
├── application/
│   └── masterdata/
│       ├── service.go         # Add department methods
│       └── service_test.go    # Add department tests
├── infrastructure/
│   └── persistence/
│       └── masterdata/
│           └── repository.go  # Add budget methods
└── interfaces/
    └── http/
        └── masterdata/
            ├── handler.go     # Add department endpoints
            └── handler_test.go # Add HTTP tests
```

---

## Success Criteria

1. All tests pass
2. No regressions in existing functionality
3. Code coverage >= 80% for new code
4. No security vulnerabilities
5. Performance < 200ms for all endpoints
