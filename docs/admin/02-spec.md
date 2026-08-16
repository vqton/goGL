# Admin/Configuration Layer — Technical Specification

> Companion to `docs/admin/00-verdict.md`, `01-brd.md`. Target state for the
> five admin modules + identity boundary + admin UI. Status: DRAFT v1.0 —
> 2026-08-16. Architectural pattern follows the repo's 4-layer skeleton
> (domain/application/persistence/http) and the existing setup/authz/audit
> implementations.

## 1. Scope

Implements the Admin/Configuration layer over the already-live infrastructure:

- **Live**: `internal/infrastructure/authorization` (Casbin v3, `enforcer.go`,
  `middleware.go`, `setup_policies.go` pattern), `internal/application/audit`
  + `persistence/audit` (`audit_logs` table), `internal/config`, `setup`
  vertical, `db.Migrate`.
- **To implement**: `user`, `options`, `system`, `backup`, `task` (all four
  layers each), identity/session, admin UI, admin authz policies.

## 2. Requirements index

| # | Req | Layer | Source |
|---|---|---|---|
| R1 | Real session-bound identity; header seam dev-only | auth/infra | BRD US-1..6 |
| R2 | User + role CRUD, custom roles over Casbin | user | BRD US-7..12 |
| R3 | Typed settings store with defaults+validation+audit | options | BRD US-13..15 |
| R4 | Backup schedule+manual, checksum, retention tiers | backup/task | BRD US-16..20 |
| R5 | Restore-to-staging + approval + audit | backup | BRD US-19..20 |
| R6 | System info/health endpoint | system | BRD US-21 |
| R7 | Job runner (queue/retry/audit) | task | BRD US-22 |
| R8 | Admin authz policies (admin-only routes) | authz | BRD AC-2 |
| R9 | Admin audit on every admin mutation | audit | BRD AC-6 |
| R10 | Admin UI: login, users/roles, settings, backup, audit | web | BRD AC-7 |

## 3. Domain model

### 3.1 `user` — extend `internal/domain/user/entity.go`

```
User{
  ID, Username, FullName,
  PasswordHash,              // argon2id encoded string; never stored plain
  RoleCodes []string,        // role codes in the Casbin role vocabulary
  Status  (active|suspended),// core.Status
  MustChangePassword bool,
  CreatedAt, UpdatedAt,      // RFC3339
  CreatedBy, UpdatedBy,
}
Role{ Code, Name, IsSystem, Description, Policies []RolePolicy }
RolePolicy{ Subject string /*"role:code"*/, Object, Action string }
```

- Role codes align with the Casbin subjects already seeded
  (`role:danh_muc`, `role:ke_toan_tong_hop`, `role:ke_toan_truong`,
  `role:giam_doc`, `role:kiem_toan`, `role:admin`). Custom roles are stored in
  the same `casbin_policies` table via the adapter; `Role` is the CRUD view.
- **Guard (BRD US-12)**: before removing/revoking the last active
  `role:admin`, reject with `ErrLastAdmin`.

### 3.2 `options` — extend `internal/domain/options/entity.go`

```
Option{ ID, Key, Category, Value string /*JSON*/, Type (string|bool|int|json),
        Description, DefaultValue, Validation map[string]string,
        UpdatedAt, UpdatedBy }
```

- `FindByKey` returns the *typed* value; missing keys fall back to
  `DefaultValue`. Validation per type (e.g. `{min,max,pattern}`).
- Seed defaults at migration: `backup.dir`, `backup.enabled`,
  `backup.retention_years`, `session.idle_minutes`, `session.max_hours`,
  `auth.max_failures`, `auth.lockout_minutes`, `auth.min_password_len`.

### 3.3 `system` — replace `internal/domain/system/entity.go`

```
SystemInfo{ Version, Commit, UptimeSeconds, DBOK bool,
            LastBackupAt *time.Time, SessionCount int, Time time.Time }
Tenant{ ID, Code, Name, Status }   // kept for future multi-company; unused in v1
```

### 3.4 `backup` — extend `internal/domain/backup/entity.go`

