package authorization_test

import (
	"testing"

	"goGL/internal/infrastructure/authorization"
)

func TestSeedPolicies_SetupRoles(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	for _, g := range [][]string{
		{"danhmuc", "role:danh_muc"},
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
		// Read roles: status/profile/balances list + check (POST).
		{"danh_muc reads status", "danhmuc", "/api/v1/setup/status", "GET", true},
		{"danh_muc reads profile", "danhmuc", "/api/v1/setup/profile", "GET", true},
		{"danh_muc lists balances", "danhmuc", "/api/v1/setup/opening-balances", "GET", true},
		{"danh_muc runs check", "danhmuc", "/api/v1/setup/opening-balances/check", "POST", true},
		{"ketoan reads status", "ketoan", "/api/v1/setup/status", "GET", true},
		{"kttruong reads status", "kttruong", "/api/v1/setup/status", "GET", true},
		{"giamdoc reads status", "giamdoc", "/api/v1/setup/status", "GET", true},
		{"kiemtoan reads status", "kiemtoan", "/api/v1/setup/status", "GET", true},
		{"read roles cannot initialize", "giamdoc", "/api/v1/setup/initialize", "POST", false},
		{"read roles cannot lock", "kiemtoan", "/api/v1/setup/opening-balances/lock", "POST", false},

		// ke_toan_tong_hop: entry/delete/import only — no lifecycle control.
		{"ketoan saves balance", "ketoan", "/api/v1/setup/opening-balances", "POST", true},
		{"ketoan deletes balance", "ketoan", "/api/v1/setup/opening-balances/b-ob1", "DELETE", true},
		{"ketoan imports", "ketoan", "/api/v1/setup/opening-balances/import", "POST", true},
		{"ketoan reads import report", "ketoan", "/api/v1/setup/opening-balances/import/job-1/report", "GET", true},
		{"ketoan cannot initialize", "ketoan", "/api/v1/setup/initialize", "POST", false},
		{"ketoan cannot lock", "ketoan", "/api/v1/setup/opening-balances/lock", "POST", false},
		{"ketoan cannot reopen", "ketoan", "/api/v1/setup/opening-balances/reopen", "POST", false},
		{"ketoan cannot activate", "ketoan", "/api/v1/setup/activate", "POST", false},
		{"ketoan cannot edit profile", "ketoan", "/api/v1/setup/profile", "PUT", false},

		// ke_toan_truong: full lifecycle + override on balance writes.
		{"kttruong initializes", "kttruong", "/api/v1/setup/initialize", "POST", true},
		{"kttruong edits profile", "kttruong", "/api/v1/setup/profile", "PUT", true},
		{"kttruong saves balance", "kttruong", "/api/v1/setup/opening-balances", "POST", true},
		{"kttruong imports", "kttruong", "/api/v1/setup/opening-balances/import", "POST", true},
		{"kttruong locks", "kttruong", "/api/v1/setup/opening-balances/lock", "POST", true},
		{"kttruong reopens", "kttruong", "/api/v1/setup/opening-balances/reopen", "POST", true},
		{"kttruong activates", "kttruong", "/api/v1/setup/activate", "POST", true},
		{"kttruong reads report", "kttruong", "/api/v1/setup/opening-balances/import/job-1/report", "GET", true},

		// admin has `* *`.
		{"admin initializes", "admin", "/api/v1/setup/initialize", "POST", true},
		{"admin locks", "admin", "/api/v1/setup/opening-balances/lock", "POST", true},

		// Unknown principal denied.
		{"unknown role denied", "stranger", "/api/v1/setup/status", "GET", false},
		{"anonymous denied", "anonymous", "/api/v1/setup/status", "GET", false},
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
