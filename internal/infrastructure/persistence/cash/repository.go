package cash

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

func rowID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
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
	id := rowID("cash_sequences", fundID, period, string(typ))
	data, _ := json.Marshal(map[string]string{
		"fund_id": fundID, "period": period, "typ": string(typ),
	})
	q := `INSERT INTO cash_sequences (id, data, fund_id, period, typ, seq)
	      VALUES (?, ?, ?, ?, ?, 1)
	      ON CONFLICT(fund_id, period, typ) DO UPDATE SET seq = seq + 1
	      RETURNING seq`
	var seq int64
	if err := r.db.QueryRowContext(ctx, q, id, string(data), fundID, period, string(typ)).Scan(&seq); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%06d", typ.RefPrefix(), period, seq), nil
}

func (r *sqliteRepository) AppendCashBookEntry(ctx context.Context, e *cash.CashBookEntry) error {
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
