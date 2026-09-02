package purchase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) purchase.Repository {
	return &sqliteRepository{db: db}
}

// --- Sequence helper ---

func (r *sqliteRepository) nextSeq(ctx context.Context, seqName string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("purchase: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO purchase_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, seqName); err != nil {
		return 0, fmt.Errorf("purchase: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE purchase_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, seqName).Scan(&seq); err != nil {
		return 0, fmt.Errorf("purchase: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("purchase: seq commit: %w", err)
	}
	return seq, nil
}

// --- Supplier CRUD ---

func (r *sqliteRepository) CreateSupplier(ctx context.Context, s *purchase.Supplier) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("purchase: supplier marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO suppliers (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		s.ID, string(data))
	if err != nil {
		return fmt.Errorf("purchase: supplier create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindSupplierByID(ctx context.Context, id string) (*purchase.Supplier, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM suppliers WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, purchase.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase: supplier find: %w", err)
	}
	var s purchase.Supplier
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, fmt.Errorf("purchase: supplier decode: %w", err)
	}
	return &s, nil
}

func (r *sqliteRepository) UpdateSupplier(ctx context.Context, s *purchase.Supplier) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("purchase: supplier marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE suppliers SET data = ? WHERE id = ?`,
		string(data), s.ID)
	if err != nil {
		return fmt.Errorf("purchase: supplier update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteSupplier(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM suppliers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("purchase: supplier delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListSuppliers(ctx context.Context, name string, status purchase.SupplierStatus, limit, offset int) ([]*purchase.Supplier, error) {
	q := `SELECT data FROM suppliers WHERE 1=1`
	var args []any
	if name != "" {
		q += ` AND json_extract(data, '$.name') LIKE ?`
		args = append(args, "%"+name+"%")
	}
	if status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, status)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase: supplier list: %w", err)
	}
	defer rows.Close()

	var out []*purchase.Supplier
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var s purchase.Supplier
		if err := json.Unmarshal([]byte(data), &s); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextSupplierNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "supplier_seq")
}

// --- Purchase Order CRUD ---

func (r *sqliteRepository) CreateOrder(ctx context.Context, o *purchase.PurchaseOrder) error {
	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("purchase: order marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO purchase_orders (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		o.ID, string(data))
	if err != nil {
		return fmt.Errorf("purchase: order create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindOrderByID(ctx context.Context, id string) (*purchase.PurchaseOrder, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM purchase_orders WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, purchase.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase: order find: %w", err)
	}
	var o purchase.PurchaseOrder
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return nil, fmt.Errorf("purchase: order decode: %w", err)
	}
	return &o, nil
}

func (r *sqliteRepository) UpdateOrder(ctx context.Context, o *purchase.PurchaseOrder) error {
	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("purchase: order marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE purchase_orders SET data = ? WHERE id = ?`,
		string(data), o.ID)
	if err != nil {
		return fmt.Errorf("purchase: order update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteOrder(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM purchase_orders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("purchase: order delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListOrders(ctx context.Context, supplierCode string, status purchase.OrderStatus, limit, offset int) ([]*purchase.PurchaseOrder, error) {
	q := `SELECT data FROM purchase_orders WHERE 1=1`
	var args []any
	if supplierCode != "" {
		q += ` AND json_extract(data, '$.supplier_code') = ?`
		args = append(args, supplierCode)
	}
	if status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, status)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase: order list: %w", err)
	}
	defer rows.Close()

	var out []*purchase.PurchaseOrder
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var o purchase.PurchaseOrder
		if err := json.Unmarshal([]byte(data), &o); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextOrderNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "purchase_order_seq")
}

// --- Goods Receipt CRUD ---

