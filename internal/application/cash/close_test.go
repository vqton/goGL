package cash_test

import (
	"context"
	"database/sql"
	"testing"

	appcash "goGL/internal/application/cash"
	domaincash "goGL/internal/domain/cash"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

type fakeNotifier struct {
	roles []string
}

func (f *fakeNotifier) Notify(_ context.Context, role, _, _ string) error {
	f.roles = append(f.roles, role)
	return nil
}

func newSvcWithNotifier(t *testing.T, d *sql.DB, n appcash.Notifier) (appcash.Service, *fakeAuditor) {
	t.Helper()
	a := &fakeAuditor{}
	return appcash.NewService(perscash.NewSqliteRepository(d), a, appcash.WithNotifier(n)), a
}

func postReceive(t *testing.T, svc appcash.Service, fundID, date string, amount int64) {
	t.Helper()
	ctx := context.Background()
	v := validVoucher()
	v.FundID = fundID
	v.RefDate = date
	v.AmountMinor = amount
	v.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "1111", AmountMinor: amount},
		{Seq: 2, CreditAcc: "131", AmountMinor: amount, ObjectID: "kh-1"},
	}
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}
	if _, err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != nil {
		t.Fatalf("post: %v", err)
	}
}

func TestService_CloseDay_Equal(t *testing.T) {
	d := openSvcDB(t)
	svc, a := newSvcWithNotifier(t, d, &fakeNotifier{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	count, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 5_000_000, []string{"thuquy", "ketoan"})
	if err != nil {
		t.Fatalf("close day: %v", err)
	}
	if count.State != "resolved" || count.Difference != 0 || count.BookBalance != 5_000_000 || count.CountedAmount != 5_000_000 {
		t.Fatalf("resolved count mismatch: %+v", count)
	}
	if len(count.Participants) != 2 {
		t.Fatalf("participants mismatch: %+v", count.Participants)
	}

	repo := perscash.NewSqliteRepository(d)
	f, _ := repo.GetFund(ctx, "fund-1")
	if len(f.ClosedDays) != 1 || f.ClosedDays[0] != "2026-08-05" {
		t.Fatalf("day not closed: %+v", f.ClosedDays)
	}

	if a.logs[len(a.logs)-1].Action != "cash.close_day" {
		t.Fatalf("expected cash.close_day audit, got %+v", a.logs[len(a.logs)-1])
	}

	// BR8: posting on a closed date is blocked.
	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != domaincash.ErrPeriodClosed {
		t.Fatalf("expected ErrPeriodClosed after close, got %v", err)
	}
}

func TestService_CloseDay_Difference(t *testing.T) {
	notifier := &fakeNotifier{}
	svc, a := newSvcWithNotifier(t, openSvcDB(t), notifier)
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	count, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_800_000, []string{"thuquy"})
	if err != nil {
		t.Fatalf("close day: %v", err)
	}
	if count.State != "open" || count.Difference != 200_000 {
		t.Fatalf("open count mismatch: %+v", count)
	}

	if len(notifier.roles) != 1 || notifier.roles[0] != "chief_accountant" {
		t.Fatalf("expected chief_accountant notification, got %+v", notifier.roles)
	}
	if a.logs[len(a.logs)-1].Action != "cash.count.open" {
		t.Fatalf("expected cash.count.open audit, got %+v", a.logs[len(a.logs)-1])
	}
}

func TestService_CloseDay_OpenCountPending(t *testing.T) {
	svc, _ := newSvcWithNotifier(t, openSvcDB(t), &fakeNotifier{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err != domaincash.ErrOpenCountPending {
		t.Fatalf("expected ErrOpenCountPending, got %v", err)
	}
}

func TestService_ListCashCounts(t *testing.T) {
	svc, _ := newSvcWithNotifier(t, openSvcDB(t), &fakeNotifier{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	_, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 5_000_000, nil)
	if err != nil {
		t.Fatalf("close day: %v", err)
	}

	counts, err := svc.ListCashCounts(ctx, "fund-1")
	if err != nil {
		t.Fatalf("list counts: %v", err)
	}
	if len(counts) != 1 || counts[0].State != "resolved" {
		t.Fatalf("counts mismatch: %+v", counts)
	}
}
