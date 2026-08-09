package sales

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/sales"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) sales.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, s *sales.SalesInvoice) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*sales.SalesInvoice, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, s *sales.SalesInvoice) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
