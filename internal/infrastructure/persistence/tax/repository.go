package tax

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/tax"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) tax.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, d *tax.TaxDeclaration) error {
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("tax: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO tax_declarations (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		d.ID, string(data))
	if err != nil {
		return fmt.Errorf("tax: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*tax.TaxDeclaration, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tax_declarations WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, tax.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("tax: find: %w", err)
	}
	var d tax.TaxDeclaration
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, fmt.Errorf("tax: decode: %w", err)
	}
	return &d, nil
}

func (r *sqliteRepository) Update(ctx context.Context, d *tax.TaxDeclaration) error {
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("tax: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE tax_declarations SET data = ? WHERE id = ?`,
		string(data), d.ID)
	if err != nil {
		return fmt.Errorf("tax: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tax.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tax_declarations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("tax: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tax.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, taxType tax.TaxType, period string) ([]*tax.TaxDeclaration, error) {
	q := `SELECT data FROM tax_declarations WHERE 1=1`
	var args []any
	if taxType != "" {
		q += ` AND json_extract(data, '$.tax_type') = ?`
		args = append(args, taxType)
	}
	if period != "" {
		q += ` AND json_extract(data, '$.period') = ?`
		args = append(args, period)
	}
	q += ` ORDER BY json_extract(data, '$.code')`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tax: list: %w", err)
	}
	defer rows.Close()

	var out []*tax.TaxDeclaration
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var d tax.TaxDeclaration
		if err := json.Unmarshal([]byte(data), &d); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context, taxType tax.TaxType) (int64, error) {
	id := fmt.Sprintf("tax_%s", string(taxType))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("tax: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO tax_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("tax: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE tax_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("tax: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("tax: seq commit: %w", err)
	}
	return seq, nil
}
