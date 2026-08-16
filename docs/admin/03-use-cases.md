# Admin/Configuration Layer (Quản trị hệ thống) — Use Cases

> UC numbering: A = Admin/Configuration. Each UC lists happy path, alternative
> paths, exception paths. Roles per `01-brd §3`; rules R1–R10 per `02-spec §2`.

## UC-A1 — Login / session

**Actor:** any user (pre-auth = anonymous). **Precondition:** store initialized.

**Happy path (A1-H)**
1. Actor opens /login → form username + password.
2. Actor submits. System validates credentials (argon2id) → generic error on
   wrong creds (no user enumeration) → creates session row, sets cookie.
3. System audits `login` (actor=username). → redirect to /system.

**Alternative A1:** First login with admin-issued initial password →
`MustChangePassword` flag → forced change-password screen before /system
(BRD Q6).
**Alternative A2:** Idle timeout → session `last_seen_at` not touched within
`session.idle_minutes` → next request 401 → re-login.

**Exceptions**
- **E1:** Wrong password ×N (default 5) → account locked for
  `auth.lockout_minutes`; audit `login_failed` + `locked`. Even correct
  password rejected until window elapses or admin unlocks.
- **E2:** Suspended user → 403 "tài khoản đã bị khóa" regardless of password.
- **E3:** Session expired (absolute `session.max_hours`) → 401 → re-login.
- **E4:** Anonymous on protected route → 401 (fail closed; header seam cannot
  impersonate when auth enabled).

## UC-A2 — Manage users

**Actor:** `ke_toan_truong` or `admin`. **Precondition:** authenticated, role OK.

**Happy path (A2-H)**
1. Actor opens Quản trị → Người dùng → list (username, full name, roles,
   status).
2. Actor "Thêm người dùng": username, full name, password (admin-set),
   roles (≥1).
3. System validates (unique username, password policy R1) → creates user →
   audit `user.create` (actor, target=username).
4. Actor "Gán vai trò": selects roles for an existing user → effective
   permissions = union of role policies (Casbin) → audit `user.roles`.

**Alternative A1:** Suspend user → next authenticated request rejected (session
dies) → audit `user.suspend` (BRD US-6/10).
**Alternative A2:** Change own password → current password required →
validated, hash replaced (not kept in history) → audit `user.password_change`.
**Alternative A3:** Activate a suspended user → audit `user.activate`.

**Exceptions**
- **E1:** Duplicate username → 409.
- **E2:** Password < policy (12 chars, complexity, ≠ last 3) → 422 field error.
- **E3:** Removing the last active `role:admin` → rejected `ErrLastAdmin`
  (guard, BRD US-12).
- **E4:** 403 (role insufficient) / 401 (session invalid).
- **E5:** Suspend self → allowed? Decision: blocked (cannot lock yourself out);
  delegate to second admin.

## UC-A3 — Manage roles (custom roles over Casbin)

**Actor:** `admin`. **Precondition:** authenticated.

**Happy path (A3-H)**
1. Actor opens Vai trò → list: system roles (read-only) + custom roles.
2. Actor "Tạo vai trò": code, name, attach policies (obj/act pairs — e.g.
   `GET /api/v1/reporting/sales`).
3. System writes role metadata to `roles` + policies to `casbin_policies`
   (upsert by deterministic id) → audit `role.create`.

**Exceptions**
- **E1:** Reserved code (already a system role) → 409.
- **E2:** Policy references a route pattern not registered → 422 warning
  (allowed but flagged) — decision: warn only.
- **E3:** Delete a role still assigned to users → blocked until unassigned,
  or cascade with confirmation → audit `role.delete`.
- **E4:** 403/401.

## UC-A4 — Configure settings

**Actor:** `admin`. **Precondition:** authenticated.

**Happy path (A4-H)**
1. Actor opens Cài đặt → categories (Backup, Session/Auth, …).
2. Actor edits `backup.retention_years` = 10 → system validates type+range →
   writes audit `options.set` old→new → UI confirms.

