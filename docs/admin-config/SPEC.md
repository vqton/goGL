# Admin/Configuration Layer - Business Requirements Document

**Version:** 1.0
**Date:** 2026-08-23
**Status:** Draft

---

## 1. Overview

### 1.1 Purpose
This document specifies the Admin/Configuration layer requirements for the goGL Vietnamese accounting system to achieve PROD readiness.

### 1.2 Scope
- User Management & Authentication
- Role-Based Access Control (RBAC)
- System Configuration
- Company Profile Management
- Security & Compliance
- Audit Trail

### 1.3 Target Users
| Role | Description |
|------|-------------|
| **System Admin** | Full system access, user management |
| **Chief Accountant** (Kế toán trưởng) | Financial data, reports, approvals |
| **Accountant** (Kế toán viên) | Daily transactions, data entry |
| **Auditor** (Kiểm toán viên) | Read-only access, audit trail |
| **Director** (Giám đốc) | Dashboard, approvals |
| **IT Admin** | System config, backup, security |

---

## 2. User Management

### 2.1 User Registration

**Happy Path:**
1. Admin navigates to `/admin/users/new`
2. Admin fills in: username, full name, email, roles
3. System validates username uniqueness
4. System generates temporary password
5. System sends invitation email (optional)
6. User receives credentials
7. User logs in with temporary password
8. System forces password change

**Alternative Paths:**
- 2a. Username exists → Show error "Username already taken"
- 2b. Email exists → Show warning, suggest merge

**Exception Paths:**
- 2e1. Email service unavailable → Log warning, proceed with manual notification
- 2e2. Database error → Show generic error, log details

### 2.2 Password Policy

| Rule | Value | Reference |
|------|-------|-----------|
| Minimum length | 8 characters | Best practice |
| Maximum length | 128 characters | Security |
| Uppercase required | Yes | Best practice |
| Lowercase required | Yes | Best practice |
| Number required | Yes | Best practice |
| Special char required | Optional | Best practice |
| Expiry | 90 days | NĐ 147/2018 |
| History | Last 5 passwords | Best practice |
| Lockout after failures | 5 attempts | Best practice |
| Lockout duration | 15 minutes | Best practice |

### 2.3 Multi-Factor Authentication (MFA)

**Implementation:**
- TOTP (Time-based One-Time Password) via authenticator apps
- Backup codes for recovery
- SMS as fallback (optional)

**Happy Path:**
1. User enables MFA in profile settings
2. System displays QR code
3. User scans with authenticator app
4. User enters verification code
5. System confirms and displays backup codes
6. User saves backup codes

**Alternative Paths:**
- 4a. Invalid code → Show error, allow retry (max 3)
- 4b. User can't scan QR → Display manual entry key

---

## 3. Role-Based Access Control

### 3.1 Default Roles

| Role | Code | Permissions |
|------|------|-------------|
| **Admin** | `role:admin` | All operations |
| **Chief Accountant** | `role:chieft_accountant` | Post entries, close periods, approve |
| **Accountant** | `role:accountant` | Create/edit entries, drafts |
| **Cashier** | `role:cashier` | Cash vouchers, receipts |
| **Auditor** | `role:kiem_toan` | Read-only all modules |
| **Director** | `role:giam_doc` | Read-only, approve vouchers |
| **Viewer** | `role:viewer` | Read-only dashboard |

### 3.2 Permission Structure

```go
type Permission struct {
    Subject  string  // role:accountant
    Object   string  // /api/v1/ledger/entries
    Action   string  // POST, GET, PUT, DELETE
}
```

### 3.3 Route Protection Matrix

| Route | Admin | Chief | Accountant | Cashier | Auditor | Director |
|-------|-------|-------|------------|---------|---------|----------|
| `GET /api/v1/*` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `POST /api/v1/ledger/entries` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `POST /api/v1/ledger/entries/:id/post` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `POST /api/v1/cash/vouchers` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `POST /api/v1/cash/vouchers/:id/approve` | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ |
| `GET /api/v1/reports/*` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `POST /api/v1/admin/*` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## 4. System Configuration

### 4.1 System Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `company.name` | string | "" | Company name |
| `company.tax_code` | string | "" | MST (10 or 13 digits) |
| `company.address` | string | "" | Registered address |
| `company.phone` | string | "" | Contact phone |
| `company.email` | string | "" | Contact email |
| `company.accounting_regime` | string | "VAS" | VAS or IFRS |
| `company.fiscal_year_start` | string | "01-01" | Fiscal year start (MM-DD) |
| `company.currency` | string | "VND" | Base currency |
| `tax.vat_rate` | int | 10 | Default VAT rate (%) |
| `tax.cit_rate` | int | 20 | Corporate income tax rate (%) |
| `backup.schedule` | string | "0 2 * * *" | Cron expression |
| `backup.retention_days` | int | 90 | Days to keep backups |
| `session.max_hours` | int | 24 | Session duration |
| `session.idle_minutes` | int | 30 | Idle timeout |
| `password.min_length` | int | 8 | Minimum password length |
| `password.expiry_days` | int | 90 | Password expiry |

### 4.2 Company Profile

