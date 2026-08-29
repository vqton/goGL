# Department Module Implementation Plan

**Objective:** Implement a production-ready Department/Organization module for the goGL Vietnamese ERP system, extending the existing `KindDepartment` in the masterdata module with hierarchical structure, cost center integration, budget tracking, employee assignment, and Vietnamese regulatory compliance.

**Duration:** 14 weeks (6 phases)  
**Target:** November 2026

---

## Phase 1: Core Department Entity (Week 1-2)

### Step 1.1: Extend Department Entity

**Context:** The current `Record` struct in `internal/domain/masterdata/entity.go` has a generic `Extra map[string]string` field. Department-specific fields (cost center, manager, budget, etc.) need to be extracted into proper typed fields.

**Tasks:**
- [ ] Add `DepartmentType` enum to entity.go
- [ ] Extend `Record` struct with department-specific fields
- [ ] Add validation constants for department types
- [ ] Ensure backward compatibility with existing `Extra` field

**Files to modify:**
- `internal/domain/masterdata/entity.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/domain/masterdata/...
```

**Exit criteria:**
- All existing tests pass
- New fields compile without errors
- JSON serialization maintains backward compatibility

---

### Step 1.2: Add Department Validation

**Context:** The current `validate()` function in `internal/application/masterdata/service.go` handles generic validation. Department-specific validation rules (BR-D001 to BR-D010) need to be added.

**Tasks:**
- [ ] Add department-specific validation in `validate()` function
- [ ] Implement code format validation (BP-XXXXX)
- [ ] Add name length validation (max 200 chars)
- [ ] Add department type enum validation
- [ ] Add cost center uniqueness check
- [ ] Add manager existence check (employee lookup)

**Files to modify:**
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- All validation rules implemented
- Error messages in Vietnamese and English
- Existing tests still pass

---

### Step 1.3: Implement Hierarchy Logic

**Context:** The current `GroupCode` field provides basic parent-child relationship. Department hierarchy needs cycle detection, level calculation, and tree construction.

**Tasks:**
- [ ] Add `GetDepartmentTree()` method to service interface
- [ ] Implement tree construction from flat list
- [ ] Add level calculation for departments
- [ ] Ensure cycle detection works for department hierarchy
- [ ] Add maximum depth validation (10 levels)

**Files to modify:**
- `internal/domain/masterdata/entity.go` (interface)
- `internal/application/masterdata/service.go` (implementation)

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Tree query returns nested structure
- Cycle detection prevents invalid hierarchies
- Level calculation is accurate

---

### Step 1.4: Create Department Tests

**Context:** Test coverage for department-specific logic needs to be established before moving to the next phase.

**Tasks:**
- [ ] Create test file for department validation
- [ ] Add test cases for all validation rules
- [ ] Add test cases for hierarchy logic
- [ ] Add test cases for tree construction
- [ ] Ensure 80%+ coverage for new code

**Files to create/modify:**
- `internal/application/masterdata/service_test.go`

**Verification:**
```bash
go test -v ./internal/application/masterdata/...
go test -cover ./internal/application/masterdata/...
```

**Exit criteria:**
- All test cases pass
- Coverage >= 80% for department code
- No regressions in existing tests

---

## Phase 2: Cost Center and Budget (Week 3-4)

### Step 2.1: Add Cost Center Field

**Context:** Cost center is a one-to-one mapping with department. The field needs to be added to the entity and validated for uniqueness.

**Tasks:**
- [ ] Add `CostCenterCode` field to Record struct
- [ ] Add validation for cost center format (CC-XXX)
- [ ] Add uniqueness check across departments
- [ ] Add cost center to export/import functionality

**Files to modify:**
- `internal/domain/masterdata/entity.go`
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Cost center field added and validated
- Uniqueness enforced
- Export/import includes cost center

---

### Step 2.2: Implement Budget Tracking

**Context:** Budget is per department per fiscal year. A new table or document structure is needed to store budget records.

**Tasks:**
- [ ] Add `DepartmentBudget` struct to entity
- [ ] Add `budget_records` table to migrations
- [ ] Implement `SetBudget()` method
- [ ] Implement `GetBudget()` method
- [ ] Add budget validation (non-negative, valid fiscal year)

**Files to modify:**
- `internal/domain/masterdata/entity.go`
- `internal/infrastructure/db/migrate.go`
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Budget can be set and retrieved
- Validation prevents invalid budgets
- Migration creates budget table

---

### Step 2.3: Create Budget Report Method

**Context:** Finance managers need to see budget vs actual expenses per department. This requires integration with GL data.

**Tasks:**
- [ ] Add `GetBudgetReport()` method to service
- [ ] Implement budget vs actual calculation
- [ ] Add variance calculation (amount and percentage)
- [ ] Support filtering by fiscal year and period

**Files to modify:**
- `internal/domain/masterdata/entity.go`
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Budget report returns accurate data
- Variance calculation is correct
- Filtering works as expected

---

### Step 2.4: Add Budget API Endpoints

**Context:** HTTP endpoints are needed for setting and retrieving budgets.

