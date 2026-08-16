package setup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"goGL/internal/domain/setup"
)

// sqliteRepository persists setup rows in the company_profiles,
// opening_balances and setup_status tables using the shared (id, data)
// JSON-document shape.
type sqliteRepository struct {
	db *sql.DB
}

// NewSqliteRepository builds the repository over a *sql.DB whose schema has
// been migrated by internal/infrastructure/db.Migrate.
func NewSqliteRepository(db *sql.DB) setup.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) SaveProfile(ctx context.Context, p *setup.CompanyProfile) error {
	if p.ID == "" {
		p.ID = setup.ProfileID
	}
	doc, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("setup: marshal profile: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO company_profiles (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		p.ID, string(doc)); err != nil {
		return fmt.Errorf("setup: save profile: %w", err)
	}
	return nil
}

func (r *sqliteRepository) GetProfile(ctx context.Context, id string) (*setup.CompanyProfile, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM company_profiles WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, err
	}
	var p setup.CompanyProfile
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return nil, fmt.Errorf("setup: decode profile: %w", err)
	}
	return &p, nil
}

func (r *sqliteRepository) SaveBalance(ctx context.Context, b *setup.OpeningBalance) error {
	doc, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("setup: marshal balance: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO opening_balances (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		b.ID, string(doc)); err != nil {
		return fmt.Errorf("setup: save balance: %w", err)
	}
	return nil
}

func (r *sqliteRepository) ListBalances(ctx context.Context, accountCode string) ([]*setup.OpeningBalance, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM opening_balances
		 WHERE ? = '' OR json_extract(data, '$.account_code') = ?
		 ORDER BY json_extract(data, '$.account_code'), json_extract(data, '$.object_code')`,
		accountCode, accountCode)
	if err != nil {
		return nil, fmt.Errorf("setup: list balances: %w", err)
	}
	defer rows.Close()

	var out []*setup.OpeningBalance
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var b setup.OpeningBalance
		if err := json.Unmarshal([]byte(data), &b); err != nil {
			return nil, fmt.Errorf("setup: decode balance: %w", err)
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) DeleteBalance(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM opening_balances WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("setup: delete balance: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return setup.ErrBalanceNotFound
	}
	return nil
}

func (r *sqliteRepository) GetStatus(ctx context.Context) (setup.SetupStatus, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM setup_status WHERE id = 'current'`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return setup.StatusEmpty, nil
	}
	if err != nil {
		return "", fmt.Errorf("setup: get status: %w", err)
	}
	var doc struct {
		Status setup.SetupStatus `json:"status"`
	}
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		return "", fmt.Errorf("setup: decode status: %w", err)
	}
	return doc.Status, nil
}

func (r *sqliteRepository) SetStatus(ctx context.Context, s setup.SetupStatus) error {
	doc, err := json.Marshal(map[string]any{"status": s})
	if err != nil {
		return fmt.Errorf("setup: marshal status: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO setup_status (id, data) VALUES ('current', ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`, string(doc)); err != nil {
		return fmt.Errorf("setup: set status: %w", err)
	}
	return nil
}
