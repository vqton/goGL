package ledger_test

import (
	"context"
	"database/sql"
	"testing"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	persledger "goGL/internal/infrastructure/persistence/ledger"
)

func newRepo(t *testing.T, d *sql.DB) domainledger.Repository {
	t.Helper()
	return persledger.NewSqliteRepository(d)
}

func TestSeedDefaultAccounts_Idempotent(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t, openTestDB(t))

	first, err := appledger.SeedDefaultAccounts(ctx, repo)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if first != appledger.DefaultChartSize() {
		t.Fatalf("first seed created %d, want %d", first, appledger.DefaultChartSize())
	}

	second, err := appledger.SeedDefaultAccounts(ctx, repo)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if second != 0 {
		t.Fatalf("second seed created %d accounts, want 0 (idempotent)", second)
	}

	accounts, err := repo.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != appledger.DefaultChartSize() {
		t.Fatalf("expected %d accounts, got %d", appledger.DefaultChartSize(), len(accounts))
	}

	// Summary accounts never allow posting; leaves do.
	byCode := map[string]*domainledger.Account{}
	for _, a := range accounts {
		byCode[a.Code] = a
	}
	if byCode["1"].AllowPost {
		t.Fatal("level-1 root must never allow posting")
	}
	if byCode["1111"].AllowPost == false {
		t.Fatal("leaf 1111 must allow posting")
	}
	if byCode["1111"].ParentCode != "111" || byCode["1111"].Type != domainledger.AccountAsset {
		t.Fatalf("unexpected seeded account: %+v", byCode["1111"])
	}
}

func TestSeedDefaultAccounts_PreservesExistingEdits(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t, openTestDB(t))

	if _, err := appledger.SeedDefaultAccounts(ctx, repo); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// A runtime rename must survive a re-seed.
	a, err := repo.GetAccountByCode(ctx, "1111")
	if err != nil {
		t.Fatalf("get 1111: %v", err)
	}
	a.Name = "Quỹ tiền mặt (sửa)"
	if err := repo.UpdateAccount(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}

	if _, err := appledger.SeedDefaultAccounts(ctx, repo); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	got, err := repo.GetAccountByCode(ctx, "1111")
	if err != nil {
		t.Fatalf("get after re-seed: %v", err)
	}
	if got.Name != "Quỹ tiền mặt (sửa)" {
		t.Fatalf("re-seed clobbered edit: %+v", got)
	}
}
