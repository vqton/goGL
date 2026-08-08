package audit

import (
	"context"
	"database/sql"
	"encoding/json"

	"goGL/internal/domain/audit"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) audit.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, l *audit.AuditLog) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, data) VALUES (?, ?)`, l.ID, string(data))
	return err
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*audit.AuditLog, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM audit_logs WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, err
	}
	var l audit.AuditLog
	if err := json.Unmarshal([]byte(data), &l); err != nil {
		return nil, err
	}
	return &l, nil
}