```
BackupArtifact{ ID, Path, SizeBytes, SHA256, CreatedAt, Tier
                 (receipt|ledger|permanent), Trigger (manual|scheduled), Status }
BackupJob{ ID, Schedule (cron), TargetDir, Enabled, LastRunAt, LastResult,
           RetentionTier, CreatedBy }
```

- Tier mapping per **Luật Kế toán Điều 41** (configurable via options):
  chứng từ ≥5y, sổ/BCTC ≥10y, hóa đơn/giao dịch ngân hàng/TSCĐ vĩnh viễn.
- Artifact naming: `gogl-<yyyyMMdd-HHmmss>-<tier>.db.bak`.

### 3.5 `task` — replace stub Task with a job runner view

```
JobRun{ ID, Name, Kind, Status (queued|running|ok|failed|retrying),
        Attempt, MaxAttempts, LastError, StartedAt, FinishedAt, Log []string }
```

- `task` module is the *infrastructure* of schedules: `ScheduleJob`,
  `RunNow`, `ListRuns`. Backup (R4) registers jobs; future modules
  (year-end close, reclassification) register more.

## 4. Application layer

### 4.1 Identity/session (R1) — new, cross-cutting

- `NewSessionService(repo, clock)`: `Login`, `Logout`, `Validate`,
  `LockAccount`, `ChangePassword`.
- Session row (server-side, revocable): `id, user_id, created_at,
  last_seen_at, expires_at, ip, user_agent`. On each `Validate`: touch
  `last_seen_at`; reject if `expires_at < now` (idle or absolute).
- Lockout: failed attempts counter per user; at `auth.max_failures` set
  `locked_until = now + auth.lockout_minutes`; log `login_failed` + `locked`.
- Password: argon2id (crypto params from `golang.org/x/crypto`); store
  `$argon2id$v=19$m=65536,t=3,p=2$salt$hash`. Reject reuse of last 3 hashes.
- `PrincipalResolver` becomes session-backed: resolve from session cookie →
  user; **`HeaderPrincipalResolver` moves behind `authorization.dev_mode`
  config flag** (default off) so prod cannot be impersonated by a header.

### 4.2 `user` service

`CreateUser`, `GetUser`, `ListUsers`, `UpdateUser`, `Suspend`,
`AssignRoles`, `CreateRole`, `UpdateRole`, `DeleteRole`, `ListRoles`.
- Role create/update/delete writes Casbin policies through the adapter
  (upsert on deterministic policy id) — single source of truth stays the
  `casbin_policies` table.
- Every mutation audits `module=user`.

### 4.3 `options` service

`Get`, `Set`, `ListByCategory`, `ResetToDefault`. `Set` validates type+rule,
writes audit with old→new value.

### 4.4 `backup` service

- `RunBackup(tier)`: `VACUUM INTO 'gogl-<ts>-<tier>.db.bak'` on the live
  connection → compute SHA-256 → record artifact → rotate old artifacts per
  tier retention.
- `RunNow`, `Schedule`, `ListArtifacts`, `Restore(artifactID)`:
  1) `VACUUM INTO <staging>.db` a copy of live first (never touch live),
  2) open staging copy, count rows on `casbin_policies`, `audit_logs`,
     `ledger_*` and compare to live snapshot counts,
  3) compute checksum of restored file,
  4) mark `restore_pending` requiring `ApproveRestore` by
     `ke_toan_truong`/admin → swaps live DB path (close/reopen pool),
  5) audit every step.
- Restore is **full replace only** (BRD Q4); no merge.

### 4.5 `system` service

`GetInfo`: reads version (build var), uptime, `SELECT 1` on DB,
`LastBackupAt` from backup repo, active session count from session repo.

### 4.6 `task` service

`Schedule`, `RunNow`, `ListRuns`, `Retry`. A single worker goroutine drains a
queue; each run audits `module=task`. Retry 3x with exponential backoff.

## 5. Persistence

- Tables added to `db.Migrate` (same `(id, data)` JSON shape):
  `users`, `roles`, `options`, `sessions`, `backup_artifacts`,
  `backup_jobs`, `task_runs`. `tenants` table can be created but unused v1.
