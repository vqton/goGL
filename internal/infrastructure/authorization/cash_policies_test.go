package authorization_test

import (
	"testing"

	"goGL/internal/infrastructure/authorization"
)

func TestSeedPolicies_CashRoles(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	for _, g := range [][]string{
		{"truongquy", "role:cashier"},
		{"ketoan", "role:cash_accountant"},
		{"kttruong", "role:chief_accountant"},
		{"giamdoc", "role:director"},
	} {
		if _, err := e.AddGroupingPolicy(g); err != nil {
			t.Fatalf("group %v: %v", g, err)
		}
	}

	cases := []struct {
		name string
		sub  string
		obj  string
		act  string
		want bool
	}{
		{"cashier posts voucher", "truongquy", "/api/v1/cash/vouchers/v-1/post", "POST", true},
		{"cashier reads book", "truongquy", "/api/v1/cash/books", "GET", true},
		{"cashier closes day", "truongquy", "/api/v1/cash/close-day", "POST", true},
		{"cashier cannot create voucher", "truongquy", "/api/v1/cash/vouchers", "POST", false},
		{"accountant creates voucher", "ketoan", "/api/v1/cash/vouchers", "POST", true},
		{"accountant reads voucher", "ketoan", "/api/v1/cash/vouchers/v-1", "GET", true},
		{"accountant reconciles", "ketoan", "/api/v1/cash/reconcile", "POST", true},
		{"accountant reaches post route", "ketoan", "/api/v1/cash/vouchers/v-1/post", "POST", true},
		{"chief can read counts", "kttruong", "/api/v1/cash/counts", "GET", true},
		{"chief can close day", "kttruong", "/api/v1/cash/close-day", "POST", true},
		{"director approves", "giamdoc", "/api/v1/cash/vouchers/v-1/approve", "POST", true},
		{"director cannot post", "giamdoc", "/api/v1/cash/vouchers/v-1/post", "POST", false},
		{"unknown role denied", "stranger", "/api/v1/cash/vouchers", "GET", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.Enforce(tc.sub, tc.obj, tc.act)
			if err != nil {
				t.Fatalf("enforce: %v", err)
			}
			if got != tc.want {
				t.Fatalf("enforce(%s, %s, %s) = %v, want %v", tc.sub, tc.obj, tc.act, got, tc.want)
			}
		})
	}
}

func TestSeedPolicies_CashRolesNotSeededTwice(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("second seed must be a no-op, got: %v", err)
	}

	if _, err := e.AddGroupingPolicy("ketoan", "role:cash_accountant"); err != nil {
		t.Fatalf("group: %v", err)
	}
	got, err := e.Enforce("ketoan", "/api/v1/cash/vouchers", "POST")
	if err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if !got {
		t.Fatal("expected cash_accountant POST create allowed after double seed")
	}
}
