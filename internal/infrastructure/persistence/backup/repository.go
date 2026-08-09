package backup

import (
	"context"
	"database/sql"

	"goGL/internal/domain/backup"
	"goGL/internal/domain/core"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) backup.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, j *backup.BackupJob) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*backup.BackupJob, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, j *backup.BackupJob) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
