package db

// Master-data tables registered into the global migration list. Kept in a
// separate file so the module can be added/removed without touching migrate.go.
func init() {
	tables = append(tables,
		"md_records",   // master-data records as (id, data) JSON documents
		"md_sequences", // per-kind auto-code counters: (kind, seq)
		"md_regimes",   // active reporting regime: (id, data)
	)
}
