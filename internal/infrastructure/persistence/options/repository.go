package options

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/options"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) options.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, o *options.Option) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByKey(ctx context.Context, key string) (*options.Option, error) {
	// TODO: implement sqlite select by key
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, o *options.Option) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
