package ledger

import (
	"context"
	"database/sql"
	"encoding/json"

	"goGL/internal/domain/ledger"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) ledger.Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) upsertDoc(ctx context.Context, table, id string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO `+table+` (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`, id, string(data))
	return err
}

func (r *sqliteRepository) getDoc(ctx context.Context, table, id string, out any) error {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM `+table+` WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), out)
}

func (r *sqliteRepository) CreateEntry(ctx context.Context, e *ledger.JournalEntry) error {
	return r.upsertDoc(ctx, "ledger_journals", e.ID, e)
}

func (r *sqliteRepository) UpdateEntry(ctx context.Context, e *ledger.JournalEntry) error {
	return r.upsertDoc(ctx, "ledger_journals", e.ID, e)
}

func (r *sqliteRepository) GetEntry(ctx context.Context, id string) (*ledger.JournalEntry, error) {
	var e ledger.JournalEntry
	if err := r.getDoc(ctx, "ledger_journals", id, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *sqliteRepository) GetEntryBySource(ctx context.Context, source ledger.EntrySource, ref string) (*ledger.JournalEntry, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM ledger_journals
		 WHERE json_extract(data, '$.source') = ? AND json_extract(data, '$.source_ref') = ?
		 ORDER BY rowid LIMIT 1`, string(source), ref).Scan(&data)
	if err != nil {
		return nil, err
	}
	var e ledger.JournalEntry
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// DeleteEntry removes a row only while it is still DRAFT (R7). The status
// guard is a CAS on the stored JSON, so a concurrent PostEntry that committed
// between the caller's read and this delete cannot be undone: the delete then
// matches nothing and reports ErrWrongState. POSTED/REVERSED rows survive.
func (r *sqliteRepository) DeleteEntry(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM ledger_journals WHERE id = ? AND json_extract(data, '$.status') = 'draft'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ledger.ErrWrongState
	}
	return nil
}

