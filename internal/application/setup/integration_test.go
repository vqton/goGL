package setup_test

// End-to-end integration test (T4.1 — narrow scope): the setup orchestrator
// wired against the REAL sqlite stack — setup/masterdata/ledger/audit repos and
// application services — proving initialize → balances → lock → activate works
// through the production seams (no fakes). Ledger books carrying opening
// balances into live postings is a documented follow-up, out of scope here.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	appaudit "goGL/internal/application/audit"
	appledger "goGL/internal/application/ledger"
	appmasterdata "goGL/internal/application/masterdata"
	appsetup "goGL/internal/application/setup"
	"goGL/internal/domain/core"
	"goGL/internal/domain/ledger"
	"goGL/internal/domain/setup"
	"goGL/internal/infrastructure/db"
	persaudit "goGL/internal/infrastructure/persistence/audit"
	persledger "goGL/internal/infrastructure/persistence/ledger"
	persmasterdata "goGL/internal/infrastructure/persistence/masterdata"
	perssetup "goGL/internal/infrastructure/persistence/setup"
)

// stack is the production wiring from cmd/server/main.go, as test fixtures.
type stack struct {
	svc    appsetup.Service
	master appmasterdata.Service
	ledger appledger.Service
	audit  appaudit.Service
	sqlDB  *sql.DB
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:setup_e2e_%p?mode=memory&cache=shared", t)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

// realStack wires the setup orchestrator onto the real repositories + services,
// exactly as cmd/server/main.go does (docs/setup §10).
func realStack(t *testing.T) *stack {
	t.Helper()
	d := openIntegrationDB(t)

	auditSvc := appaudit.NewService(persaudit.NewSqliteRepository(d))
	masterdataSvc := appmasterdata.NewService(persmasterdata.NewSqliteRepository(d))
	ledgerSvc := appledger.NewService(persledger.NewSqliteRepository(d))

	// main.go seeds the ledger chart at startup; setup's balance validation
	// runs against it, so the test stack must mirror that (see T4.1 follow-up:
	// the masterdata COA and the ledger chart still disagree on 131/331 — the
	// ledger chart makes 131 a non-postable summary while setup spec R10 names
	// it the mandatory-object TK).
	ledgerRepo := persledger.NewSqliteRepository(d)
	if _, err := appledger.SeedDefaultAccounts(context.Background(), ledgerRepo); err != nil {
		t.Fatalf("seed ledger chart: %v", err)
	}

	svc := appsetup.NewService(
		perssetup.NewSqliteRepository(d),
		appsetup.Dependencies{
			Regime:   masterdataSvc,
			Seeder:   masterdataSvc,
			Objects:  masterdataSvc,
			Periods:  ledgerSvc,
			Accounts: ledgerSvc,
			Postings: ledgerSvc,
			Audit:    auditSvc,
		},
	)
	return &stack{svc: svc, master: masterdataSvc, ledger: ledgerSvc, audit: auditSvc, sqlDB: d}
}

func integrationProfile() *setup.CompanyProfile {
	return &setup.CompanyProfile{
		ID:                  setup.ProfileID,
		Name:                "Cty TNHH SX Thép ABC",
		TaxCode:             "0101234567",
		Address:             "Số 1, đường ABC, Hà Nội",
		LegalRepresentative: "Nguyễn Văn A",
		AccountingCurrency:  "VND",
		FiscalYearStart:     "2026-01-01",
		AccountingRegime:    "TT99-2025",
	}
}

func initializeSystem(t *testing.T, svc appsetup.Service) {
	t.Helper()
	view, err := svc.Initialize(context.Background(), &appsetup.InitializeRequest{
		Profile:         integrationProfile(),
		Regime:          "TT99-2025",
		FiscalYearStart: "2026-01-01",
		SeedAccounts:    true,
		OpenPeriods:     true,
	}, "ketoan")
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if view.Status != setup.StatusBalancesDraft {
		t.Fatalf("status = %s, want %s", view.Status, setup.StatusBalancesDraft)
	}
}

// TestSetup_FullLifecycle drives the setup wizard through the real stack and
// asserts each state transition plus the side effects on masterdata/ledger
// (regime, seeded accounts, opened periods).
func TestSetup_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	st := realStack(t)

	// --- EMPTY: no profile yet, status empty -------------------------------
	status, err := st.svc.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != setup.StatusEmpty {
		t.Fatalf("initial status = %s, want %s", status.Status, setup.StatusEmpty)
	}

	// --- initialize → BALANCES_DRAFT (steps 1–5) ---------------------------
	initializeSystem(t, st.svc)

	view, err := st.svc.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if view.Profile == nil {
		t.Fatal("profile missing after initialize")
	}
	if view.Profile.Name != "Cty TNHH SX Thép ABC" {
		t.Fatalf("profile name = %q", view.Profile.Name)
	}
	if view.Regime != "TT99-2025" {
		t.Fatalf("regime = %q, want TT99-2025", view.Regime)
	}
	done := map[string]bool{}
	for _, s := range view.Steps {
		done[s.Key] = s.Done
	}
	for _, key := range []string{"profile", "regime", "accounts", "periods"} {
		if !done[key] {
			t.Errorf("step %q should be done at %s", key, view.Status)
		}
	}
	if done["lock"] || done["active"] {
		t.Errorf("lock/active steps should not be done at %s", view.Status)
	}

	// --- side effects on the real providers ----------------------------------
	// masterdata: regime switched + COA seeded (TT 99/2025 Phụ lục 2 subset).
	// Read via SQL: masterdata.List caps page size at 100 while the seed chart
	// has 172 accounts.
	mrows, err := st.sqlDB.QueryContext(ctx, `SELECT json_extract(data, '$.code') FROM md_records WHERE json_extract(data, '$.kind') = 'account'`)
	if err != nil {
		t.Fatalf("query seeded accounts: %v", err)
	}
	defer mrows.Close()
	found := map[string]bool{}
	for mrows.Next() {
		var code string
		if err := mrows.Scan(&code); err != nil {
			t.Fatalf("scan account: %v", err)
		}
		found[code] = true
	}
	if err := mrows.Err(); err != nil {
		t.Fatalf("account rows: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("no accounts seeded through the real masterdata service")
	}
	for _, code := range []string{"1111", "131", "5111", "1121"} {
		if !found[code] {
			t.Errorf("seeded COA missing %s", code)
		}
	}
	// ledger: 12 monthly periods opened for fiscal year 2026.
	periods, err := st.ledger.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("list periods: %v", err)
	}
	if len(periods) != 12 {
		t.Fatalf("opened %d periods, want 12", len(periods))
	}
	openSet := map[string]bool{}
	for _, p := range periods {
		if p.Status != ledger.PeriodOpen {
			t.Errorf("period %s status = %s, want open", p.ID, p.Status)
		}
		openSet[p.ID] = true
	}
	for m := 1; m <= 12; m++ {
		if !openSet[fmt.Sprintf("2026-%02d", m)] {
			t.Errorf("period 2026-%02d not opened", m)
		}
	}

	// --- opening balances (step 4) -------------------------------------------
	// NOTE: only TKs postable in the ledger chart are usable here (1111/1121/
	// 5111). Spec R10's mandatory-object TK 131 is a non-postable summary in
	// the ledger chart, so the R10 object-mandatory path is covered by the
	// HTTP tests (fake chart marks 131 postable) — see T4.1 follow-up.
	rows := []*setup.OpeningBalance{
		{AccountCode: "1111", Debit: core.Money{AmountMinor: 500_000_000}, Credit: core.Money{}},
		{AccountCode: "1121", Debit: core.Money{AmountMinor: 45_500_000}, Credit: core.Money{}},
		{AccountCode: "5111", Debit: core.Money{}, Credit: core.Money{AmountMinor: 545_500_000}},
	}
	for _, b := range rows {
		saved, err := st.svc.SaveBalance(ctx, b, "ketoan")
		if err != nil {
			t.Fatalf("save balance %s: %v", b.AccountCode, err)
		}
		if saved.ID == "" {
			t.Fatalf("balance %s saved without ID", b.AccountCode)
		}
		if saved.Status != setup.BalanceDraft {
			t.Fatalf("balance %s status = %s, want draft", b.AccountCode, saved.Status)
		}
	}
	// R10 (object-mandatory TK rejected without object) is covered by HTTP
	// tests; here the real stack rejects 131 as not postable, which is the
	// ledger-chart divergence documented as a T4.1 follow-up.

	// --- check: sheet must balance (R9) --------------------------------------
	check, err := st.svc.CheckBalances(ctx)
	if err != nil {
		t.Fatalf("check balances: %v", err)
	}
	if !check.Balanced {
		t.Fatalf("expected balanced sheet, diff = %d", check.Diff)
	}
	if check.Debit != 545_500_000 || check.Credit != 545_500_000 {
		t.Fatalf("totals = %d/%d, want 545500000/545500000", check.Debit, check.Credit)
	}

	// --- lock → reopen (draft-only edits, reason mandatory) -------------------
	if err := st.svc.Lock(ctx, "ketoan"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	status, _ = st.svc.Status(ctx)
	if status.Status != setup.StatusBalancesLocked {
		t.Fatalf("status after lock = %s, want %s", status.Status, setup.StatusBalancesLocked)
	}
	if _, err := st.svc.SaveBalance(ctx, &setup.OpeningBalance{
		AccountCode: "1121", Debit: core.Money{AmountMinor: 1_000}, Credit: core.Money{},
	}, "ketoan"); !errors.Is(err, setup.ErrBalanceLocked) {
		t.Fatalf("save after lock = %v, want ErrBalanceLocked", err)
	}
	if err := st.svc.Reopen(ctx, "ketoan", ""); !errors.Is(err, setup.ErrReopenBlocked) {
		t.Fatalf("reopen without reason = %v, want ErrReopenBlocked", err)
	}
	if err := st.svc.Reopen(ctx, "ketoan", "Sửa sai số liệu"); err != nil {
		t.Fatalf("reopen with reason: %v", err)
	}
	status, _ = st.svc.Status(ctx)
	if status.Status != setup.StatusBalancesDraft {
		t.Fatalf("status after reopen = %s, want %s", status.Status, setup.StatusBalancesDraft)
	}

	// --- activate (step 5) → ACTIVE ------------------------------------------
	if err := st.svc.Lock(ctx, "ketoan"); err != nil {
		t.Fatalf("relock: %v", err)
	}
	if err := st.svc.Activate(ctx, "ketoan"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	status, err = st.svc.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != setup.StatusActive {
		t.Fatalf("final status = %s, want %s", status.Status, setup.StatusActive)
	}
	// Re-initialize / lock after activate must be rejected.
	if _, err := st.svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: integrationProfile(), Regime: "TT99-2025",
	}, "ketoan"); !errors.Is(err, setup.ErrAlreadyInitialized) {
		t.Fatalf("initialize after activate = %v, want ErrAlreadyInitialized", err)
	}
	if err := st.svc.Lock(ctx, "ketoan"); !errors.Is(err, setup.ErrWrongState) {
		t.Fatalf("lock after activate = %v, want ErrWrongState", err)
	}

	// --- audit trail captured every mutation (R13) ----------------------------
	actions := map[string]bool{}
	arows, err := st.sqlDB.QueryContext(ctx, `SELECT json_extract(data, '$.action') FROM audit_logs`)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	defer arows.Close()
	for arows.Next() {
		var a string
		if err := arows.Scan(&a); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		actions[a] = true
	}
	if err := arows.Err(); err != nil {
		t.Fatalf("audit rows: %v", err)
	}
	for _, want := range []string{
		"initialize.profile", "initialize.regime", "initialize.accounts", "initialize.periods",
		"balance.upsert", "balances.lock", "balances.reopen", "activate",
	} {
		if !actions[want] {
			t.Errorf("audit trail missing %q", want)
		}
	}
}
