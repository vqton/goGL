package cash_test

import (
	"context"
	"sync"
	"testing"

	appcash "goGL/internal/application/cash"
	domainaudit "goGL/internal/domain/audit"
	domaincash "goGL/internal/domain/cash"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

type noopAuditor struct{}

func (noopAuditor) Record(_ context.Context, _ *domainaudit.AuditLog) error { return nil }

func TestConcurrent_PostSameVoucher_NoDoubleEntry(t *testing.T) {
	svc := appcash.NewService(perscash.NewSqliteRepository(openSvcDB(t)), noopAuditor{})
	ctx := context.Background()

	mustCreateFund(t, svc, "fund-race")
	v := validVoucher()
	v.FundID = "fund-race"
	v.RefDate = "2026-08-10"
	v.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "1111", AmountMinor: v.AmountMinor},
		{Seq: 2, CreditAcc: "5111", AmountMinor: v.AmountMinor},
	}
	if err := svc.CreateVoucher(ctx, "ketoan01", v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}
	if _, err := svc.ApproveVoucher(ctx, "giamdoc01", v.ID); err != nil {
		t.Fatalf("approve voucher: %v", err)
	}

	const workers = 16
	var wg sync.WaitGroup
	ok, wrong := make(chan error, workers), make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.PostVoucher(ctx, "cashier01", v.ID)
			if err == domaincash.ErrWrongState {
				ok <- nil
				return
			}
			ok <- err
		}()
	}
	wg.Wait()
	close(ok)
	close(wrong)
	for err := range ok {
		if err != nil {
			t.Fatalf("concurrent post failed: %v", err)
		}
	}

	// BR10: exactly one cash book entry despite 16 concurrent posts.
	entries, err := svc.GetCashBook(ctx, "fund-race", "", "")
	if err != nil {
		t.Fatalf("get cash book: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 cash book entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Balance != v.AmountMinor {
		t.Errorf("balance = %d, want %d", entries[0].Balance, v.AmountMinor)
	}

	// Voucher is posted, not duplicated.
	got, err := svc.GetVoucher(ctx, v.ID)
	if err != nil {
		t.Fatalf("get voucher: %v", err)
	}
	if got.State != domaincash.VoucherPosted {
		t.Errorf("state = %s, want posted", got.State)
	}
}

func TestConcurrent_CreateVouchers_UniqueRefNos(t *testing.T) {
	svc := appcash.NewService(perscash.NewSqliteRepository(openSvcDB(t)), noopAuditor{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-seq")

	const n = 24
	refs := make(chan string, n)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := validVoucher()
			v.FundID = "fund-seq"
			v.RefDate = "2026-08-10"
			v.Lines = []domaincash.VoucherLine{
				{Seq: 1, DebitAcc: "1111", AmountMinor: v.AmountMinor},
				{Seq: 2, CreditAcc: "5111", AmountMinor: v.AmountMinor},
			}
			if err := svc.CreateVoucher(ctx, "cashier01", v); err != nil {
				errs <- err
				return
			}
			refs <- v.RefNo
		}()
	}
	wg.Wait()
	close(refs)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent create failed: %v", err)
	}

	seen := make(map[string]bool, n)
	for r := range refs {
		if seen[r] {
			t.Fatalf("duplicate RefNo %q under concurrency", r)
		}
		seen[r] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct refs, got %d", n, len(seen))
	}

	list, err := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-seq"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != n {
		t.Errorf("expected %d persisted vouchers, got %d", n, len(list))
	}
}

func TestConcurrent_CloseDaySameDate(t *testing.T) {
	svc := appcash.NewService(perscash.NewSqliteRepository(openSvcDB(t)), noopAuditor{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-count")

	const workers = 8
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CloseDay(ctx, "cashier01", "fund-count", "2026-08-31", 0, []string{"thuquy01"})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	// At least one close succeeds; others report an already-closed day
	// (the winner persisted ClosedDays before they re-read the fund).
	succeeded, pending := 0, 0
	for err := range results {
		switch err {
		case nil:
			succeeded++
		case domaincash.ErrOpenCountPending, domaincash.ErrPeriodClosed:
			pending++
		default:
			t.Fatalf("unexpected close-day error: %v", err)
		}
	}
	if succeeded < 1 {
		t.Fatal("expected at least one close-day to succeed")
	}
	_ = pending

	counts, err := svc.ListCashCounts(ctx, "fund-count")
	if err != nil {
		t.Fatalf("list counts: %v", err)
	}
	if len(counts) != 1 {
		t.Fatalf("expected exactly 1 cash count, got %d", len(counts))
	}

	// ClosedDays must not contain duplicate dates even if two close-day
	// attempts raced the append (BR: close-day is idempotent).
	funds, err := svc.ListFunds(ctx)
	if err != nil {
		t.Fatalf("list funds: %v", err)
	}
	var closed []string
	for _, f := range funds {
		if f.ID == "fund-count" {
			closed = f.ClosedDays
		}
	}
	seen := make(map[string]bool)
	for _, d := range closed {
		if seen[d] {
			t.Fatalf("duplicate closed day %q", d)
		}
		seen[d] = true
	}
}
