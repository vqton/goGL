package cash

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/cash"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) cash.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) upsertDoc(ctx context.Context, table, id string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO `+table+` (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`, id, string(data))
	return err
}

func (r *sqliteRepository) getDoc(ctx context.Context, table, id string, out any) error {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM `+table+` WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), out)
}

func (r *sqliteRepository) CreateFund(ctx context.Context, f *cash.Fund) error {
	return r.upsertDoc(ctx, "cash_funds", f.ID, f)
}

func (r *sqliteRepository) GetFund(ctx context.Context, id string) (*cash.Fund, error) {
	var f cash.Fund
	if err := r.getDoc(ctx, "cash_funds", id, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *sqliteRepository) ListFunds(ctx context.Context) ([]*cash.Fund, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM cash_funds ORDER BY json_extract(data, '$.name')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*cash.Fund
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var f cash.Fund
		if err := json.Unmarshal([]byte(data), &f); err != nil {
			return nil, err
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateVoucher(ctx context.Context, v *cash.Voucher) error {
	return r.upsertDoc(ctx, "cash_vouchers", v.ID, v)
}

func (r *sqliteRepository) UpdateVoucher(ctx context.Context, v *cash.Voucher) error {
	return r.upsertDoc(ctx, "cash_vouchers", v.ID, v)
}

func (r *sqliteRepository) GetVoucher(ctx context.Context, id string) (*cash.Voucher, error) {
	var v cash.Voucher
	if err := r.getDoc(ctx, "cash_vouchers", id, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *sqliteRepository) ListVouchers(ctx context.Context, f cash.VoucherFilter) ([]*cash.Voucher, error) {
	q := `SELECT data FROM cash_vouchers WHERE 1=1`
	var args []any
	if f.FundID != "" {
		q += ` AND json_extract(data, '$.fund_id') = ?`
		args = append(args, f.FundID)
	}
	if f.State != "" {
		q += ` AND json_extract(data, '$.state') = ?`
		args = append(args, string(f.State))
	}
	if f.Type != "" {
		q += ` AND json_extract(data, '$.type') = ?`
		args = append(args, string(f.Type))
	}
	if f.From != "" {
		q += ` AND json_extract(data, '$.ref_date') >= ?`
		args = append(args, f.From)
	}
	if f.To != "" {
		q += ` AND json_extract(data, '$.ref_date') <= ?`
		args = append(args, f.To)
	}
	q += ` ORDER BY json_extract(data, '$.ref_date'), json_extract(data, '$.ref_no')`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*cash.Voucher
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var v cash.Voucher
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextRefNo(ctx context.Context, fundID, period string, typ cash.VoucherType) (string, error) {
	// Atomic counter over the JSON document. INSERT OR IGNORE seeds the
	// counter at 0, then UPDATE bumps it and RETURNING hands back the new
	// value — so the first ref is 000001 and the two statements give the
	// same answer on first and subsequent use (a single INSERT..ON CONFLICT
	// ..RETURNING would return 0 on the insert path). Deterministic id =
	// cash.RowID keeps the seed idempotent across re-creates.
	id := cash.RowID("cash_sequences", fundID, period, string(typ))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO cash_sequences (id, data) VALUES (?, json_object('fund_id', ?, 'period', ?, 'typ', ?, 'seq', 0))`,
		id, fundID, period, string(typ)); err != nil {
		return "", err
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE cash_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%06d", typ.RefPrefix(), period, seq), nil
}

func (r *sqliteRepository) AppendCashBookEntry(ctx context.Context, e *cash.CashBookEntry) error {
	return r.upsertDoc(ctx, "cash_book", e.ID, e)
}

// DeleteCashBookEntry removes a cash-book row. Used to compensate a failed
// post so the book and voucher states stay consistent.
func (r *sqliteRepository) DeleteCashBookEntry(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cash_book WHERE id = ?`, id)
	return err
}

// UpdateCashBookEntry rewrites a cash-book row, used to re-chain running
// balances after a back-dated post.
func (r *sqliteRepository) UpdateCashBookEntry(ctx context.Context, e *cash.CashBookEntry) error {
	return r.upsertDoc(ctx, "cash_book", e.ID, e)
}

func (r *sqliteRepository) ListCashBook(ctx context.Context, fundID, from, to string) ([]*cash.CashBookEntry, error) {
	q := `SELECT data FROM cash_book WHERE json_extract(data, '$.fund_id') = ?`
	args := []any{fundID}
	if from != "" {
		q += ` AND json_extract(data, '$.entry_date') >= ?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND json_extract(data, '$.entry_date') <= ?`
		args = append(args, to)
	}
	q += ` ORDER BY json_extract(data, '$.entry_date'), rowid`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*cash.CashBookEntry
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var e cash.CashBookEntry
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateCashCount(ctx context.Context, c *cash.CashCount) error {
	return r.upsertDoc(ctx, "cash_counts", c.ID, c)
}

func (r *sqliteRepository) GetCashCount(ctx context.Context, id string) (*cash.CashCount, error) {
	var c cash.CashCount
	if err := r.getDoc(ctx, "cash_counts", id, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *sqliteRepository) ListCashCounts(ctx context.Context, fundID string) ([]*cash.CashCount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM cash_counts WHERE json_extract(data, '$.fund_id') = ?
		 ORDER BY json_extract(data, '$.count_date') DESC`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*cash.CashCount
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c cash.CashCount
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateReconciliation(ctx context.Context, rec *cash.Reconciliation) error {
	return r.upsertDoc(ctx, "cash_reconciliations", rec.ID, rec)
}

func (r *sqliteRepository) GetReconciliation(ctx context.Context, id string) (*cash.Reconciliation, error) {
	var rec cash.Reconciliation
	if err := r.getDoc(ctx, "cash_reconciliations", id, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *sqliteRepository) ListReconciliations(ctx context.Context, fundID string) ([]*cash.Reconciliation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM cash_reconciliations WHERE json_extract(data, '$.fund_id') = ?
		 ORDER BY json_extract(data, '$.created_at') DESC`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*cash.Reconciliation
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var rec cash.Reconciliation
		if err := json.Unmarshal([]byte(data), &rec); err != nil {
			return nil, err
		}
		out = append(out, &rec)
	}
	return out, rows.Err()
}
