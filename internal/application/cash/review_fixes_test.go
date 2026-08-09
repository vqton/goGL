package cash_test

import (
	"context"
	"testing"

	appcash "goGL/internal/application/cash"
	domaincash "goGL/internal/domain/cash"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

// TestService_PostVoucher_UnauthorizedActor locks in R6: the poster must be
// neither the preparer nor the approver of the voucher.
func TestService_PostVoucher_UnauthorizedActor(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	// Poster == preparer.
	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "thuquy", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	approveVoucher(t, svc, v.ID, "giamdoc")
	if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != domaincash.ErrUnauthorizedActor {
		t.Fatalf("post by preparer: expected ErrUnauthorizedActor, got %v", err)
	}

	// Poster == approver.
	v2 := validVoucher()
	v2.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v2); err != nil {
		t.Fatalf("create: %v", err)
	}
	approveVoucher(t, svc, v2.ID, "giamdoc")
	if _, err := svc.PostVoucher(ctx, "giamdoc", v2.ID); err != domaincash.ErrUnauthorizedActor {
		t.Fatalf("post by approver: expected ErrUnauthorizedActor, got %v", err)
	}

	// Distinct actor posts fine.
	v3 := validVoucher()
	v3.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v3); err != nil {
		t.Fatalf("create: %v", err)
	}
	approveVoucher(t, svc, v3.ID, "giamdoc")
	if _, err := svc.PostVoucher(ctx, "thuquy", v3.ID); err != nil {
		t.Fatalf("post by distinct cashier: %v", err)
	}
}

// TestService_ReconcileMonth_InvalidSigners locks in UC-5's three-way
// electronic sign: exactly three distinct, non-empty signers.
func TestService_ReconcileMonth_InvalidSigners(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	good := []string{"thuquy", "ketoan", "giamdoc"}
	for name, signers := range map[string][]string{
		"too few":   {"thuquy", "ketoan"},
		"empty":     {},
		"blank":     {"", "ketoan", "giamdoc"},
		"duplicate": {"thuquy", "thuquy", "giamdoc"},
		"all same":  {"thuquy", "thuquy", "thuquy"},
	} {
		if _, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 5_000_000, signers); err != domaincash.ErrInvalidSigners {
			t.Fatalf("%s: expected ErrInvalidSigners, got %v", name, err)
		}
	}

	// The good set still works and records all three signers.
	rec, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 5_000_000, good)
	if err != nil {
		t.Fatalf("reconcile with valid signers: %v", err)
	}
	if len(rec.SignedBy) != 3 {
		t.Fatalf("expected 3 signers, got %+v", rec.SignedBy)
	}
}

type denyVoidApprover struct{}

func (denyVoidApprover) CanApproveVoid(context.Context, string) (bool, error) { return false, nil }

// TestService_VoidPosted_ApprovalDenied locks in Điều 30: voiding a posted
// voucher needs the chief accountant's sign-off via the VoidApprover seam,
// and a refused approval leaves the cash book untouched.
func TestService_VoidPosted_ApprovalDenied(t *testing.T) {
	ctx := context.Background()
	svc := appcash.NewService(perscash.NewSqliteRepository(openSvcDB(t)), &fakeAuditor{}, appcash.WithVoidApprover(denyVoidApprover{}))
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	vouchers, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	orig := vouchers[0]
	if _, err := svc.VoidVoucher(ctx, "ketoan", orig.ID, "hủy"); err != domaincash.ErrUnauthorizedActor {
		t.Fatalf("expected ErrUnauthorizedActor, got %v", err)
	}
	entries, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if len(entries) != 1 || entries[0].Balance != 5_000_000 {
		t.Fatalf("cash book must be untouched: %+v", entries)
	}
}

// TestService_ResolveCashCount_ClosesDay locks in UC-4: a mismatched count
// keeps the day open until it is resolved, then the day locks.
func TestService_ResolveCashCount_ClosesDay(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	// Mismatched close leaves the count open and the day unlocked.
	count, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, []string{"A"})
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if count.State != domaincash.CashCountOpen {
		t.Fatalf("mismatched count must stay open, got %s", count.State)
	}

	// A fresh count on the same date is blocked while one is pending.
	if _, err := svc.CreateCashCount(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err != domaincash.ErrOpenCountPending {
		t.Fatalf("expected ErrOpenCountPending, got %v", err)
	}

	// Resolving signs the resolution and closes the day.
	resolved, err := svc.ResolveCashCount(ctx, "ketoan", count.ID, "đã đối soát, chênh lệch do nhập nhầm")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.State != domaincash.CashCountResolved || resolved.Resolution == "" {
		t.Fatalf("resolve mismatch: %+v", resolved)
	}

	// Day is now closed: a new count for the same date is rejected.
	if _, err := svc.CreateCashCount(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err != domaincash.ErrPeriodClosed {
		t.Fatalf("expected ErrPeriodClosed after resolve, got %v", err)
	}
}

// TestService_BackdatedPost_RebuildsBalances locks in the rebuildBalances fix:
// posting an earlier-dated voucher after a later one re-derives every running
// balance so the ledger stays chronologically consistent.
func TestService_BackdatedPost_RebuildsBalances(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	// Later-dated voucher posted first.
	postReceive(t, svc, "fund-1", "2026-08-10", 5_000_000)
	// Back-dated voucher posted second.
	postReceive(t, svc, "fund-1", "2026-08-05", 3_000_000)

	entries, err := svc.GetCashBook(ctx, "fund-1", "", "")
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	first, second := entries[0], entries[1]
	if first.EntryDate != "2026-08-05" || first.Balance != 3_000_000 {
		t.Fatalf("back-dated entry wrong: %+v", first)
	}
	if second.EntryDate != "2026-08-10" || second.Balance != 8_000_000 {
		t.Fatalf("later entry not re-derived: %+v", second)
	}
}
