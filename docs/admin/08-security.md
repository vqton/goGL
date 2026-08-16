# 08 — Security review (Admin/Configuration layer)

Scope: the Admin/Configuration layer — identity/session, user & role
management, settings, backup/restore, job runner, system info, admin audit
viewer, and the admin UI. Written against the **target state** in
`02-spec.md`; revisit and verify per commit when implemented (mirror the
setup/ledger security reviews).

Review checklist: [ ] authz matrix, [ ] identity/session hardening,
[ ] password handling, [ ] backup/restore safety, [ ] audit trail completeness,
[ ] input validation, [ ] secrets handling, [ ] documented residual risks.

## 1. Attack surface

| Surface | Transport | Authz gate | Notes |
|---|---|---|---|
| JSON API `/api/v1/user`, `/options`, `/system`, `/backup`, `/task`, `/audit` | HTTP | Casbin middleware | Fail-closed; admin/chief-only on mutating routes |
| `/api/v1/auth/*` (login, logout, change-password) | HTTP | pre-auth (anonymous allowed only for login) | Rate-limit + lockout on login; never leak user existence |
| Admin UI `/login`, `/system/**` | HTTP | session-backed principal + role gate | Root group renders server-side; mutators must re-check principal server-side |
| Backup artifacts (files) | filesystem | OS permissions on target dir | Artifacts contain the full ledger — treat like the live DB |

## 2. Identity & session hardening

- **Passwords**: argon2id (`m=65536,t=3,p=2`, salt per-user, ≥12 chars,
  complexity, reuse-of-last-3 rejected). Never store/log plaintext; generic
  login error prevents user enumeration (NĐ 13 Điều 24).
- **Sessions**: server-side row + httpOnly cookie; idle + absolute expiry;
  revocable on suspend/logout. CSRF defense required on the web UI (cookie-
  based auth) — add per-form CSRF tokens before PROD.
- **Lockout**: N failures (default 5) → locked window (default 15 min);
  admin unlock path. Audit `login_failed`/`locked`.
- **Impersonation**: `X-User-Id` header seam must be dev-only
  (`authorization.dev_mode`, default off). In PROD the header must be ignored
  by `PrincipalResolver` (AC-1). **This is the single highest-risk fix.**
- **Session fixation**: rotate session id on login; logout invalidates server
  row + clears cookie.

## 3. Authorization matrix (admin routes — target)

From `04-processes §8`. Mutating admin endpoints are **admin / chief-accountant
only**; no anonymous grants; system roles read-only; `role:admin` covered by
the built-in `* *` policy but only reachable through an authenticated admin
session.

| Route | anon | danh_muc | ke_toan_tong_hop | kiem_toan | ke_toan_truong | giam_doc | admin |
|---|---|---|---|---|---|---|---|
| `POST /auth/login` | ✓ | – | – | – | – | – | – |
| `POST /auth/logout`, `/auth/change-password` | – | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `GET /system/info` | – | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `GET /user`, `/user/:id`, `/user/roles` | – | – | – | – | ✓ | – | ✓ |
| `POST/PUT /user`, `/user/:id/*`, `/user/roles` | – | – | – | – | ✓ | – | ✓ |
| `GET /options` (read) | – | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `PUT /options` | – | – | – | – | – | – | ✓ |
| `GET /backup`, `/backup/artifacts` | – | – | – | – | ✓ | – | ✓ |
| `POST /backup/run` | – | – | – | – | ✓ | – | ✓ |
| `POST /backup/schedule` | – | – | – | – | – | – | ✓ |
| `POST /backup/restore`, `/backup/restore/approve` | – | – | – | – | ✓ | – | ✓ |
| `GET/POST /task/*` | – | – | – | – | – | – | ✓ |
| `GET /audit` | – | – | – | ✓ | ✓ | – | ✓ |

## 4. Backup & restore safety

- Artifact integrity: SHA-256 recorded at creation and re-verified before
  restore; DB `PRAGMA integrity_check` on staged copy.
- Restore two-step: stage → verify (row counts vs live snapshot) → approve →
  swap. **Live DB never implicitly mutated**; writer present → 409
  (`04-processes §5`, UC-A6).
- Retention rotation deletes only artifacts past their tier and keeps the
  newest valid backup; deletion is audited.
- Backup dir: os.MkdirAll with 0700; refuse world-readable target; check
  writability before scheduling.
- Artifacts contain all ledger data — same classification as the live DB;
  never back up into a web-servable directory.

## 5. Audit trail completeness (admin)

Every admin mutation → `audit_logs` (`04-processes §7`): module, action,
target, actor, timestamp. No user-triggerable endpoint can write/delete
audit rows. Viewer is read-only (`kiểm toán`, `kế toán trưởng`, `admin`).

## 6. Input validation & secrets

- All admin inputs validated server-side (username charset, password policy,
  role codes, settings type+rule, cron syntax, artifact ids). No reflection
  into SQL — parameterized queries + JSON `(id, data)` rows.
- Config: no secrets in `config.yaml` defaults; DB DSN may contain a path only
  in dev. Session secrets from env (`GOGL_SESSION_SECRET`), never committed.
  Audit payloads must not include passwords or session tokens.

## 7. Residual risks (documented)

1. **Web UI CSRF** — must be addressed when the cookie-based session lands
   (per-form tokens); JSON API is header/cookie-based and relies on same-origin
   policy + CORS config (none today).
2. **No 2FA/SSO** — password-only auth in v1; acceptable for internal single-
   company deployment, revisit with Luật GDĐT/ANM compliance for larger rollouts.
3. **Restore is full-replace** — accidental approval of an old artifact loses
   post-backup postings; mitigated by approval step + "date of backup" warning
   (UI) + restore drill.
4. **Dev seam risk** — if `authorization.dev_mode` is accidentally on in PROD,
   impersonation via header returns. Gate with an env assertion: refuse to
   start with `dev_mode` when `GOGL_ENV=production`.
5. **Backup target is local** — a disk failure takes out DB + backups
   together unless the target is a mounted/off-box volume; document the
   off-box mount as the PROD requirement.

## 8. Security test checklist (implement with roadmap phases)

- [ ] Impersonation-reject: `X-User-Id: admin` on `/api/v1/user` → 401 when
  auth enabled (P0 AC-1).
- [ ] Lockout after N failures; unlock path; no user enumeration (P0).
- [ ] Session expiry (idle + absolute), revocation on suspend (P0/P1).
- [ ] Authz matrix test: every admin route × all roles + anonymous (P4.1).
- [ ] Backup→restore round-trip with integrity check; restore abort on
  checksum/count mismatch (P2).
- [ ] Retention rotation unit tests (P2).
- [ ] Settings validation: type/rule rejection; audit old→new (P3).
- [ ] Password hash round-trip; reuse rejection; no plaintext in DB/logs (P0).
- [ ] CSRF token presence on all admin POST/PUT forms (P4).
