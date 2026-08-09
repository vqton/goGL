package fixedasset

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/fixedasset"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) fixedasset.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, a *fixedasset.FixedAsset) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, a *fixedasset.FixedAsset) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
