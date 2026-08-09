package setup

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/setup"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) setup.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) SaveProfile(ctx context.Context, p *setup.CompanyProfile) error {
	// TODO: implement sqlite upsert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) GetProfile(ctx context.Context, id string) (*setup.CompanyProfile, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) SaveBalance(ctx context.Context, b *setup.InitialBalance) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}
