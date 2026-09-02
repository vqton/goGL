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
	"opening_balance_imports",
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
	"sessions",
	"backup_artifacts",
	"restore_plans",
	"task_runs",
	"password_history",
	"documents",
	"document_sequences",
	"budget_plans",
	"budget_sequences",
	"tax_declarations",
	"tax_sequences",
	"bank_transactions",
	"bank_sequences",
	"tools_cards",
	"tools_transactions",
	"tools_sequences",
	"contracts",
	"contract_sequences",
	"loans",
	"fixedasset_sequences",
	"depreciation_entries",
	"sales_orders",
	"sales_returns",
	"sales_sequences",
	"suppliers",
	"supplier_sequences",
	"purchase_orders",
	"purchase_order_sequences",
	"goods_receipts",
	"goods_receipt_sequences",
	"purchase_invoices",
	"purchase_invoice_sequences",
	"purchase_payments",
	"purchase_payment_sequences",
	"purchase_sequences",
	"inventory_items",
	"inventory_warehouses",
	"inventory_stock_cards",
	"inventory_stock_movements",
	"inventory_stock_valuation_layers",
	"inventory_physical_counts",
	"inventory_sequences",
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