**Tasks:**
- [ ] Add `POST /master-data/department/:code/budget` endpoint
- [ ] Add `GET /master-data/department/:code/budget` endpoint
- [ ] Add `GET /master-data/department/report/cost` endpoint
- [ ] Add proper error handling and validation

**Files to modify:**
- `internal/interfaces/http/masterdata/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/interfaces/http/masterdata/...
```

**Exit criteria:**
- All endpoints respond correctly
- Error handling is consistent
- API documentation updated

---

## Phase 3: Employee Integration (Week 5-6)

### Step 3.1: Add Employee Count Calculation

**Context:** Department needs to show real-time employee count. This can be calculated from employee records.

**Tasks:**
- [ ] Add `EmployeeCount` field to Record struct
- [ ] Implement `CalculateEmployeeCount()` method
- [ ] Update count on employee assignment/transfer
- [ ] Add count to tree view response

**Files to modify:**
- `internal/domain/masterdata/entity.go`
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Employee count is accurate
- Count updates in real-time
- Tree view shows count

---

### Step 3.2: Implement Department Transfer

**Context:** Employees need to be transferable between departments with audit trail.

**Tasks:**
- [ ] Add `TransferEmployee()` method to service
- [ ] Add transfer audit trail logging
- [ ] Add effective date validation
- [ ] Add reason code validation (min 10 chars)

