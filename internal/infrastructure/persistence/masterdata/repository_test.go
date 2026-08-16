package masterdata_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/masterdata"
	"goGL/internal/infrastructure/db"
	persistence "goGL/internal/infrastructure/persistence/masterdata"
)

func dbMigrate(d *sql.DB) error {
	return db.Migrate(context.Background(), d)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:md_repo_%p?mode=memory&cache=shared", t)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := dbMigrate(d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

func newRepo(t *testing.T) masterdata.Repository {
	t.Helper()
	return persistence.NewSqliteRepository(openTestDB(t))
}

func TestUpsertGetRoundtrip(t *testing.T) {
	repo := newRepo(t)
	rec := &masterdata.Record{
		Kind:  masterdata.KindCustomer,
		Code:  "KH-00001",
		Name:  "Công ty ABC",
		State: masterdata.StateActive,
		Extra: map[string]string{"tax_code": "0101234567"},
	}
	if err := repo.Upsert(context.Background(), rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.Get(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Code != "KH-00001" || got.Extra["tax_code"] != "0101234567" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	byCode, err := repo.GetByCode(context.Background(), masterdata.KindCustomer, "KH-00001")
	if err != nil {
		t.Fatalf("get by code: %v", err)
	}
	if byCode.ID != rec.ID {
		t.Fatalf("id mismatch: %s != %s", byCode.ID, rec.ID)
	}
}

func TestGetByCodeNotFound(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.GetByCode(context.Background(), masterdata.KindCustomer, "nope")
	if err != masterdata.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUpsertIsIdempotentByID(t *testing.T) {
	repo := newRepo(t)
	rec := &masterdata.Record{Kind: masterdata.KindSupplier, Code: "NCC-1", Name: "A"}
	if err := repo.Upsert(context.Background(), rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	rec.Name = "A2"
	if err := repo.Upsert(context.Background(), rec); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	got, err := repo.Get(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "A2" {
		t.Fatalf("expected overwritten name, got %q", got.Name)
	}
}

func TestListOrderedByCode(t *testing.T) {
	repo := newRepo(t)
	for _, c := range []string{"KH-2", "KH-1", "KH-10"} {
		if err := repo.Upsert(context.Background(), &masterdata.Record{
			Kind: masterdata.KindCustomer, Code: c, Name: c}); err != nil {
			t.Fatalf("upsert %s: %v", c, err)
		}
	}
	got, err := repo.List(context.Background(), masterdata.KindCustomer)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 rows, got %d", len(got))
	}
	if got[0].Code != "KH-1" {
		t.Fatalf("expected KH-1 first, got %s", got[0].Code)
	}
}

func TestListKindIsolation(t *testing.T) {
	repo := newRepo(t)
	_ = repo.Upsert(context.Background(), &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-1", Name: "c"})
	_ = repo.Upsert(context.Background(), &masterdata.Record{Kind: masterdata.KindSupplier, Code: "NCC-1", Name: "s"})
	got, err := repo.List(context.Background(), masterdata.KindCustomer)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Kind != masterdata.KindCustomer {
		t.Fatalf("kind leak: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	repo := newRepo(t)
	rec := &masterdata.Record{Kind: masterdata.KindReason, Code: "LD-1", Name: "x"}
	if err := repo.Upsert(context.Background(), rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := repo.Delete(context.Background(), rec.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(context.Background(), rec.ID); err != masterdata.ErrNotFound {
		t.Fatalf("want ErrNotFound on re-delete, got %v", err)
	}
}

func TestNextCodeSequence(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	for i, want := range []int64{1, 2, 3} {
		n, err := repo.NextCode(ctx, masterdata.KindCustomer)
		if err != nil {
			t.Fatalf("next: %v", err)
		}
		if n != want {
			t.Fatalf("step %d: want %d, got %d", i, want, n)
		}
	}
	// Sequences are per-kind.
	n, err := repo.NextCode(ctx, masterdata.KindSupplier)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if n != 1 {
		t.Fatalf("supplier seq should start at 1, got %d", n)
	}
}

func TestRegimeRoundtrip(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	if r, _ := repo.GetRegime(ctx); r != "" {
		t.Fatalf("expected empty regime initially, got %q", r)
	}
	if err := repo.SetRegime(ctx, "TT99-2025", "alice"); err != nil {
		t.Fatalf("set regime: %v", err)
	}
	r, err := repo.GetRegime(ctx)
	if err != nil {
		t.Fatalf("get regime: %v", err)
	}
	if r != "TT99-2025" {
		t.Fatalf("want TT99-2025, got %q", r)
	}
	// Idempotent overwrite.
	if err := repo.SetRegime(ctx, "TT200-2014", "bob"); err != nil {
		t.Fatalf("re-set regime: %v", err)
	}
	if r, _ = repo.GetRegime(ctx); r != "TT200-2014" {
		t.Fatalf("want TT200-2014, got %q", r)
	}
}