```go
type CompanyProfile struct {
    ID                   string    `json:"id"`
    Name                 string    `json:"name"`
    TaxCode              string    `json:"tax_code"`
    Address              string    `json:"address"`
    Phone                string    `json:"phone"`
    Email                string    `json:"email"`
    Website              string    `json:"website"`
    LegalRepresentative   string    `json:"legal_representative"`
    Position             string    `json:"position"`
    AccountingRegime     string    `json:"accounting_regime"` // VAS or IFRS
    FiscalYearStart      string    `json:"fiscal_year_start"`
    AccountingCurrency   string    `json:"accounting_currency"`
    BankName             string    `json:"bank_name"`
    BankAccount          string    `json:"bank_account"`
    BankBranch           string    `json:"bank_branch"`
    TaxAuthority         string    `json:"tax_authority"`
    RegistrationDate     string    `json:"registration_date"`
    CharterCapital       int64     `json:"charter_capital"`
    EmployeeCount        int       `json:"employee_count"`
    IndustryCode         string    `json:"industry_code"`
    Logo                 string    `json:"logo"` // base64 or URL
}
```

---

## 5. Audit Trail Requirements

### 5.1 What to Log

| Event | Data Captured |
|-------|---------------|
| Login/Logout | User, timestamp, IP, success/fail |
| User CRUD | Actor, target, changes |
| Financial entries | Actor, entry ID, amount, accounts |
| Approvals | Actor, entry ID, decision |
| Config changes | Actor, key, old value, new value |
| Data export | Actor, module, format, row count |
| Backup/Restore | Actor, operation, result |

### 5.2 Audit Log Structure

```go
type AuditLog struct {
    ID        string `json:"id"`
    UserCode  string `json:"user_code"`
    Module    string `json:"module"`
    Action    string `json:"action"`
    TargetID  string `json:"target_id"`
    Details   string `json:"details"` // JSON of changes
    IP        string `json:"ip"`
    UserAgent string `json:"user_agent"`
    Timestamp string `json:"timestamp"`
}
```

### 5.3 Retention Policy

- Minimum 5 years retention (NĐ 147/2018)
- Exportable to Excel/PDF
- Immutable (no delete/update)

---

## 6. Security Requirements

### 6.1 Authentication

| Requirement | Status |
|-------------|--------|
| Password hashing (argon2id) | ✅ Implemented |
| Session management | ✅ Implemented |
| Idle timeout | ✅ Implemented |
| Account lockout | ✅ Implemented |
| Multi-factor authentication | ❌ Required |
| Password expiry | ❌ Required |
| Password history | ❌ Required |
| Single sign-on (SSO) | ❌ Optional |

### 6.2 Authorization

| Requirement | Status |
|-------------|--------|
| RBAC enforcement | ✅ Implemented |
| Route-level protection | ✅ Implemented |
| Data-level isolation | ❌ Required (multi-company) |
| API rate limiting | ❌ Required |

### 6.3 Data Protection

| Requirement | Status |
|-------------|--------|
| HTTPS enforcement | ❌ Required |
| CORS configuration | ❌ Required |
| CSRF protection | ❌ Required |
| SQL injection prevention | ✅ (parameterized queries) |
| XSS prevention | ✅ (JSON API) |
| Data encryption at rest | ❌ Required |
| Data encryption in transit | ❌ Required |

---

## 7. Integration Requirements

### 7.1 E-Invoice Integration (NĐ 123/2020)

| Provider | Status | Priority |
|----------|--------|----------|
| MISA e-Invoice | ❌ Not integrated | Critical |
| VNPT e-Invoice | ❌ Not integrated | Critical |
| Viettel e-Invoice | ❌ Not integrated | High |

### 7.2 Tax Filing Integration

| Service | Status | Priority |
|---------|--------|----------|
| eTax (NĐ 126/2020) | ❌ Not integrated | Critical |
| VAT declaration | ❌ Not integrated | Critical |
| CIT declaration | ❌ Not integrated | High |
| PIT declaration | ❌ Not integrated | High |

### 7.3 Banking Integration

| Service | Status | Priority |
|---------|--------|----------|
| Bank reconciliation | ❌ Not integrated | Medium |
| Payment gateway | ❌ Not integrated | Low |
| Exchange rates (SBV) | ❌ Not integrated | High |

---

## 8. Non-Functional Requirements

### 8.1 Performance

| Metric | Target |
|--------|--------|
| API response time | < 200ms (p95) |
| Concurrent users | 100+ |
| Database size | 10GB+ |
| Backup duration | < 5 minutes |

### 8.2 Availability

| Metric | Target |
|--------|--------|
| Uptime | 99.9% |
| RPO (Recovery Point Objective) | 1 hour |
| RTO (Recovery Time Objective) | 4 hours |

### 8.3 Compliance

| Standard | Status |
|----------|--------|
| Circular 99/2025 | ❌ Required |
| ISO 27001 | ❌ Recommended |
| GDPR (if applicable) | ❌ Check |

---

## 9. Acceptance Criteria

### 9.1 User Management

- [ ] Admin can create, edit, suspend, delete users
- [ ] Password policy enforced (length, complexity, expiry)
- [ ] MFA can be enabled/disabled per user
- [ ] Account lockout after 5 failed attempts
- [ ] Password history prevents reuse of last 5 passwords

### 9.2 RBAC

- [ ] Default roles created on first run
- [ ] Custom roles can be created with specific permissions
- [ ] Users cannot access routes without permission
- [ ] Changes to permissions take effect immediately

### 9.3 Configuration

- [ ] Company profile can be updated
- [ ] System options persist across restarts
- [ ] Configuration changes are audited

### 9.4 Security

- [ ] Sessions expire after configured timeout
- [ ] Idle sessions are terminated
- [ ] All sensitive operations are logged
- [ ] Audit trail cannot be modified

---

## 10. Out of Scope

- Multi-tenant architecture (future phase)
- Mobile native app (future phase)
- AI-powered features (future phase)
- Advanced analytics/BI (future phase)

---

*Document prepared by BA Lead and Chief Accountant review team.*
