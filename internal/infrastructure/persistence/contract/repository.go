package contract

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/contract"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) contract.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, c *contract.Contract) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("contract: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO contracts (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		c.ID, string(data))
	if err != nil {
		return fmt.Errorf("contract: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*contract.Contract, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM contracts WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, contract.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("contract: find: %w", err)
	}
	var c contract.Contract
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("contract: decode: %w", err)
	}
	return &c, nil
}

func (r *sqliteRepository) Update(ctx context.Context, c *contract.Contract) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("contract: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE contracts SET data = ? WHERE id = ?`,
		string(data), c.ID)
	if err != nil {
		return fmt.Errorf("contract: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contract.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM contracts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("contract: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contract.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, ctype contract.ContractType, state contract.ContractState) ([]*contract.Contract, error) {
	q := `SELECT data FROM contracts WHERE 1=1`
	var args []any
	if ctype != "" {
		q += ` AND json_extract(data, '$.type') = ?`
		args = append(args, ctype)
	}
	if state != "" {
		q += ` AND json_extract(data, '$.state') = ?`
		args = append(args, state)
	}
	q += ` ORDER BY json_extract(data, '$.code')`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("contract: list: %w", err)
	}
	defer rows.Close()

	var out []*contract.Contract
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c contract.Contract
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context) (int64, error) {
	id := "contract_seq"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("contract: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO contract_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("contract: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE contract_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("contract: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("contract: seq commit: %w", err)
	}
	return seq, nil
}

func (r *sqliteRepository) CreateLoan(ctx context.Context, l *contract.LoanAgreement) error {
	data, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("contract: loan marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO loans (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		l.ID, string(data))
	if err != nil {
		return fmt.Errorf("contract: loan create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindLoanByID(ctx context.Context, id string) (*contract.LoanAgreement, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM loans WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, contract.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("contract: loan find: %w", err)
	}
	var l contract.LoanAgreement
	if err := json.Unmarshal([]byte(data), &l); err != nil {
		return nil, fmt.Errorf("contract: loan decode: %w", err)
	}
	return &l, nil
}

func (r *sqliteRepository) UpdateLoan(ctx context.Context, l *contract.LoanAgreement) error {
	data, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("contract: loan marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE loans SET data = ? WHERE id = ?`,
		string(data), l.ID)
	if err != nil {
		return fmt.Errorf("contract: loan update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contract.ErrNotFound
	}
	return nil
}
