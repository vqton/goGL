package bank

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/bank"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) bank.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, t *bank.BankTransaction) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("bank: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO bank_transactions (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		t.ID, string(data))
	if err != nil {
		return fmt.Errorf("bank: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*bank.BankTransaction, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM bank_transactions WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, bank.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bank: find: %w", err)
	}
	var t bank.BankTransaction
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("bank: decode: %w", err)
	}
	return &t, nil
}

func (r *sqliteRepository) Update(ctx context.Context, t *bank.BankTransaction) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("bank: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE bank_transactions SET data = ? WHERE id = ?`,
		string(data), t.ID)
	if err != nil {
		return fmt.Errorf("bank: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return bank.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM bank_transactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("bank: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return bank.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, accountNo string, txType bank.TransactionType) ([]*bank.BankTransaction, error) {
	q := `SELECT data FROM bank_transactions WHERE 1=1`
	var args []any
	if accountNo != "" {
		q += ` AND json_extract(data, '$.account_no') = ?`
		args = append(args, accountNo)
	}
	if txType != "" {
		q += ` AND json_extract(data, '$.type') = ?`
		args = append(args, txType)
	}
	q += ` ORDER BY json_extract(data, '$.code')`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("bank: list: %w", err)
	}
	defer rows.Close()

	var out []*bank.BankTransaction
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var t bank.BankTransaction
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context) (int64, error) {
	id := "bank_seq"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("bank: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO bank_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("bank: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE bank_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("bank: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("bank: seq commit: %w", err)
	}
	return seq, nil
}
