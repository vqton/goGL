# Admin/Configuration Layer - PROD Readiness Analysis

**Date:** 2026-08-23
**Analyst:** BA Lead (20+ years) + Vietnamese Chief Accountant (20+ years)

---

## Executive Summary

**Verdict: NOT PROD-READY**

The current Admin/Configuration layer provides a solid skeleton but lacks critical features required for Vietnamese production accounting. Here's the gap analysis:

---

## 1. Current Implementation Status

| Module | Status | Lines | Completeness |
|--------|--------|-------|--------------|
| **user** | ✅ Implemented | 455 | 85% - missing password expiry, MFA |
| **auth** | ✅ Implemented | 214 | 90% - missing MFA, password history |
| **session** | ✅ Implemented | 70 | 95% - basic but functional |
| **authz** | ✅ Implemented | ~400 | 95% - RBAC working |
| **options** | ✅ Implemented | 142 | 70% - missing validation rules |
| **system** | ✅ Implemented | 59 | 80% - basic health check |
| **backup** | ✅ Implemented | 355 | 90% - VACUUM INTO working |
| **audit** | ✅ Implemented | 73 | 85% - basic logging |
| **setup** | ✅ Implemented | 863 | 95% - full wizard |

---

## 2. CRITICAL GAPS for Vietnamese PROD

### 2.1 Missing Regulatory Compliance Features

| Requirement | Status | Reference | Impact |
|-------------|--------|-----------|--------|
| **E-Invoice Integration** | ❌ Missing | NĐ 123/2020, TT 78/2021 | **BLOCKER** - Cannot issue legal invoices |
| **eTax Filing Integration** | ❌ Missing | NĐ 126/2020 | **BLOCKER** - Cannot file tax returns |
| **Circular 99/2025 Compliance** | ❌ Missing | TT 99/2025/TT-BTC | **BLOCKER** - New accounting standards effective 2026 |
| **Digital Signature** | ❌ Missing | NĐ 130/2018 | Required for e-invoices |
| **Multi-Currency (USD/EUR)** | ⚠️ Partial | TT 99/2025 | Only VND hardcoded |
| **Transfer Pricing Docs** | ❌ Missing | TT 32/2020 | FDI companies require |

### 2.2 Missing Security Features (for PROD)

| Feature | Status | Reference | Impact |
|---------|--------|-----------|--------|
| **Multi-Factor Authentication** | ❌ Missing | ISO 27001 | **HIGH** - Accounting systems need MFA |
| **Password Expiry Policy** | ❌ Missing | NĐ 147/2018 | Required for SOX-like compliance |
| **Password History** | ❌ Missing | Best practice | Prevents password reuse |
| **Session Recording** | ❌ Missing | Audit trail | Need to track user actions |
| **IP Whitelisting** | ❌ Missing | Security | Restrict access by IP |
| **Concurrent Session Limit** | ❌ Missing | Security | Prevent shared accounts |

### 2.3 Missing Operational Features

| Feature | Status | Reference | Impact |
|---------|--------|-----------|--------|
| **Multi-Company/Branch** | ❌ Missing | Vietnamese accounting | **HIGH** - Most businesses have branches |
| **Fiscal Year Management** | ⚠️ Partial | TT 200/2014 | Only 12 monthly periods |
| **Chart of Accounts Import** | ❌ Missing | TT 200/2014, TT 133/2016 | Need to import standard COA |
| **Opening Balance Import** | ✅ Implemented | Setup module | CSV import working |
| **Bank Reconciliation** | ❌ Missing | Cash module | Not implemented |
| **Exchange Rate Management** | ❌ Missing | TT 99/2025 | Daily rates from SBV |

### 2.4 Missing Reporting Features

| Report | Status | Reference | Impact |
|--------|--------|-----------|--------|
| **Financial Statements (BCTC)** | ❌ Missing | TT 200/2014, TT 99/2025 | **BLOCKER** - Balance Sheet, P&L, Cash Flow |
| **Tax Reports (Báo cáo thuế)** | ❌ Missing | NĐ 126/2020 | VAT, CIT, PIT declarations |
| **Audit Trail Export** | ⚠️ Partial | Audit module | No export to Excel/PDF |
| **Management Reports** | ❌ Missing | Business need | Cost center, project reporting |

---

## 3. Comparison with Vietnamese Market Leaders

