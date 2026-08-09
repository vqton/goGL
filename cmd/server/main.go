package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/casbin/casbin/v3"

	"goGL/internal/application/audit"
	"goGL/internal/application/backup"
	"goGL/internal/application/bank"
	"goGL/internal/application/budget"
	"goGL/internal/application/cash"
	"goGL/internal/application/contract"
	"goGL/internal/application/costing"
	"goGL/internal/application/document"
	"goGL/internal/application/fixedasset"
	"goGL/internal/application/inventory"
	"goGL/internal/application/invoice"
	"goGL/internal/application/ledger"
	"goGL/internal/application/masterdata"
	appoptions "goGL/internal/application/options"
	"goGL/internal/application/payroll"
	"goGL/internal/application/purchase"
	"goGL/internal/application/reporting"
	"goGL/internal/application/sales"
	"goGL/internal/application/setup"
	"goGL/internal/application/system"
	"goGL/internal/application/task"
	"goGL/internal/application/tax"
	"goGL/internal/application/tools"
	"goGL/internal/application/user"
	"goGL/internal/config"
	"goGL/internal/infrastructure/authorization"
	"goGL/internal/infrastructure/db"
	persaudit "goGL/internal/infrastructure/persistence/audit"
	persbackup "goGL/internal/infrastructure/persistence/backup"
	persbank "goGL/internal/infrastructure/persistence/bank"
	persbudget "goGL/internal/infrastructure/persistence/budget"
	perscash "goGL/internal/infrastructure/persistence/cash"
	perscontract "goGL/internal/infrastructure/persistence/contract"
	perscosting "goGL/internal/infrastructure/persistence/costing"
	persdocument "goGL/internal/infrastructure/persistence/document"
	persfixedasset "goGL/internal/infrastructure/persistence/fixedasset"
	persinventory "goGL/internal/infrastructure/persistence/inventory"
	persinvoice "goGL/internal/infrastructure/persistence/invoice"
	persledger "goGL/internal/infrastructure/persistence/ledger"
	persmasterdata "goGL/internal/infrastructure/persistence/masterdata"
	persoptions "goGL/internal/infrastructure/persistence/options"
	perspayroll "goGL/internal/infrastructure/persistence/payroll"
	perspurchase "goGL/internal/infrastructure/persistence/purchase"
	persreporting "goGL/internal/infrastructure/persistence/reporting"
	perssales "goGL/internal/infrastructure/persistence/sales"
	perssetup "goGL/internal/infrastructure/persistence/setup"
	perssystem "goGL/internal/infrastructure/persistence/system"
	perstask "goGL/internal/infrastructure/persistence/task"
	perstax "goGL/internal/infrastructure/persistence/tax"
	perstools "goGL/internal/infrastructure/persistence/tools"
	persuser "goGL/internal/infrastructure/persistence/user"
	httpaudit "goGL/internal/interfaces/http/audit"
	httpauthz "goGL/internal/interfaces/http/authz"
	httpbackup "goGL/internal/interfaces/http/backup"
	httpbank "goGL/internal/interfaces/http/bank"
	httpbudget "goGL/internal/interfaces/http/budget"
	httpcash "goGL/internal/interfaces/http/cash"
	httpcontract "goGL/internal/interfaces/http/contract"
	httpcosting "goGL/internal/interfaces/http/costing"
	httpdocument "goGL/internal/interfaces/http/document"
	httpfixedasset "goGL/internal/interfaces/http/fixedasset"
	httpinventory "goGL/internal/interfaces/http/inventory"
	httpinvoice "goGL/internal/interfaces/http/invoice"
	httpledger "goGL/internal/interfaces/http/ledger"
	httpmasterdata "goGL/internal/interfaces/http/masterdata"
	httpoptions "goGL/internal/interfaces/http/options"
	httppayroll "goGL/internal/interfaces/http/payroll"
	httppurchase "goGL/internal/interfaces/http/purchase"
	httpreporting "goGL/internal/interfaces/http/reporting"
	httpsales "goGL/internal/interfaces/http/sales"
	httpsetup "goGL/internal/interfaces/http/setup"
	httpsystem "goGL/internal/interfaces/http/system"
	httptask "goGL/internal/interfaces/http/task"
	httptax "goGL/internal/interfaces/http/tax"
	httptools "goGL/internal/interfaces/http/tools"
	httpuser "goGL/internal/interfaces/http/user"
	httpweb "goGL/internal/interfaces/http/web"
	httpcashweb "goGL/internal/interfaces/http/webcash"
)

