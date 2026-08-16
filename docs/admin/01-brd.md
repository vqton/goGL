# Admin/Configuration Layer — Business Requirements (BRD)

> Owner: BA lead + chief accountant (goGL). Scope: the five admin modules
> (`user`, `options`, `system`, `backup`, `task`) plus the identity boundary
> and admin UI that make the product operable. Status: DRAFT v1.0 — 2026-08-16.
> Companion docs: `00-verdict.md`, `02-spec.md`, `03-use-cases.md`,
> `04-processes.md`, `05-ui.md`, `06-roadmap.md`, `08-security.md`.

## 1. Business context

goGL's cashier, ledger, masterdata, and setup modules are PROD-pilot
ready. They solve the *bookkeeping* problem. The Admin/Configuration layer
solves the *operation* problem: **who** may do what, **what** the system is
configured to do, **that** the books survive failure, and **how** we prove it.

Statutory pressure (verified 2026-08-16):
- **TT 99/2025/TT-BTC Điều 28** (eff 01/01/2026) — accounting software *must*
  support phân quyền truy cập, quản lý người sử dụng, bảo mật thông tin, và
  lưu dấu vết sửa chữa theo trình tự thời gian.
- **Luật Kế toán 88/2015 Điều 26, 41** (VBHN 25/VBHN-VPQH 2025) — electronic
  books must be securely stored **and backed up**; documents retained 5/10/
  vĩnh viễn years by type.
- **NĐ 13/2023/NĐ-CP arts. 20, 24** — access control, strong passwords,
  lockout; periodic access review.
- **Luật ANM 116/2025, Luật Dữ liệu 60/2024** — security duties for systems
  processing VN data; staff accounts hold PII.

Without this layer, the product fails TT 99 Điều 28 as a matter of law, and a
production outage means the books are unrecoverable.

## 2. Goals / non-goals

### Goals
- G1. Establish a **real identity boundary**: authenticated sessions replace
  the `X-User-Id` dev header in production.
- G2. **Manage users & roles** so every transaction is attributable and role
  grants follow TT 99 Điều 28 / NĐ 13 Điều 20.
- G3. **Protect the data**: scheduled, checksummed, retention-aware backups
  and a safe restore path (Luật Kế toán Điều 26/41).
- G4. **Configure operations** via a typed, validated, audited settings store.
- G5. **Run background jobs** (backup schedules now; year-end close later).
- G6. **Show system health/info** to operators.
- G7. **Audit admin actions** — full trail from the same `audit_logs` used by
  business modules.
- G8. **Admin UI** for all of the above (login → manage → verify), matching
  the existing Tailwind template pattern.

### Non-goals (v1)
- Multi-tenancy / SaaS isolation (single-company product).
- LDAP/SSO/2FA-as-a-service; token-based SSO out of scope until sessions are
  stable (2FA via TOTP is a P2 candidate).
- Full workflow/approval engine (BRAVO-style) — only the audit + role grants
  needed for statutory control.
- Realtime presence / concurrent-edit conflict resolution.
- Public cloud backup targets (S3 etc.) in v1 — local path + mounted volume;
  target abstraction deferred.

## 3. Personas

| Persona | Role(s) | Needs |
|---|---|---|
| **Chủ doanh nghiệp / Giám đốc** | `giam_doc` | Read access to books + system status; approve nothing in v1 (approval flows deferred) |
| **Kế toán trưởng** | `ke_toan_truong` | Manage users/roles, run + approve restore, lock/unlock books, view audit |
| **Kế toán tổng hợp** | `ke_toan_tong_hop` | Day-to-day entry; read-only system info |
| **Kế toán viên (phân hệ)** | `danh_muc`, `cash`, `ledger` roles | Do their job; must not see admin functions |
| **Kiểm toán** | `kiem_toan` | Read-only access incl. audit trail, no mutation |
| **Quản trị hệ thống (operator)** | `role:admin` | Full control: users, settings, backup/restore, jobs, audit |

> Roles shown are the seeded set; custom roles are addable in v1 via the user
> module (role = named set of Casbin policies).

## 4. User stories / requirements

### Identity & session (G1, G7)
- US-1. A user logs in with username + password; wrong credentials produce a
  generic error (no user enumeration). *NĐ 13 Điều 24; security*
- US-2. After N consecutive failures (default 5), the account locks for a
  window (default 15 min); admin can unlock. *NĐ 13 Điều 24*
- US-3. Sessions expire on idle (default 30 min) and on absolute lifetime
  (default 12 h); logout invalidates the session. *NĐ 13 Điều 24*