**Files to modify:**
- `internal/application/masterdata/service.go`
- `internal/infrastructure/persistence/masterdata/repository.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Employee transfer works correctly
- Audit trail is logged
- Effective date is respected

---

### Step 3.3: Create Employee List Endpoint

**Context:** HR managers need to see employees per department.

**Tasks:**
- [ ] Add `GET /master-data/department/:code/employees` endpoint
- [ ] Implement employee list filtering by department
- [ ] Add pagination support
- [ ] Add employee count to department response

**Files to modify:**
- `internal/interfaces/http/masterdata/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/interfaces/http/masterdata/...
```

**Exit criteria:**
- Employee list endpoint works
- Pagination works correctly
- Count is accurate

---

## Phase 4: Reporting and Export (Week 7-8)

### Step 4.1: Implement Cost Report Generation

**Context:** Finance managers need departmental cost reports for Circular 99/2025 compliance.

**Tasks:**
- [ ] Add `GenerateCostReport()` method
- [ ] Implement cost allocation by department
- [ ] Add GL account mapping
- [ ] Add export format support (PDF, Excel)

**Files to modify:**
- `internal/application/masterdata/service.go`
- `internal/interfaces/http/masterdata/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/application/masterdata/...
```

**Exit criteria:**
- Cost report generation works
- Export formats are correct
- Compliance with Circular 99/2025

---

### Step 4.2: Add CSV/Excel Export

**Context:** Departments need to be exportable with hierarchy information.

**Tasks:**
- [ ] Update `exportCSV()` to include department fields
- [ ] Add hierarchy path to export
- [ ] Add Excel export support
- [ ] Add export filtering options

**Files to modify:**
- `internal/interfaces/http/masterdata/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/interfaces/http/masterdata/...
```

**Exit criteria:**
- Export includes all department fields
- Hierarchy path is correct
- Excel export works

---

## Phase 5: UI/UX Implementation (Week 9-12)

### Step 5.1: Department List Screen

**Context:** HR managers need a list view of all departments with filtering and search.

**Tasks:**
- [ ] Create `web/templates/masterdata/department_list.html`
- [ ] Add department table with sorting
- [ ] Add filter controls (status, type, search)
- [ ] Add pagination controls
- [ ] Add responsive design

**Files to create:**
- `web/templates/masterdata/department_list.html`

**Verification:**
- Manual testing in browser
- Verify all filters work
- Verify pagination works

**Exit criteria:**
- List view displays correctly
- All filters work
- Responsive on mobile/tablet

---

### Step 5.2: Department Tree View

**Context:** HR managers need a visual tree view of the department hierarchy.

**Tasks:**
- [ ] Create `web/templates/masterdata/department_tree.html`
- [ ] Implement expandable/collapsible tree nodes
- [ ] Add drag-and-drop reorganization
- [ ] Add employee count badges
- [ ] Add status indicators

**Files to create:**
- `web/templates/masterdata/department_tree.html`

**Verification:**
- Manual testing in browser
- Verify tree expansion works
- Verify drag-and-drop works

**Exit criteria:**
- Tree view displays correctly
- All interactions work
- Responsive on mobile/tablet

---

### Step 5.3: Department Create/Edit Form

**Context:** HR managers need a form to create and edit departments.

**Tasks:**
- [ ] Create `web/templates/masterdata/department_form.html`
- [ ] Add form fields per BRD spec
- [ ] Add client-side validation
- [ ] Add parent department dropdown
- [ ] Add manager employee dropdown
- [ ] Add cost center dropdown

**Files to create:**
- `web/templates/masterdata/department_form.html`

**Verification:**
- Manual testing in browser
- Verify all validations work
- Verify form submission works

**Exit criteria:**
- Form displays correctly
- All validations work
- Form submission succeeds

---

### Step 5.4: Department Detail Screen

**Context:** HR managers need to view full department information.

**Tasks:**
- [ ] Create `web/templates/masterdata/department_detail.html`
- [ ] Add department info display
- [ ] Add employee list display
- [ ] Add budget vs actual display
- [ ] Add audit trail display

**Files to create:**
- `web/templates/masterdata/department_detail.html`

**Verification:**
- Manual testing in browser
- Verify all data displays correctly
- Verify links work

**Exit criteria:**
- Detail view displays correctly
- All data is accurate
- Links work as expected

---

### Step 5.5: Cost Report Screen

**Context:** Finance managers need to view budget vs actual expenses.

**Tasks:**
- [ ] Create `web/templates/masterdata/department_cost.html`
- [ ] Add budget vs actual table
- [ ] Add variance highlighting
- [ ] Add export buttons (PDF, Excel)
- [ ] Add chart visualization (optional)

**Files to create:**
- `web/templates/masterdata/department_cost.html`

**Verification:**
- Manual testing in browser
- Verify data accuracy
- Verify export works

**Exit criteria:**
- Cost report displays correctly
- Data is accurate
- Export works as expected

---

## Phase 6: Integration and Polish (Week 13-14)

### Step 6.1: Integrate with Payroll Module

**Context:** Payroll needs department data for salary reports.

**Tasks:**
- [ ] Add department lookup in payroll handler
- [ ] Add department field to salary reports
- [ ] Add department filtering to payroll queries

**Files to modify:**
- `internal/interfaces/http/payroll/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/interfaces/http/payroll/...
```

**Exit criteria:**
- Payroll reports include department data
- Filtering works correctly

---

### Step 6.2: Integrate with Fixed Assets

**Context:** Fixed assets need department assignment for asset tracking.

**Tasks:**
- [ ] Add department lookup in fixed asset handler
- [ ] Add department field to asset register
- [ ] Add department filtering to asset queries

**Files to modify:**
- `internal/interfaces/http/fixedasset/handler.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./internal/interfaces/http/fixedasset/...
```

**Exit criteria:**
- Asset register includes department data
- Filtering works correctly

---

### Step 6.3: API Documentation

**Context:** API documentation is needed for developers and integration.

**Tasks:**
- [ ] Create OpenAPI/Swagger specification
- [ ] Document all endpoints
- [ ] Add request/response examples
- [ ] Add error code documentation

**Files to create:**
- `docs/api/department.yaml`

**Verification:**
- OpenAPI validator passes
- All endpoints documented
- Examples are correct

**Exit criteria:**
- API documentation is complete
- Documentation is accurate
- Examples work as documented

---

### Step 6.4: Performance Optimization

**Context:** Department queries need to be optimized for production use.

**Tasks:**
- [ ] Add database indexes for common queries
- [ ] Implement query result caching
- [ ] Optimize tree construction algorithm
- [ ] Add pagination to large result sets

**Files to modify:**
- `internal/infrastructure/persistence/masterdata/repository.go`
- `internal/application/masterdata/service.go`

**Verification:**
```bash
go build ./...
go vet ./...
go test ./...
```

**Exit criteria:**
- Query response time < 200ms
- Tree query response time < 100ms
- No memory leaks

---

### Step 6.5: Security Review

**Context:** All endpoints need RBAC enforcement and security review.

**Tasks:**
- [ ] Verify RBAC enforcement on all endpoints
- [ ] Add input sanitization
- [ ] Add rate limiting
- [ ] Add audit logging for all mutations

**Files to modify:**
- `internal/interfaces/http/masterdata/handler.go`
- `internal/infrastructure/authorization/`

**Verification:**
- Manual security testing
- Penetration testing
- Code review

**Exit criteria:**
- All endpoints are protected
- Input is sanitized
- Audit trail is complete

---

## Verification Commands

After each phase, run:
```bash
go build ./...
go vet ./...
go test ./...
```

After all phases, run:
```bash
go test -v ./...
go test -cover ./...
go test -race ./...
```

## Rollback Strategy

Each step is designed to be reversible:
- Entity changes: Revert struct fields
- Service changes: Revert method implementations
- Repository changes: Revert query changes
- Handler changes: Revert endpoint registrations
- UI changes: Revert template files

## Dependencies

- Phase 1 depends on: None
- Phase 2 depends on: Phase 1
- Phase 3 depends on: Phase 1
- Phase 4 depends on: Phase 2
- Phase 5 depends on: Phase 1-4
- Phase 6 depends on: Phase 1-5

## Parallel Work

- Phase 2 and Phase 3 can run in parallel after Phase 1
- Phase 4 depends on Phase 2
- Phase 5 depends on all previous phases
- Phase 6 can start after Phase 5

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Circular 99/2025 interpretation | Consult with Vietnamese accounting experts |
| Performance with large hierarchies | Implement pagination and caching early |
| Integration with existing modules | Early integration testing |
| UI/UX complexity | Use existing Tailwind components |
