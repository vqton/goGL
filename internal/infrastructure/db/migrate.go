package db

import (
	"context"
	"database/sql"
	"fmt"
)

var tables = []string{
	"cash_funds",
	"cash_vouchers",
	"cash_book",
	"cash_counts",
	"cash_reconciliations",
	"bank_transactions",
	"purchase_invoices",
	"sales_invoices",
	"invoices",
	"stock_movements",
	"tools_cards",
	"fixed_assets",
	"tax_declarations",
	"payslips",
	"cost_sheets",
	"ledger_accounts",
	"ledger_journals",
	"ledger_sequences",
	"ledger_periods",
	"ledger_templates",
	"contracts",
	"loan_agreements",
	"budget_plans",
	"financial_reports",
	"company_profiles",
	"opening_balances",
	"setup_status",
	"catalog_items",
	"users",
	"roles",
	"tenants",
	"system_options",
	"documents",
	"tasks",
	"audit_logs",
	"backup_jobs",
	"casbin_policies",
	"cash_sequences",
}

// validTableName checks that a table name contains only letters, digits, and
// underscores — safe for direct interpolation in DDL.
func validTableName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func Migrate(ctx context.Context, d *sql.DB) error {
	for _, t := range tables {
		if !validTableName(t) {
			return fmt.Errorf("db: invalid table name %q", t)
		}
		q := `CREATE TABLE IF NOT EXISTS ` + t + ` (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL
		)`
		if _, err := d.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
