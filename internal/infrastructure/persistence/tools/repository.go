package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/tools"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) tools.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, c *tools.ToolCard) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("tools: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO tools_cards (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		c.ID, string(data))
	if err != nil {
		return fmt.Errorf("tools: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*tools.ToolCard, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tools_cards WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, tools.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("tools: find: %w", err)
	}
	var c tools.ToolCard
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("tools: decode: %w", err)
	}
	return &c, nil
}

func (r *sqliteRepository) Update(ctx context.Context, c *tools.ToolCard) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("tools: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE tools_cards SET data = ? WHERE id = ?`,
		string(data), c.ID)
	if err != nil {
		return fmt.Errorf("tools: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tools.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tools_cards WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("tools: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tools.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, category string, state tools.CardState, limit, offset int) ([]*tools.ToolCard, error) {
	q := `SELECT data FROM tools_cards WHERE 1=1`
	var args []any
	if category != "" {
		q += ` AND json_extract(data, '$.category') = ?`
		args = append(args, category)
	}
	if state != "" {
		q += ` AND json_extract(data, '$.state') = ?`
		args = append(args, state)
	}
	q += ` ORDER BY json_extract(data, '$.code') LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tools: list: %w", err)
	}
	defer rows.Close()

	var out []*tools.ToolCard
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c tools.ToolCard
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context) (int64, error) {
	id := "tools_seq"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("tools: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO tools_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("tools: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE tools_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("tools: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("tools: seq commit: %w", err)
	}
	return seq, nil
}

// --- Transaction methods ---

func (r *sqliteRepository) CreateTransaction(ctx context.Context, tx *tools.ToolTransaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("tools: tx marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO tools_transactions (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		tx.ID, string(data))
	if err != nil {
		return fmt.Errorf("tools: tx create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindTransactionByID(ctx context.Context, id string) (*tools.ToolTransaction, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tools_transactions WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, tools.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("tools: tx find: %w", err)
	}
	var tx tools.ToolTransaction
	if err := json.Unmarshal([]byte(data), &tx); err != nil {
		return nil, fmt.Errorf("tools: tx decode: %w", err)
	}
	return &tx, nil
}

func (r *sqliteRepository) ListTransactions(ctx context.Context, toolCardID string, txType tools.TransactionType, limit, offset int) ([]*tools.ToolTransaction, error) {
	q := `SELECT data FROM tools_transactions WHERE 1=1`
	var args []any
	if toolCardID != "" {
		q += ` AND json_extract(data, '$.tool_card_id') = ?`
		args = append(args, toolCardID)
	}
	if txType != "" {
		q += ` AND json_extract(data, '$.transaction_type') = ?`
		args = append(args, txType)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tools: tx list: %w", err)
	}
	defer rows.Close()

	var out []*tools.ToolTransaction
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var tx tools.ToolTransaction
		if err := json.Unmarshal([]byte(data), &tx); err != nil {
			return nil, err
		}
		out = append(out, &tx)
	}
	return out, rows.Err()
}

// --- Stock methods ---

func (r *sqliteRepository) GetStock(ctx context.Context, toolCardID string) (int, error) {
	var quantity int
	err := r.db.QueryRowContext(ctx,
		`SELECT json_extract(data, '$.quantity') FROM tools_cards WHERE id = ?`,
		toolCardID).Scan(&quantity)
	if err == sql.ErrNoRows {
		return 0, tools.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("tools: stock find: %w", err)
	}
	return quantity, nil
}

func (r *sqliteRepository) AdjustStock(ctx context.Context, toolCardID string, quantity int) error {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tools_cards WHERE id = ?`, toolCardID).Scan(&data)
	if err == sql.ErrNoRows {
		return tools.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("tools: adjust find: %w", err)
	}
	var c tools.ToolCard
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return fmt.Errorf("tools: adjust decode: %w", err)
	}
	c.Quantity = quantity
	newData, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("tools: adjust marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE tools_cards SET data = ? WHERE id = ?`,
		string(newData), toolCardID)
	if err != nil {
		return fmt.Errorf("tools: adjust update: %w", err)
	}
	return nil
}

func (r *sqliteRepository) DecrementStock(ctx context.Context, toolCardID string, quantity int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tools_cards
		 SET data = json_set(data, '$.quantity', json_extract(data, '$.quantity') - ?)
		 WHERE id = ? AND json_extract(data, '$.quantity') >= ?`,
		quantity, toolCardID, quantity)
	if err != nil {
		return fmt.Errorf("tools: decrement: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Check if card exists
		var exists int
		if err := r.db.QueryRowContext(ctx,
			`SELECT 1 FROM tools_cards WHERE id = ?`, toolCardID).Scan(&exists); err != nil {
			return tools.ErrNotFound
		}
		return tools.ErrInsufficientStock
	}
	return nil
}

func (r *sqliteRepository) IncrementStock(ctx context.Context, toolCardID string, quantity int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tools_cards
		 SET data = json_set(data, '$.quantity', json_extract(data, '$.quantity') + ?)
		 WHERE id = ?`,
		quantity, toolCardID)
	if err != nil {
		return fmt.Errorf("tools: increment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tools.ErrNotFound
	}
	return nil
}
