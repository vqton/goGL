package db_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/infrastructure/db"
)

// openTestDB opens a uniquely-named in-memory SQLite database.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	d, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func tableExists(t *testing.T, d *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := d.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name,
	).Scan(&n); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return n > 0
}

func TestMigrate_CreatesCashTables(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()

	if err := db.Migrate(ctx, d); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, name := range []string{
		"cash_funds", "cash_vouchers", "cash_book", "cash_counts", "cash_reconciliations", "cash_sequences",
	} {
		if !tableExists(t, d, name) {
			t.Fatalf("expected table %q to exist after migrate", name)
		}
	}
}

func TestMigrate_CashSequencesSchema(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()

	if err := db.Migrate(ctx, d); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows, err := d.Query(`PRAGMA table_info(cash_sequences)`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()

	cols := map[string]string{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		cols[name] = typ
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}

	// cash_sequences is a plain JSON-document table like every other row;
	// the counter lives in the 'data' doc (see repository.NextRefNo).
	for _, name := range []string{"id", "data"} {
		if cols[name] == "" {
			t.Fatalf("expected column %q in cash_sequences, got %v", name, cols)
		}
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()

	if err := db.Migrate(ctx, d); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.Migrate(ctx, d); err != nil {
		t.Fatalf("second migrate must be a no-op, got: %v", err)
	}
}
