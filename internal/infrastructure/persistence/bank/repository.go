package bank

import (
	"context"
	"database/sql"

	"goGL/internal/domain/bank"
	"goGL/internal/domain/core"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) bank.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, t *bank.BankTransaction) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*bank.BankTransaction, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, t *bank.BankTransaction) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
