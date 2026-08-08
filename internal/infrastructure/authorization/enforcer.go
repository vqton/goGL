package authorization

import (
	"database/sql"

	"github.com/casbin/casbin/v3"
)

// DefaultPolicies bootstraps a fresh deployment: a built-in admin role with
// full access, and the default "admin" user made a member of it. Swap the
// grouping rule once real user provisioning exists.
var DefaultPolicies = struct {
	Policies [][]string // p rules (sub, obj, act)
	Grouping [][]string // g rules (user, role)
}{
	Policies: [][]string{
		{"role:admin", "*", "*"},
		{"role:cashier", "/api/v1/cash/vouchers/*/post", "POST"},
		{"role:cashier", "/api/v1/cash/books*", "GET"},
		{"role:cashier", "/api/v1/cash/close-day", "POST"},
		{"role:cash_accountant", "/api/v1/cash/vouchers", "*"},
		{"role:cash_accountant", "/api/v1/cash/vouchers/*", "*"},
		{"role:cash_accountant", "/api/v1/cash/reconcile", "POST"},
		{"role:chief_accountant", "/api/v1/cash/*", "*"},
		{"role:director", "/api/v1/cash/*/approve", "POST"},
	},
	Grouping: [][]string{
		{"admin", "role:admin"},
	},
}

// NewEnforcer builds a casbin enforcer over the given SQLite connection. The
// embedded RBAC model is loaded and all policy rules are read from storage.
func NewEnforcer(db *sql.DB) (*casbin.Enforcer, error) {
	m, err := RBACModel()
	if err != nil {
		return nil, err
	}
	return casbin.NewEnforcer(m, NewSqliteAdapter(db))
}

// SeedDefaultPolicies inserts the bootstrap policies unless the policy store
// already contains any rules. Safe to call on every startup: it is a no-op
// once any rule exists.
func SeedDefaultPolicies(e *casbin.Enforcer) error {
	policies, err := e.GetPolicy()
	if err != nil {
		return err
	}
	grouping, err := e.GetGroupingPolicy()
	if err != nil {
		return err
	}
	if len(policies) > 0 || len(grouping) > 0 {
		return nil
	}

	for _, rule := range DefaultPolicies.Policies {
		if ok, err := e.AddPolicy(rule); err != nil {
			return err
		} else if !ok {
			return nil
		}
	}
	for _, rule := range DefaultPolicies.Grouping {
		if ok, err := e.AddGroupingPolicy(rule); err != nil {
			return err
		} else if !ok {
			return nil
		}
	}
	return nil
}
