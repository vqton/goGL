package system

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/system"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) system.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, t *system.Tenant) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*system.Tenant, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, t *system.Tenant) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