- US-4. Password policy enforced: min length (12), complexity (letter+digit+
  symbol), must differ from previous 3. *NĐ 13 Điều 24*
- US-5. Every login success/failure and logout is recorded in `audit_logs`
  (actor=username, action=login/login_failed/logout). *Luật ANM, TT 99 Điều 28*
- US-6. A suspended/deleted user's session dies at the next request; their
  historical records remain attributable (identity is never hard-deleted).

### User & role management (G2, G7)
- US-7. Admin creates a user: username, full name, password (or invite +
  first-login change), status active/suspended. *TT 99 Điều 28(c)*
- US-8. Admin assigns 1..N roles; a user's effective permissions = union of
  role policies (Casbin). *TT 99 Điều 28(c)*
- US-9. Admin can create a custom role and attach policies to it (UI over the
  Casbin policy store). *Differentiation*
- US-10. Admin suspends/unsuspends a user; suspension takes effect at the
  next authenticated request.
- US-11. Every user/role mutation is audited (who, whom, what, when).
- US-12. Only `role:admin` and `ke_toan_truong` reach user/role management
  endpoints; nobody can revoke their own `role:admin` in a way that locks the
  system (guard: at least one active admin must remain).

### Settings (G4, G7)
- US-13. Admins read/update typed settings (string/bool/int/JSON) grouped by
  category; each has a schema-defined default.
- US-14. Settings changes are validated and audited (old → new value).
- US-15. Business modules read settings via the options service without
  owning them (e.g., backup dir, upload limits, session policy knobs).

### Backup & restore (G3, G7)
- US-16. Admin configures a backup schedule (daily/weekly + time) and target
  directory; system runs it via the task runner. *Luật Kế toán Điều 26*
- US-17. Manual backup on demand. Backup = SQLite snapshot (`VACUUM INTO`) +
  checksum + timestamped artifact.
- US-18. Retention tiers per **Luật Kế toán Điều 41**: chứng từ ≥ 5 năm; sổ/
  BCTC ≥ 10 năm; hóa đơn/giao dịch ngân hàng/TSCĐ vĩnh viễn. Backups carry a
  tier tag; rotation honors the longest applicable tier.
- US-19. Restore is a two-step, audited flow: pick artifact → restore to a
  **staging copy** → verify (row counts, checksum) → operator confirms →
  approved by `ke_toan_truong`/admin → swap. Restore never mutates the live
  DB implicitly.
- US-20. Backup/restore events (success, failure, verification) are audited
  and surfaced on the admin dashboard.

### System info (G6)
- US-21. `GET /system/info`: app version, commit, uptime, DB reachability,
  last successful backup time, active session count. Read-only, available to
  all authenticated roles.

### Background jobs (G5)
- US-22. A job runner executes scheduled jobs (backup first), with queued
  execution, retry (3x, backoff), per-run audit, and failure alerting surface
  (audit + UI badge).

## 5. Acceptance criteria (summary — detailed in 02-spec)

- AC-1. Login with valid credentials works; `X-User-Id` alone **cannot** be
  used to impersonate when auth is enabled (header seam dev-only).
- AC-2. Role matrix enforced end-to-end on admin routes (401 anonymous, 403
  wrong role, 200 granted) — verified by tests.
- AC-3. Password hashing + lockout + session expiry verified by tests; no
  plaintext password in DB or logs.
- AC-4. Backup runs on schedule and on demand; artifacts are checksummed;
  restore round-trips a database through staging and verifies before swap.
- AC-5. Retention rotation deletes only artifacts older than their tier and
  never the newest valid backup.
- AC-6. Every US-* admin action has a corresponding `audit_logs` row with
  actor/action/target/timestamp.
- AC-7. All admin UI reachable through the Tailwind template pattern; login
  gating on `/system` section.

## 6. Open questions / decisions needed

- Q1. Session store: server-side DB table (`sessions`) vs signed stateless
  token? Recommend server-side (revocable on suspend) for v1.
- Q2. Backup target in v1: local path only, or also SMB/NFS mount / S3?
  Recommend local path + documented mount point; S3 deferred.
- Q3. Role granularity: reuse seeded roles + custom, or fixed role set? Recommend
  custom roles with guard that seeded roles are immutable system roles.
- Q4. Does restore need to *delete* live DB rows first (replace) or
  merge? Recommend full replace only (no merge) in v1.
- Q5. Are user accounts tied to a real HR record / employee id? Recommend
  plain `username + full_name` in v1; employee link deferred.
- Q6. Invite flow (email) or admin-set-password only? Recommend admin-set
  initial password + forced change at first login (no email dependency).
