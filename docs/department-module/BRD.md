# Business Requirements Document: Department/Organization Module

**Document Version:** 1.0  
**Date:** August 2026  
**Module:** Department/Organization (Phòng ban/Cơ cấu tổ chức)  
**Status:** Draft for Review

---

## Executive Summary

This BRD defines the requirements for a production-ready Department/Organization module for the goGL Vietnamese ERP system. The module enables enterprises to model their organizational hierarchy, assign cost centers, track departmental budgets, and integrate with payroll, fixed assets, and cost accounting modules. The design is benchmarked against MISA AMIS, FAST Accounting, and BRAVO ERP with compliance to Circular 99/2025 and TT 200/2014.

---

## 1. Business Context

### 1.1 Vietnamese Accounting Regulatory Requirements

| Regulation | Requirement | Module Impact |
|------------|-------------|---------------|
| **Circular 99/2025/TT-BTC** (effective Jan 1, 2026) | New accounting regime for enterprises; departmental cost allocation mandatory for management reporting | Department hierarchy with cost center mapping |
| **TT 200/2014/TT-BTC** | Accounting chart organization; department codes in chart of accounts | Department code format standards |
| **NĐ 123/2020/NĐ-CP** | Electronic invoice requirements; department info on invoices | Department field on invoice documents |
| **NĐ 126/2020/NĐ-CP** | Tax administration; cost allocation for CIT deduction | Department-level cost tracking for tax |
| **Labor Code 2019** | Employee-Department relationship; labor contract registration | Department-Employee linkage |

### 1.2 Market Comparison

| Feature | MISA AMIS | FAST Accounting | BRAVO ERP | goGL Target |
|---------|-----------|-----------------|-----------|-------------|
| **Department Hierarchy** | Multi-level tree | Flat + cost center | Full org chart | ✅ Multi-level tree |
| **Cost Center** | Yes (kpi_000) | Yes (phong_ban) | Yes (trung_tam) | ✅ Yes |
| **Budget Tracking** | Per department | Per department | Per department | ✅ Per department |
| **Manager Assignment** | Yes | Yes | Yes | ✅ Yes |
| **Employee Count** | Yes | Yes | Yes | ✅ Yes |
| **Department Transfer** | Yes | Limited | Yes | ✅ Yes |
| **Audit Trail** | Yes | Yes | Yes | ✅ Yes |
| **API Integration** | REST | REST | REST | ✅ REST |

### 1.3 Business Goals

1. **Enable multi-level organizational hierarchy** supporting Vietnamese enterprise structures (Ban Giám đốc → Phòng ban → Bộ phận → Tổ nhóm)
2. **Integrate with cost accounting** for departmental cost allocation and budget tracking
3. **Comply with Vietnamese accounting standards** (Circular 99/2025, TT 200/2014)
4. **Support department transfers** with full audit trail and effective dating
5. **Enable departmental reporting** for management and statutory purposes

---

## 2. User Roles & Personas

### 2.1 Primary Users

| Role | Responsibilities | Key Actions |
|------|------------------|-------------|
| **HR Manager** | Manage department structure, assign managers | Create/Edit/Deactivate departments, Assign managers |
| **Finance Manager** | Track departmental budgets, allocate costs | Set budgets, View cost reports, Allocate expenses |
| **Department Head** | View own department, request changes | View department info, Request transfers |
| **System Admin** | Configure module, manage permissions | Set up department types, Configure approval workflows |

### 2.2 User Stories

#### Epic 1: Department Hierarchy Management

| ID | User Story | Priority | Acceptance Criteria |
|----|------------|----------|---------------------|
| US-D001 | As HR Manager, I want to create a new department with parent, code, name, and manager | P0 | Department created with auto-generated code (BP-XXXXX), parent validated, manager exists |
| US-D002 | As HR Manager, I want to view the department hierarchy as a tree | P0 | Tree view shows all levels, expandable/collapsible, shows employee count |
| US-D003 | As HR Manager, I want to move a department to a different parent | P1 | Validation prevents cycles, effective date required, audit trail logged |
| US-D004 | As HR Manager, I want to deactivate a department with reason | P1 | Cannot deactivate if active employees, reason required, cascade to sub-departments |
| US-D005 | As HR Manager, I want to reactivate a deactivated department | P2 | Parent must be active, validation of dependencies |

#### Epic 2: Cost Center & Budget

| ID | User Story | Priority | Acceptance Criteria |
|----|------------|----------|---------------------|
| US-D010 | As Finance Manager, I want to assign a cost center code to each department | P0 | Cost center unique, linked to GL account, valid format (CC-XXX) |
| US-D011 | As Finance Manager, I want to set annual budget per department | P0 | Budget amount positive, fiscal year validated, version history maintained |
| US-D012 | As Finance Manager, I want to view budget vs actual per department | P1 | Real-time calculation, variance highlighting, export to Excel |
| US-D013 | As Finance Manager, I want to transfer budget between departments | P2 | Both departments active, approval workflow, audit trail |

