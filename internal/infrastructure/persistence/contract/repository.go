package contract

import (
	"context"
	"database/sql"

	"goGL/internal/domain/contract"
	"goGL/internal/domain/core"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) contract.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, c *contract.Contract) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*contract.Contract, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, c *contract.Contract) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}

func (r *sqliteRepository) CreateLoan(ctx context.Context, l *contract.LoanAgreement) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}
