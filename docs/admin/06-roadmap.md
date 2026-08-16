# Admin/Configuration Layer (Quản trị hệ thống) — Implementation Roadmap

> Mirrors the cash/ledger/masterdata/setup phase pattern (P0→P5). **Precondition
> P0 is a hard gate.** Each phase ends with `go build ./... && go vet ./... &&
> go test ./...` green and a commit. Test-first (red-green-refactor); ≥ 80%
> coverage on service + repository. Tasks tracked in `tasks/admin-todo.md`.

Dependencies **already implemented**: Casbin authz (enforcer, adapter,
middleware, per-module policy files), audit (`audit_logs`), setup vertical
(first-run gateway), `db.Migrate`, Tailwind web stack, `websetup` handler
pattern. The layer is mostly **new modules on live seams** — the risk is the
identity boundary and backup safety, not infrastructure.

## P0 — Identity boundary: sessions + login (hard gate) [1 wk]

- A0.1 Migration: add `users`, `roles`, `sessions` tables to `db.Migrate`
  (same `(id, data)` JSON shape); seed default admin user + roles metadata
  when store empty (mirror `SeedDefaultPolicies`).
- A0.2 Session service: `Login/Logout/Validate`, server-side session row,
  idle + absolute expiry (BRD US-3), lockout after N failures (BRD US-2),
  argon2id password hashing + policy (BRD US-4, US-1), audit on every auth
  event (`module=auth`: login/login_failed/locked/logout).
- A0.3 Move `HeaderPrincipalResolver` behind `authorization.dev_mode`
  (default off); add session-backed `PrincipalResolver`; **`X-User-Id` alone
  must NOT authenticate when auth enabled** (AC-1).
- A0.4 `POST /auth/login`, `/auth/logout`, `/auth/change-password` (pre-auth
  routes); authz matrix test (anonymous → 401, no impersonation).
- A0.5 Login page (`05-ui §2.1`).

**Checkpoint:** build/vet/test green; lockout + expiry + impersonation-reject
tests pass; no plaintext password in DB or logs.

## P1 — User & role management [1–2 wks]

- P1.1 Implement `user` service + repo: `CreateUser`, `GetUser`, `ListUsers`,
  `UpdateUser`, `Suspend`, `AssignRoles`; `MustChangePassword` flag; guard
  `ErrLastAdmin` (BRD US-12).
- P1.2 Roles over Casbin: `CreateRole`/`UpdateRole`/`DeleteRole` write role
  metadata to `roles` + policies to `casbin_policies` (upsert by deterministic
  id); system roles read-only; custom roles join the enforcer.
- P1.3 `user` HTTP endpoints per `02-spec §6` + `admin_policies.go` (explicit
  objects; admin + kế toán trưởng only); audit every mutation.
- P1.4 Web UI: Users + Roles tabs (`05-ui §2.2–2.3`).
- Tests: authz matrix, suspend kills next request, last-admin guard, role
  policy upsert/idempotency, audit rows on every mutation.

## P2 — Backup, restore, retention [1–2 wks]

- P2.1 Migration: `backup_artifacts`, `backup_jobs` tables.
- P2.2 `backup` service: `RunBackup` (`VACUUM INTO` + SHA-256 + artifact row),
  schedule config, rotation by tier retention (Luật Kế toán Điều 41: chứng từ
  ≥5y, sổ/BCTC ≥10y, hóa đơn/ngân hàng/TSCĐ vĩnh viễn; keep newest valid ≥1).
- P2.3 `Restore` two-step: stage copy → verify (row counts vs live snapshot +
  checksum) → approve (`ke_toan_truong`/admin, self-approve blocked by
  default) → swap DB path (close/reopen pool). Never mutates live implicitly.
- P2.4 `system/info` (version, uptime, DBOK, last backup, session count) —
  read for all auth'd roles.
- P2.5 Web UI: Backup tab + restore wizard (`05-ui §2.5–2.6`).
- Tests: backup→restore round-trip, checksum equality, retention rotation
  (old deleted / newest kept), concurrent-run serialization, restore abort on
  verification mismatch, approval role matrix.

## P3 — Settings + job runner [1 wk]

- P3.1 Migration: `options`, `task_runs` tables; seed default options.
- P3.2 `options` service: typed get/set (string|bool|int|json), default
  fallback, validation, audit old→new (BRD US-13..15).
- P3.3 `task` service: worker goroutine queue, `Schedule/RunNow/ListRuns/
  Retry`, retry ×3 backoff, per-run audit; backup registers as the first job
  consumer.
- P3.4 Web UI: Settings tab + audit viewer (`05-ui §2.4, §2.7`).
- Tests: typed validation, default fallback, retry/backoff, job audit,
  settings audit old→new.

## P4 — Integration hardening + regression [1 wk]

- P4.1 Full admin authz sweep: every `/user`, `/options`, `/system`,
  `/backup`, `/task`, `/audit` route in the policy matrix (`04-processes §8`);
  fail-closed audit (no anonymous grants).
- P4.2 Restore-under-load guard (writer present → 409), backup dir permissions
  check, session revocation on suspend, parallel-login concurrency test.
- P4.3 Regression: rerun full `go test ./...` incl. authz + audit suites;
  bench backup of a 12-month ledger book; security review (`08-security.md`).
- P4.4 README/docs: operator runbook (install → configure → login → first
  backup → restore drill) + `docs/admin/` updates.

## P5 — Security review + PROD sign-off [1 wk]

- P5.1 Security review pass per `08-security.md` checklist; fix findings.
- P5.2 Coverage gate ≥ 80% on service+repo; benchmarks; vet clean.
- P5.3 Restore drill against a seeded prod-like DB (documented runbook).
- P5.4 **Sign-off checklist:** AC-1..AC-7 from `01-brd §5` all green.

## Ordering rationale

- **Identity first (P0)** — without it `role:admin` is spoofable and no admin
  capability is meaningful; also unblocks user/role P1.
- **Data protection second (P2)** — statutory (Luật Kế toán Điều 26/41, TT 99
  Điều 28); a running system with no backup is a liability.
- **Settings/jobs last (P3)** — they power backup schedules and future
  year-end flows, but are not on the critical path.
- **Custom roles, tenancy, SSO, 2FA, S3 targets** — explicitly deferred
  (BRD §2 non-goals / Q1–Q6).

## Full PROD checkpoint

cash + ledger + masterdata + setup + **admin layer** all running on one
TT 99/2025 fiscal year, with real users, sessions, scheduled backups, restore
drill passed, and a defensible audit trail.
