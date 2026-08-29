package session

import (
	"context"
	"database/sql"
	"encoding/json"

	"goGL/internal/domain/session"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) session.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, s *session.Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, data) VALUES (?, ?)`, s.ID, string(data))
	return err
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*session.Session, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM sessions WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, err
	}
	var s session.Session
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sqliteRepository) Update(ctx context.Context, s *session.Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE sessions SET data = ? WHERE id = ?`, string(data), s.ID)
	return err
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (r *sqliteRepository) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE json_extract(data, '$.user_id') = ?`, userID)
	return err
}

func (r *sqliteRepository) CountActive(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE json_extract(data, '$.expires_at') > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`).
		Scan(&n)
	return n, err
}

func (r *sqliteRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE json_extract(data, '$.user_id') = ? AND json_extract(data, '$.expires_at') > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		userID).Scan(&n)
	return n, err
}

func (r *sqliteRepository) ListByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM sessions WHERE json_extract(data, '$.user_id') = ? ORDER BY rowid`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*session.Session
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var s session.Session
		if err := json.Unmarshal([]byte(data), &s); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}
