package user

import (
	"context"
	"database/sql"

	"goGL/internal/domain/core"
	"goGL/internal/domain/user"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) user.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, u *user.User) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	// TODO: implement sqlite select
	return nil, core.ErrNotImplemented
}

func (r *sqliteRepository) Update(ctx context.Context, u *user.User) error {
	// TODO: implement sqlite update
	return core.ErrNotImplemented
}

func (r *sqliteRepository) SaveRole(ctx context.Context, role *user.Role) error {
	// TODO: implement sqlite insert
	return core.ErrNotImplemented
}
