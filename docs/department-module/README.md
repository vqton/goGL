# Department Module - Executive Summary

**Document Version:** 1.0  
**Date:** August 2026  
**Module:** Department/Organization (Phòng ban/Cơ cấu tổ chức)  
**Status:** Ready for Review

---

## Overview

This document summarizes the complete analysis and specification for the Department/Organization module for the goGL Vietnamese ERP system. The module enables enterprises to model their organizational hierarchy, assign cost centers, track departmental budgets, and integrate with payroll, fixed assets, and cost accounting modules.

---

## Key Findings

### 1. Vietnamese Accounting Standards Compliance

| Regulation | Requirement | Module Impact |
|------------|-------------|---------------|
| **Circular 99/2025/TT-BTC** | Departmental cost allocation mandatory | Department hierarchy with cost center mapping |
| **TT 200/2014/TT-BTC** | Department codes in chart of accounts | Department code format standards |
| **NĐ 123/2020/NĐ-CP** | Department info on invoices | Department field on invoice documents |
| **Labor Code 2019** | Employee-Department relationship | Department-Employee linkage |

### 2. Market Comparison

| Feature | MISA AMIS | FAST Accounting | BRAVO ERP | goGL Target |
|---------|-----------|-----------------|-----------|-------------|
| Department Hierarchy | Multi-level tree | Flat + cost center | Full org chart | Multi-level tree |
| Cost Center | Yes | Yes | Yes | Yes |
| Budget Tracking | Per department | Per department | Per department | Per department |
| Manager Assignment | Yes | Yes | Yes | Yes |
| Employee Count | Yes | Yes | Yes | Yes |
| Department Transfer | Yes | Limited | Yes | Yes |
| Audit Trail | Yes | Yes | Yes | Yes |

### 3. Current State

The masterdata module already has `KindDepartment` with basic CRUD operations. The implementation will extend this existing foundation rather than create a new standalone module.

---

## Deliverables

| Document | Description | Location |
|----------|-------------|----------|
| **BRD.md** | Business Requirements Document | `docs/department-module/BRD.md` |
| **SPEC.md** | Detailed Specification with Use Cases | `docs/department-module/SPEC.md` |
| **UI-WIREFRAMES.md** | UI/UX Wireframes and Component Specs | `docs/department-module/UI-WIREFRAMES.md` |
| **ROADMAP.md** | Implementation Roadmap (14 weeks) | `docs/department-module/ROADMAP.md` |

---

## Implementation Plan

### Phases

1. **Phase 1 (Week 1-2):** Core Department Entity
2. **Phase 2 (Week 3-4):** Cost Center and Budget
3. **Phase 3 (Week 5-6):** Employee Integration
4. **Phase 4 (Week 7-8):** Reporting and Export
5. **Phase 5 (Week 9-12):** UI/UX Implementation
6. **Phase 6 (Week 13-14):** Integration and Polish

### Total Duration: 14 weeks

### Target Completion: November 2026

---

## Key Design Decisions

### 1. Extend Existing Module

Decision: Extend `KindDepartment` in the masterdata module rather than create a standalone department module.

Rationale:
- Reuses existing CRUD infrastructure
- Maintains consistency with other master data types
- Reduces development time
- Leverages existing audit trail

### 2. Hierarchical Structure

Decision: Implement multi-level tree hierarchy with maximum depth of 10 levels.

Rationale:
- Matches Vietnamese enterprise structures
- Supports Ban Giám đốc → Phòng ban → Bộ phận → Tổ nhóm
- Enables flexible organizational modeling

### 3. Cost Center Integration

Decision: Embed cost center code in department entity (one-to-one mapping).

Rationale:
- Simplifies queries
- Reduces joins
- Matches FAST and MISA approach
- Easy to extend to separate module later if needed

---

## Open Questions

| ID | Question | Impact | Owner |
|----|----------|--------|-------|
| OQ-001 | Should department support multi-company hierarchy? | High | Product |
| OQ-002 | Should budget approval workflow be included in v1? | Medium | Product |
| OQ-003 | Should department history be versioned? | Medium | Data |
| OQ-004 | Should cost center be a separate module? | Low | Architecture |

---

## Next Steps

1. **Review and approve** BRD and specification documents
2. **Resolve open questions** with product owner
3. **Begin Phase 1** implementation (Core Department Entity)
4. **Schedule weekly reviews** to track progress

---

## References

### Vietnamese Accounting Standards
- Circular 99/2025/TT-BTC (effective Jan 1, 2026)
- TT 200/2014/TT-BTC (accounting chart and reporting)
- NĐ 123/2020/NĐ-CP (electronic invoices)
- NĐ 126/2020/NĐ-CP (tax administration)

### Market Leaders
- MISA AMIS (amismisa.vn) - 45% market share, 250,000+ enterprises
- FAST Accounting (fast.com.vn) - 25% market share, 23,000+ customers
- BRAVO ERP (bravo.com.vn) - 10% market share, enterprise segment

### Industry Sources
- Vietnam Association of Accountants and Auditors (VAA)
- Vietnam Tax Consultants' Association (VTCA)
- Big 4 Vietnam (E&Y, PwC, Deloitte, KPMG)