### MISA SME (45% market share)

| Feature | MISA | goGL | Gap |
|---------|------|------|-----|
| E-Invoice | ✅ Built-in | ❌ | Critical |
| eTax | ✅ Built-in | ❌ | Critical |
| Multi-company | ✅ | ❌ | High |
| Bank integration | ✅ | ❌ | Medium |
| Mobile app | ✅ | ❌ | Low |
| AI assistant | ✅ (2025+) | ❌ | Low |
| IFRS support | ✅ | ❌ | Medium |

### Fast Accounting (25% market share)

| Feature | Fast | goGL | Gap |
|---------|------|------|-----|
| Multi-warehouse | ✅ | ❌ | High |
| BOM support | ✅ | ❌ | Medium |
| Custom reports | ✅ | ❌ | High |
| API integration | ✅ | ⚠️ Basic | Medium |

### BRAVO ERP (10% market share)

| Feature | BRAVO | goGL | Gap |
|---------|-------|------|-----|
| Full ERP | ✅ | ❌ | High |
| Manufacturing | ✅ | ❌ | Medium |
| English UI | ✅ | ✅ | None |
| Multi-currency | ✅ | ⚠️ | High |

---

## 4. Critical Path to PROD

### Phase 1: MUST HAVE (Blocking)

| # | Task | Effort | Dependencies |
|---|------|--------|--------------|
| 1 | E-Invoice Integration (NĐ 123/2020) | 4-6 weeks | Tax API access |
| 2 | eTax Filing Integration | 3-4 weeks | Tax API access |
| 3 | Digital Signature Integration | 2-3 weeks | CA certificate |
| 4 | Financial Statements (BCTC) | 4-6 weeks | Ledger module |
| 5 | Multi-Factor Authentication | 2-3 weeks | None |
| 6 | Password Policy Enhancement | 1-2 weeks | None |

### Phase 2: SHOULD HAVE (Important)

| # | Task | Effort | Dependencies |
|---|------|--------|--------------|
| 7 | Multi-Company Support | 6-8 weeks | Architecture refactor |
| 8 | Exchange Rate Management | 2-3 weeks | SBV API |
| 9 | Bank Reconciliation | 3-4 weeks | Bank API |
| 10 | Advanced Reporting | 4-6 weeks | Ledger module |
| 11 | Audit Trail Export | 2-3 weeks | None |

### Phase 3: NICE TO HAVE

| # | Task | Effort | Dependencies |
|---|------|--------|--------------|
| 12 | Mobile App | 8-12 weeks | API layer |
| 13 | AI Assistant | 6-8 weeks | ML infrastructure |
| 14 | IFRS Dual Reporting | 4-6 weeks | Ledger refactor |

---

## 5. Regulatory Reference Matrix

| Regulation | Description | Status | Impact |
|------------|-------------|--------|--------|
| **TT 200/2014** | Accounting regime for enterprises | Partial | Core |
| **TT 133/2016** | Accounting for SMEs | Partial | Core |
| **TT 99/2025** | New accounting standards (2026) | Not implemented | **BLOCKER** |
| **NĐ 123/2020** | E-invoice management | Not implemented | **BLOCKER** |
| **NĐ 126/2020** | Tax administration | Not implemented | **BLOCKER** |
| **NĐ 130/2018** | Digital signatures | Not implemented | High |
| **TT 78/2021** | E-invoice guidance | Not implemented | **BLOCKER** |
| **TT 32/2020** | Transfer pricing | Not implemented | FDI only |
| **NĐ 147/2018** | Cybersecurity | Partial | High |

---

## 6. Recommendation

**DO NOT DEPLOY TO PROD** without completing Phase 1 items. The system is suitable for:
- ✅ Development/testing
- ✅ Internal training
- ✅ Demo/proof of concept
- ❌ Production use
- ❌ Legal compliance
- ❌ Tax filing

---

## 7. Next Steps

1. **Immediate:** Implement MFA and password policies (2-3 weeks)
2. **Short-term:** Integrate e-invoice and eTax (8-10 weeks)
3. **Medium-term:** Complete financial statements and multi-company (10-14 weeks)
4. **Long-term:** Advanced features (mobile, AI, IFRS)

---

*This analysis is based on Vietnamese accounting regulations as of August 2026 and market comparison with MISA, Fast, and BRAVO.*
