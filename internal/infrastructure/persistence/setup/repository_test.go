package setup_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/core"
	"goGL/internal/domain/setup"
	"goGL/internal/infrastructure/db"
	persistence "goGL/internal/infrastructure/persistence/setup"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:setup_repo_%p?mode=memory&cache=shared", t)
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

func newRepo(t *testing.T) setup.Repository {
	t.Helper()
	return persistence.NewSqliteRepository(openTestDB(t))
}

func TestProfileRoundtrip(t *testing.T) {
	repo := newRepo(t)
	p := &setup.CompanyProfile{
		ID:                 setup.ProfileID,
		Name:               "Cty TNHH SX Thép ABC",
		TaxCode:            "0101234567",
		FiscalYearStart:    "2026-01-01",
		AccountingRegime:   "TT99-2025",
		AccountingCurrency: "VND",
		Status:             setup.StatusProfiled,
	}
	if err := repo.SaveProfile(context.Background(), p); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := repo.GetProfile(context.Background(), setup.ProfileID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != p.Name || got.TaxCode != p.TaxCode || got.Status != setup.StatusProfiled {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestSaveProfileUpsertIsIdempotent(t *testing.T) {
	repo := newRepo(t)
	p := &setup.CompanyProfile{ID: setup.ProfileID, Name: "A", TaxCode: "0101234567", Status: setup.StatusProfiled}
	if err := repo.SaveProfile(context.Background(), p); err != nil {
		t.Fatalf("save: %v", err)
	}
	p.Name = "A2"
	if err := repo.SaveProfile(context.Background(), p); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	got, err := repo.GetProfile(context.Background(), setup.ProfileID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "A2" {
		t.Fatalf("expected overwritten name, got %q", got.Name)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	repo := newRepo(t)
	if _, err := repo.GetProfile(context.Background(), setup.ProfileID); err != sql.ErrNoRows {
		t.Fatalf("want sql.ErrNoRows, got %v", err)
	}
}

func TestBalanceUpsertByIdempotentByID(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	b := &setup.OpeningBalance{
		ID:          setup.BalanceID("1111", ""),
		AccountCode: "1111",
		Period:      core.Period{From: "2026-01-01", To: "2026-01-01"},
		Debit:       core.Money{AmountMinor: 500_000_000, Currency: "VND"},
		Status:      setup.BalanceDraft,
	}
	if err := repo.SaveBalance(ctx, b); err != nil {
		t.Fatalf("save: %v", err)
	}
	b.Debit.AmountMinor = 600_000_000
	if err := repo.SaveBalance(ctx, b); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	list, err := repo.ListBalances(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 row after re-save, got %d", len(list))
	}
	if list[0].Debit.AmountMinor != 600_000_000 {
		t.Fatalf("expected overwritten debit, got %+v", list[0].Debit)
	}
}

func TestListBalancesFilterByAccount(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	for _, acct := range []string{"1111", "131"} {
		if err := repo.SaveBalance(ctx, &setup.OpeningBalance{
			ID: setup.BalanceID(acct, ""), AccountCode: acct,
			Debit: core.Money{AmountMinor: 100, Currency: "VND"}, Status: setup.BalanceDraft,
		}); err != nil {
			t.Fatalf("save %s: %v", acct, err)
		}
	}
	list, err := repo.ListBalances(ctx, "1111")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].AccountCode != "1111" {
		t.Fatalf("filter leak: %+v", list)
	}
}

func TestDeleteBalance(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	id := setup.BalanceID("1111", "")
	if err := repo.SaveBalance(ctx, &setup.OpeningBalance{
		ID: id, AccountCode: "1111", Status: setup.BalanceDraft,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := repo.DeleteBalance(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ := repo.ListBalances(ctx, "")
	if len(list) != 0 {
		t.Fatalf("want 0 rows after delete, got %d", len(list))
	}
}

func TestSaveProfileDefaultsID(t *testing.T) {
	repo := newRepo(t)
	p := &setup.CompanyProfile{Name: "A", TaxCode: "0101234567"}
	if err := repo.SaveProfile(context.Background(), p); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := repo.GetProfile(context.Background(), setup.ProfileID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "A" {
		t.Fatalf("want saved under ProfileID, got %+v", got)
	}
}

func TestDeleteBalanceNotFound(t *testing.T) {
	repo := newRepo(t)
	err := repo.DeleteBalance(context.Background(), setup.BalanceID("1111", ""))
	if !errors.Is(err, setup.ErrBalanceNotFound) {
		t.Fatalf("want ErrBalanceNotFound, got %v", err)
	}
}

// closedDB drives every Exec/Query error branch by handing the repository a
// database whose pool is closed.
func TestRepositoryDBErrorPaths(t *testing.T) {
	repo := persistence.NewSqliteRepository(openTestDB(t))
	_ = repo
	ctx := context.Background()

	closed := persistence.NewSqliteRepository(mustClosedDB(t))
	p := &setup.CompanyProfile{Name: "A", TaxCode: "0101234567"}
	if err := closed.SaveProfile(ctx, p); err == nil {
		t.Fatal("SaveProfile on closed db must error")
	}
	if _, err := closed.GetProfile(ctx, setup.ProfileID); err == nil {
		t.Fatal("GetProfile on closed db must error")
	}
	if err := closed.SaveBalance(ctx, &setup.OpeningBalance{ID: "x"}); err == nil {
		t.Fatal("SaveBalance on closed db must error")
	}
	if _, err := closed.ListBalances(ctx, ""); err == nil {
		t.Fatal("ListBalances on closed db must error")
	}
	if err := closed.DeleteBalance(ctx, "x"); err == nil {
		t.Fatal("DeleteBalance on closed db must error")
	}
	if _, err := closed.GetStatus(ctx); err == nil {
		t.Fatal("GetStatus on closed db must error")
	}
	if err := closed.SetStatus(ctx, setup.StatusProfiled); err == nil {
		t.Fatal("SetStatus on closed db must error")
	}
	if err := closed.SaveBalances(ctx, []*setup.OpeningBalance{{ID: "x"}}); err == nil {
		t.Fatal("SaveBalances on closed db must error")
	}
	if err := closed.SaveImportJob(ctx, &setup.ImportJob{ID: "x"}); err == nil {
		t.Fatal("SaveImportJob on closed db must error")
	}
	if _, err := closed.GetImportJob(ctx, "x"); err == nil {
		t.Fatal("GetImportJob on closed db must error")
	}
}

func mustClosedDB(t *testing.T) *sql.DB {
	t.Helper()
	d := openTestDB(t)
	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return d
}

func TestStatusRoundtrip(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	st, err := repo.GetStatus(ctx)
	if err != nil {
		t.Fatalf("get initial status: %v", err)
	}
	if st != setup.StatusEmpty {
		t.Fatalf("want StatusEmpty initially, got %s", st)
	}
	if err := repo.SetStatus(ctx, setup.StatusProfiled); err != nil {
		t.Fatalf("set: %v", err)
	}
	st, err = repo.GetStatus(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if st != setup.StatusProfiled {
		t.Fatalf("want profiled, got %s", st)
	}
	if err := repo.SetStatus(ctx, setup.StatusAccountsSeeded); err != nil {
		t.Fatalf("re-set: %v", err)
	}
	if st, _ = repo.GetStatus(ctx); st != setup.StatusAccountsSeeded {
		t.Fatalf("want accounts_seeded, got %s", st)
	}
}

func TestSaveBalancesBatchTx(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	bs := []*setup.OpeningBalance{
		{ID: "a", AccountCode: "1111", Debit: core.Money{AmountMinor: 100, Currency: "VND"}},
		{ID: "b", AccountCode: "331", Credit: core.Money{AmountMinor: 100, Currency: "VND"}},
	}
	if err := repo.SaveBalances(ctx, bs); err != nil {
		t.Fatalf("save batch: %v", err)
	}
	list, err := repo.ListBalances(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 rows, got %d", len(list))
	}
	// batched upsert overwrites existing rows (idempotent re-import)
	if err := repo.SaveBalances(ctx, []*setup.OpeningBalance{
		{ID: "a", AccountCode: "1111", Debit: core.Money{AmountMinor: 150, Currency: "VND"}},
	}); err != nil {
		t.Fatalf("re-save batch: %v", err)
	}
	list, _ = repo.ListBalances(ctx, "")
	if len(list) != 2 {
		t.Fatalf("want 2 rows after upsert, got %d", len(list))
	}
}

func TestImportJobRoundtrip(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	job := &setup.ImportJob{
		ID: "job-1", Status: setup.JobErrored, Total: 2, Created: 1, Updated: 0,
		Errors: []setup.RowError{{Row: 3, Message: "account does not exist"}},
		DryRun: true, CreatedBy: "user-1", CreatedAt: "2026-01-01T00:00:00Z",
	}
	if err := repo.SaveImportJob(ctx, job); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := repo.GetImportJob(ctx, "job-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "job-1" || got.Status != setup.JobErrored || got.Total != 2 || len(got.Errors) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if !got.DryRun || got.CreatedBy != "user-1" || got.Errors[0].Row != 3 {
		t.Fatalf("roundtrip metadata mismatch: %+v", got)
	}
	// save overwrites (idempotent re-upload)
	job.Status = setup.JobOK
	job.Errors = nil
	if err := repo.SaveImportJob(ctx, job); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	if got, _ = repo.GetImportJob(ctx, "job-1"); got.Status != setup.JobOK || len(got.Errors) != 0 {
		t.Fatalf("re-save mismatch: %+v", got)
	}
	if _, err := repo.GetImportJob(ctx, "nope"); !errors.Is(err, setup.ErrImportNotFound) {
		t.Fatalf("missing job: err = %v, want ErrImportNotFound", err)
	}
}
