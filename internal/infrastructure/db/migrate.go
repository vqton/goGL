package db

import (
	"context"
	"database/sql"
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
	"ledger_journal_entries",
	"contracts",
	"loan_agreements",
	"budget_plans",
	"financial_reports",
	"company_profiles",
	"opening_balances",
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

func Migrate(ctx context.Context, d *sql.DB) error {
	for _, t := range tables {
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
