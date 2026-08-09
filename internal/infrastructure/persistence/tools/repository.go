package tools

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/tools"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) tools.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, t *tools.ToolsCard) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*tools.ToolsCard, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, t *tools.ToolsCard) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}
