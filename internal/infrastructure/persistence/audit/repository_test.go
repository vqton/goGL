package audit_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/audit"
	"goGL/internal/infrastructure/db"
	persaudit "goGL/internal/infrastructure/persistence/audit"
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

func TestRepository_CreateAndFind(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	repo := persaudit.NewSqliteRepository(d)

	in := &audit.AuditLog{
		ID:        "log-1",
		UserCode:  "truongquy",
		Module:    "cash",
		Action:    "voucher.create",
		TargetID:  "v-1",
		Timestamp: "2026-08-08T10:00:00Z",
	}
	if err := repo.Create(ctx, in); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByID(ctx, "log-1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.UserCode != in.UserCode || got.Module != in.Module ||
		got.Action != in.Action || got.TargetID != in.TargetID || got.Timestamp != in.Timestamp {
		t.Fatalf("round trip mismatch: got %+v", got)
	}
}

func TestRepository_FindByID_Missing(t *testing.T) {
	d := openTestDB(t)
	repo := persaudit.NewSqliteRepository(d)

	_, err := repo.FindByID(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for missing audit log")
	}
	// The repository wraps sql.ErrNoRows into a domain error
	if err == sql.ErrNoRows {
		t.Fatal("expected domain error, got raw sql.ErrNoRows")
	}
}

func TestRepository_ListRecent(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	repo := persaudit.NewSqliteRepository(d)

	logs := []audit.AuditLog{
		{ID: "1", Module: "setup", Action: "setup.activate", TargetID: "s", Timestamp: "2026-08-08T12:00:00Z"},
		{ID: "2", Module: "cash", Action: "voucher.create", TargetID: "v-1", Timestamp: "2026-08-08T11:00:00Z"},
		{ID: "3", Module: "setup", Action: "balances.lock", TargetID: "s", Timestamp: "2026-08-08T10:00:00Z"},
		{ID: "4", Module: "setup", Action: "balances.reopen", TargetID: "s", Timestamp: "2026-08-08T09:00:00Z"},
	}
	for i := range logs {
		if err := repo.Create(ctx, &logs[i]); err != nil {
			t.Fatalf("create %s: %v", logs[i].ID, err)
		}
	}

	got, err := repo.ListRecent(ctx, "setup", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0].ID != "1" || got[1].ID != "3" {
		t.Fatalf("module-filtered list mismatch: %+v", got)
	}

	all, err := repo.ListRecent(ctx, "", 10)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 4 || all[0].ID != "1" || all[3].ID != "4" {
		t.Fatalf("all-modules list must be newest-first: %+v", all)
	}
	// timestamps are RFC3339 with an offset, so the lexicographic ORDER BY on
	// the same timezone holds; entries equal on timestamp break by insertion.
}
