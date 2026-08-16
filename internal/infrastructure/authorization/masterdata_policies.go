package authorization

// Master-data RBAC policies. Appended at init so the module can be added and
// removed without touching the core enforcer file.
//
// Roles:
//   - danh_muc, ke_toan_tong_hop  — full catalog maintenance (create/update,
//     deactivate/activate, delete, import)
//   - ke_toan_truong              — additionally: merge, forced deactivate,
//     chart-of-accounts seed, regime switch
//   - giam_doc, kiem_toan         — read-only
//
// Objects use keyMatch2, so "*" matches a single path segment; route depth is
// mirrored explicitly (e.g. "/api/v1/master-data/*/*").
func init() {
	readRoles := []string{
		"role:danh_muc",
		"role:ke_toan_tong_hop",
		"role:ke_toan_truong",
		"role:giam_doc",
		"role:kiem_toan",
	}
	for _, role := range readRoles {
		for _, obj := range []string{
			"/api/v1/master-data",
			"/api/v1/master-data/*",
			"/api/v1/master-data/*/*",
		} {
			DefaultPolicies.Policies = append(DefaultPolicies.Policies, []string{role, obj, "GET"})
		}
		// Web UI page (served under the /web group, same handler).
		DefaultPolicies.Policies = append(DefaultPolicies.Policies,
			[]string{role, "/web/master-data", "GET"},
			[]string{role, "/web/master-data/*", "GET"},
		)
	}

	for _, role := range []string{"role:danh_muc", "role:ke_toan_tong_hop"} {
		DefaultPolicies.Policies = append(DefaultPolicies.Policies,
			[]string{role, "/api/v1/master-data/*", "POST"},              // create + import
			[]string{role, "/api/v1/master-data/*/*", "PUT"},             // update
			[]string{role, "/api/v1/master-data/*/*", "DELETE"},          // delete (unreferenced only)
			[]string{role, "/api/v1/master-data/*/*/deactivate", "POST"}, // guarded deactivate
			[]string{role, "/api/v1/master-data/*/*/activate", "POST"},
			[]string{role, "/api/v1/master-data/*/*/references", "POST"}, // reference-count seam
		)
	}

	// Chief accountant only: merge, forced deactivate, seed, regime switch.
	DefaultPolicies.Policies = append(DefaultPolicies.Policies,
		[]string{"role:ke_toan_truong", "/api/v1/master-data/*/merge", "POST"},
		[]string{"role:ke_toan_truong", "/api/v1/master-data/*/*/deactivate-force", "POST"},
		[]string{"role:ke_toan_truong", "/api/v1/master-data/*/*/deactivate", "POST"},
		[]string{"role:ke_toan_truong", "/api/v1/master-data/accounts/seed", "POST"},
		[]string{"role:ke_toan_truong", "/api/v1/master-data/settings/regime", "POST"},
	)
}
