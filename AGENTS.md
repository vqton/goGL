# AGENTS.md

Go ERP/general-ledger app (`module goGL`, Go 1.26.4): Gin HTTP API backed by SQLite.
This is a **scaffolded skeleton** — 24 feature modules × 4 layers exist, but almost
every method is a `TODO` stub. Only the DB, migration, and config infrastructure is
implemented. `go build ./...`, `go vet ./...`, `go test ./...` all pass (no tests).

## Layout & architecture

Each feature module (cash, ledger, invoice, payroll, …) spans four packages — one
file each — wired **manually** in `cmd/server/main.go`. No DI container, no codegen.

- `internal/domain/<m>/entity.go` — entities + `Repository` interface
- `internal/application/<m>/service.go` — `Service` interface + `NewService(repo)`
- `internal/infrastructure/persistence/<m>/repository.go` — `NewSqliteRepository(db)`
- `internal/interfaces/http/<m>/handler.go` — `Handler` + `Register(*gin.RouterGroup)`

All handlers register under `/api/v1/<m>/...`. Stubs return `core.ErrNotImplemented`
(services/repos) or HTTP 501 (handlers). To add a module, create all four packages
and register it in `main.go`. `internal/domain/core/types.go` holds shared types
(`Money`, `Period`, `Status`) and `ErrNotImplemented`.

## Authorization (Casbin)

Access control is implemented with **Casbin v3** (`github.com/casbin/casbin/v3`) and
wired as cross-cutting infrastructure, not a feature module:

- `internal/infrastructure/authorization/` — embedded RBAC model (`rbac_model.conf`),
  a custom SQLite adapter (`sqliteAdapter` implementing `persist.Adapter` +
  `persist.BatchAdapter`), `NewEnforcer(db)` / `SeedDefaultPolicies(e)`, and the Gin
  `AuthorizationMiddleware`. It is the only real (non-stub) infrastructure besides
  db/config.
- Policies persist in the `casbin_policies` table — same `(id, data)` JSON-document
  shape as every other table; `id` is a deterministic SHA-256 of the rule, so
  re-adding a rule is an upsert. The table is created by `db.Migrate`.
- Enforcement triple: `sub` = principal (user id), `obj` = matched route pattern
  via `c.FullPath()`, `act` = HTTP method. `*` wildcards and `keyMatch2` are
  supported in the matcher.
- The middleware mounts on the `/api/v1` group only when `cfg.Authorization.Enabled`.
  A `PrincipalResolver` supplies the subject per request — `HeaderPrincipalResolver`
  reads it from the `identity_header` (default `X-User-Id`) as a **dev seam** until
  real authentication exists; anonymous (missing header) fails closed.
- `internal/interfaces/http/authz/handler.go` is the one working module: policy
  list/add/remove and an enforcement check under `/api/v1/authz/...` (itself
  protected, so only roles with access can manage policies).
- Startup seeds the built-in `role:admin` (`* *`) role and links the default
  `admin` user to it when the store is empty.

Front end: `internal/interfaces/http/web/handler.go` (the one handler that registers
on the root engine, not `/api/v1`) serves the landing page and static assets. Pages
are Go `html/templates` in `web/templates/` (layout `base.html`, pages define
`title`/`content` blocks; render the `base` template). Tailwind v4 compiles
`web/css/input.css` (`@import "tailwindcss";`) to the generated
`web/static/css/app.css` — gitignored, never edit by hand.

## Persistence

- SQLite via `modernc.org/sqlite` (pure Go; driver name `"sqlite"`; plain `database/sql`).
- Every table has the same shape: `(id TEXT PRIMARY KEY, data TEXT NOT NULL)` — i.e.
  JSON-document rows.
- Migrations are hand-rolled in `internal/infrastructure/db/migrate.go`: only
  `CREATE TABLE IF NOT EXISTS` over a fixed table list. No ALTER, no migration tool.
- Entities carry `bson:"…"` tags — leftovers from a former MongoDB intent. Don't add
  more. `go.mongodb.org/mongo-driver/v2` and `quic-go` in go.mod are merely indirect
  deps of gin v1.12.0; this code uses neither.

## Commands

- Run: `go run ./cmd/server` — from repo root only, because config is hardcoded to
  `config.Load("config.yaml")`. Creates `gogl.db` in CWD and listens on `:8080`.
- `go mod tidy` after changing imports. Direct deps: casbin/casbin/v3, gin-gonic/gin,
  goccy/go-yaml, modernc.org/sqlite.
- Tests live only in the authorization surface so far: the infrastructure package
  (`internal/infrastructure/authorization/`) and the authz HTTP handler
  (`internal/interfaces/http/authz/`). Run them with
  `go test ./internal/infrastructure/authorization/... ./internal/interfaces/http/authz/...`.
- CSS (needs Node, first run `npm install`): `npm run build:css` compiles Tailwind
  once; `npm run watch:css` rebuilds on change. After adding Tailwind classes to a
  template, rebuild or the generated `app.css` goes stale.

## Git

- Repo initialized on `master`, **zero commits**, everything untracked. `.gitignore`
  covers `node_modules/`, generated `web/static/css/app.css`, and runtime `gogl.db`.
- A custom `.git/hooks/pre-commit` (gofmt + `go vet` + `go test` on staged Go files)
  exists only in this checkout — it isn't tracked, so fresh clones won't have it.

## Skills

The globally installed skills are routed by a 5W1H decision table at
`/root/.opencode/skills/SKILLS-5W1H.md`. When unsure which skill applies, check its
**When** column first (or `ask-matt`); skills load by name via the `skill` tool.

CodeGraph index (`.codegraph/`) is available for code exploration.
