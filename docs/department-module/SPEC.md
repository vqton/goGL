# Department Module - Detailed Specification

**Document Version:** 1.0  
**Date:** August 2026  
**Module:** Department/Organization (Phòng ban/Cơ cấu tổ chức)

---

## 1. Use Cases

### UC-D001: Create Department

**Actor:** HR Manager  
**Precondition:** User authenticated, has catalog.write permission  
**Postcondition:** Department created with auto-generated code

**Main Flow:**
1. HR Manager navigates to Department Management screen
2. HR Manager clicks "Thêm phòng ban" (Add Department)
3. System displays department form with fields:
   - Tên phòng ban (Department Name) - required
   - Tên tiếng Anh (English Name) - optional
   - Phòng ban cha (Parent Department) - dropdown, optional for root
   - Loại phòng ban (Type) - dropdown: executive/operational/support
   - Mã trung tâm chi phí (Cost Center Code) - optional
   - Quản lý (Manager) - employee dropdown
   - Ngân sách năm (Annual Budget) - VND input
   - Điện thoại (Phone) - optional
   - Email - optional
   - Địa chỉ (Address) - optional
4. HR Manager fills required fields
5. HR Manager clicks "Lưu" (Save)
6. System validates input:
   - Name required, max 200 chars
   - Parent exists and active (if provided)
   - No cycle in hierarchy
   - Cost center unique (if provided)
   - Manager is active employee (if provided)
7. System generates auto-code: BP-XXXXX
8. System creates department record
9. System logs audit trail
10. System displays success message

**Alternative Flows:**
- 6a. Validation fails → System displays error message
- 6b. Parent department inactive → System displays warning
- 6c. Cost center already assigned → System displays error

**Exception Flows:**
- Database error → System displays generic error, logs details

---

### UC-D002: View Department Tree

**Actor:** HR Manager  
**Precondition:** User authenticated, has catalog.read permission  
**Postcondition:** Department tree displayed

**Main Flow:**
1. HR Manager navigates to Department Management screen
2. System loads department tree from database
3. System displays tree with:
   - Expandable/collapsible nodes
   - Department code and name
   - Employee count badge
   - Active/inactive status indicator
4. HR Manager can:
   - Click node to view details
   - Drag node to reorganize (with confirmation)
   - Filter by status (active/inactive/all)
   - Search by code or name

**Alternative Flows:**
- 3a. No departments → System displays empty state
- 3b. Tree too large → System enables virtual scrolling

---

### UC-D003: Transfer Department

**Actor:** HR Manager  
**Precondition:** Department exists, user has catalog.write permission  
**Postcondition:** Department moved to new parent

**Main Flow:**
1. HR Manager selects department to transfer
2. HR Manager clicks "Chuyển phòng ban" (Transfer Department)
3. System displays transfer form:
   - Phòng ban hiện tại (Current Parent)
   - Phòng ban mới (New Parent) - dropdown
   - Ngày hiệu lực (Effective Date) - required
   - Lý do (Reason) - required, min 10 chars
4. HR Manager selects new parent and fills fields
5. HR Manager clicks "Xác nhận" (Confirm)
6. System validates:
   - New parent exists and active
   - No cycle would be created
   - Effective date not in past
7. System updates department parent
8. System recalculates hierarchy levels
9. System logs audit trail with reason
10. System displays success message

**Alternative Flows:**
- 6a. Cycle detected → System displays error with cycle path
- 6b. New parent inactive → System displays warning

---

### UC-D004: Deactivate Department

**Actor:** HR Manager  
**Precondition:** Department exists, user has catalog.write permission  
**Postcondition:** Department deactivated

**Main Flow:**
1. HR Manager selects department to deactivate
2. HR Manager clicks "Ngừng hoạt động" (Deactivate)
3. System checks:
   - No active employees in department
   - No active sub-departments
4. System displays deactivation form:
   - Lý do (Reason) - required, min 10 chars
5. HR Manager fills reason
6. HR Manager clicks "Xác nhận" (Confirm)
7. System sets department state to inactive
8. System logs audit trail
9. System displays success message

