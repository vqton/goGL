package authorization_test

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/casbin/casbin/v3"
	_ "modernc.org/sqlite"

	"goGL/internal/infrastructure/authorization"
)

// openTestDB opens a uniquely-named in-memory SQLite database and creates the
// policy table, mirroring db.Migrate for the casbin_policies table.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	db, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS casbin_policies (
		id   TEXT PRIMARY KEY,
		data TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create casbin_policies: %v", err)
	}
	return db
}

// newEnforcer builds an enforcer backed by the test database.
func newEnforcer(t *testing.T, db *sql.DB) *casbin.Enforcer {
	t.Helper()

	m, err := authorization.RBACModel()
	if err != nil {
		t.Fatalf("build model: %v", err)
	}
	e, err := casbin.NewEnforcer(m, authorization.NewSqliteAdapter(db))
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	return e
}

func TestAdapter_SaveLoadRoundTrip(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)

	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddGroupingPolicy("alice", "role:viewer"); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}
	if err := e.SavePolicy(); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	// A fresh enforcer must see exactly what was persisted.
	reloaded := newEnforcer(t, db)
	got, err := reloaded.GetPolicy()
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if len(got) != 1 || len(got[0]) != 3 {
		t.Fatalf("policy set mismatch: %v", got)
	}
	gotG, err := reloaded.GetGroupingPolicy()
	if err != nil {
		t.Fatalf("get grouping policy: %v", err)
	}
	if len(gotG) != 1 || len(gotG[0]) != 2 {
		t.Fatalf("grouping set mismatch: %v", gotG)
	}

	if ok, err := reloaded.Enforce("alice", "/api/v1/cash/vouchers", "GET"); err != nil || !ok {
		t.Fatalf("expected alice GET allowed, ok=%v err=%v", ok, err)
	}
}

func TestAdapter_AddPolicyAutosaves(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)

	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddGroupingPolicy("alice", "role:viewer"); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	// AutoSave must have persisted without an explicit SavePolicy call.
	reloaded := newEnforcer(t, db)
	if ok, err := reloaded.Enforce("alice", "/api/v1/cash/vouchers", "GET"); err != nil || !ok {
		t.Fatalf("expected alice GET allowed after autosave, ok=%v err=%v", ok, err)
	}
	if ok, err := reloaded.Enforce("alice", "/api/v1/cash/vouchers", "POST"); err != nil || ok {
		t.Fatalf("expected alice POST denied after autosave, ok=%v err=%v", ok, err)
	}
}

func TestAdapter_RemovePolicyPersists(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)

	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddGroupingPolicy("alice", "role:viewer"); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	if removed, err := e.RemovePolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil || !removed {
		t.Fatalf("remove policy: removed=%v err=%v", removed, err)
	}

	reloaded := newEnforcer(t, db)
	if ok, err := reloaded.Enforce("alice", "/api/v1/cash/vouchers", "GET"); err != nil || ok {
		t.Fatalf("expected alice GET denied after removal, ok=%v err=%v", ok, err)
	}
}

func TestAdapter_RemoveFilteredPolicyPersists(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)

	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddPolicy("role:admin", "*", "*"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	// Remove every p rule whose subject is role:viewer.
	if removed, err := e.RemoveFilteredPolicy(0, "role:viewer"); err != nil || !removed {
		t.Fatalf("remove filtered policy: removed=%v err=%v", removed, err)
	}

	reloaded := newEnforcer(t, db)
	got, err := reloaded.GetPolicy()
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if len(got) != 1 || got[0][0] != "role:admin" {
		t.Fatalf("unexpected remaining policies: %v", got)
	}
}

func TestAdapter_AddPolicyIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)

	rule := []string{"role:viewer", "/api/v1/cash/*", "GET"}
	if _, err := e.AddPolicy(rule); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddPolicy(rule); err != nil {
		t.Fatalf("add duplicate policy: %v", err)
	}

	// The store must contain one row, not two.
	rows, err := db.Query(`SELECT COUNT(*) FROM casbin_policies`)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no row returned")
	}
	var n int
	if err := rows.Scan(&n); err != nil {
		t.Fatalf("scan count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 stored rule, got %d", n)
	}
}
