# 08 — Security review (setup module)

Scope: the "khởi tạo hệ thống" (setup) module — its HTTP surface, the
server-rendered wizard, the Casbin authorization posture, the audit trail and
the CSV import pipeline. Verified against the code at commit `05d6b4c` +
`e150c32` plus the T3 import/report work.

Review checklist: [x] authz matrix, [x] audit trail completeness, [x] MST /
sensitive data not logged outside audit, [x] input validation, [x] transaction
atomicity, [x] documented residual risks.

## 1. Attack surface

| Surface | Transport | Authz gate | Notes |
|---|---|---|---|
| JSON API `/api/v1/setup/**` | HTTP | Casbin middleware (only when `authorization.enabled`) | Fail-closed: missing/unknown principal → 403; anonymous → 403 |
| Web wizard `/setup/**` | HTTP | none (no Casbin on root group) + per-handler `requireActor` | Renders server-side; only trusted operator network expected |
| CSV import + report + errors.csv | HTTP | Casbin on `/api/v1` routes | Template v1 header enforced server-side |

## 2. Authorization matrix (JSON API)

Derived from `internal/infrastructure/authorization/setup_policies.go` +
`setup_policies_test.go`. `keyMatch2` is used, so `*` is single-segment;
no broad `POST /api/v1/setup/*` exists — lock/reopen/activate stay
chief-accountant-only even for the general accountant.

| Route | danh_muc | ke_toan_tong_hop | ke_toan_truong | giam_doc | kiem_toan | admin |
|---|---|---|---|---|---|---|
| `GET  /setup/status` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `GET  /setup/profile` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `PUT  /setup/profile` | ✗ | ✗ | ✓ | ✗ | ✗ | ✓ |
| `POST /setup/initialize` | ✗ | ✗ | ✓ | ✗ | ✗ | ✓ |
| `GET  /setup/opening-balances` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `POST /setup/opening-balances` | ✗ | ✓ | ✓ | ✗ | ✗ | ✓ |
| `DELETE /setup/opening-balances/:id` | ✗ | ✓ | ✓ | ✗ | ✗ | ✓ |
| `POST /setup/opening-balances/check` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `POST /setup/opening-balances/import` | ✗ | ✓ | ✓ | ✗ | ✗ | ✓ |
| `GET  /setup/opening-balances/import/:id/report` | ✗ | ✓ | ✓ | ✗ | ✗ | ✓ |
| `GET  /setup/opening-balances/import/:id/errors.csv` | ✗ | ✓ | ✓ | ✗ | ✗ | ✓ |
| `POST /setup/opening-balances/lock` | ✗ | ✗ | ✓ | ✗ | ✗ | ✓ |
| `POST /setup/opening-balances/reopen` | ✗ | ✗ | ✓ | ✗ | ✗ | ✓ |
| `POST /setup/activate` | ✗ | ✗ | ✓ | ✗ | ✗ | ✓ |

Design intent (spec §8/§9): read-mostly for the read roles; the general
accountant performs balance data entry and CSV import; the chief accountant
owns the lifecycle (initialize, lock, reopen with reason, activate, profile
edits) and has write override on balances/import.

The **web wizard** is registered on the root engine and is *not* protected by
Casbin; each mutating handler calls `requireActor`, which fails closed (401)
when the identity header is absent. This is the same pattern as `webcash` /
`webledger` and is an accepted dev-seam posture (see §6).

## 3. Audit trail (R13)

Every mutation is recorded via the `audit` module (`internal/application/audit`)
with `module: "setup"`. Actions:

| Action | Payload `target_id` | Trigger |
|---|---|---|
| `initialize.profile` | `company-profile` | profile saved during Initialize |
| `initialize.regime` | regime code (e.g. `TT99-2025`) | regime validated + periods configured |
| `initialize.accounts` | `company-profile` | COA seed |
| `initialize.periods` | `company-profile` | fiscal periods opened |
| `profile.update` | `company-profile` | PUT /profile |
| `balance.upsert` | deterministic balance id (sha256 of account+object) | single-row save / import commit |
| `balance.delete` | balance id | delete |
| `balances.import` | import **job id** (content hash) | CSV import (dry-run or commit) |
| `balances.lock` | `company-profile` | lock |
| `balances.reopen` | `company-profile` | reopen (reason validated but not stored; see §6) |
| `activate` | `company-profile` | activation |

Each entry carries `user_code`, `module`, `action`, `target_id`, `timestamp`
(RFC3339 UTC). The dashboard renders the newest 15 setup entries.

