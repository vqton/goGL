# Admin/Configuration Layer - Implementation Roadmap

**Date:** 2026-08-23
**Timeline:** 16 weeks (4 months)

---

## Overview

This roadmap transforms the goGL Admin/Configuration layer from skeleton to PROD-ready, focusing on Vietnamese regulatory compliance and security best practices.

---

## Phase 1: Security Foundation (Weeks 1-4)

### Sprint 1.1: MFA Implementation (Week 1-2)

| Task | Files | Effort |
|------|-------|--------|
| Add MFA entity to user domain | `internal/domain/user/mfa.go` | 2h |
| Add TOTP generation/verification | `internal/domain/user/totp.go` | 4h |
| Add backup codes generation | `internal/domain/user/backup_codes.go` | 2h |
| Update user repository for MFA | `internal/infrastructure/persistence/user/repository.go` | 2h |
| Update user service for MFA | `internal/application/user/service.go` | 4h |
| Add MFA HTTP handlers | `internal/interfaces/http/user/handler.go` | 4h |
| Write tests | `*_test.go` | 4h |

**Acceptance Criteria:**
- [ ] User can enable MFA via profile settings
- [ ] QR code generated for authenticator apps
- [ ] Backup codes generated and stored securely
- [ ] MFA verification required on login when enabled
- [ ] Backup codes can be used for recovery

### Sprint 1.2: Password Policies (Week 2-3)

| Task | Files | Effort |
|------|-------|--------|
| Add password expiry tracking | `internal/domain/user/entity.go` | 2h |
| Add password history table | `internal/infrastructure/db/migrate.go` | 1h |
| Implement expiry check in login | `internal/application/auth/service.go` | 3h |
| Implement history check on change | `internal/application/auth/service.go` | 3h |
| Add password policy config | `internal/domain/options/entity.go` | 2h |
| Add UI for policy settings | `internal/interfaces/http/options/handler.go` | 3h |
| Write tests | `*_test.go` | 4h |

**Acceptance Criteria:**
- [ ] Passwords expire after configured days (default 90)
- [ ] Users forced to change expired password on login
- [ ] Password history prevents reuse of last 5 passwords
- [ ] Policy configurable via system options

### Sprint 1.3: Session Security (Week 3-4)

| Task | Files | Effort |
|------|-------|--------|
| Add session IP tracking | `internal/domain/session/entity.go` | 1h |
| Add concurrent session limit | `internal/application/auth/service.go` | 3h |
| Add session invalidation on password change | Already implemented | 0h |
| Add "logout all devices" feature | `internal/application/auth/service.go` | 2h |
| Add session list UI | `internal/interfaces/http/auth/handler.go` | 3h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Session IP logged and validated
- [ ] Concurrent session limit enforced
- [ ] User can view active sessions
- [ ] User can invalidate specific sessions
- [ ] All sessions invalidated on password change

### Sprint 1.4: API Security (Week 4)

| Task | Files | Effort |
|------|-------|--------|
| Add CORS configuration | `cmd/server/main.go` | 2h |
| Add CSRF protection middleware | `internal/infrastructure/http/middleware.go` | 4h |
| Add rate limiting middleware | `internal/infrastructure/http/middleware.go` | 3h |
| Add request logging | `internal/infrastructure/http/middleware.go` | 3h |
| Write tests | `*_test.go` | 2h |

**Acceptance Criteria:**
- [ ] CORS properly configured for frontend
- [ ] CSRF tokens required for state-changing operations
- [ ] Rate limiting prevents brute force attacks
- [ ] All requests logged with user, IP, timestamp

**Checkpoint 1:** Security foundation complete. All auth flows tested.

---

## Phase 2: Configuration Management (Weeks 5-8)

### Sprint 2.1: System Options Enhancement (Week 5-6)

| Task | Files | Effort |
|------|-------|--------|
| Add validation rules to options | `internal/domain/options/entity.go` | 3h |
| Add option categories | `internal/domain/options/entity.go` | 2h |
| Add bulk update endpoint | `internal/interfaces/http/options/handler.go` | 3h |
| Add option groups UI | Web templates | 4h |
| Add option search/filter | `internal/application/options/service.go` | 2h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Options organized by category (Company, Tax, Security, etc.)
- [ ] Validation rules enforced (e.g., tax code format)
- [ ] Bulk update supported
- [ ] Options searchable and filterable

