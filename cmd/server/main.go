package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/casbin/casbin/v3"

	"goGL/internal/application/audit"
	"goGL/internal/application/auth"
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
	apptask "goGL/internal/application/task"
	"goGL/internal/application/tax"
	"goGL/internal/application/tools"
	appuser "goGL/internal/application/user"
	"goGL/internal/config"
	domainbackup "goGL/internal/domain/backup"
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
	perssession "goGL/internal/infrastructure/persistence/session"
	perssetup "goGL/internal/infrastructure/persistence/setup"
	perssystem "goGL/internal/infrastructure/persistence/system"
	perstask "goGL/internal/infrastructure/persistence/task"
	perstax "goGL/internal/infrastructure/persistence/tax"
	perstools "goGL/internal/infrastructure/persistence/tools"
	persuser "goGL/internal/infrastructure/persistence/user"
	httpaudit "goGL/internal/interfaces/http/audit"
	httpauth "goGL/internal/interfaces/http/auth"
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
	httpwebledger "goGL/internal/interfaces/http/webledger"
	httpwebsetup "goGL/internal/interfaces/http/websetup"
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

	ledgerRepo := persledger.NewSqliteRepository(sqlDB)
	if n, err := ledger.SeedDefaultAccounts(ctx, ledgerRepo); err != nil {
		log.Fatalf("seed ledger chart of accounts: %v", err)
	} else if n > 0 {
		log.Printf("seeded %d default ledger accounts", n)
	}

	// Cross-module seams for the setup ORCHESTRATOR (docs/setup §10): the
	// concrete masterdata / ledger / audit services satisfy the narrow setup
	// interfaces structurally.
	auditSvc := audit.NewService(persaudit.NewSqliteRepository(sqlDB))
	masterdataSvc := masterdata.NewService(persmasterdata.NewSqliteRepository(sqlDB))
	ledgerSvc := ledger.NewService(ledgerRepo)

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

	httpwebledger.NewHandler(ledgerSvc, cfg.Authorization.IdentityHeader).Register(r)
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
	fixedAssetRepo := persfixedasset.NewSqliteRepository(sqlDB)
	fixedAssetDeprRepo := persfixedasset.NewSqliteDepreciationRepository(sqlDB)
	httpfixedasset.NewHandler(fixedasset.NewServiceWithDepreciation(fixedAssetRepo, fixedAssetDeprRepo)).Register(v1)
	httptax.NewHandler(tax.NewService(perstax.NewSqliteRepository(sqlDB))).Register(v1)
	httppayroll.NewHandler(payroll.NewService(perspayroll.NewSqliteRepository(sqlDB))).Register(v1)
	httpcosting.NewHandler(costing.NewService(perscosting.NewSqliteRepository(sqlDB))).Register(v1)
	httpledger.NewHandler(ledgerSvc, cfg.Authorization.IdentityHeader).Register(v1)
	httpcontract.NewHandler(contract.NewService(perscontract.NewSqliteRepository(sqlDB))).Register(v1)
	httpbudget.NewHandler(budget.NewService(persbudget.NewSqliteRepository(sqlDB))).Register(v1)
	httpreporting.NewHandler(reporting.NewService(persreporting.NewSqliteRepository(sqlDB))).Register(v1)
	setupSvc := setup.NewService(
		perssetup.NewSqliteRepository(sqlDB),
		setup.Dependencies{
			Regime:   masterdataSvc,
			Seeder:   masterdataSvc,
			Objects:  masterdataSvc,
			Periods:  ledgerSvc,
			Accounts: ledgerSvc,
			Postings: ledgerSvc,
			Audit:    auditSvc,
		},
	)
	httpsetup.NewHandler(setupSvc, cfg.Authorization.IdentityHeader).Register(v1)
	httpwebsetup.NewHandler(setupSvc, cfg.Authorization.IdentityHeader).Register(r)
	httpmasterdata.NewHandler(masterdataSvc).Register(v1)
	// Session repository (shared by auth and system modules)
	sessionRepo := perssession.NewSqliteRepository(sqlDB)

	// User management service with casbin integration
	userSvc := appuser.NewService(
		persuser.NewSqliteRepository(sqlDB),
		authzEnforcer,
		auditSvc,
		[]string{"role:admin"},
		8, // minPasswordLen
	)
	httpuser.NewHandler(userSvc, cfg.Authorization.IdentityHeader).Register(v1)

	// Auth service (login/logout/session management)
	authSvc := auth.NewService(
		persuser.NewSqliteRepository(sqlDB),
		sessionRepo,
		auditSvc,
		auth.Policy{
			CookieName:           "session",
			MaxHours:             24,
			IdleMinutes:          30,
			MaxFailures:          5,
			LockoutMinutes:       15,
			MinPasswordLen:       8,
			PasswordExpiryDays:   90,  // Circular 99/2025 compliance
			PasswordHistoryCount: 5,   // Prevent password reuse
		},
	)
	httpauth.NewHandler(authSvc, "session", 24).Register(v1)

	// System info service
	httpsystem.NewHandler(system.NewService(
		"dev",                    // version
		"unknown",                // commit
		"1.21",                   // goVersion
		time.Now(),               // startedAt
		perssystem.NewSqliteRepository(sqlDB),
		sessionRepo,
		&backupLastBackupProvider{repo: persbackup.NewSqliteRepository(sqlDB)},
	)).Register(v1)

	// System options service
	httpoptions.NewHandler(appoptions.NewService(
		persoptions.NewSqliteRepository(sqlDB),
		auditSvc,
	), cfg.Authorization.IdentityHeader).Register(v1)

	httpdocument.NewHandler(document.NewService(persdocument.NewSqliteRepository(sqlDB))).Register(v1)

	// Backup service
	backupDir := filepath.Join(".", "backups")
	_ = os.MkdirAll(backupDir, 0o755)
	backupSvc := backup.NewService(
		sqlDB,
		persbackup.NewSqliteRepository(sqlDB),
		auditSvc,
		backupDir,
		cfg.Database.DSN,
		10, // maxBackups
	)
	httpbackup.NewHandler(backupSvc, cfg.Authorization.IdentityHeader).Register(v1)

	// Task runner service with registry
	taskRegistry := apptask.Registry{
		"backup": func(ctx context.Context) error {
			_, err := backupSvc.CreateBackup(ctx, "system", "default", "scheduled")
			return err
		},
	}
	httptask.NewHandler(apptask.NewService(
		taskRegistry,
		perstask.NewSqliteRepository(sqlDB),
		auditSvc,
	), cfg.Authorization.IdentityHeader).Register(v1)

	httpaudit.NewHandler(audit.NewService(persaudit.NewSqliteRepository(sqlDB))).Register(v1)
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

// backupLastBackupProvider adapts the backup repository to satisfy the
// system.LastBackupProvider interface.
type backupLastBackupProvider struct {
	repo domainbackup.Repository
}

func (p *backupLastBackupProvider) LastBackupAt(ctx context.Context) (string, error) {
	arts, err := p.repo.ListArtifacts(ctx)
	if err != nil {
		return "", err
	}
	if len(arts) == 0 {
		return "", nil
	}
	newest := arts[0]
	for _, a := range arts[1:] {
		if a.CreatedAt.After(newest.CreatedAt) {
			newest = a
		}
	}
	return newest.CreatedAt.UTC().Format(time.RFC3339), nil
}
