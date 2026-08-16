# Admin/Configuration Layer (Quản trị hệ thống) — Processes & Data Flows

## 1. Identity / session lifecycle

```
┌─ pre-auth (AnonymousSubject) ────────────────┐
│ POST /auth/login {username, password}         │
│   ├─ N failures ≥ max → locked_until = now+X  │  audit login_failed|locked
│   ├─ wrong creds → 401 generic                │
│   ├─ suspended → 403                          │
│   └─ ok → session row (sessions table) +      │  audit login
│        httpOnly cookie                        │
└──────────────────────────────────────────────┘
             │ every protected request
             ▼
   PrincipalResolver (session-backed) ── user id ──► Casbin Enforce(sub, obj, act)
             │ session valid? idle? absolute?        │ allowed? ──► handler
             │ expired → 401, delete session         │ denied  ──► 403
             └─ logout → delete session → audit logout
```

- Header seam (`X-User-Id`) moves behind `authorization.dev_mode` (default
  **off**) — production identity is session-only. Anonymous always fails
  closed unless a policy grants it (none do on admin routes).

## 2. User & role management flow (over the Casbin single source of truth)

```
Quản trị → Người dùng / Vai trò
  │
  ├─ CreateUser → users row (argon2id hash, RoleCodes) → audit user.create
  ├─ AssignRoles → update users.RoleCodes → audit user.roles
  │   └─ effective permissions = union of Casbin role policies (no parallel model)
  ├─ CreateRole  → roles row + casbin_policies rows (upsert, deterministic id)
  │                  └─ custom policy (role:X, obj, act) joins the enforcer
  ├─ Suspend     → users.status=suspended → sessions invalidated → audit
  └─ guard: last active role:admin cannot be revoked (ErrLastAdmin)
```

## 3. Settings flow

```
options.Set(key, value)
  ├─ schema exists? (options seeded at migrate) ──no──► 404
  ├─ validate type + rule (string|bool|int|json; min/max/pattern) ──fail──► 422
  ├─ write JSON value → audit options.set {old → new}
  └─ business modules read via FindByKey → typed default if never written
```

## 4. Backup lifecycle (statutory retention)

```
trigger: schedule (task runner) | manual (admin/kt)
  ▼
VACUUM INTO gogl-<ts>-<tier>.db.bak   (tier = receipt|ledger|permanent)
  ▼
SHA-256 → backup_artifacts row (id, path, size, hash, tier, trigger, status)
  ▼
rotation: delete artifacts older than tier retention, keep newest valid ≥ 1
          tier map (default, configurable): chứng từ ≥5y · sổ/BCTC ≥10y
          · hóa đơn/giao dịch ngân hàng/TSCĐ vĩnh viễn   (Luật Kế toán Điều 41)
  ▼
audit backup.run{ok|failed} → dashboard last-backup badge
```

## 5. Restore flow (two-step, never mutates live implicitly)

```
restore request {artifactID}
  ├─ stage: copy live → <staging>.db, verify counts vs live snapshot,
  │           checksum restored file ──mismatch──► abort + audit backup.restore_failed
  ├─ show verification report (read-only; live untouched) → audit backup.restore_stage
  ├─ approval: ke_toan_truong/admin (self-approve blocked unless allowed)
  │             → audit backup.restore_approve
  └─ swap: close/reopen DB pool on staging path → audit backup.restore
```

## 6. Job runner flow

```
scheduler tick ──enqueue──► queue ──worker──► run{queued→running→ok}
                                       │ retry ×3 (backoff) on failure
                                       └─ audit task.run {name, status, log tail}
   future consumers: year-end close, reclassification sweeps, e-invoice sync
```

## 7. Admin audit trail (every admin mutation lands in audit_logs)

```
module=user    : create, update, suspend, activate, roles, password_change
module=role    : create, update, delete
module=options : set, reset
module=backup  : run, run_failed, restore_stage, restore_approve, restore, restore_failed
module=task    : run, retry, failed
module=auth    : login, login_failed, locked, logout, session_expired
readers: kiem_toan / ke_toan_truong / admin (viewer UC-A8; audit write is
         service-internal only — no user-triggerable endpoint writes logs)
```

## 8. Authz policy map (admin routes — mirror `setup_policies.go` style)

| Route group | anonymous | danh_muc | kế toán tổng hợp | kiểm toán | kế toán trưởng | giam_doc | admin |
|---|---|---|---|---|---|---|---|
| `/auth/login` | ✅ | — | — | — | — | — | — |
| `/auth/logout`, `/auth/change-password` | — | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/system/info` | — | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/user` (CRUD), `/user/roles` | — | — | — | — | ✅ | — | ✅ |
| `/options` write | — | — | — | — | — | — | ✅ |
| `/options` read | — | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/backup/run`, `/backup/artifacts` | — | — | — | — | ✅ | — | ✅ |
| `/backup/schedule` | — | — | — | — | — | — | ✅ |
| `/backup/restore`, `/backup/restore/approve` | — | — | — | — | ✅ | — | ✅ |
| `/task/*` | — | — | — | — | — | — | ✅ |
| `/audit` (read) | — | — | — | ✅ | ✅ | — | ✅ |
