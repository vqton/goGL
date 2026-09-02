# AGENTS.md

Go ERP/general-ledger app (`module goGL`, Go 1.26.4): Gin HTTP API backed by SQLite.
24 feature modules × 4 layers, wired manually in `cmd/server/main.go`.
`go build ./...`, `go vet ./...`, `go test ./...` all pass.

## Quick reference

- **Run:** `go run ./cmd/server` — from repo root only, creates `gogl.db`, listens on `:8080`
- **Test all:** `go test ./...` (42 passing packages)
- **Test one package:** `go test ./internal/application/cash/...`
- **Test one file:** `go test ./internal/application/cash -run TestCreateVoucher`
- **Build CSS:** `npm install && npm run build:css` (Tailwind v4, needs Node)
- **Watch CSS:** `npm run watch:css`
- **E2E tests:** `npm run test:e2e` (Playwright, starts server automatically)

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

## Testing patterns

- Tests use **in-memory SQLite** databases (`file:<name>?mode=memory&cache=shared`)
- Each test gets a fresh DB via `db.Migrate()` — no cleanup needed
- Service tests create real DB + fake auditors/repos, no mocking framework
- Domain tests are pure Go, no DB dependency
- HTTP handler tests use `httptest.NewRecorder()` with real service + DB
- Run a single test: `go test ./internal/application/cash -run TestCreateVoucher`

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
- Tests exist across many packages (69 test files, 42 passing packages). Run all with
  `go test ./...` or target a single package, e.g.
  `go test ./internal/application/cash/...`.
- CSS (needs Node, first run `npm install`): `npm run build:css` compiles Tailwind
  once; `npm run watch:css` rebuilds on change. After adding Tailwind classes to a
  template, rebuild or the generated `app.css` goes stale.
- E2E tests use Playwright (`npm run test:e2e`). The `playwright.config.ts` starts
  the server via `go run ./cmd/server` on `:8080` automatically.

## Git

- Branch: `main`. `.gitignore` covers `node_modules/`, generated `web/static/css/app.css`,
  and runtime `gogl.db`.
- A custom `.git/hooks/pre-commit` (gofmt + `go vet` + `go test` on staged Go files)
  exists only in this checkout — it isn't tracked, so fresh clones won't have it.

## Module Status

Fully implemented modules (with tests):
- **inventory** — 103 tests: Item/Warehouse CRUD, StockCard, StockMovement (FIFO/Weighted Average), StockTransfer, StockAdjustment, PhysicalCount, NRV Write-Down/Reversal
- **purchase** — 99 tests: Supplier, PurchaseOrder, GoodsReceipt, PurchaseInvoice, Payment
- **sales** — 60 tests: SalesInvoice, SalesOrder, SalesReturn, Customer balance
- **fixedasset** — Entity with depreciation, batch processor, approval workflow
- **tools** — Entity with GL fields, ToolTransaction, 6 transaction types, atomic stock operations

Stub modules (return `ErrNotImplemented`):
- bank, budget, cash, contract, costing, invoice, ledger, masterdata, payroll, reporting, setup, tax

## Inventory Module (Vietnamese Accounting Compliant)

API endpoints under `/api/v1/inventory/...`:
- `POST/GET /items`, `GET/PUT /items/:id`, `DELETE /items/:id`, `GET /items/code/:code`
- `POST/GET /warehouses`, `GET/PUT /warehouses/:id`, `DELETE /warehouses/:id`, `GET /warehouses/code/:code`
- `GET /stock`, `GET /stock/:itemCode/:warehouseCode`
- `POST /movements`, `GET /movements`, `GET /movements/:id`, `POST /movements/:id/confirm`
- `POST /transfer`, `POST /adjust`
- `POST /counts`, `GET /counts`, `GET/PUT /counts/:id`, `POST /counts/:id/complete`, `POST /counts/:id/reconcile`
- `POST /writedown`, `POST /writedown/reverse`

Key patterns:
- FIFO valuation uses `StockValuationLayer` entities (oldest-first consumption)
- Weighted Average maintains running average on stock card
- Physical Count auto-populates system qty from stock cards on complete
- NRV Write-Down: Dr. 515xxx / Cr. 152xxx; Reversal: Dr. 152xxx / Cr. 515xxx
- Movement code prefixes: PN (receipt), PX (dispatch), PCCD (transfer-in), PCCT (transfer-out), DK (adjustment)
- All monetary values stored as `int64` (VND, no decimals)

## Skills

The globally installed skills are routed by a 5W1H decision table at
`/root/.opencode/skills/SKILLS-5W1H.md`. When unsure which skill applies, check its
**When** column first (or `ask-matt`); skills load by name via the `skill` tool.

CodeGraph index (`.codegraph/`) is available for code exploration.

## Architecture diagram

Interactive HTML diagram generated with Archify:
- `docs/architecture.html` — self-contained, open in browser
- `docs/architecture.json` — source IR (edit to update)
- Regenerate: `node /root/.config/opencode/skills/archify/bin/archify.mjs deliver architecture docs/architecture.json docs/architecture.html --quality showcase --json`
