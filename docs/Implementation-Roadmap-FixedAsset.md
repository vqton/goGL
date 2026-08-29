# Implementation Roadmap - Fixed Asset Module

**Version**: 1.0  
**Date**: 2026-08-29  
**Target**: Production-Ready FA Module

---

## Phase 0: Planning & Design (Week 1)

### Tasks
- [ ] Finalize BRD with stakeholders
- [ ] Review Vietnamese accounting standards
- [ ] Design database schema
- [ ] Design API contracts
- [ ] Create wireframes
- [ ] Set up development environment

### Deliverables
- [x] Analysis document (docs/fixed-asset-analysis.md)
- [x] BRD (docs/BRD-FixedAsset-Module.md)
- [x] Use Cases (docs/UseCases-FixedAsset.md)
- [ ] Database schema
- [ ] API specification (OpenAPI)

### Exit Criteria
- BRD approved by BA Lead and Chief Accountant
- Technical design reviewed

---

## Phase 1: Entity & Repository (Week 2)

### Tasks
- [ ] Enhance FixedAsset entity with all required fields
- [ ] Add depreciation-related fields
- [ ] Add classification fields per Circular 45
- [ ] Create Repository interface
- [ ] Implement SQLite repository
- [ ] Create migration tables
- [ ] Write unit tests

### Deliverables
- [ ] Enhanced entity.go
- [ ] Repository implementation
- [ ] Migration scripts
- [ ] Unit tests (80% coverage)

### Exit Criteria
- All entity fields defined
- Repository CRUD operations working
- Unit tests passing

---

## Phase 2: Service Layer - Lifecycle (Week 3)

### Tasks
- [ ] Implement CreateAsset method
- [ ] Implement GetAsset method
- [ ] Implement UpdateAsset method
- [ ] Implement DeleteAsset method
- [ ] Implement ListAssets method
- [ ] Add validation rules
- [ ] Add code generation (FA-XXXXX)
- [ ] Write unit tests

### Deliverables
- [ ] Service implementation
- [ ] Validation logic
- [ ] Unit tests

### Exit Criteria
- CRUD operations working
- Validation rules enforced
- Unit tests passing

---

## Phase 3: Depreciation Engine (Week 4)

### Tasks
- [ ] Implement straight-line calculation
- [ ] Implement declining-balance calculation
- [ ] Implement units-of-output calculation
- [ ] Implement monthly depreciation posting
- [ ] Implement depreciation schedule generation
- [ ] Add depreciation life validation (Annex I)
- [ ] Write unit tests with known values

### Deliverables
- [ ] Depreciation calculator
- [ ] Depreciation poster
- [ ] Unit tests

### Exit Criteria
- All three methods produce correct results
- Depreciation posting to correct accounts
- Unit tests passing with known values

---

## Phase 4: Business Operations (Week 5)

### Tasks
- [ ] Implement asset transfer
- [ ] Implement asset revaluation
- [ ] Implement liquidation workflow
- [ ] Implement state transitions
- [ ] Add approval workflows
- [ ] Write unit tests

### Deliverables
- [ ] Transfer service
- [ ] Revaluation service
- [ ] Liquidation service
- [ ] Unit tests

### Exit Criteria
- All operations working correctly
- State transitions valid
- Unit tests passing

---

## Phase 5: HTTP Handlers (Week 6)

### Tasks
- [ ] Implement create handler
- [ ] Implement get handler
- [ ] Implement update handler
- [ ] Implement list handler
- [ ] Implement depreciation handlers
- [ ] Implement transfer handler
- [ ] Implement liquidation handler
- [ ] Add error handling
- [ ] Add authorization (Casbin)
- [ ] Write integration tests

### Deliverables
- [ ] HTTP handlers
- [ ] Integration tests

### Exit Criteria
- All endpoints working
- Authorization enforced
- Integration tests passing

---

## Phase 6: Reports & UI (Week 7)

### Tasks
- [ ] Implement asset register report
- [ ] Implement depreciation report
- [ ] Implement depreciation schedule report
- [ ] Create web UI templates
- [ ] Add export to Excel/PDF
- [ ] Write UI tests

### Deliverables
- [ ] Report generators
- [ ] Web UI templates
- [ ] UI tests

### Exit Criteria
- Reports generating correctly
- UI functional
- UI tests passing

---

## Phase 7: Integration & Testing (Week 8)

### Tasks
- [ ] Integrate with GL module
- [ ] Test depreciation posting
- [ ] Test liquidation accounting
- [ ] Performance testing
- [ ] Security testing
- [ ] Compliance verification
- [ ] User acceptance testing

### Deliverables
- [ ] Integration tests
- [ ] Performance report
- [ ] Security report
- [ ] Compliance checklist

### Exit Criteria
- All integration tests passing
- Performance meets requirements
- Security audit passed
- Compliance verified

---

## Phase 8: Deployment (Week 9)

### Tasks
- [ ] Production deployment plan
- [ ] Data migration (if any)
- [ ] User training
- [ ] Go-live support
- [ ] Post-deployment monitoring

### Deliverables
- [ ] Deployment plan
- [ ] Training materials
- [ ] Runbook

### Exit Criteria
- Successful production deployment
- Users trained
- System stable

---

## Resource Requirements

### Team
- 1 Backend Developer (Go)
- 1 Frontend Developer (HTML/Templates)
- 1 QA Engineer
- 1 BA/Accountant (part-time)

### Infrastructure
- SQLite database
- Development/staging/production environments
- CI/CD pipeline

### External Dependencies
- Vietnamese accounting standards documentation
- Circular 45/2013/TT-BTC
- Circular 99/2025/TT-BTC
- VAS 03

---

## Risk Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Incorrect depreciation calculation | Medium | High | Extensive testing with known values |
| Non-compliance with regulations | Low | Critical | Review by Chief Accountant |
| Performance issues with many assets | Low | Medium | Optimize queries, add indexing |
| Integration errors with GL | Medium | High | Comprehensive integration tests |

---

## Success Criteria

- [ ] All unit tests passing (80% coverage)
- [ ] All integration tests passing
- [ ] Depreciation calculations verified against manual calculations
- [ ] Financial statements compliant with VAS 03
- [ ] Tax reports accurate
- [ ] User acceptance testing passed
- [ ] Performance requirements met
- [ ] Security audit passed

---

## Timeline Summary

| Week | Phase | Milestone |
|------|-------|-----------|
| 1 | Planning | BRD approved |
| 2 | Entity/Repo | Core data model ready |
| 3 | Service | CRUD operations working |
| 4 | Depreciation | Depreciation engine working |
| 5 | Operations | Business operations working |
| 6 | HTTP | API endpoints working |
| 7 | Reports/UI | Reports and UI working |
| 8 | Integration | System fully integrated |
| 9 | Deployment | Production ready |

**Total Duration**: 9 weeks  
**Target Completion**: Week 40, 2026

---

*Roadmap complete.*
