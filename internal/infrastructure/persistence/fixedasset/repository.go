package fixedasset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/fixedasset"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) fixedasset.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, a *fixedasset.FixedAsset) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("fixedasset: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO fixed_assets (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		a.ID, string(data))
	if err != nil {
		return fmt.Errorf("fixedasset: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM fixed_assets WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fixedasset.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fixedasset: find: %w", err)
	}
	var a fixedasset.FixedAsset
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		return nil, fmt.Errorf("fixedasset: decode: %w", err)
	}
	return &a, nil
}

func (r *sqliteRepository) Update(ctx context.Context, a *fixedasset.FixedAsset) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("fixedasset: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE fixed_assets SET data = ? WHERE id = ?`,
		string(data), a.ID)
	if err != nil {
		return fmt.Errorf("fixedasset: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fixedasset.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM fixed_assets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("fixedasset: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fixedasset.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, assetType fixedasset.AssetType, state fixedasset.AssetState, limit, offset int) ([]*fixedasset.FixedAsset, error) {
	q := `SELECT data FROM fixed_assets WHERE 1=1`
	var args []any
	if assetType != "" {
		q += ` AND json_extract(data, '$.asset_type') = ?`
		args = append(args, assetType)
	}
	if state != "" {
		q += ` AND json_extract(data, '$.state') = ?`
		args = append(args, state)
	}
	q += ` ORDER BY json_extract(data, '$.code')`
	q += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: list: %w", err)
	}
	defer rows.Close()

	var out []*fixedasset.FixedAsset
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var a fixedasset.FixedAsset
		if err := json.Unmarshal([]byte(data), &a); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context) (int64, error) {
	id := "fixedasset_seq"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("fixedasset: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO fixedasset_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, id); err != nil {
		return 0, fmt.Errorf("fixedasset: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE fixedasset_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, id).Scan(&seq); err != nil {
		return 0, fmt.Errorf("fixedasset: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("fixedasset: seq commit: %w", err)
	}
	return seq, nil
}

// --- DepreciationEntry repository ---

type sqliteDepreciationRepo struct {
	db *sql.DB
}

func NewSqliteDepreciationRepository(db *sql.DB) fixedasset.DepreciationEntryRepository {
	return &sqliteDepreciationRepo{db: db}
}

func (r *sqliteDepreciationRepo) Create(ctx context.Context, e *fixedasset.DepreciationEntry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("depreciation: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO depreciation_entries (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		e.ID, string(data))
	if err != nil {
		return fmt.Errorf("depreciation: create: %w", err)
	}
	return nil
}

func (r *sqliteDepreciationRepo) FindByID(ctx context.Context, id string) (*fixedasset.DepreciationEntry, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM depreciation_entries WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fixedasset.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("depreciation: find: %w", err)
	}
	var e fixedasset.DepreciationEntry
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, fmt.Errorf("depreciation: decode: %w", err)
	}
	return &e, nil
}

func (r *sqliteDepreciationRepo) Update(ctx context.Context, e *fixedasset.DepreciationEntry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("depreciation: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE depreciation_entries SET data = ? WHERE id = ?`,
		string(data), e.ID)
	if err != nil {
		return fmt.Errorf("depreciation: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fixedasset.ErrNotFound
	}
	return nil
}

func (r *sqliteDepreciationRepo) ListByAsset(ctx context.Context, assetID string) ([]*fixedasset.DepreciationEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM depreciation_entries WHERE json_extract(data, '$.asset_id') = ?
		 ORDER BY json_extract(data, '$.period')`, assetID)
	if err != nil {
		return nil, fmt.Errorf("depreciation: list by asset: %w", err)
	}
	defer rows.Close()
	return scanDepreciationEntries(rows)
}

func (r *sqliteDepreciationRepo) ListByPeriod(ctx context.Context, period string) ([]*fixedasset.DepreciationEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM depreciation_entries WHERE json_extract(data, '$.period') = ?
		 ORDER BY json_extract(data, '$.asset_code')`, period)
	if err != nil {
		return nil, fmt.Errorf("depreciation: list by period: %w", err)
	}
	defer rows.Close()
	return scanDepreciationEntries(rows)
}

func (r *sqliteDepreciationRepo) FindByAssetAndPeriod(ctx context.Context, assetID, period string) (*fixedasset.DepreciationEntry, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM depreciation_entries
		 WHERE json_extract(data, '$.asset_id') = ?
		   AND json_extract(data, '$.period') = ?`, assetID, period).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fixedasset.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("depreciation: find by asset and period: %w", err)
	}
	var e fixedasset.DepreciationEntry
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, fmt.Errorf("depreciation: decode: %w", err)
	}
	return &e, nil
}

func (r *sqliteDepreciationRepo) IsPeriodPosted(ctx context.Context, period string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM depreciation_entries
		 WHERE json_extract(data, '$.period') = ?
		   AND json_extract(data, '$.status') = 'posted'`, period).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("depreciation: check period posted: %w", err)
	}
	return count > 0, nil
}

func scanDepreciationEntries(rows *sql.Rows) ([]*fixedasset.DepreciationEntry, error) {
	var out []*fixedasset.DepreciationEntry
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var e fixedasset.DepreciationEntry
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
