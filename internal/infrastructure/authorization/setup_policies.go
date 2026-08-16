package authorization

// Setup RBAC policies (module khởi tạo hệ thống). Appended at init so the
// module can be added and removed without touching the core enforcer file.
//
// Route map (§4 of docs/setup/02-spec.md):
//   - read roles (danh_muc, ke_toan_tong_hop, ke_toan_truong, giam_doc,
//     kiem_toan) — status, profile read, opening-balances list + check
//   - ke_toan_tong_hop — opening-balances entry/delete + CSV import
//   - ke_toan_truong — initialize, profile edit, lock/reopen/activate, and the
//     balance write/import routes (override rights, §8)
//   - role:admin covers everything via the built-in `* *` rule.
//
// Policy objects are exact except where a wildcard is genuinely needed
// (DELETE on /opening-balances/:id). keyMatch2 `*` is multi-segment, so no
// broad `POST /api/v1/setup/*` is granted: lock/reopen must stay chief-only.
func init() {
	readRoles := []string{
		"role:danh_muc",
		"role:ke_toan_tong_hop",
		"role:ke_toan_truong",
		"role:giam_doc",
		"role:kiem_toan",
	}
	for _, role := range readRoles {
		DefaultPolicies.Policies = append(DefaultPolicies.Policies,
			[]string{role, "/api/v1/setup/status", "GET"},
			[]string{role, "/api/v1/setup/profile", "GET"},
			[]string{role, "/api/v1/setup/opening-balances", "GET"},
			[]string{role, "/api/v1/setup/opening-balances/check", "POST"},
		)
	}

	// General accountant: opening-balances data entry, delete, CSV import.
	for _, obj := range []string{
		"/api/v1/setup/opening-balances",
		"/api/v1/setup/opening-balances/import",
	} {
		DefaultPolicies.Policies = append(DefaultPolicies.Policies,
			[]string{"role:ke_toan_tong_hop", obj, "POST"})
	}
	DefaultPolicies.Policies = append(DefaultPolicies.Policies,
		[]string{"role:ke_toan_tong_hop", "/api/v1/setup/opening-balances/*", "DELETE"},
		[]string{"role:ke_toan_tong_hop", "/api/v1/setup/opening-balances/import/*/report", "GET"},
		[]string{"role:ke_toan_tong_hop", "/api/v1/setup/opening-balances/import/*/errors.csv", "GET"},
	)

	// Chief accountant: lifecycle control + override on balance writes/import.
	for _, rule := range [][]string{
		{"role:ke_toan_truong", "/api/v1/setup/initialize", "POST"},
		{"role:ke_toan_truong", "/api/v1/setup/profile", "PUT"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances", "POST"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances/import", "POST"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances/import/*/report", "GET"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances/import/*/errors.csv", "GET"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances/lock", "POST"},
		{"role:ke_toan_truong", "/api/v1/setup/opening-balances/reopen", "POST"},
		{"role:ke_toan_truong", "/api/v1/setup/activate", "POST"},
	} {
		DefaultPolicies.Policies = append(DefaultPolicies.Policies, rule)
	}
}
