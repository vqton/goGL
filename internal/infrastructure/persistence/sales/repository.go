package sales

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/sales"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) sales.Repository {
	return &sqliteRepository{db: db}
}

// --- Invoice CRUD ---

func (r *sqliteRepository) CreateInvoice(ctx context.Context, inv *sales.SalesInvoice) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("sales: invoice marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO sales_invoices (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		inv.ID, string(data))
	if err != nil {
		return fmt.Errorf("sales: invoice create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindInvoiceByID(ctx context.Context, id string) (*sales.SalesInvoice, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM sales_invoices WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, sales.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sales: invoice find: %w", err)
	}
	var inv sales.SalesInvoice
	if err := json.Unmarshal([]byte(data), &inv); err != nil {
		return nil, fmt.Errorf("sales: invoice decode: %w", err)
	}
	return &inv, nil
}

func (r *sqliteRepository) UpdateInvoice(ctx context.Context, inv *sales.SalesInvoice) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("sales: invoice marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE sales_invoices SET data = ? WHERE id = ?`,
		string(data), inv.ID)
	if err != nil {
		return fmt.Errorf("sales: invoice update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sales.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteInvoice(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sales_invoices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sales: invoice delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sales.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListInvoices(ctx context.Context, customerCode string, status sales.InvoiceStatus, limit, offset int) ([]*sales.SalesInvoice, error) {
	q := `SELECT data FROM sales_invoices WHERE 1=1`
	var args []any
	if customerCode != "" {
		q += ` AND json_extract(data, '$.customer_code') = ?`
		args = append(args, customerCode)
	}
	if status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, status)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sales: invoice list: %w", err)
	}
	defer rows.Close()

	var out []*sales.SalesInvoice
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var inv sales.SalesInvoice
		if err := json.Unmarshal([]byte(data), &inv); err != nil {
			return nil, err
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextInvoiceNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "sales_invoice_seq")
}

// --- Order CRUD ---

func (r *sqliteRepository) CreateOrder(ctx context.Context, o *sales.SalesOrder) error {
	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("sales: order marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO sales_orders (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		o.ID, string(data))
	if err != nil {
		return fmt.Errorf("sales: order create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindOrderByID(ctx context.Context, id string) (*sales.SalesOrder, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM sales_orders WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, sales.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sales: order find: %w", err)
	}
	var o sales.SalesOrder
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return nil, fmt.Errorf("sales: order decode: %w", err)
	}
	return &o, nil
}

func (r *sqliteRepository) UpdateOrder(ctx context.Context, o *sales.SalesOrder) error {
	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("sales: order marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE sales_orders SET data = ? WHERE id = ?`,
		string(data), o.ID)
	if err != nil {
		return fmt.Errorf("sales: order update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sales.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteOrder(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sales_orders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sales: order delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sales.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListOrders(ctx context.Context, customerCode string, status sales.OrderStatus, limit, offset int) ([]*sales.SalesOrder, error) {
	q := `SELECT data FROM sales_orders WHERE 1=1`
	var args []any
	if customerCode != "" {
		q += ` AND json_extract(data, '$.customer_code') = ?`
		args = append(args, customerCode)
	}
	if status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, status)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sales: order list: %w", err)
	}
	defer rows.Close()

	var out []*sales.SalesOrder
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var o sales.SalesOrder
		if err := json.Unmarshal([]byte(data), &o); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextOrderNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "sales_order_seq")
}

// --- Return CRUD ---

func (r *sqliteRepository) CreateReturn(ctx context.Context, ret *sales.SalesReturn) error {
	data, err := json.Marshal(ret)
	if err != nil {
		return fmt.Errorf("sales: return marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO sales_returns (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		ret.ID, string(data))
	if err != nil {
		return fmt.Errorf("sales: return create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindReturnByID(ctx context.Context, id string) (*sales.SalesReturn, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM sales_returns WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, sales.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sales: return find: %w", err)
	}
	var ret sales.SalesReturn
	if err := json.Unmarshal([]byte(data), &ret); err != nil {
		return nil, fmt.Errorf("sales: return decode: %w", err)
	}
	return &ret, nil
}

func (r *sqliteRepository) UpdateReturn(ctx context.Context, ret *sales.SalesReturn) error {
	data, err := json.Marshal(ret)
	if err != nil {
		return fmt.Errorf("sales: return marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE sales_returns SET data = ? WHERE id = ?`,
		string(data), ret.ID)
	if err != nil {
		return fmt.Errorf("sales: return update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sales.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListReturns(ctx context.Context, customerCode string, limit, offset int) ([]*sales.SalesReturn, error) {
	q := `SELECT data FROM sales_returns WHERE 1=1`
	var args []any
	if customerCode != "" {
		q += ` AND json_extract(data, '$.customer_code') = ?`
		args = append(args, customerCode)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sales: return list: %w", err)
	}
	defer rows.Close()

	var out []*sales.SalesReturn
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var ret sales.SalesReturn
		if err := json.Unmarshal([]byte(data), &ret); err != nil {
			return nil, err
		}
		out = append(out, &ret)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextReturnNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "sales_return_seq")
}

// --- Customer balance ---

func (r *sqliteRepository) GetCustomerBalance(ctx context.Context, customerCode string) (core.Money, error) {
	var total int64
	q := `SELECT COALESCE(SUM(json_extract(data, '$.total_amount.amount_minor')), 0)
	       FROM sales_invoices
	       WHERE json_extract(data, '$.customer_code') = ?
	         AND json_extract(data, '$.status') NOT IN ('voided', 'returned')`
	if err := r.db.QueryRowContext(ctx, q, customerCode).Scan(&total); err != nil {
		return core.Money{}, fmt.Errorf("sales: customer balance: %w", err)
	}
	return core.Money{AmountMinor: total, Currency: "VND"}, nil
}

// --- Sequence helper ---

func (r *sqliteRepository) nextSeq(ctx context.Context, seqName string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("sales: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sales_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, seqName); err != nil {
		return 0, fmt.Errorf("sales: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE sales_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, seqName).Scan(&seq); err != nil {
		return 0, fmt.Errorf("sales: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("sales: seq commit: %w", err)
	}
	return seq, nil
}
