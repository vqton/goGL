package purchase

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) purchase.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, p *purchase.PurchaseInvoice) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*purchase.PurchaseInvoice, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, p *purchase.PurchaseInvoice) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