// PostEntry posts a journal entry atomically (R10): inside a single
// transaction it acquires the next per-form-per-period VoucherNo (unless the
// entry already carries one from its source system) and CAS-updates the DRAFT
// row. The R5 duplicate-key guard and the draft CAS are one statement, so the
// duplicate check runs under the write lock that statement takes — concurrent
// posts of the same (Source, SourceRef) serialize and never double-post. The
// sequence increment is the first statement, so concurrent numbered posts
// serialize before the counter is read. Returns the entry that owns the key:
// the freshly posted one, or the already-posted holder (idempotent retry).
func (r *sqliteRepository) PostEntry(ctx context.Context, e *ledger.JournalEntry, form string) (*ledger.JournalEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if form != "" {
		id := ledger.RowID("seq", form, e.Period)
		seed, _ := json.Marshal(ledger.VoucherSeq{Form: form, Period: e.Period, N: 1})
		var seq int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO ledger_sequences (id, data) VALUES (?, ?)
			ON CONFLICT(id) DO UPDATE SET data = json_set(
				data, '$.n', json_extract(data, '$.n') + 1)
			RETURNING json_extract(data, '$.n')`, id, string(seed)).Scan(&seq); err != nil {
			return nil, err
		}
		e.VoucherNo = ledger.FormatVoucherNo(form, seq, e.Period)
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	// The draft transition is blocked only when another POSTED entry already
	// holds the (Source, SourceRef) key — that entry is then returned, making a
	// concurrent duplicate post an idempotent retry (R5). SQLite evaluates the
	// NOT EXISTS subquery under the write lock this statement acquires, so the
	// check is serialized against concurrent posts on both the numbered and
	// pre-numbered paths. A duplicate that is still DRAFT does not block the
	// post here; the service rejects posting a draft onto a draft-held key.
	res, err := tx.ExecContext(ctx,
		`UPDATE ledger_journals SET data = ?
		 WHERE id = ? AND json_extract(data, '$.status') = 'draft'
		   AND NOT EXISTS (
		     SELECT 1 FROM ledger_journals x
		     WHERE x.id <> ? AND json_extract(x.data, '$.source') = ?
		       AND json_extract(x.data, '$.source_ref') = ?
		       AND json_extract(x.data, '$.status') = 'posted')`,
		string(data), e.ID, e.ID, string(e.Source), e.SourceRef)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		// The write lock is held: disambiguate ErrWrongState (not a draft, or
		// the row is gone) from the idempotent R5 case (a posted holder).
		var cur string
		if err := tx.QueryRowContext(ctx, `SELECT data FROM ledger_journals WHERE id = ?`, e.ID).Scan(&cur); err != nil {
			return nil, ledger.ErrWrongState
		}
		var current ledger.JournalEntry
		if err := json.Unmarshal([]byte(cur), &current); err != nil {
			return nil, err
		}
		if current.Status != ledger.EntryDraft {
			return nil, ledger.ErrWrongState
		}
		var dup string
		if err := tx.QueryRowContext(ctx,
			`SELECT data FROM ledger_journals
			 WHERE id <> ? AND json_extract(data, '$.source') = ?
			   AND json_extract(data, '$.source_ref') = ?
			   AND json_extract(data, '$.status') = 'posted'
			 ORDER BY rowid LIMIT 1`,
			e.ID, string(e.Source), e.SourceRef).Scan(&dup); err != nil {
			// The NOT EXISTS guard blocked us, so a posted holder must exist;
			// defensively report the duplicate rather than a wrong-state.
			return nil, ledger.ErrDuplicateSource
		}
		var existing ledger.JournalEntry
		if err := json.Unmarshal([]byte(dup), &existing); err != nil {
			return nil, err
		}
		return &existing, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return e, nil
}

func (r *sqliteRepository) ListEntries(ctx context.Context, f ledger.EntryFilter) ([]*ledger.JournalEntry, error) {
	q := `SELECT data FROM ledger_journals WHERE 1=1`
	var args []any
	if f.Period != "" {
		q += ` AND json_extract(data, '$.period') = ?`
		args = append(args, f.Period)
	}
	if f.Source != "" {
		q += ` AND json_extract(data, '$.source') = ?`
		args = append(args, string(f.Source))
	}
	if f.Status != "" {
		q += ` AND json_extract(data, '$.status') = ?`
		args = append(args, string(f.Status))
	}
	if f.FromDate != "" {
		q += ` AND json_extract(data, '$.voucher_date') >= ?`
		args = append(args, f.FromDate)
	}
	if f.ToDate != "" {
		q += ` AND json_extract(data, '$.voucher_date') <= ?`
		args = append(args, f.ToDate)
	}
	q += ` ORDER BY json_extract(data, '$.voucher_date'), json_extract(data, '$.period'), rowid`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ledger.JournalEntry = []*ledger.JournalEntry{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var e ledger.JournalEntry
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateAccount(ctx context.Context, a *ledger.Account) error {
	return r.upsertDoc(ctx, "ledger_accounts", a.ID, a)
}

func (r *sqliteRepository) UpdateAccount(ctx context.Context, a *ledger.Account) error {
	return r.upsertDoc(ctx, "ledger_accounts", a.ID, a)
}

func (r *sqliteRepository) GetAccount(ctx context.Context, id string) (*ledger.Account, error) {
	var a ledger.Account
	if err := r.getDoc(ctx, "ledger_accounts", id, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *sqliteRepository) GetAccountByCode(ctx context.Context, code string) (*ledger.Account, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM ledger_accounts WHERE json_extract(data, '$.code') = ?`, code).Scan(&data)
	if err != nil {
		return nil, err
	}
	var a ledger.Account
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *sqliteRepository) ListAccounts(ctx context.Context) ([]*ledger.Account, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM ledger_accounts ORDER BY json_extract(data, '$.code')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ledger.Account = []*ledger.Account{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var a ledger.Account
		if err := json.Unmarshal([]byte(data), &a); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreatePeriod(ctx context.Context, p *ledger.AccountingPeriod) error {
	return r.upsertDoc(ctx, "ledger_periods", p.ID, p)
}

func (r *sqliteRepository) GetPeriod(ctx context.Context, id string) (*ledger.AccountingPeriod, error) {
	var p ledger.AccountingPeriod
	if err := r.getDoc(ctx, "ledger_periods", id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *sqliteRepository) ListPeriods(ctx context.Context) ([]*ledger.AccountingPeriod, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM ledger_periods ORDER BY json_extract(data, '$.id') DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ledger.AccountingPeriod = []*ledger.AccountingPeriod{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var p ledger.AccountingPeriod
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}
