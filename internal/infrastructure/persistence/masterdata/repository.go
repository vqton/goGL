package masterdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"goGL/internal/domain/masterdata"
)

// sqliteRepository persists masterdata.Record rows in the md_records table
// using the shared (id, data) JSON-document shape.
type sqliteRepository struct {
	db *sql.DB
}

// NewSqliteRepository builds the repository over a *sql.DB whose schema has
// been migrated by internal/infrastructure/db.Migrate.
func NewSqliteRepository(db *sql.DB) *sqliteRepository {
	return &sqliteRepository{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(sc rowScanner) (*masterdata.Record, error) {
	var data string
	if err := sc.Scan(&data); err != nil {
		return nil, err
	}
	var rec masterdata.Record
	if err := json.Unmarshal([]byte(data), &rec); err != nil {
		return nil, fmt.Errorf("masterdata: decode row: %w", err)
	}
	return &rec, nil
}

func (r *sqliteRepository) Upsert(ctx context.Context, rec *masterdata.Record) error {
	if rec.ID == "" {
		rec.ID = masterdata.RecordID(rec.Kind, rec.Code)
	}
	doc, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("masterdata: marshal: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO md_records (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		rec.ID, string(doc)); err != nil {
		return fmt.Errorf("masterdata: upsert %s/%s: %w", rec.Kind, rec.Code, err)
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (*masterdata.Record, error) {
	rec, err := scanRecord(r.db.QueryRowContext(ctx,
		`SELECT data FROM md_records WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, masterdata.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("masterdata: get %s: %w", id, err)
	}
	return rec, nil
}

func (r *sqliteRepository) GetByCode(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
	rec, err := scanRecord(r.db.QueryRowContext(ctx,
		`SELECT data FROM md_records
		 WHERE json_extract(data, '$.kind') = ? AND json_extract(data, '$.code') = ?`,
		string(kind), code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, masterdata.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("masterdata: get %s/%s: %w", kind, code, err)
	}
	return rec, nil
}

func (r *sqliteRepository) List(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM md_records
		 WHERE json_extract(data, '$.kind') = ?
		 ORDER BY json_extract(data, '$.code')`, string(kind))
	if err != nil {
		return nil, fmt.Errorf("masterdata: list %s: %w", kind, err)
	}
	defer rows.Close()

	var out []*masterdata.Record
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM md_records WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("masterdata: delete %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return masterdata.ErrNotFound
	}
	return nil
}

// NextCode atomically advances the per-kind sequence. md_sequences uses the
// shared (id, data) shape: id = kind, data = {"seq":N}. The read-modify-write
// inside one transaction is safe for the expected single-writer workload.
func (r *sqliteRepository) NextCode(ctx context.Context, kind masterdata.Kind) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("masterdata: seq begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	// Ensure the row exists with seq = 0.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO md_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, string(kind)); err != nil {
		return 0, fmt.Errorf("masterdata: seq insert: %w", err)
	}
	// Bump the JSON counter atomically.
	if _, err := tx.ExecContext(ctx,
		`UPDATE md_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ?`, string(kind)); err != nil {
		return 0, fmt.Errorf("masterdata: seq bump: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`SELECT json_extract(data, '$.seq') FROM md_sequences WHERE id = ?`,
		string(kind)).Scan(&seq); err != nil {
		return 0, fmt.Errorf("masterdata: seq read: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("masterdata: seq commit: %w", err)
	}
	return seq, nil
}

func (r *sqliteRepository) GetRegime(ctx context.Context) (string, error) {
	var data string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM md_regimes WHERE id = 'current'`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("masterdata: get regime: %w", err)
	}
	var doc struct {
		Regime string `json:"regime"`
	}
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		return "", fmt.Errorf("masterdata: decode regime: %w", err)
	}
	return doc.Regime, nil
}

func (r *sqliteRepository) SetRegime(ctx context.Context, regime, actor string) error {
	doc, err := json.Marshal(map[string]string{
		"regime":     regime,
		"updated_by": actor,
		"updated_at": nowRFC3339(),
	})
	if err != nil {
		return fmt.Errorf("masterdata: marshal regime: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO md_regimes (id, data) VALUES ('current', ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`, string(doc)); err != nil {
		return fmt.Errorf("masterdata: set regime: %w", err)
	}
	return nil
}