func main() {
	cfg := config.Load("config.yaml")

	ctx := context.Background()
	sqlDB, err := db.Open(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(ctx, sqlDB); err != nil {
		log.Fatalf("migrate sqlite: %v", err)
	}

	authzEnforcer, err := authorization.NewEnforcer(sqlDB)
	if err != nil {
		log.Fatalf("init authorization: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(authzEnforcer); err != nil {
		log.Fatalf("seed authorization policies: %v", err)
	}

	r := gin.Default()
	httpweb.NewHandler().Register(r)
	v1 := r.Group("/api/v1")

	cashSvc := cash.NewService(
		perscash.NewSqliteRepository(sqlDB),
		audit.NewService(persaudit.NewSqliteRepository(sqlDB)),
		cash.WithNotifier(logNotifier{}),
		cash.WithVoidApprover(&casbinVoidApprover{enforcer: authzEnforcer}),
	)
	httpcashweb.NewHandler(cashSvc, cfg.Authorization.IdentityHeader).Register(r)

	if cfg.Authorization.Enabled {
		v1.Use(authorization.AuthorizationMiddleware(
			authzEnforcer,
			authorization.HeaderPrincipalResolver(cfg.Authorization.IdentityHeader),
		))
	}

	httpcash.NewHandler(cashSvc, cfg.Authorization.IdentityHeader).Register(v1)
	httpbank.NewHandler(bank.NewService(persbank.NewSqliteRepository(sqlDB))).Register(v1)
	httppurchase.NewHandler(purchase.NewService(perspurchase.NewSqliteRepository(sqlDB))).Register(v1)
	httpsales.NewHandler(sales.NewService(perssales.NewSqliteRepository(sqlDB))).Register(v1)
	httpinvoice.NewHandler(invoice.NewService(persinvoice.NewSqliteRepository(sqlDB))).Register(v1)
	httpinventory.NewHandler(inventory.NewService(persinventory.NewSqliteRepository(sqlDB))).Register(v1)
	httptools.NewHandler(tools.NewService(perstools.NewSqliteRepository(sqlDB))).Register(v1)
	httpfixedasset.NewHandler(fixedasset.NewService(persfixedasset.NewSqliteRepository(sqlDB))).Register(v1)
	httptax.NewHandler(tax.NewService(perstax.NewSqliteRepository(sqlDB))).Register(v1)
	httppayroll.NewHandler(payroll.NewService(perspayroll.NewSqliteRepository(sqlDB))).Register(v1)
	httpcosting.NewHandler(costing.NewService(perscosting.NewSqliteRepository(sqlDB))).Register(v1)
	httpledger.NewHandler(ledger.NewService(persledger.NewSqliteRepository(sqlDB))).Register(v1)
	httpcontract.NewHandler(contract.NewService(perscontract.NewSqliteRepository(sqlDB))).Register(v1)
	httpbudget.NewHandler(budget.NewService(persbudget.NewSqliteRepository(sqlDB))).Register(v1)
	httpreporting.NewHandler(reporting.NewService(persreporting.NewSqliteRepository(sqlDB))).Register(v1)
	httpsetup.NewHandler(setup.NewService(perssetup.NewSqliteRepository(sqlDB))).Register(v1)
	httpmasterdata.NewHandler(masterdata.NewService(persmasterdata.NewSqliteRepository(sqlDB))).Register(v1)
	httpuser.NewHandler(user.NewService(persuser.NewSqliteRepository(sqlDB))).Register(v1)
	httpsystem.NewHandler(system.NewService(perssystem.NewSqliteRepository(sqlDB))).Register(v1)
	httpoptions.NewHandler(appoptions.NewService(persoptions.NewSqliteRepository(sqlDB))).Register(v1)
	httpdocument.NewHandler(document.NewService(persdocument.NewSqliteRepository(sqlDB))).Register(v1)
	httptask.NewHandler(task.NewService(perstask.NewSqliteRepository(sqlDB))).Register(v1)
	httpaudit.NewHandler(audit.NewService(persaudit.NewSqliteRepository(sqlDB))).Register(v1)
	httpbackup.NewHandler(backup.NewService(persbackup.NewSqliteRepository(sqlDB))).Register(v1)
	httpauthz.NewHandler(authzEnforcer).Register(v1)

	if err := r.Run(cfg.Server.HTTPAddr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

// logNotifier implements cash.Notifier by logging to the server log — the
// dev-seam until real notification channels (email/SMS) exist. CloseDay uses
// it to alert the chief accountant of a count mismatch (R7).
type logNotifier struct{}

func (logNotifier) Notify(_ context.Context, recipientRole, subject, body string) error {
	log.Printf("[notify] role=%s subject=%q body=%q", recipientRole, subject, body)
	return nil
}

// casbinVoidApprover implements cash.VoidApprover against the Casbin enforcer:
// voiding an already-posted voucher (Điều 30) requires the chief accountant's
// approval, so the actor must hold role:chief_accountant, role:director, or
// role:admin (the last via role inheritance through the admin role).
type casbinVoidApprover struct {
	enforcer *casbin.Enforcer
}

func (a *casbinVoidApprover) CanApproveVoid(_ context.Context, actor string) (bool, error) {
	roles, err := a.enforcer.GetRolesForUser(actor)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		switch role {
		case "role:chief_accountant", "role:director", "role:admin":
			return true, nil
		}
	}
	return false, nil
}
