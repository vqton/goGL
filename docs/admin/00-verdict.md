# Admin/Configuration Layer (Quản trị hệ thống) — PROD-Readiness Verdict

> Status: **NOT PROD-READY** — the Admin/Configuration layer is the last gap
> between a demo and a deployable system. Authorization infra and audit are
> **implemented**, setup is a **full vertical**, but the five admin modules
> (`user`, `options`, `system`, `backup`, `task`) are still 4-layer stubs.
> Capabilities below describe the **target state**. Date: 2026-08-16.

## 1. Executive verdict

| Criterion | Assessment | Evidence |
|---|---|---|
| Authorization infra | 🟢 Implemented | Casbin v3 enforcer + SQLite adapter (`casbin_policies` table), gin middleware, `HeaderPrincipalResolver` dev seam, `SeedDefaultPolicies` with `role:admin` (`* *`) + business roles (`danh_muc`, `ke_toan_tong_hop`, `ke_toan_truong`, `giam_doc`, `kiem_toan`). Policy files + tests for cash/ledger/masterdata/setup. Covered by tests |
| Audit | 🟢 Implemented | `audit_logs (id, data)` JSON rows; `audit.Service` (`Record`, `ListRecent`) + SQLite repo + tests; `/api/v1/audit` handler registered; setup dashboard shows recent trail |
| Setup (first-run gateway) | 🟢 Implemented | Full vertical: profile (statutory), regime (TT 99/2025 default / TT 133/2016), COA seed via masterdata, balanced opening balances + CSV import, status machine `EMPTY→…→ACTIVE`, lock/reopen/activate, idempotent, audited, authz-wired. See `docs/setup/` |
| Config | 🟡 Minimal | `config.yaml`: server.http_addr, database.dsn, authorization.{enabled, identity_header}. Missing: secrets/TLS, CORS, logging level, backup dir, session/token config, feature flags, upload limits |
| Identity (user) | 🔴 Stub | `User{ID, Username, FullName, RoleCodes, Status}` + `Role{Code, Name, Permissions}` — no password, no auth method (sessions/JWT), no lockout, no created/updated audit fields, no pagination/list. Service (`CreateUser`, `GetUser`, `AssignRole`) returns `core.ErrNotImplemented`; repo `Create/FindByID/Update/SaveRole` stub; HTTP → 501 |
| Settings (options) | 🔴 Stub | `Option{ID, Key, Value, Category}` — no typing (bool/int/json), no defaults, no validation, no schema/version. All layers stub |
| System/tenancy | 🔴 Stub | `Tenant{ID, Code, Name, Status}` — single-company product has no tenancy need yet; but no company-scope enforcement, no "system info/version/health" endpoint. Stub |
| Backup | 🔴 Stub | `BackupJob{ID, Schedule, Target, LastRun, Status}` — no scheduler, no `.db` snapshot/restore, no rotation, no integrity check, no retention policy. Stub |
| Task/jobs | 🔴 Stub | `Task{ID, Title, DueDate, Assignee, Status}` — no job registry, no queue, no retry, no cron. Stub |
| Migrations | 🟢 Ready | `db.Migrate` covers all module tables; `casbin_policies` + `audit_logs` live; admin tables (users, roles, options, tenants, backup_jobs, tasks) need adding to the fixed list |
| Laws/data-protection mapping | 🔴 Gaps | No **identity boundary** (header seam fails closed but is a dev token — violates "người dùng được phân quyền" TT 99 Điều 28 bảo mật). No **password/session policy** (NĐ 13/2023 PDPD arts. 20, 24; Luật ANM 116/2025). No **backup/retention** honoring Luật Kế toán Điều 41 (5/10/vĩnh viễn) + Bộ Tài chính guidance on electronic-archive custody. No **khóa sổ** logging beyond setup |
| UI | 🔴 Missing | No admin screens: no login, no user/role management UI, no settings page, no backup screen, no audit viewer outside setup dashboard |
| Tests | 🟡 Partial | Only authz + audit tested. Zero tests for user/options/system/backup/task |
| Integration seams | 🟡 Partial | Main.go wires admin services into `/api/v1` but everything returns 501; audit is reusable (setup calls it); no auth resolver swap seam beyond header |

