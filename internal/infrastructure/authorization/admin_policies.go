package authorization

// Admin/configuration RBAC policies for the identity, user/role, options,
// system, backup, and task modules.
//
// Roles:
//   - role:admin               — everything (already covered by `* *`); listed
//     here only for documentation.
//   - role:kiem_toan (auditor) — read-only view of users, roles, options,
//     system info, backups, and audit logs; cannot mutate anything.
//   - role:giam_doc (director) — read-only view of the same surfaces.
//   - role:ke_toan_tong_hop    — read-only on system/backup status; backup
//     operations (run/restore) are admin-only.
func init() {
	viewRoles := []string{"role:kiem_toan", "role:giam_doc"}
	viewObjects := []string{
		"/api/v1/users", "/api/v1/users/*",
		"/api/v1/roles", "/api/v1/roles/*",
		"/api/v1/options", "/api/v1/options/*",
		"/api/v1/system/*",
		"/api/v1/backup/*",
		"/api/v1/tasks/*",
		"/api/v1/audit/*",
	}
	for _, role := range viewRoles {
		for _, obj := range viewObjects {
			DefaultPolicies.Policies = append(DefaultPolicies.Policies, []string{role, obj, "GET"})
		}
	}

	// kiem_toan may also list audit logs through the audit module's own routes.
	DefaultPolicies.Policies = append(DefaultPolicies.Policies,
		[]string{"role:kiem_toan", "/api/v1/audit/logs", "GET"},
		[]string{"role:kiem_toan", "/api/v1/audit/logs/*", "GET"},
	)
}
