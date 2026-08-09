package costing

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/costing"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) costing.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, c *costing.CostSheet) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*costing.CostSheet, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, c *costing.CostSheet) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
