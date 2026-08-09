package authorization

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"

	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
)

// policyTable stores every casbin rule. It follows the repo-wide
// (id TEXT PRIMARY KEY, data TEXT NOT NULL) JSON-document table shape; the
// table itself is created by db.Migrate alongside all other tables.
const policyTable = "casbin_policies"

// sqliteAdapter persists casbin policy rules in the (id, data) JSON-document
// table. The id is a deterministic hash of the rule so re-adding an identical
// rule is an upsert rather than a duplicate row.
type sqliteAdapter struct {
	db *sql.DB
}

// NewSqliteAdapter returns a casbin persist.Adapter backed by *sql.DB.
func NewSqliteAdapter(db *sql.DB) *sqliteAdapter {
	return &sqliteAdapter{db: db}
}

var _ persist.Adapter = (*sqliteAdapter)(nil)
var _ persist.BatchAdapter = (*sqliteAdapter)(nil)

// ruleID derives the deterministic row id for a policy rule.
func ruleID(ptype string, rule []string) string {
	h := sha256.New()
	h.Write([]byte(ptype))
	h.Write([]byte{0})
	for _, field := range rule {
		h.Write([]byte(field))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ruleRow marshals a rule to the stored JSON array, ptype first.
func ruleRow(ptype string, rule []string) (string, error) {
	row := append([]string{ptype}, rule...)
	data, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// upsertRule writes a single rule row, replacing an identical existing one.
func upsertRule(exec interface {
	Exec(string, ...any) (sql.Result, error)
}, ptype string, rule []string) error {
	data, err := ruleRow(ptype, rule)
	if err != nil {
		return err
	}
	_, err = exec.Exec(
		`INSERT INTO `+policyTable+` (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		ruleID(ptype, rule), data,
	)
	return err
}

// LoadPolicy loads all policy rules from the store into the model.
func (a *sqliteAdapter) LoadPolicy(m model.Model) error {
	rows, err := a.db.Query(`SELECT data FROM ` + policyTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return err
		}
		var rule []string
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			return err
		}
		if err := persist.LoadPolicyArray(rule, m); err != nil {
			return err
		}
	}
	return rows.Err()
}

// SavePolicy wipes the store and writes every rule currently in the model.
func (a *sqliteAdapter) SavePolicy(m model.Model) error {
	if _, err := a.db.Exec(`DELETE FROM ` + policyTable); err != nil {
		return err
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for sec, assertions := range m {
		if sec != "p" && sec != "g" {
			continue
		}
		for ptype, ast := range assertions {
			for _, rule := range ast.Policy {
				if err := upsertRule(tx, ptype, rule); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

// AddPolicy persists a single rule (Auto-Save support).
func (a *sqliteAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	return upsertRule(a.db, ptype, rule)
}

// AddPolicies persists several rules in one transaction (Batch support).
func (a *sqliteAdapter) AddPolicies(_ string, ptype string, rules [][]string) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, rule := range rules {
		if err := upsertRule(tx, ptype, rule); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RemovePolicy deletes a single rule.
func (a *sqliteAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	_, err := a.db.Exec(`DELETE FROM `+policyTable+` WHERE id = ?`, ruleID(ptype, rule))
	return err
}

// RemovePolicies deletes several rules in one transaction (Batch support).
func (a *sqliteAdapter) RemovePolicies(_ string, ptype string, rules [][]string) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, rule := range rules {
		if _, err := tx.Exec(`DELETE FROM `+policyTable+` WHERE id = ?`, ruleID(ptype, rule)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RemoveFilteredPolicy deletes every rule of the given type whose fields match
// the filter starting at fieldIndex. Empty filter values are skipped.
func (a *sqliteAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	rows, err := a.db.Query(`SELECT id, data FROM ` + policyTable)
	if err != nil {
		return err
	}

	var ids []string
	for rows.Next() {
		var id, data string
		if err := rows.Scan(&id, &data); err != nil {
			rows.Close()
			return err
		}
		var rule []string
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			rows.Close()
			return err
		}
		if len(rule) == 0 || rule[0] != ptype {
			continue
		}
		if filterMatches(rule[1:], fieldIndex, fieldValues) {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, id := range ids {
		if _, err := a.db.Exec(`DELETE FROM `+policyTable+` WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

// filterMatches reports whether fields matches fieldValues at fieldIndex.
func filterMatches(fields []string, fieldIndex int, fieldValues []string) bool {
	for i, want := range fieldValues {
		if want == "" {
			continue
		}
		idx := fieldIndex + i
		if idx >= len(fields) || fields[idx] != want {
			return false
		}
	}
	return true
}
