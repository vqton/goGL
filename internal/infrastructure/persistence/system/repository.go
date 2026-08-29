package system

import (
	"context"
	"database/sql"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) *sqliteRepository {
	return &sqliteRepository{db: db}
}

// Ping verifies the database connection is healthy.
func (r *sqliteRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