#### Epic 3: Employee Integration

| ID | User Story | Priority | Acceptance Criteria |
|----|------------|----------|---------------------|
| US-D020 | As HR Manager, I want to assign employees to departments | P0 | Employee exists, department active, effective date, one primary department |
| US-D021 | As HR Manager, I want to transfer an employee between departments | P1 | Old department released, new department assigned, effective date, reason code |
| US-D022 | As HR Manager, I want to view employee count per department | P1 | Real-time count, filter by status, drill-down to employee list |

#### Epic 4: Reporting & Compliance

| ID | User Story | Priority | Acceptance Criteria |
|----|------------|----------|---------------------|
| US-D030 | As Finance Manager, I want departmental cost reports per Circular 99/2025 | P0 | Cost allocation by department, GL account mapping, export format compliant |
| US-D031 | As Finance Manager, I want organizational chart export | P1 | PDF/PNG export, Vietnamese labels, hierarchical layout |
| US-D032 | As System Admin, I want audit trail of all department changes | P0 | Who/what/when/why logged, immutable, searchable |

---

## 3. Functional Requirements

### 3.1 Department Entity

```go
// Department represents an organizational unit in the enterprise hierarchy.
type Department struct {
    ID               string            `json:"id"`
    Code             string            `json:"code"`              // BP-XXXXX (auto-generated)
    Name             string            `json:"name"`              // Vietnamese name (required)
    NameEN           string            `json:"name_en,omitempty"` // English name (optional)
    ParentCode       string            `json:"parent_code,omitempty"` // Parent department code
    Level            int               `json:"level"`             // 0=root, 1=sub, etc.
    CostCenterCode   string            `json:"cost_center_code,omitempty"` // Linked cost center
    ManagerCode      string            `json:"manager_code,omitempty"`     // Employee code of manager
    BudgetAnnual     int64             `json:"budget_annual,omitempty"`    // Annual budget VND
    EmployeeCount    int               `json:"employee_count"`            // Calculated field
    State            string            `json:"state"`             // active/inactive
    DepartmentType   string            `json:"department_type"`   // executive/operational/support
    Phone            string            `json:"phone,omitempty"`
    Email            string            `json:"email,omitempty"`
    Address          string            `json:"address,omitempty"`
    ValidFrom        string            `json:"valid_from,omitempty"`
    ValidTo          string            `json:"valid_to,omitempty"`
    Extra            map[string]string `json:"extra,omitempty"`   // Extensible attributes
    CreatedBy        string            `json:"created_by,omitempty"`
    CreatedAt        string            `json:"created_at"`
    UpdatedBy        string            `json:"updated_by,omitempty"`
    UpdatedAt        string            `json:"updated_at"`
    DeactivatedBy    string            `json:"deactivated_by,omitempty"`
    DeactivatedAt    string            `json:"deactivated_at,omitempty"`
    DeactivateReason string            `json:"deactivate_reason,omitempty"`
}
```

### 3.2 Business Rules

| Rule ID | Description | Validation |
|---------|-------------|------------|
| BR-D001 | Department code auto-generated with prefix BP-XXXXX | Format: `BP-[0-9]{5}` |
| BR-D002 | Department name required, max 200 chars | Non-empty, trimmed |
| BR-D003 | Parent must exist, be active, same tenant | Cycle detection (max depth 32) |
| BR-D004 | Maximum hierarchy depth: 10 levels | Prevent excessive nesting |
| BR-D005 | Cost center code must be unique across departments | One-to-one mapping |
| BR-D006 | Manager must be an active employee | Employee lookup, state=active |
| BR-D007 | Cannot deactivate department with active employees | Employee count check |
| BR-D008 | Budget must be non-negative | Amount >= 0 |
| BR-D009 | Department type must be: executive/operational/support | Enum validation |
| BR-D010 | Deactivation requires reason (min 10 chars) | Reason field mandatory |

