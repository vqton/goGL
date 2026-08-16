package ledger_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	d, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })

	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

func newRepo(t *testing.T, d *sql.DB) ledger.Repository {
	t.Helper()
	return persledger.NewSqliteRepository(d)
}

func validEntry() *ledger.JournalEntry {
	return &ledger.JournalEntry{
		ID:          "e-1",
		VoucherDate: "2026-08-05",
		Period:      "2026-08",
		Source:      ledger.SourceManual,
		Description: "Test entry",
		Status:      ledger.EntryDraft,
		CreatedBy:   "ketoan",
		Lines: []ledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	}
}

func validAccount() *ledger.Account {
	return &ledger.Account{
		ID:        ledger.RowID("account", "1111"),
		Code:      "1111",
		Name:      "Tiền mặt - VND",
		Type:      ledger.AccountAsset,
		Level:     2,
		Status:    ledger.AccountActive,
		AllowPost: true,
	}
}

func TestRepository_EntryRoundTrip(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()
	e := validEntry()

	if err := repo.CreateEntry(ctx, e); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	got, err := repo.GetEntry(ctx, e.ID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got.ID != e.ID || got.Period != "2026-08" || len(got.Lines) != 2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if got.Lines[0].Debit != 5_000_000 || got.Lines[1].Credit != 5_000_000 {
		t.Fatalf("line amounts lost: %+v", got.Lines)
	}

	e.Status = ledger.EntryPosted
	if err := repo.UpdateEntry(ctx, e); err != nil {
		t.Fatalf("update entry: %v", err)
	}
	got, err = repo.GetEntry(ctx, e.ID)
	if err != nil {
		t.Fatalf("get updated entry: %v", err)
	}
	if got.Status != ledger.EntryPosted {
		t.Fatalf("status = %q, want posted", got.Status)
	}
}

func TestRepository_GetEntryNotFound(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	_, err := repo.GetEntry(context.Background(), "nope")
	if err != sql.ErrNoRows {
		t.Fatalf("get missing entry: got %v, want sql.ErrNoRows", err)
	}
}

func TestRepository_ListEntriesFilterByPeriod(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	for _, e := range []*ledger.JournalEntry{
		validEntry(),
		func() *ledger.JournalEntry {
			e := validEntry()
			e.ID = "e-2"
			e.VoucherDate = "2026-09-01"
			e.Period = "2026-09"
			return e
		}(),
	} {
		if err := repo.CreateEntry(ctx, e); err != nil {
			t.Fatalf("create entry: %v", err)
		}
	}

	got, err := repo.ListEntries(ctx, ledger.EntryFilter{Period: "2026-08"})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(got) != 1 || got[0].ID != "e-1" {
		t.Fatalf("period filter: got %d entries (%s), want 1 (e-1)", len(got), idsOf(got))
	}
}

func idsOf(es []*ledger.JournalEntry) string {
	out := ""
	for i, e := range es {
		if i > 0 {
			out += ","
		}
		out += e.ID
	}
	return out
}

func TestRepository_AccountAndPeriodRoundTrip(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	a := validAccount()
	if err := repo.CreateAccount(ctx, a); err != nil {
		t.Fatalf("create account: %v", err)
	}
	byID, err := repo.GetAccount(ctx, a.ID)
	if err != nil {
		t.Fatalf("get account by id: %v", err)
	}
	if byID.Name != "Tiền mặt - VND" {
		t.Fatalf("account name mismatch: %+v", byID)
	}
	byCode, err := repo.GetAccountByCode(ctx, "1111")
	if err != nil {
		t.Fatalf("get account by code: %v", err)
	}
	if byCode.ID != a.ID {
		t.Fatalf("by code id mismatch: %+v", byCode)
	}

	accounts, err := repo.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	p := &ledger.AccountingPeriod{ID: "2026-08", Year: 2026, Month: 8, Status: ledger.PeriodOpen}
	if err := repo.CreatePeriod(ctx, p); err != nil {
		t.Fatalf("create period: %v", err)
	}
	got, err := repo.GetPeriod(ctx, "2026-08")
	if err != nil {
		t.Fatalf("get period: %v", err)
	}
	if got.Status != ledger.PeriodOpen {
		t.Fatalf("period status = %q, want open", got.Status)
	}
	periods, err := repo.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("list periods: %v", err)
	}
	if len(periods) != 1 || periods[0].ID != "2026-08" {
		t.Fatalf("expected 1 period, got %+v", periods)
	}
}
