package document

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/document"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) document.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Create(ctx context.Context, d *document.Document) error {
	doc, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("document: marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO documents (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		d.ID, string(doc))
	if err != nil {
		return fmt.Errorf("document: create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindByID(ctx context.Context, id string) (*document.Document, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM documents WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, document.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("document: find: %w", err)
	}
	var doc document.Document
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		return nil, fmt.Errorf("document: decode: %w", err)
	}
	return &doc, nil
}

func (r *sqliteRepository) Update(ctx context.Context, d *document.Document) error {
	doc, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("document: marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET data = ? WHERE id = ?`,
		string(doc), d.ID)
	if err != nil {
		return fmt.Errorf("document: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return document.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("document: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return document.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, owner string, docType document.DocumentType, state document.DocumentState) ([]*document.Document, error) {
	q := `SELECT data FROM documents WHERE 1=1`
	var args []any
	if owner != "" {
		q += ` AND json_extract(data, '$.owner') = ?`
		args = append(args, owner)
	}
	if docType != "" {
		q += ` AND json_extract(data, '$.type') = ?`
		args = append(args, string(docType))
	}
	if state != "" {
		q += ` AND json_extract(data, '$.state') = ?`
		args = append(args, string(state))
	}
	q += ` ORDER BY json_extract(data, '$.created_at') DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("document: list: %w", err)
	}
	defer rows.Close()

	var out []*document.Document
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var d document.Document
		if err := json.Unmarshal([]byte(data), &d); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) NextCode(ctx context.Context) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("document: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO document_sequences (id, data) VALUES ('document', '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`); err != nil {
		return 0, fmt.Errorf("document: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE document_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = 'document' RETURNING json_extract(data, '$.seq')`).Scan(&seq); err != nil {
		return 0, fmt.Errorf("document: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("document: seq commit: %w", err)
	}
	return seq, nil
}