- Each repo = `NewSqliteRepository(db)` mirroring `persistence/audit`.
- `casbin_policies` continues to be the policy source of truth; `roles` table
  holds role metadata (code, name, system flag).

## 6. HTTP API

All under `/api/v1/<module>`, registered in `cmd/server/main.go` on the
protected group (behind `AuthorizationMiddleware`).

| Method | Path | Role | Req |
|---|---|---|---|
| POST | `/api/v1/auth/login` | anon (pre-auth) | R1 |
| POST | `/api/v1/auth/logout` | auth'd | R1 |
| POST | `/api/v1/auth/change-password` | auth'd | R1 |
| GET | `/api/v1/user` | admin/kt | R2 |
| POST | `/api/v1/user` | admin/kt | R2 |
| GET/PUT | `/api/v1/user/:id` | admin/kt | R2 |
| POST | `/api/v1/user/:id/suspend` | admin/kt | R2 |
| POST | `/api/v1/user/:id/roles` | admin/kt | R2 |
| GET/POST | `/api/v1/user/roles` | admin/kt | R2 |
| PUT/DELETE | `/api/v1/user/roles/:code` | admin/kt | R2 |
| GET/PUT | `/api/v1/options` | admin (read: all auth'd) | R3 |
| GET | `/api/v1/system/info` | all auth'd (read) | R6 |
| GET | `/api/v1/backup` | admin/kt | R4 |
| POST | `/api/v1/backup/run` | admin/kt | R4 |
| POST | `/api/v1/backup/schedule` | admin | R4 |
| GET | `/api/v1/backup/artifacts` | admin/kt | R4 |
| POST | `/api/v1/backup/restore` | admin/kt | R5 |
| POST | `/api/v1/backup/restore/approve` | kt/admin | R5 |
| GET | `/api/v1/task/runs` | admin | R7 |
| POST | `/api/v1/task/:name/run` | admin | R7 |
| GET | `/api/v1/audit?module=&limit=` | kiem_toan/admin/kt | R9 |

- Admin route policy file `admin_policies.go` mirrors `setup_policies.go`:
  explicit objects, no broad wildcards; `role:admin` covered by built-in `* *`.
- Login/change-password are pre-middleware (no subject yet); middleware allows
  the two paths by matching `AnonymousSubject`-compatible policies or by
  mounting them before the enforcer group.

## 7. UI (web layer)

New section under `internal/interfaces/http/webadmin/` using the existing
Tailwind + `html/templates` pattern (`base.html` + block templates) and
`identity_header` for dev rendering (replaced by session resolver in PROD).

- `/login` — username/password, lockout notice, generic error.
- `/system/` — admin dashboard: tabs Users / Roles / Settings / Backup /
  Audit / System info. Audit tab reuses `audit_logs` reader.
- `/system/backup/restore` — two-step restore wizard with staging verify
  results + confirm + approval fields.

## 8. Testing

- Authz matrix tests (pattern of `setup_policies_test.go`): admin routes
  denied for anonymous, `danh_muc`, `kiem_toan`; granted for admin/kt.
- Session: expiry, idle-touch, lockout after N, revocation on suspend.
- Password: hash round-trip, reuse rejection, no plaintext in DB.
- Backup/restore: round-trip through staging, checksum equality, retention
  rotation (old deleted, newest kept), concurrent run serialized.
- Options: typed get/set, default fallback, validation reject, audit row.
- Task: retry/backoff, per-run audit.
- `go test ./internal/domain/... ./internal/application/... ./internal/infrastructure/... ./internal/interfaces/http/...` plus the existing authz suite.

## 9. Migration & sequencing

1. Add tables + seed default options + roles metadata.
2. Implement session/auth; move header seam behind `dev_mode`.
3. Implement user (CRUD + roles over Casbin).
4. Implement system/info.
5. Implement backup + task (runner) + retention.
6. Implement options CRUD.
7. Admin UI + policies + login.
8. Full admin authz + audit wiring; tests.