### 3.3 API Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/api/v1/masterdata/department` | Create department | catalog.write |
| `GET` | `/api/v1/masterdata/department/:code` | Get department by code | catalog.read |
| `PUT` | `/api/v1/masterdata/department/:code` | Update department | catalog.write |
| `DELETE` | `/api/v1/masterdata/department/:code` | Soft delete (deactivate) | catalog.write |
| `GET` | `/api/v1/masterdata/department` | List departments (with filters) | catalog.read |
| `GET` | `/api/v1/masterdata/department/tree` | Get department tree | catalog.read |
| `POST` | `/api/v1/masterdata/department/:code/activate` | Activate department | catalog.write |
| `POST` | `/api/v1/masterdata/department/:code/budget` | Set/update budget | catalog.write |
| `GET` | `/api/v1/masterdata/department/:code/budget` | Get budget report | catalog.read |
| `POST` | `/api/v1/masterdata/department/:code/transfer` | Transfer department (change parent) | catalog.write |
| `GET` | `/api/v1/masterdata/department/:code/employees` | List employees in department | catalog.read |
| `GET` | `/api/v1/masterdata/department/report/cost` | Departmental cost report | catalog.read |
| `GET` | `/api/v1/masterdata/department/export` | Export departments (CSV/Excel) | catalog.read |
| `POST` | `/api/v1/masterdata/department/import` | Import departments (CSV) | catalog.import |

### 3.4 Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Department Management Flow                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │   HR     │    │  Finance │    │  Payroll │    │  Fixed   │ │
│  │ Manager  │    │ Manager  │    │  Module  │    │  Assets  │ │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘ │
│       │               │               │               │         │
│       ▼               ▼               ▼               ▼         │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Department Module                         │ │
│  │  • Hierarchy Management                                     │ │
│  │  • Cost Center Assignment                                   │ │
│  │  • Budget Tracking                                          │ │
│  │  • Employee Assignment                                      │ │
│  └─────────────────────────────────────────────────────────────┘ │
│       │               │               │               │         │
│       ▼               ▼               ▼               ▼         │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │  SQLite  │    │    GL    │    │  Salary  │    │  Asset   │ │
│  │   DB     │    │ Accounts │    │  Reports │    │ Register │ │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Non-Functional Requirements

| Category | Requirement | Target |
|----------|-------------|--------|
| **Performance** | Department tree query | < 100ms for 1000 nodes |
| **Performance** | Employee list by department | < 200ms for 10,000 employees |
| **Scalability** | Maximum departments | 10,000 per tenant |
| **Scalability** | Maximum hierarchy depth | 10 levels |
| **Availability** | Department API uptime | 99.9% |
| **Security** | RBAC enforcement | All endpoints protected |
| **Audit** | Change logging | 100% of mutations logged |
| **Compliance** | Vietnamese accounting standards | Circular 99/2025, TT 200/2014 |

---

## 5. Integration Points

### 5.1 Upstream Dependencies

| Module | Integration | Data Flow |
|--------|-------------|-----------|
| **User Module** | Employee lookup | Department → Employee assignment |
| **Options Module** | Department types | Options → Department type enum |
| **Auth Module** | Permission check | Auth → RBAC enforcement |

### 5.2 Downstream Consumers

| Module | Integration | Data Flow |
|--------|-------------|-----------|
| **Payroll Module** | Department salary reports | Department → Payroll aggregation |
| **Fixed Assets Module** | Department asset register | Department → Asset location |
| **Cost Accounting** | Department cost allocation | Department → Cost center mapping |
| **General Ledger** | Departmental journal entries | Department → GL department code |
| **Reporting Module** | Departmental financial reports | Department → Report dimensions |

---

## 6. Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Department CRUD response time | < 200ms | API monitoring |
| Tree query response time | < 100ms | API monitoring |
| Data accuracy | 100% | Unit tests + integration tests |
| Vietnamese regulatory compliance | 100% | Audit checklist |
| User adoption | 90% of HR/Finance users | Usage analytics |
| Bug rate | < 1 bug per 1000 transactions | Issue tracking |

---

## 7. Open Questions

| ID | Question | Impact | Owner |
|----|----------|--------|-------|
| OQ-001 | Should department support multi-company (consolidated) hierarchy? | High | Product |
| OQ-002 | Should budget approval workflow be included in v1? | Medium | Product |
| OQ-003 | Should department history be versioned (SCD Type 2)? | Medium | Data |
| OQ-004 | Should cost center be a separate module or embedded? | Low | Architecture |

---

## Appendices

### Appendix A: Vietnamese Accounting Standards Reference

1. **Circular 99/2025/TT-BTC** - New accounting regime effective Jan 1, 2026
2. **TT 200/2014/TT-BTC** - Accounting chart and reporting standards
3. **NĐ 123/2020/NĐ-CP** - Electronic invoice requirements
4. **NĐ 126/2020/NĐ-CP** - Tax administration procedures

### Appendix B: Competitive Analysis Sources

1. **MISA AMIS** - amismisa.vn (45% market share, 250,000+ enterprises)
2. **FAST Accounting** - fast.com.vn (25% market share, 23,000+ customers)
3. **BRAVO ERP** - bravo.com.vn (10% market share, enterprise segment)

### Appendix C: Related Documents

- `docs/admin-config/SPEC.md` - Admin configuration specification
- `docs/admin-config/ROADMAP.md` - Implementation roadmap
- `internal/domain/masterdata/entity.go` - Current masterdata entity