**MST guarantee:** the MST (tax code) is stored only in the company profile
row (`company_profiles`). It never appears in audit `target_id` (profile
actions use the constant `company-profile`, not the MST) and no audit message
concatenates profile data. Domain errors are static sentinels or
`ValidationError{Field, MessageVn, MessageEn}` with no data interpolation, so
error responses/redirects (`?err=`) do not echo the MST or any other profile
field.

## 4. CSV import hardening (T3)

- **Template version rejection:** the header row is validated byte-for-byte
  (case/space-normalized) against `account,object_type,object_code,debit,credit`
  (`importColumns`); a missing or shuffled header is rejected with
  `ErrInvalidImport` (422) *before* any row is processed — a future v2 template
  can never be silently mis-parsed.
- **Per-row validation:** account exists + postable (R7), đối tượng required
  (R10) / optional (R10b), debit+credit parsed as integer VND minor units, no
  row may set both or neither (R8), object must exist + active (R10).
- **Atomic commit:** valid rows are written in **one transaction**
  (`SaveBalances`) — a failure rolls the whole batch back; import rows are
  never half-applied (spec §5.4 "one tx per batch").
- **Deterministic ids:** each row's id is `sha256("OB" + account + object)` so
  re-imports upsert instead of duplicating; the same hash feeds idempotency
  counting (created vs updated).
- **Job retention:** every upload (dry-run and commit) persists a
  `ImportJob` under a content-derived id; `GET .../report` re-serves the
  per-row errors and `GET .../errors.csv` exports `row,column,message` for
  offline fixing. Rows referenced only by line number; the operator maps them
  to their source file.
- **Lock guard:** import (and balance save) is refused once status is
  `balances_locked`/`active` (`ErrBalanceLocked`).

## 5. Input validation summary

| Field | Rule |
|---|---|
| name / address / legal_representative | required, non-blank |
| tax_code (MST) | normalized (whitespace stripped); 10 or 13 digits |
| accounting_currency | VND only in v1 |
| fiscal_year_start | 1st of a month, exactly 12 months |
| accounting_regime | whitelist (`TT99-2025`, …); unknown → 422 |
| balance rows | amount = integer VND minor; account/object existence + postability |

## 6. Residual risks / accepted posture

1. **Identity header dev seam (HIGH until replaced).** Both the JSON API
   principal resolver and the web wizard trust `X-User-Id`
   (`config.yaml: identity_header`) verbatim. Any caller who can set the
   header can impersonate `ketoan`/`kttruong`. This is the documented pre-auth
   seam; the deployment must sit behind a gateway that strips it until real
   authentication lands. Enabling `authorization.enabled` gives *some* cover
   (role membership still matters) but is not identity proof.
2. **Web wizard outside Casbin.** `/setup/**` is readable by anyone who can
   reach the server; mutating actions require only a non-empty header, not a
   role. Accepted while the wizard is an internal operator tool; keep it off
   the public network.
3. **Reopen reason (R12) is validated but not persisted.** The reason is
   required and gates the reopen, but the audit entry only records the
   reopen action, not the reason text. Tracked follow-up: store the reason in
   the audit `target_id` or a dedicated field.
4. **`balances.import` audits dry-runs too.** A dry-run that found errors
   still writes an audit entry (with the job id) — by design for traceability,
   but operators must not read it as a data change.
5. **MST at rest.** The MST is stored in the `company_profiles` JSON row in
   plaintext. No field-level encryption exists anywhere in the app (JSON-doc
   rows); at-rest protection is the DB/file layer's job.
6. **No rate limiting / CSRF.** The JSON API is a dev surface; the web wizard
   form posts have no CSRF token. Follow-up for the operator deployment.
7. **Job errors expose account codes.** `errors.csv` and the report contain
   account/object codes and reason text (not values); treat the report as
   internal data.

## 7. Verification

- `internal/infrastructure/authorization/setup_policies_test.go` — full matrix
  incl. deny cases (read roles cannot initialize/lock; general accountant
  cannot lock/reopen/activate/edit profile; anonymous/unknown denied).
- `internal/application/audit` + `internal/interfaces/http/websetup` tests —
  trail recorded for every mutation and rendered on the dashboard.
- `internal/application/setup` tests — template rejection, per-row errors,
  batch atomicity, lock guard, job persistence + report/errors.csv.
- `go test ./...` green; `go vet ./...` clean.