### Sprint 2.2: Company Profile Enhancement (Week 6-7)

| Task | Files | Effort |
|------|-------|--------|
| Add logo upload support | `internal/interfaces/http/setup/handler.go` | 3h |
| Add company info to reports | Web templates | 4h |
| Add company info to API responses | `internal/interfaces/http/` | 3h |
| Add company validation | `internal/application/setup/service.go` | 2h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Company logo uploadable (base64 or file)
- [ ] Company info displayed on all reports
- [ ] Company info included in API responses
- [ ] Tax code format validated (10 or 13 digits)

### Sprint 2.3: Fiscal Year Management (Week 7-8)

| Task | Files | Effort |
|------|-------|--------|
| Add fiscal year entity | `internal/domain/ledger/entity.go` | 2h |
| Add fiscal year repository | `internal/infrastructure/persistence/ledger/repository.go` | 3h |
| Add fiscal year service | `internal/application/ledger/service.go` | 4h |
| Add fiscal year handlers | `internal/interfaces/http/ledger/handler.go` | 3h |
| Add period management | `internal/application/ledger/service.go` | 4h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Fiscal year configurable (start month)
- [ ] Periods auto-generated from fiscal year
- [ ] Period open/close controlled by chief accountant
- [ ] Transactions blocked in closed periods

**Checkpoint 2:** Configuration management complete. Company profile and fiscal year working.

---

## Phase 3: Vietnamese Compliance (Weeks 9-12)

### Sprint 3.1: E-Invoice Integration (Week 9-10)

| Task | Files | Effort |
|------|-------|--------|
| Define e-invoice domain | `internal/domain/einvoice/entity.go` | 4h |
| Create e-invoice service interface | `internal/application/einvoice/service.go` | 3h |
| Implement MISA adapter | `internal/infrastructure/einvoice/misa.go` | 8h |
| Implement VNPT adapter | `internal/infrastructure/einvoice/vnpt.go` | 8h |
| Add e-invoice handlers | `internal/interfaces/http/einvoice/handler.go` | 4h |
| Write tests | `*_test.go` | 4h |

**Acceptance Criteria:**
- [ ] E-invoices created from sales transactions
- [ ] XML format compliant with NĐ 123/2020
- [ ] Digital signature applied
- [ ] Submitted to tax authority
- [ ] Status tracking (pending, accepted, rejected)

### Sprint 3.2: eTax Integration (Week 10-11)

| Task | Files | Effort |
|------|-------|--------|
| Define tax report domain | `internal/domain/tax/entity.go` | 3h |
| Create tax service interface | `internal/application/tax/service.go` | 3h |
| Implement eTax adapter | `internal/infrastructure/tax/etax.go` | 8h |
| Add tax report handlers | `internal/interfaces/http/tax/handler.go` | 4h |
| Add VAT report generation | `internal/application/tax/service.go` | 4h |
| Write tests | `*_test.go` | 4h |

**Acceptance Criteria:**
- [ ] VAT reports generated from transactions
- [ ] CIT reports generated
- [ ] PIT reports generated
- [ ] Reports submitted via eTax portal
- [ ] Submission status tracked

### Sprint 3.3: Financial Statements (Week 11-12)

| Task | Files | Effort |
|------|-------|--------|
| Define report domain | `internal/domain/reporting/entity.go` | 3h |
| Create report service | `internal/application/reporting/service.go` | 4h |
| Implement Balance Sheet | `internal/application/reporting/service.go` | 6h |
| Implement Income Statement | `internal/application/reporting/service.go` | 6h |
| Implement Cash Flow Statement | `internal/application/reporting/service.go` | 6h |
| Add report handlers | `internal/interfaces/http/reporting/handler.go` | 3h |
| Write tests | `*_test.go` | 4h |

