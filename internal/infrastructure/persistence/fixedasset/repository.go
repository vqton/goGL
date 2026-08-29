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
