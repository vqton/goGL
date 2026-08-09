package authorization_test

import (
	"testing"

	"goGL/internal/infrastructure/authorization"
)

func TestNewEnforcer_SeedsDefaultPoliciesOnEmptyStore(t *testing.T) {
	db := openTestDB(t)

	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	// The default admin user must have full access via the admin role.
	if ok, err := e.Enforce("admin", "/api/v1/cash/vouchers", "GET"); err != nil || !ok {
		t.Fatalf("expected admin GET allowed, ok=%v err=%v", ok, err)
	}
	if ok, err := e.Enforce("admin", "/api/v1/users", "DELETE"); err != nil || !ok {
		t.Fatalf("expected admin DELETE allowed, ok=%v err=%v", ok, err)
	}

	// Any other subject must be denied.
	if ok, err := e.Enforce("bob", "/api/v1/cash/vouchers", "GET"); err != nil || ok {
		t.Fatalf("expected bob denied, ok=%v err=%v", ok, err)
	}

	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if len(policies) != len(authorization.DefaultPolicies.Policies) {
		t.Fatalf("expected %d seeded p rules, got %d", len(authorization.DefaultPolicies.Policies), len(policies))
	}
	grouping, err := e.GetGroupingPolicy()
	if err != nil {
		t.Fatalf("get grouping policy: %v", err)
	}
	if len(grouping) != len(authorization.DefaultPolicies.Grouping) {
		t.Fatalf("expected %d seeded g rules, got %d", len(authorization.DefaultPolicies.Grouping), len(grouping))
	}
}

func TestSeedDefaultPolicies_IsIdempotent(t *testing.T) {
	db := openTestDB(t)

	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := authorization.SeedDefaultPolicies(e); err != nil {
			t.Fatalf("seed defaults (pass %d): %v", i, err)
		}
	}

	// Seeding again must not duplicate rules.
	rows, err := db.Query(`SELECT COUNT(*) FROM casbin_policies`)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no count row")
	}
	var n int
	if err := rows.Scan(&n); err != nil {
		t.Fatalf("scan count: %v", err)
	}
	expected := len(authorization.DefaultPolicies.Policies) + len(authorization.DefaultPolicies.Grouping)
	if n != expected {
		t.Fatalf("expected %d stored rules, got %d", expected, n)
	}
}
