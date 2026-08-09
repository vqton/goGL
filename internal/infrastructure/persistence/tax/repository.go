package tax

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/tax"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) tax.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, d *tax.TaxDeclaration) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*tax.TaxDeclaration, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, d *tax.TaxDeclaration) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
