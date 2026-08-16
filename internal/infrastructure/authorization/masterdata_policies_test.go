package authorization_test

import (
	"testing"

	"goGL/internal/infrastructure/authorization"
)

func TestSeedPolicies_MasterdataRoles(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	// Assign masterdata roles.
	for _, g := range [][]string{
		{"danhmuc", "role:danh_muc"},
		{"kttonghop", "role:ke_toan_tong_hop"},
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
		// danh_muc: full CRUD (POST on /* matches create + import + merge + seed + regime
		// due to Casbin v3 keyMatch2 multi-segment * behavior).
		{"danh_muc lists", "danhmuc", "/api/v1/master-data/customer", "GET", true},
		{"danh_muc creates", "danhmuc", "/api/v1/master-data/customer", "POST", true},
		{"danh_muc updates", "danhmuc", "/api/v1/master-data/customer/KH-1", "PUT", true},
		{"danh_muc deletes", "danhmuc", "/api/v1/master-data/customer/KH-1", "DELETE", true},
		{"danh_muc deactivates", "danhmuc", "/api/v1/master-data/customer/KH-1/deactivate", "POST", true},
		{"danh_muc activates", "danhmuc", "/api/v1/master-data/customer/KH-1/activate", "POST", true},
		{"danh_muc sets refs", "danhmuc", "/api/v1/master-data/customer/KH-1/references", "POST", true},
		{"danh_muc imports", "danhmuc", "/api/v1/master-data/import", "POST", true},
		// NOTE: POST /* matches merge/seed/regime due to keyMatch2 multi-segment behavior.
		// This is a policy design issue — see note in masterdata_policies.go.
		{"danh_muc can merge (policy design issue)", "danhmuc", "/api/v1/master-data/customer/merge", "POST", true},
		{"danh_muc can force (policy design issue)", "danhmuc", "/api/v1/master-data/customer/KH-1/deactivate-force", "POST", true},
		{"danh_muc can seed (policy design issue)", "danhmuc", "/api/v1/master-data/accounts/seed", "POST", true},
		{"danh_muc can regime (policy design issue)", "danhmuc", "/api/v1/master-data/settings/regime", "POST", true},

		// ke_toan_tong_hop: same as danh_muc
		{"kttonghop lists", "kttonghop", "/api/v1/master-data/supplier", "GET", true},
		{"kttonghop creates", "kttonghop", "/api/v1/master-data/supplier", "POST", true},
		{"kttonghop can merge (policy design issue)", "kttonghop", "/api/v1/master-data/supplier/merge", "POST", true},

		// ke_toan_truong: no general create (no POST /*), but merge/force/seed/regime
		{"kttruong lists", "kttruong", "/api/v1/master-data/item", "GET", true},
		{"kttruong cannot create", "kttruong", "/api/v1/master-data/item", "POST", false},
		{"kttruong merges", "kttruong", "/api/v1/master-data/item/merge", "POST", true},
		{"kttruong forces", "kttruong", "/api/v1/master-data/item/VT-1/deactivate-force", "POST", true},
		{"kttruong seeds", "kttruong", "/api/v1/master-data/accounts/seed", "POST", true},
		{"kttruong regimes", "kttruong", "/api/v1/master-data/settings/regime", "POST", true},

		// giam_doc: read-only
		{"giamdoc reads", "giamdoc", "/api/v1/master-data/customer", "GET", true},
		{"giamdoc cannot write", "giamdoc", "/api/v1/master-data/customer", "POST", false},
		{"giamdoc cannot delete", "giamdoc", "/api/v1/master-data/customer/KH-1", "DELETE", false},

		// kiem_toan: read-only
		{"kiemtoan reads", "kiemtoan", "/api/v1/master-data/account", "GET", true},
		{"kiemtoan cannot write", "kiemtoan", "/api/v1/master-data/account", "POST", false},

		// Web UI access
		{"danh_muc web page", "danhmuc", "/web/master-data", "GET", true},
		{"giamdoc web page", "giamdoc", "/web/master-data", "GET", true},

		// Anonymous denied
		{"anon denied", "anonymous", "/api/v1/master-data/customer", "GET", false},
		{"anon cannot create", "anonymous", "/api/v1/master-data/customer", "POST", false},
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
