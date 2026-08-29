package budget

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/budget"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) budget.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, p *budget.BudgetPlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("budget: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO budget_plans (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		p.ID, string(data))
	if err != nil {
		return fmt.Errorf("budget: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*budget.BudgetPlan, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM budget_plans WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, budget.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("budget: find: %w", err)
	}
	var p budget.BudgetPlan
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return nil, fmt.Errorf("budget: decode: %w", err)
	}
	return &p, nil
}

func (r *sqliteRepository) Update(ctx context.Context, p *budget.BudgetPlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("budget: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE budget_plans SET data = ? WHERE id = ?`,
		string(data), p.ID)
	if err != nil {
		return fmt.Errorf("budget: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return budget.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM budget_plans WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("budget: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return budget.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, fiscalYear int, department string) ([]*budget.BudgetPlan, error) {
	q := `SELECT data FROM budget_plans WHERE 1=1`
	var args []any
	if fiscalYear > 0 {
		q += ` AND json_extract(data, '$.fiscal_year') = ?`
		args = append(args, fiscalYear)
	}
	if department != "" {
		q += ` AND json_extract(data, '$.department') = ?`
		args = append(args, department)
	}
	q += ` ORDER BY json_extract(data, '$.code')`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("budget: list: %w", err)
	}
	defer rows.Close()

	var out []*budget.BudgetPlan
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var p budget.BudgetPlan
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context, fiscalYear int) (int64, error) {
	id := fmt.Sprintf("budget_%d", fiscalYear)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("budget: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO budget_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("budget: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE budget_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("budget: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("budget: seq commit: %w", err)
	}
	return seq, nil
}