**Verdict: the Admin/Configuration layer cannot be used in production today.**
Authorization enforcement works and setup is real, but there is **no identity
boundary** (anyone who can set `X-User-Id` is any user — the header is a dev
seam, and `role:admin` grants `* *` to whoever presents `admin`), **no
user/role management**, **no backup**, **no operational settings**, and **no
background jobs**. A PROD-pilot cashier/ledger/setup deployment works
functionally but has **no way to establish who is acting, no way to protect or
recover the books, and no audit surface for admin actions**. This is the layer
that makes the product *operable* rather than *demonstrable*.

## 2. What "production-ready" means for this layer

Production-ready = the system can be **operated by non-developers in a real
company** with a defensible identity boundary, recoverable data, and
enforceable statutory controls. Concretely:

1. **Real identity + session** — a login with credentials (password hashed,
   e.g. argon2/bcrypt — no plaintext), server session or signed token, logout,
   session expiry + lockout policy (NĐ 13/2023 arts. 20, 24; Luật ANM
   116/2025), and a `PrincipalResolver` that reads the authenticated principal
   **instead of a spoofable header**. Fail closed stays.
2. **User & role management** — CRUD users, assign 1..N roles, activate/
   suspend; roles mirror the seeded Casbin roles plus custom ones; only
   `role:admin`/`ke_toan_truong` manage accounts; every change audited.
3. **Operational settings** — a typed key/value store (string/bool/number/
   JSON) with defaults + validation + change audit; per-module config (company
   info override, currency, number formats, backup settings).
4. **Backup & restore with retention** — scheduled snapshot of the SQLite file
   (or `VACUUM INTO`), verify + checksum, rotation honoring Luật Kế toán Điều
   41 retention (5 năm / 10 năm / vĩnh viễn), one-click restore to a staging
   copy with confirmation, restore never mutates the live DB without a second
   audit + approval.
5. **Background jobs** — a small job runner (queued, retryable, auditable)
   used by backup schedules and later by year-end close, reclassification
   sweeps, e-invoice sync.
6. **System health/info** — `GET /api/v1/system/info` (version, uptime, DB
   reachable, last backup), used by ops and the UI footer.
7. **Full admin audit** — every admin action (login, logout, user grant,
   role change, setting change, backup run, restore) writes `audit_logs` with
   actor, action, target, timestamp.
8. **Authz for admin APIs** — a `role:admin`-gated policy set for
   `/api/v1/user`, `/options`, `/system`, `/backup`, `/task`; no admin route
   reachable by anonymous or low-privilege roles.

## 3. Regulatory basis (verified current, 2026-08-16)

Sources checked: thuvienphapluat.vn / vbpl.vn / congbao.chinhphu.vn /
mof.gov.vn / Bộ Công an feeds (cross-referenced with MISA/Fast/Bravo release
notes and community practice).

- **Luật Kế toán 2015 (88/2015/QH13), consolidated 41/VBHN-VPQH (16/03/2026)**
  — **Điều 26**: sổ kế toán phải được quản lý, bảo quản, lưu trữ bằng
  phương tiện điện tử **có biện pháp bảo mật, an toàn thông tin** và phải
  được **sao lưu** theo quy định. **Điều 41**: tài liệu kế toán phải lưu trữ
  tối thiểu **5 năm** (chứng từ kế toán), **10 năm** (sổ kế toán, báo cáo
  tài chính, báo cáo quyết toán), **vĩnh viễn** (tài liệu có liên quan đến
  thanh lý, bàn giao tài sản cố định, hóa đơn, chứng từ của giao dịch thanh
  toán qua ngân hàng). → **backup module = statutory obligation, not nice-to-have**.