**Alternative Flows:**
- 3a. Active employees exist → System blocks deactivation, shows employee list
- 3b. Active sub-departments exist → System warns, asks to deactivate children first

---

### UC-D005: Set Department Budget

**Actor:** Finance Manager  
**Precondition:** Department exists, user has catalog.write permission  
**Postcondition:** Budget set for department

**Main Flow:**
1. Finance Manager selects department
2. Finance Manager clicks "Đặt ngân sách" (Set Budget)
3. System displays budget form:
   - Năm tài chính (Fiscal Year) - dropdown
   - Ngân sách năm (Annual Budget) - VND input
   - Ghi chú (Notes) - optional
4. Finance Manager fills fields
5. Finance Manager clicks "Lưu" (Save)
6. System validates:
   - Fiscal year valid
   - Budget non-negative
7. System creates/updates budget record
8. System logs audit trail
9. System displays success message

**Alternative Flows:**
- 6a. Budget already exists for year → System asks to update
- 6b. Department inactive → System blocks budget setting

---

### UC-D006: View Department Cost Report

**Actor:** Finance Manager  
**Precondition:** Department exists, user has catalog.read permission  
**Postcondition:** Cost report displayed

**Main Flow:**
1. Finance Manager navigates to Cost Reports
2. Finance Manager selects department
3. Finance Manager selects period (month/quarter/year)
4. System queries:
   - Budget amount
   - Actual expenses from GL
   - Variance calculation
5. System displays report:
   - Budget vs Actual table
   - Variance percentage
   - Trend chart
   - Export options (PDF/Excel)

**Alternative Flows:**
- 5a. No budget set → System displays "Chưa đặt ngân sách"
- 5b. No actual data → System displays only budget

---

## 2. Processes

### 2.1 Department Lifecycle Process

```
┌─────────────────────────────────────────────────────────────────┐
│                    Department Lifecycle                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │   Draft  │───▶│  Active  │───▶│ Inactive │───▶│ Deleted  │ │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘ │
│       │               │               │                         │
│       ▼               ▼               ▼                         │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐                 │
│  │  Create  │    │ Transfer │    │ Activate │                 │
│  │  Update  │    │  Budget  │    │          │                 │
│  │  Delete  │    │  Assign  │    │          │                 │
│  └──────────┘    └──────────┘    └──────────┘                 │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Employee Transfer Process

```
┌─────────────────────────────────────────────────────────────────┐
│                    Employee Department Transfer                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  1. HR Manager initiates transfer                                │
│     │                                                            │
│  2. Select employee, new department, effective date              │
│     │                                                            │
│  3. System validates:                                            │
│     ├── Employee exists and active                               │
│     ├── New department active                                    │
│     ├── Effective date valid                                     │
│     └── Reason provided                                          │
│     │                                                            │
│  4. System updates:                                              │
│     ├── Employee.department = new department                     │
│     ├── Old department.employee_count--                          │
│     ├── New department.employee_count++                          │
│     └── Audit trail logged                                       │
│     │                                                            │
│  5. System notifies:                                             │
│     ├── Old department manager                                   │
│     ├── New department manager                                   │
│     └── Employee                                                 │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 Budget Allocation Process

```
┌─────────────────────────────────────────────────────────────────┐
│                    Department Budget Allocation                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  1. Finance Manager sets annual budget                           │
│     │                                                            │
│  2. System stores budget record                                  │
│     │                                                            │
│  3. Monthly actual expenses recorded in GL                       │
│     │                                                            │
│  4. System calculates:                                           │
│     ├── Budget remaining = Budget - Actual                       │
│     ├── Variance = Actual - Budget                               │
│     └── Utilization % = Actual / Budget * 100                   │
│     │                                                            │
│  5. System generates alerts:                                     │
│     ├── 80% utilized → Warning                                   │
│     ├── 100% utilized → Critical                                 │
│     └── >100% utilized → Block (optional)                        │
│     │                                                            │
│  6. Finance Manager reviews and adjusts                         │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```
