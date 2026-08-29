package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"goGL/internal/domain/task"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) task.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) CreateRun(ctx context.Context, run *task.JobRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO task_runs (id, data) VALUES (?, ?)`, run.ID, string(data))
	return err
}

func (r *sqliteRepository) FindRun(ctx context.Context, id string) (*task.JobRun, error) {
	var data string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM task_runs WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, task.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var run task.JobRun
	if err := json.Unmarshal([]byte(data), &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *sqliteRepository) UpdateRun(ctx context.Context, run *task.JobRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE task_runs SET data = ? WHERE id = ?`, string(data), run.ID)
	return err
}

func (r *sqliteRepository) ListRuns(ctx context.Context) ([]*task.JobRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM task_runs ORDER BY rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*task.JobRun
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var run task.JobRun
		if err := json.Unmarshal([]byte(data), &run); err != nil {
			return nil, err
		}
		out = append(out, &run)
	}
	return out, rows.Err()
}
