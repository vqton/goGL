# Department Module - Implementation Roadmap

**Document Version:** 1.0  
**Date:** August 2026  
**Module:** Department/Organization (Phòng ban/Cơ cấu tổ chức)  
**Status:** Planning

---

## 1. Implementation Overview

### 1.1 Scope

This roadmap covers the full implementation of the Department/Organization module for the goGL Vietnamese ERP system, including:
- Department entity with hierarchical structure
- Cost center integration
- Budget tracking
- Employee assignment
- Reporting and compliance
- UI/UX implementation

### 1.2 Dependencies

| Dependency | Type | Status | Impact |
|------------|------|--------|--------|
| masterdata module | Internal | Exists | Foundation - extend KindDepartment |
| user module | Internal | Exists | Employee lookup |
| auth module | Internal | Exists | RBAC enforcement |
| options module | Internal | Exists | Department type enum |
| audit module | Internal | Exists | Change logging |

---

## 2. Phased Implementation

### Phase 1: Core Department Entity (Week 1-2)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T1.1 | Extend Department entity with new fields | All fields per BRD spec | `internal/domain/masterdata/entity.go` |
| T1.2 | Add department-specific validation | Business rules BR-D001 to BR-D010 | `internal/application/masterdata/service.go` |
| T1.3 | Implement hierarchy logic | Parent-child, cycle detection, level calc | `internal/application/masterdata/service.go` |
| T1.4 | Add tree query method | GetDepartmentTree returns nested structure | `internal/infrastructure/persistence/masterdata/repository.go` |
| T1.5 | Create department-specific tests | 100% coverage of new logic | `internal/application/masterdata/service_test.go` |

**Milestone:** Department CRUD with hierarchy works end-to-end.

### Phase 2: Cost Center and Budget (Week 3-4)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T2.1 | Add cost center field and validation | Unique per department | `internal/domain/masterdata/entity.go` |
| T2.2 | Implement budget tracking | Set/Get budget per department/year | `internal/application/masterdata/service.go` |
| T2.3 | Create budget report method | Budget vs actual calculation | `internal/application/masterdata/service.go` |
| T2.4 | Add budget API endpoints | POST/GET for budget | `internal/interfaces/http/masterdata/handler.go` |
| T2.5 | Create budget tests | Unit and integration tests | `internal/application/masterdata/service_test.go` |

**Milestone:** Cost center and budget tracking functional.

### Phase 3: Employee Integration (Week 5-6)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T3.1 | Add employee count calculation | Real-time count per department | `internal/application/masterdata/service.go` |
| T3.2 | Implement department transfer | Employee moves between departments | `internal/application/masterdata/service.go` |
| T3.3 | Create employee list endpoint | List employees by department | `internal/interfaces/http/masterdata/handler.go` |
| T3.4 | Add transfer audit trail | Who/when/why logged | `internal/infrastructure/persistence/masterdata/repository.go` |
| T3.5 | Create employee integration tests | End-to-end transfer tests | `internal/application/masterdata/service_test.go` |

**Milestone:** Employee-department integration complete.

### Phase 4: Reporting and Export (Week 7-8)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T4.1 | Implement cost report generation | Budget vs actual by department | `internal/application/masterdata/service.go` |
| T4.2 | Add CSV/Excel export | Export departments with hierarchy | `internal/interfaces/http/masterdata/handler.go` |
| T4.3 | Create PDF org chart export | Visual org chart generation | `internal/interfaces/http/masterdata/handler.go` |
| T4.4 | Add Circular 99/2025 compliance | Departmental cost allocation reports | `internal/application/masterdata/service.go` |
| T4.5 | Create reporting tests | Report accuracy tests | `internal/application/masterdata/service_test.go` |

**Milestone:** Reporting and compliance complete.

### Phase 5: UI/UX Implementation (Week 9-12)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T5.1 | Department list screen | Table view with filters | `web/templates/masterdata/department_list.html` |
| T5.2 | Department tree view | Interactive tree component | `web/templates/masterdata/department_tree.html` |
| T5.3 | Department create/edit form | Form with validation | `web/templates/masterdata/department_form.html` |
| T5.4 | Department detail screen | Full department info display | `web/templates/masterdata/department_detail.html` |
| T5.5 | Cost report screen | Budget vs actual visualization | `web/templates/masterdata/department_cost.html` |
| T5.6 | Responsive design | Mobile/tablet support | All templates |

**Milestone:** Complete UI/UX implementation.

### Phase 6: Integration and Polish (Week 13-14)

| Task | Description | Acceptance Criteria | Files |
|------|-------------|---------------------|-------|
| T6.1 | Integrate with payroll module | Department salary reports | `internal/interfaces/http/payroll/handler.go` |
| T6.2 | Integrate with fixed assets | Department asset register | `internal/interfaces/http/fixedasset/handler.go` |
| T6.3 | Add API documentation | OpenAPI/Swagger spec | `docs/api/department.yaml` |
| T6.4 | Performance optimization | Query optimization, caching | Various |
| T6.5 | Security review | RBAC enforcement audit | All handlers |

**Milestone:** Full integration and production readiness.

---

## 3. Timeline Summary

```
Week  1-2:  [████████████] Phase 1: Core Department Entity
Week  3-4:  [████████████] Phase 2: Cost Center and Budget
Week  5-6:  [████████████] Phase 3: Employee Integration
Week  7-8:  [████████████] Phase 4: Reporting and Export
Week  9-12: [████████████████████████] Phase 5: UI/UX Implementation
Week 13-14: [████████████] Phase 6: Integration and Polish
```

**Total Duration:** 14 weeks  
**Target Completion:** November 2026

---

## 4. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Circular 99/2025 interpretation | Medium | High | Consult with Vietnamese accounting experts |
| Performance with large hierarchies | Low | Medium | Implement pagination and caching |
| Integration with existing modules | Medium | Medium | Early integration testing |
| UI/UX complexity | Medium | Low | Use existing Tailwind components |

---

## 5. Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Department CRUD response time | < 200ms | API monitoring |
| Tree query response time | < 100ms | API monitoring |
| Test coverage | > 80% | go test -cover |
| Vietnamese regulatory compliance | 100% | Audit checklist |
| User adoption | 90% of HR/Finance users | Usage analytics |
