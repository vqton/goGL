package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"goGL/internal/domain/backup"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) backup.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) CreateArtifact(ctx context.Context, a *backup.BackupArtifact) error {
	return insert(ctx, r.db, "backup_artifacts", a.ID, a)
}

func (r *sqliteRepository) FindArtifact(ctx context.Context, id string) (*backup.BackupArtifact, error) {
	var a backup.BackupArtifact
	if err := find(ctx, r.db, "backup_artifacts", id, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *sqliteRepository) ListArtifacts(ctx context.Context) ([]*backup.BackupArtifact, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM backup_artifacts ORDER BY rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*backup.BackupArtifact
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var a backup.BackupArtifact
		if err := json.Unmarshal([]byte(data), &a); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) DeleteArtifact(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM backup_artifacts WHERE id = ?`, id)
	return err
}

func (r *sqliteRepository) CreateJob(ctx context.Context, j *backup.BackupJob) error {
	return insert(ctx, r.db, "backup_jobs", j.ID, j)
}

func (r *sqliteRepository) FindJob(ctx context.Context, id string) (*backup.BackupJob, error) {
	var j backup.BackupJob
	if err := find(ctx, r.db, "backup_jobs", id, &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *sqliteRepository) UpdateJob(ctx context.Context, j *backup.BackupJob) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE backup_jobs SET data = ? WHERE id = ?`, string(data), j.ID)
	return err
}

func (r *sqliteRepository) CreatePlan(ctx context.Context, p *backup.RestorePlan) error {
	return insert(ctx, r.db, "restore_plans", p.ID, p)
}

func (r *sqliteRepository) FindPlan(ctx context.Context, id string) (*backup.RestorePlan, error) {
	var p backup.RestorePlan
	if err := find(ctx, r.db, "restore_plans", id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *sqliteRepository) UpdatePlan(ctx context.Context, p *backup.RestorePlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE restore_plans SET data = ? WHERE id = ?`, string(data), p.ID)
	return err
}

func insert(ctx context.Context, db *sql.DB, table, id string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO `+table+` (id, data) VALUES (?, ?)`, id, string(data))
	return err
}

func find(ctx context.Context, db *sql.DB, table, id string, out any) error {
	var data string
	err := db.QueryRowContext(ctx, `SELECT data FROM `+table+` WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return backup.ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), out)
}