func (r *sqliteRepository) CreateReceipt(ctx context.Context, g *purchase.GoodsReceipt) error {
	data, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("purchase: receipt marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO goods_receipts (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		g.ID, string(data))
	if err != nil {
		return fmt.Errorf("purchase: receipt create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindReceiptByID(ctx context.Context, id string) (*purchase.GoodsReceipt, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM goods_receipts WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, purchase.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase: receipt find: %w", err)
	}
	var g purchase.GoodsReceipt
	if err := json.Unmarshal([]byte(data), &g); err != nil {
		return nil, fmt.Errorf("purchase: receipt decode: %w", err)
	}
	return &g, nil
}

func (r *sqliteRepository) UpdateReceipt(ctx context.Context, g *purchase.GoodsReceipt) error {
	data, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("purchase: receipt marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE goods_receipts SET data = ? WHERE id = ?`,
		string(data), g.ID)
	if err != nil {
		return fmt.Errorf("purchase: receipt update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListReceipts(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.GoodsReceipt, error) {
	q := `SELECT data FROM goods_receipts WHERE 1=1`
	var args []any
	if supplierCode != "" {
		q += ` AND json_extract(data, '$.supplier_code') = ?`
		args = append(args, supplierCode)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase: receipt list: %w", err)
	}
	defer rows.Close()

	var out []*purchase.GoodsReceipt
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var g purchase.GoodsReceipt
		if err := json.Unmarshal([]byte(data), &g); err != nil {
			return nil, err
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) HasReceiptsForOrder(ctx context.Context, poID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM goods_receipts
		 WHERE json_extract(data, '$.po_id') = ?`, poID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("purchase: check receipts for order: %w", err)
	}
	return count > 0, nil
}

func (r *sqliteRepository) NextReceiptNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "goods_receipt_seq")
}

// --- Purchase Invoice CRUD ---

func (r *sqliteRepository) CreateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("purchase: invoice marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO purchase_invoices (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		inv.ID, string(data))
	if err != nil {
		return fmt.Errorf("purchase: invoice create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindInvoiceByID(ctx context.Context, id string) (*purchase.PurchaseInvoice, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM purchase_invoices WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, purchase.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase: invoice find: %w", err)
	}
	var inv purchase.PurchaseInvoice
	if err := json.Unmarshal([]byte(data), &inv); err != nil {
		return nil, fmt.Errorf("purchase: invoice decode: %w", err)
	}
	return &inv, nil
}

func (r *sqliteRepository) UpdateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("purchase: invoice marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE purchase_invoices SET data = ? WHERE id = ?`,
		string(data), inv.ID)
	if err != nil {
		return fmt.Errorf("purchase: invoice update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteInvoice(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM purchase_invoices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("purchase: invoice delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListInvoices(ctx context.Context, supplierCode string, status purchase.InvoiceStatus, limit, offset int) ([]*purchase.PurchaseInvoice, error) {
	q := `SELECT data FROM purchase_invoices WHERE 1=1`
	var args []any
	if supplierCode != "" {
		q += ` AND json_extract(data, '$.supplier_code') = ?`
		args = append(args, supplierCode)
	}
	if status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, status)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase: invoice list: %w", err)
	}
	defer rows.Close()

	var out []*purchase.PurchaseInvoice
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var inv purchase.PurchaseInvoice
		if err := json.Unmarshal([]byte(data), &inv); err != nil {
			return nil, err
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextInvoiceNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "purchase_invoice_seq")
}

// --- Payment CRUD ---

func (r *sqliteRepository) CreatePayment(ctx context.Context, p *purchase.Payment) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("purchase: payment marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO purchase_payments (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		p.ID, string(data))
	if err != nil {
		return fmt.Errorf("purchase: payment create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindPaymentByID(ctx context.Context, id string) (*purchase.Payment, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM purchase_payments WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, purchase.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase: payment find: %w", err)
	}
	var p purchase.Payment
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return nil, fmt.Errorf("purchase: payment decode: %w", err)
	}
	return &p, nil
}

func (r *sqliteRepository) UpdatePayment(ctx context.Context, p *purchase.Payment) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("purchase: payment marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE purchase_payments SET data = ? WHERE id = ?`,
		string(data), p.ID)
	if err != nil {
		return fmt.Errorf("purchase: payment update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return purchase.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListPayments(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.Payment, error) {
	q := `SELECT data FROM purchase_payments WHERE 1=1`
	var args []any
	if supplierCode != "" {
		q += ` AND json_extract(data, '$.supplier_code') = ?`
		args = append(args, supplierCode)
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase: payment list: %w", err)
	}
	defer rows.Close()

	var out []*purchase.Payment
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var p purchase.Payment
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextPaymentNo(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "purchase_payment_seq")
}

// --- Supplier balance ---

func (r *sqliteRepository) GetSupplierBalance(ctx context.Context, supplierCode string) (core.Money, error) {
	var invoiceTotal, paymentTotal int64

	// Sum of posted invoices (excludes draft and reconciled/paid)
	invQ := `SELECT COALESCE(SUM(json_extract(data, '$.total_amount.amount_minor')), 0)
	       FROM purchase_invoices
	       WHERE json_extract(data, '$.supplier_code') = ?
	         AND json_extract(data, '$.status') NOT IN ('draft', 'reconciled')`
	if err := r.db.QueryRowContext(ctx, invQ, supplierCode).Scan(&invoiceTotal); err != nil {
		return core.Money{}, fmt.Errorf("purchase: supplier balance (invoices): %w", err)
	}

	// Sum of approved/processed payments
	payQ := `SELECT COALESCE(SUM(json_extract(data, '$.amount.amount_minor')), 0)
	       FROM purchase_payments
	       WHERE json_extract(data, '$.supplier_code') = ?
	         AND json_extract(data, '$.status') IN ('approved', 'processed')`
	if err := r.db.QueryRowContext(ctx, payQ, supplierCode).Scan(&paymentTotal); err != nil {
		return core.Money{}, fmt.Errorf("purchase: supplier balance (payments): %w", err)
	}

	outstanding := invoiceTotal - paymentTotal
	if outstanding < 0 {
		outstanding = 0
	}
	return core.Money{AmountMinor: outstanding, Currency: "VND"}, nil
}
