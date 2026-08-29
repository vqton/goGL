package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"goGL/internal/domain/user"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) user.Repository {
	return &sqliteRepository{db: db}
}

const userTable = "users"
const roleTable = "roles"

func (r *sqliteRepository) Create(ctx context.Context, u *user.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO `+userTable+` (id, data) VALUES (?, ?)`, u.ID, string(data))
	return err
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM `+userTable+` WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var u user.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *sqliteRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM `+userTable+` WHERE json_extract(data, '$.username') = ?`, username).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var u user.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *sqliteRepository) Update(ctx context.Context, u *user.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE `+userTable+` SET data = ? WHERE id = ?`, string(data), u.ID)
	return err
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM `+userTable+` WHERE id = ?`, id)
	return err
}

func (r *sqliteRepository) List(ctx context.Context) ([]*user.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM `+userTable+` ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*user.User
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var u user.User
		if err := json.Unmarshal([]byte(data), &u); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SaveRole(ctx context.Context, role *user.Role) error {
	data, err := json.Marshal(role)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO `+roleTable+` (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		role.Code, string(data))
	return err
}

func (r *sqliteRepository) FindRoleByCode(ctx context.Context, code string) (*user.Role, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM `+roleTable+` WHERE id = ?`, code).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrRoleNotFound
	}
	if err != nil {
		return nil, err
	}
	var role user.Role
	if err := json.Unmarshal([]byte(data), &role); err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *sqliteRepository) ListRoles(ctx context.Context) ([]*user.Role, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM `+roleTable+` ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*user.Role
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var role user.Role
		if err := json.Unmarshal([]byte(data), &role); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) DeleteRole(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM `+roleTable+` WHERE id = ?`, code)
	return err
}

// Password history methods

func (r *sqliteRepository) AddPasswordHistory(ctx context.Context, userID, hash string) error {
	data, err := json.Marshal(map[string]string{"user_id": userID, "hash": hash})
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO password_history (id, data) VALUES (?, ?)`,
		userID+"_"+hash[:8], string(data))
	return err
}

func (r *sqliteRepository) GetPasswordHistory(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM password_history WHERE json_extract(data, '$.user_id') = ? ORDER BY rowid DESC LIMIT ?`,
		userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var record map[string]string
		if err := json.Unmarshal([]byte(data), &record); err != nil {
			return nil, err
		}
		hashes = append(hashes, record["hash"])
	}
	return hashes, rows.Err()
}

func (r *sqliteRepository) UpdatePasswordChangedAt(ctx context.Context, userID string, t time.Time) error {
	// This is handled by Update() since PasswordChangedAt is part of the User struct.
	return nil
}
