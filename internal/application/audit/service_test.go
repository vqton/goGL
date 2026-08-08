package audit_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/application/audit"
	domainaudit "goGL/internal/domain/audit"
	"goGL/internal/infrastructure/db"
	persaudit "goGL/internal/infrastructure/persistence/audit"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	d, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })

	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

func TestService_RecordAssignsIDAndTimestamp(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	svc := audit.NewService(persaudit.NewSqliteRepository(d))

	in := &domainaudit.AuditLog{
		UserCode: "ketoan",
		Module:   "cash",
		Action:   "voucher.create",
		TargetID: "v-2",
	}
	if err := svc.Record(ctx, in); err != nil {
		t.Fatalf("record: %v", err)
	}
	if in.ID == "" {
		t.Fatal("expected Record to assign an ID")
	}
	if in.Timestamp == "" {
		t.Fatal("expected Record to assign a timestamp")
	}

	got, err := svc.GetLog(ctx, in.ID)
	if err != nil {
		t.Fatalf("get log: %v", err)
	}
	if got.Action != in.Action || got.TargetID != in.TargetID {
		t.Fatalf("stored log mismatch: got %+v", got)
	}
}

func TestService_RecordKeepsProvidedIDAndTimestamp(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	svc := audit.NewService(persaudit.NewSqliteRepository(d))

	in := &domainaudit.AuditLog{
		ID:        "log-preset",
		UserCode:  "admin",
		Module:    "cash",
		Action:    "voucher.void",
		TargetID:  "v-3",
		Timestamp: "2026-08-08T10:00:00Z",
	}
	if err := svc.Record(ctx, in); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err := svc.GetLog(ctx, "log-preset")
	if err != nil {
		t.Fatalf("get log: %v", err)
	}
	if got.ID != "log-preset" || got.Timestamp != in.Timestamp {
		t.Fatalf("expected provided id/timestamp preserved, got %+v", got)
	}
}