- **Thông tư 99/2025/TT-BTC** (eff 01/01/2026, replaces TT 200/2014) —
  **Điều 28**: phần mềm kế toán phải đáp ứng — (a) chấp hành các quy định về
  kế toán và thuế, chống trùng lắp, sửa chữa phải lưu **dấu vết theo trình tự
  thời gian** (không được xóa dữ liệu gốc); (b) **bảo đảm an toàn, bảo mật
  thông tin**; (c) phân quyền truy cập, quản lý người sử dụng; (d) khóa sổ
  theo quy định; (e) xuất sổ, báo cáo theo đúng biểu mẫu. → identity + role
  management + audit are *software prerequisites for using the software legally*.
- **Luật Quản lý thuế 2025 (108/2025/QH15)** (eff 01/07/2026) + **NĐ 254/2026/NĐ-CP**
  (eff 01/07/2026, replaces NĐ 123/2020 detail guidance) — electronic
  invoicing end-to-end; operator identity on e-invoice issuance must be
  traceable. → user accounts must persist and be attributable per transaction.
- **Nghị định 13/2023/NĐ-CP (PDPD)** — **Điều 20**: rà soát, cập nhật quyền
  truy cập định kỳ; **Điều 24**: biện pháp quản lý truy cập (đăng nhập, xác
  thực, mật khẩu mạnh, khóa tài khoản khi sai nhiều lần). **Điều 35-36**:
  kiểm soát đơn vị xử lý dữ liệu; retention of access logs. → password/lockout
  + access-log policy is a data-protection duty, not optional.
- **Luật An ninh mạng 2025 (116/2025/QH15)** (eff 01/07/2026) — network
  security obligations for systems processing personal data in VN; log
  retention for system administration.
- **Luật Giao dịch điện tử (20/2023), VBHN 36/VBHN-VPQH (2026)** — legal
  value of electronic signatures/records; amended by Luật Dữ liệu 60/2024,
  Luật ANM 116/2025, Luật CĐS 148/2025. → system records (audit trail) are
  admissible evidence if integrity is provable.
- **Luật Dữ liệu 60/2024/QH15** (eff 01/07/2025) — personal data processing
  duties; staff accounts hold PII (full name, username).
- **Chữ ký số**: NĐ 130/2018/NĐ-CP + NĐ 48/2024/NĐ-CP (VBHN 06/VBHN-BTTTT);
  Luật GDĐT 20/2023 amendments — relevant when approval steps (khóa sổ,
  restore, role grant) require strong authentication; scoped out of v1 except
  as an auth factor option.
- **TT 32/2017/TT-BTC + TT 98/2018/TT-BTC** — lưu trữ, bảo quản tài liệu điện
  tử kế toán và chuyển đổi sang điện tử: nguyên tắc "lưu trữ trên hệ thống có
  biện pháp kỹ thuật bảo đảm không bị thay đổi" → backup + checksum must
  protect against silent modification.

## 4. Competitor scan (all active and TT 99-ready, 2026)

| Product | Admin/Configuration capabilities goGL must match |
|---|---|
| **MISA AMIS Kế toán** (web, R-series 2026: R91 21/04/2026, R92 15/05/2026) | "Các tiện ích và thiết lập → Quản lý người dùng và phân quyền": users + roles, per-function permission matrix (sử dụng/thêm/sửa/xóa/in), active/inactive, audit of admin changes. Đăng nhập bằng mật khẩu/số điện thoại/email; khóa tài khoản sau N lần sai; nhật ký đăng nhập. Tự động sao lưu dữ liệu theo lịch, khôi phục theo yêu cầu |
| **Fast Accounting** | "Quản trị Hệ thống" phân hệ: user + password policy, phân quyền theo menu/chức năng, đơn vị cơ sở, sao lưu/phục hồi dữ liệu, nhật ký hoạt động hệ thống |
| **BRAVO 10 ERP** | "Quản trị hệ thống": workflow định nghĩa (approval chains), phân quyền theo vai trò, nhật ký sử dụng, tự động sao lưu theo lịch, ISO/IEC 27001:2022 certified |
| **Community practice** (go-admin, casbin-admin) | RBAC UI over Casbin policies (role + policy editors), operation logs, JWT sessions; **lesson**: keep the policy layer as the single source of truth and build UI on top of it rather than a parallel permission model |

