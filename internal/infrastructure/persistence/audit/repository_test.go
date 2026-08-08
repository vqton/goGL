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

	if _, err := repo.FindByID(context.Background(), "nope"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
