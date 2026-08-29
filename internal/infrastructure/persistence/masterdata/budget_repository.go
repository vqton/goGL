package masterdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"goGL/internal/domain/masterdata"
)

// BudgetTableName is the table for budget records.
const BudgetTableName = "md_budgets"

func (r *sqliteRepository) UpsertBudget(ctx context.Context, b *masterdata.BudgetRecord) error {
	if b.ID == "" {
		b.ID = fmt.Sprintf("%s_%d", b.DepartmentCode, b.FiscalYear)
	}
	doc, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("masterdata: marshal budget: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`, BudgetTableName),
		b.ID, string(doc)); err != nil {
		return fmt.Errorf("masterdata: upsert budget %s: %w", b.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetBudget(ctx context.Context, departmentCode string, fiscalYear int) (*masterdata.BudgetRecord, error) {
	id := fmt.Sprintf("%s_%d", departmentCode, fiscalYear)
	var data string
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT data FROM %s WHERE id = ?`, BudgetTableName),
		id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, masterdata.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("masterdata: get budget %s: %w", id, err)
	}
	var rec masterdata.BudgetRecord
	if err := json.Unmarshal([]byte(data), &rec); err != nil {
		return nil, fmt.Errorf("masterdata: decode budget: %w", err)
	}
	return &rec, nil
}

func (r *sqliteRepository) ListBudgets(ctx context.Context, fiscalYear int) ([]*masterdata.BudgetRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT data FROM %s ORDER BY id`, BudgetTableName))
	if err != nil {
		return nil, fmt.Errorf("masterdata: list budgets: %w", err)
	}
	defer rows.Close()

	var out []*masterdata.BudgetRecord
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("masterdata: scan budget: %w", err)
		}
		var rec masterdata.BudgetRecord
		if err := json.Unmarshal([]byte(data), &rec); err != nil {
			return nil, fmt.Errorf("masterdata: decode budget: %w", err)
		}
		if fiscalYear == 0 || rec.FiscalYear == fiscalYear {
			out = append(out, &rec)
		}
	}
	return out, rows.Err()
}

func (r *sqliteRepository) DeleteBudget(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, BudgetTableName), id)
	if err != nil {
		return fmt.Errorf("masterdata: delete budget %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return masterdata.ErrNotFound
	}
	return nil
}