**Differentiation gap:** goGL already has the Casbin core and audit. The table
stakes are: (1) swap the header seam for a real session-bound principal, (2)
build user/role management UI directly on the Casbin role model (roles are
already seeded for business flows), (3) ship backup with statutory retention,
(4) expose audit as an admin screen. **Do not** build multi-tenancy, LDAP/SSO,
or full workflow engines before the single-company core is sound.

## 5. Key points — what must change

1. **Identity boundary first.** Replace `HeaderPrincipalResolver` usage in
   PROD with session-bound resolution; the header seam stays only as a dev
   toggle behind `authorization.dev_mode`. Passwords must be hashed
   (argon2id/bcrypt), sessions server-side or signed+expiring, lockout after N
   failures (NĐ 13 Điều 24). `role:admin`'s `* *` policy is fine only because
   reaching that role requires an authenticated admin session.
2. **Implement the five stub modules against the existing seams.** `user`
   (accounts + role assignment writing into the same role vocabulary Casbin
   uses), `options` (typed settings with defaults + audit), `system` (info/
   health; hold tenancy for later), `backup` (SQLite snapshot via `VACUUM
   INTO`, retention tiers per Luật Kế toán Điều 41, restore-to-staging),
   `task` (job runner to drive backup schedules, later year-end close).
3. **Add tables to `db.Migrate`**: `users`, `roles`, `options`, `tenants`,
   `backup_jobs`, `tasks`, `sessions` (if server-side) — same `(id, data)`
   JSON shape.
4. **Wire admin policies** (mirror `setup_policies.go` pattern): only
   `role:admin` (+ `ke_toan_truong` for user/password management) touches
   admin APIs; read-only `system/info` open to all authenticated roles.
5. **Audit everything admin** — reuse `audit.Service`; admin actions land in
   `audit_logs` with actor/target/timestamp; audit viewer screen (read-only).
6. **Backup = statutory.** Scheduled + manual, checksummed, rotated by
   retention tier; restore always to staging copy first, requires
   `ke_toan_truong` approval, and is itself audited (TT 99 Điều 28, Luật Kế
   toán Điều 26/41, TT 32/2017).
7. **Admin UI** — a `/system` section: login, user & role management,
   settings, backup/restore, audit viewer. Reuse the existing Tailwind +
   `html/templates` pattern from `websetup`.
8. **Tests** — authz matrix for admin routes, session expiry/lockout,
   password hashing round-trip, backup+restore round-trip with integrity
   check, retention rotation, audit assertions on every admin mutation.

## 6. Recommendation

1. **Treat this as the next vertical.** Setup's dashboard already shows audit
   trails and the authz core is live; the layer now needs the identity
   boundary + admin modules to make cash/ledger/setup *operable* in a real
   company.
2. **Ship P0 gate first:** real session auth + `user` module + admin authz
   policies + login UI + `system/info`. Without identity, every other admin
   capability is meaningless and `role:admin` is spoofable via the header.
3. **Then P1: backup + retention + audit viewer** (statutory data protection).
4. **Then P2: options + task runner + settings/backup UI**.
5. **Full PROD checkpoint** = cash + ledger + masterdata + setup + admin
   layer all running on one TT 99/2025 fiscal year with real users, sessions,
   scheduled backups, and a defensible audit trail.