**Acceptance Criteria:**
- [ ] Balance Sheet (Bảng cân đối kế toán) compliant with TT 200/2014
- [ ] Income Statement (Kết quả kinh doanh) compliant
- [ ] Cash Flow Statement (Lưu chuyển tiền tệ) compliant
- [ ] Reports exportable to Excel/PDF
- [ ] Reports comply with Circular 99/2025

### Sprint 3.4: Exchange Rate Management (Week 12)

| Task | Files | Effort |
|------|-------|--------|
| Add exchange rate entity | `internal/domain/forex/entity.go` | 2h |
| Add exchange rate repository | `internal/infrastructure/persistence/forex/repository.go` | 3h |
| Add SBV API integration | `internal/infrastructure/forex/sbv.go` | 4h |
| Add exchange rate service | `internal/application/forex/service.go` | 3h |
| Add exchange rate handlers | `internal/interfaces/http/forex/handler.go` | 2h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Daily exchange rates fetched from SBV
- [ ] Manual rate entry supported
- [ ] Rates applied to foreign currency transactions
- [ ] Rate history maintained

**Checkpoint 3:** Vietnamese compliance complete. E-invoice, eTax, and financial statements working.

---

## Phase 4: Audit & Reporting (Weeks 13-16)

### Sprint 4.1: Enhanced Audit Trail (Week 13-14)

| Task | Files | Effort |
|------|-------|--------|
| Enhance audit log structure | `internal/domain/audit/entity.go` | 3h |
| Add IP/UserAgent tracking | `internal/infrastructure/http/middleware.go` | 3h |
| Add audit log search/filter | `internal/application/audit/service.go` | 4h |
| Add audit log export | `internal/application/audit/service.go` | 4h |
| Add audit log UI | Web templates | 4h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] All API requests logged
- [ ] Audit logs searchable by user, module, action, date
- [ ] Audit logs exportable to Excel/CSV
- [ ] Audit logs immutable (no delete/update)

### Sprint 4.2: Management Reports (Week 14-15)

| Task | Files | Effort |
|------|-------|--------|
| Add dashboard entity | `internal/domain/dashboard/entity.go` | 2h |
| Add dashboard service | `internal/application/dashboard/service.go` | 4h |
| Add dashboard handlers | `internal/interfaces/http/dashboard/handler.go` | 3h |
| Add dashboard UI | Web templates | 6h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Dashboard shows key metrics (revenue, expenses, profit)
- [ ] Dashboard shows user activity
- [ ] Dashboard shows system health
- [ ] Dashboard configurable by user

### Sprint 4.3: Backup Enhancement (Week 15-16)

| Task | Files | Effort |
|------|-------|--------|
| Add backup scheduling | `internal/application/backup/service.go` | 4h |
| Add backup verification | `internal/application/backup/service.go` | 3h |
| Add backup notifications | `internal/application/backup/service.go` | 2h |
| Add backup restore testing | `internal/application/backup/service.go` | 3h |
| Add backup UI | Web templates | 4h |
| Write tests | `*_test.go` | 3h |

**Acceptance Criteria:**
- [ ] Backups scheduled via cron
- [ ] Backups verified after creation
- [ ] Admin notified of backup success/failure
- [ ] Restore tested periodically
- [ ] Backup retention policy enforced

### Sprint 4.4: Documentation & Training (Week 16)

| Task | Effort |
|------|--------|
| API documentation (OpenAPI/Swagger) | 4h |
| User manual | 4h |
| Admin guide | 4h |
| Deployment guide | 4h |
| Security hardening guide | 4h |

**Checkpoint 4:** PROD-ready. All features complete, documented, tested.

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| E-invoice API changes | High | Abstract adapter interface, monitor changes |
| Tax regulation changes | High | Modular design, quick updates |
| Security vulnerabilities | High | Regular audits, penetration testing |
| Performance issues | Medium | Load testing, optimization |
| User adoption | Medium | Training, documentation |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Test coverage | > 80% |
| Security audit | Pass |
| Performance (p95) | < 200ms |
| Vietnamese compliance | 100% |
| Documentation | Complete |

---

*Roadmap prepared by BA Lead and Development Team.*
