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

func TestService_ListRecent(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	svc := audit.NewService(persaudit.NewSqliteRepository(d))

	for _, l := range []domainaudit.AuditLog{
		{UserCode: "u1", Module: "setup", Action: "balances.lock", TargetID: "s", Timestamp: "2026-08-08T10:00:00Z"},
		{UserCode: "u2", Module: "cash", Action: "voucher.create", TargetID: "v-1", Timestamp: "2026-08-08T11:00:00Z"},
		{UserCode: "u3", Module: "setup", Action: "balances.reopen", TargetID: "s", Timestamp: "2026-08-08T09:00:00Z"},
	} {
		if err := svc.Record(ctx, &l); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	trail, err := svc.ListRecent(ctx, "setup", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(trail) != 2 {
		t.Fatalf("want 2 setup logs, got %d: %+v", len(trail), trail)
	}
	if trail[0].Action != "balances.lock" || trail[1].Action != "balances.reopen" {
		t.Fatalf("setup trail must be newest-first: %+v", trail)
	}

	limited, err := svc.ListRecent(ctx, "", 1)
	if err != nil {
		t.Fatalf("limited list: %v", err)
	}
	if len(limited) != 1 || limited[0].Action != "voucher.create" {
		t.Fatalf("limit 1 must return newest entry: %+v", limited)
	}
}
