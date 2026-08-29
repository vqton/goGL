package options

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"goGL/internal/domain/options"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) options.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, o *options.Option) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO system_options (id, data) VALUES (?, ?)`, o.ID, string(data))
	return err
}

func (r *sqliteRepository) FindByKey(ctx context.Context, key string) (*options.Option, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM system_options WHERE json_extract(data, '$.key') = ?`, key).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, options.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var o options.Option
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *sqliteRepository) Update(ctx context.Context, o *options.Option) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE system_options SET data = ? WHERE id = ?`, string(data), o.ID)
	return err
}

func (r *sqliteRepository) List(ctx context.Context) ([]*options.Option, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM system_options ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*options.Option
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var o options.Option
		if err := json.Unmarshal([]byte(data), &o); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}
