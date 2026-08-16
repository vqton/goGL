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
		{"role:cashier", "/api/v1/cash/book", "GET"},
		{"role:cashier", "/api/v1/cash/close-day", "POST"},
		{"role:cash_accountant", "/api/v1/cash/vouchers", "*"},
		{"role:cash_accountant", "/api/v1/cash/vouchers/*", "*"},
		{"role:cash_accountant", "/api/v1/cash/reconcile", "POST"},
		{"role:chief_accountant", "/api/v1/cash/*", "*"},
		{"role:director", "/api/v1/cash/*/approve", "POST"},

		// Ledger (kế toán tổng hợp). ke_toan_tong_hop maintains the GL (accounts,
		// manual entries, templates, retries); ke_toan_truong approves posts,
		// reversals and period control; giam_doc and kiem_toan are read-only.
		{"role:ke_toan_tong_hop", "/api/v1/ledger/*", "GET"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/accounts", "POST"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/accounts/*", "PATCH"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/entries", "POST"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/entries/*", "DELETE"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/templates", "POST"},
		{"role:ke_toan_tong_hop", "/api/v1/ledger/postings/*", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/*", "GET"},
		{"role:ke_toan_truong", "/api/v1/ledger/entries/*/post", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/entries/*/reverse", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/periods/*/open", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/periods/*/close", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/periods/*/reopen", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/periods/*/close/run", "POST"},
		{"role:ke_toan_truong", "/api/v1/ledger/opening-balances", "POST"},
		{"role:giam_doc", "/api/v1/ledger/*", "GET"},
		{"role:kiem_toan", "/api/v1/ledger/*", "GET"},
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
		if _, err := e.AddPolicy(rule); err != nil {
			return err
		}
	}
	for _, rule := range DefaultPolicies.Grouping {
		if _, err := e.AddGroupingPolicy(rule); err != nil {
			return err
		}
	}
	return nil
}