**Alternative A1:** Business module reads a key that was never written → typed
default returned (R3 fallback), no audit.
**Alternative A2:** Reset key to default → audit `options.reset`.

**Exceptions**
- **E1:** Value fails type/rule validation (e.g. string in int) → 422.
- **E2:** Unknown key → 404 (key schema must exist).
- **E3:** 403/401.

## UC-A5 — Backup (scheduled + manual)

**Actor:** system (scheduled) or `admin`/`ke_toan_truong` (manual).
**Precondition:** DB open; backup dir writable.

**Happy path (A5-H)**
1. Trigger (schedule fires or actor clicks "Sao lưu ngay").
2. System `VACUUM INTO` snapshot → SHA-256 → write
   `gogl-<ts>-<tier>.db.bak` → record artifact → rotate per tier retention →
   audit `backup.run` (success, artifact id).
3. Admin dashboard shows last-success time + artifact list.

**Alternative A1:** Scheduled run races manual run → serialized via task queue
(one backup at a time) → second waits or is dropped with `already_running`.
**Alternative A2:** Target dir missing → attempt mkdir; if fails → job marked
`failed`, audit `backup.failed`, retry (3x backoff), dashboard badge.

**Exceptions**
- **E1:** SQLite busy → retry with backoff; if still busy → failed + audit.
- **E2:** Disk full → failed + audit + surface alert (UI badge + audit entry).
- **E3:** 403/401.

## UC-A6 — Restore (two-step, approved)

**Actor:** `ke_toan_truong` (approve) + operator/admin (initiate).
**Precondition:** ≥1 artifact; actor role OK.

**Happy path (A6-H)**
1. Actor opens Sao lưu → Khôi phục → selects artifact + confirms intent.
2. System copies live DB → staging, opens staging copy, compares row counts
   (`casbin_policies`, `audit_logs`, key ledger tables) vs live snapshot,
   checksums restored file → shows verification report. **Live DB untouched.**
   Audit `backup.restore_stage`.
3. Actor (or a second authorized admin) **approves** → system swaps live DB
   path (close/reopen pool) → audit `backup.restore_approve` → dashboard
   shows restored-to timestamp.

**Alternative A1:** Verification fails (counts/checksum mismatch) → restore
**aborted**, no swap, audit `backup.restore_failed`.
**Alternative A2:** Only one admin available → self-approve blocked unless
`backup.allow_self_approve` (default false; decision per BRD Q6).

**Exceptions**
- **E1:** No artifacts → 404.
- **E2:** Approval by wrong role → 403.
- **E3:** Live DB path locked by a writer → block with 409 "đang có phát sinh".

## UC-A7 — System info / health

**Actor:** any authenticated user. **Precondition:** session valid.

**Happy path (A7-H):** `GET /system/info` → version, uptime, DBOK=true,
last backup, session count → footer + ops dashboard.

**Exceptions:** **E1** DB unreachable → DBOK=false + 200 (info still
returns); **E2** 401 anonymous.

## UC-A8 — Audit viewer

**Actor:** `kiem_toan`, `ke_toan_truong`, `admin`. **Precondition:** role OK.

**Happy path (A8-H):** open Nhật ký hệ thống → filter module/action/actor/
time range → rows from `audit_logs` (read-only) → export CSV.
**Exceptions:** **E1** 403 for other roles; **E2** empty result → empty state.

## UC-A9 — Background job runner

**Actor:** system (scheduler) / `admin` (manual trigger). **Precondition:** job
registered.

**Happy path (A9-H):** scheduler enqueues `backup` → worker runs → status
queued→running→ok → audit `task.run` with log tail. Dashboard shows runs.
**Exceptions:** **E1** retry budget exhausted → failed + audit; **E2** worker
down at tick → job stays queued, picked up next tick; **E3** 403.
