package authorization_test

import (
	"testing"

	"goGL/internal/infrastructure/authorization"
)

func TestSeedPolicies_LedgerRoles(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	for _, g := range [][]string{
		{"ketoan", "role:ke_toan_tong_hop"},
		{"kttruong", "role:ke_toan_truong"},
		{"giamdoc", "role:giam_doc"},
		{"kiemtoan", "role:kiem_toan"},
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
		// ke_toan_tong_hop — general accountant
		{"accountant reads accounts", "ketoan", "/api/v1/ledger/accounts", "GET", true},
		{"accountant creates account", "ketoan", "/api/v1/ledger/accounts", "POST", true},
		{"accountant updates account", "ketoan", "/api/v1/ledger/accounts/account-1111", "PATCH", true},
		{"accountant reads one account", "ketoan", "/api/v1/ledger/accounts/account-1111", "GET", true},
		{"accountant creates entry", "ketoan", "/api/v1/ledger/entries", "POST", true},
		{"accountant reads entry", "ketoan", "/api/v1/ledger/entries/e-1", "GET", true},
		{"accountant deletes draft", "ketoan", "/api/v1/ledger/entries/e-1", "DELETE", true},
		{"accountant manages templates", "ketoan", "/api/v1/ledger/templates", "POST", true},
		{"accountant reposts source", "ketoan", "/api/v1/ledger/postings/CASH/v-1", "POST", true},
		{"accountant reads books", "ketoan", "/api/v1/ledger/books/general-journal", "GET", true},
		{"accountant cannot post entry", "ketoan", "/api/v1/ledger/entries/e-1/post", "POST", false},
		{"accountant cannot reverse", "ketoan", "/api/v1/ledger/entries/e-1/reverse", "POST", false},
		{"accountant cannot close period", "ketoan", "/api/v1/ledger/periods/2026-08/close", "POST", false},
		{"accountant cannot set opening balances", "ketoan", "/api/v1/ledger/opening-balances", "POST", false},

		// ke_toan_truong — chief accountant
		{"chief posts entry", "kttruong", "/api/v1/ledger/entries/e-1/post", "POST", true},
		{"chief reverses entry", "kttruong", "/api/v1/ledger/entries/e-1/reverse", "POST", true},
		{"chief opens period", "kttruong", "/api/v1/ledger/periods/2026-08/open", "POST", true},
		{"chief closes period", "kttruong", "/api/v1/ledger/periods/2026-08/close", "POST", true},
		{"chief reopens period", "kttruong", "/api/v1/ledger/periods/2026-08/reopen", "POST", true},
		{"chief runs close templates", "kttruong", "/api/v1/ledger/periods/2026-08/close/run", "POST", true},
		{"chief sets opening balances", "kttruong", "/api/v1/ledger/opening-balances", "POST", true},
		{"chief reads books", "kttruong", "/api/v1/ledger/books/trial-balance", "GET", true},
		{"chief reads entries", "kttruong", "/api/v1/ledger/entries/e-1", "GET", true},
		{"chief cannot create account", "kttruong", "/api/v1/ledger/accounts", "POST", false},
		{"chief cannot update account", "kttruong", "/api/v1/ledger/accounts/account-1111", "PATCH", false},
		{"chief cannot delete entry", "kttruong", "/api/v1/ledger/entries/e-1", "DELETE", false},
		{"chief cannot create entry", "kttruong", "/api/v1/ledger/entries", "POST", false},

		// giam_doc — director (read-only)
		{"director reads books", "giamdoc", "/api/v1/ledger/books/general-journal", "GET", true},
		{"director reads accounts", "giamdoc", "/api/v1/ledger/accounts", "GET", true},
		{"director reads periods", "giamdoc", "/api/v1/ledger/periods", "GET", true},
		{"director cannot post", "giamdoc", "/api/v1/ledger/entries/e-1/post", "POST", false},
		{"director cannot close period", "giamdoc", "/api/v1/ledger/periods/2026-08/close", "POST", false},

		// kiem_toan — internal audit (read-only)
		{"auditor reads books", "kiemtoan", "/api/v1/ledger/books/detail", "GET", true},
		{"auditor reads entries", "kiemtoan", "/api/v1/ledger/entries/e-1", "GET", true},
		{"auditor cannot post", "kiemtoan", "/api/v1/ledger/entries/e-1/post", "POST", false},
		{"auditor cannot create account", "kiemtoan", "/api/v1/ledger/accounts", "POST", false},

		{"unknown role denied", "stranger", "/api/v1/ledger/entries", "GET", false},
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
