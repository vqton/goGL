package document

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/document"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) document.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, d *document.Document) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*document.Document, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, d *document.Document) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
