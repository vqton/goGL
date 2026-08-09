package budget

import (
	"context"
	"database/sql"

	"goGL/internal/domain/budget"
	"goGL/internal/domain/core"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) budget.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, p *budget.BudgetPlan) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*budget.BudgetPlan, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, p *budget.BudgetPlan) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
