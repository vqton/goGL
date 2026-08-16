package setup_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	appsetup "goGL/internal/application/setup"
	"goGL/internal/domain/core"
	"goGL/internal/domain/ledger"
	"goGL/internal/domain/masterdata"
	domainsetup "goGL/internal/domain/setup"
	"goGL/internal/infrastructure/db"
	persistence "goGL/internal/infrastructure/persistence/setup"

	_ "modernc.org/sqlite"
)

// benchHarness is the full service over an in-memory SQLite DB plus the fake
// seams, pre-initialized to balances_draft. It is shared across iterations:
// import upserts are idempotent (deterministic ids), so repeated runs measure
// steady-state throughput, not first-write latency.
func benchHarness(b *testing.B, accounts map[string]*ledger.Account) appsetup.Service {
	b.Helper()
	dsn := fmt.Sprintf("file:bench_%p?mode=memory&cache=shared", b)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	b.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewSqliteRepository(d)
	fo := &fakeObjects{recs: map[string]*masterdata.Record{
		"customer:KH-0001": {Code: "KH-0001", Kind: masterdata.KindCustomer, State: masterdata.StateActive},
	}}
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime:   &fakeRegime{},
		Seeder:   &fakeSeeder{n: len(accounts)},
		Objects:  fo,
		Periods:  &fakePeriods{},
		Accounts: &fakeAccounts{accts: accounts},
		Postings: &fakePostings{},
		Audit:    &fakeAudit{},
	})
	if _, err := svc.Initialize(context.Background(), &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "bench"); err != nil {
		b.Fatalf("init: %v", err)
	}
	return svc
}

func benchAccounts() map[string]*ledger.Account {
	codes := []string{"1111", "1121", "5111", "5112", "5113", "5211", "5212", "5213",
		"1110", "1112", "1113", "1114", "1115", "1116", "1117", "1118"}
	accts := make(map[string]*ledger.Account, len(codes))
	for _, c := range codes {
		accts[c] = &ledger.Account{Code: c, Status: ledger.AccountActive, AllowPost: true, Level: 2}
	}
	return accts
}

// BenchmarkImportBalances measures the balanced 100-row commit path: template
// check, per-row parse/validate, one batched upsert tx, job persistence, audit.
func BenchmarkImportBalances100(b *testing.B) {
	svc := benchHarness(b, benchAccounts())
	ctx := context.Background()
	header := []string{"account", "object_type", "object_code", "debit", "credit"}
	rows := make([][]string, 0, 101)
	rows = append(rows, header)
	for i := 0; i < 50; i++ {
		rows = append(rows, []string{"1111", "", "", "100", ""})
		rows = append(rows, []string{"5111", "", "", "", "100"})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ImportBalances(ctx, rows, "bench", false); err != nil {
			b.Fatalf("import: %v", err)
		}
	}
}

// BenchmarkConcurrentSaveBalance exercises the single-writer path: the service
// mutex serializes writers, and SQLite + the deterministic-id upsert make
// re-saves idempotent. 8 goroutines × distinct accounts.
func BenchmarkConcurrentSaveBalance(b *testing.B) {
	svc := benchHarness(b, benchAccounts())
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			ob := &domainsetup.OpeningBalance{
				AccountCode: fmt.Sprintf("111%d", g),
				Debit:       core.Money{AmountMinor: 100, Currency: "VND"},
			}
			for i := 0; i < b.N; i++ {
				if _, err := svc.SaveBalance(ctx, ob, "bench"); err != nil {
					errs <- err
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		b.Fatalf("save: %v", err)
	}
}
