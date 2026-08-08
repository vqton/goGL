package cash_test

import (
	"context"
	"database/sql"
	"testing"

	appcash "goGL/internal/application/cash"
	domaincash "goGL/internal/domain/cash"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

type fakeLedger struct {
	entries []appcash.LedgerEntry
}

func (f *fakeLedger) Post(_ context.Context, e appcash.LedgerEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func newSvcWithLedger(t *testing.T, d *sql.DB, l appcash.LedgerWriter) (appcash.Service, *fakeAuditor) {
	t.Helper()
	a := &fakeAuditor{}
	return appcash.NewService(perscash.NewSqliteRepository(d), a, appcash.WithLedger(l)), a
}

func approveVoucher(t *testing.T, svc appcash.Service, id, approver string) {
	t.Helper()
	if _, err := svc.ApproveVoucher(context.Background(), approver, id); err != nil {
		t.Fatalf("approve voucher: %v", err)
	}
}

func createApproved(t *testing.T, svc appcash.Service, v *domaincash.Voucher) string {
	t.Helper()
	if err := svc.CreateVoucher(context.Background(), "ketoan", v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}
	approveVoucher(t, svc, v.ID, "giamdoc")
	return v.ID
}

func TestService_PostVoucher_Receive(t *testing.T) {
	d := openSvcDB(t)
	ledger := &fakeLedger{}
	svc, a := newSvcWithLedger(t, d, ledger)
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	id := createApproved(t, svc, v)

	posted, err := svc.PostVoucher(ctx, "thuquy", id)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if posted.State != domaincash.VoucherPosted || posted.PostedBy != "thuquy" || posted.PostedAt == "" {
		t.Fatalf("posted voucher mismatch: %+v", posted)
	}

	entries, err := svc.GetCashBook(ctx, "fund-1", "", "")
	if err != nil {
		t.Fatalf("cash book: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RefNo != "PT/2026-08/000001" || e.Receive != 5_000_000 || e.Pay != 0 || e.Balance != 5_000_000 {
		t.Fatalf("entry mismatch: %+v", e)
	}

	if len(ledger.entries) != 1 {
		t.Fatalf("expected 1 ledger post, got %d", len(ledger.entries))
	}
	le := ledger.entries[0]
	if le.Account != "1111" || le.Debit != 5_000_000 || le.Credit != 0 || le.VoucherID != id || le.RefNo != "PT/2026-08/000001" {
		t.Fatalf("ledger entry mismatch: %+v", le)
	}

	if a.logs[len(a.logs)-1].Action != "voucher.post" {
		t.Fatalf("expected voucher.post audit log, got %+v", a.logs[len(a.logs)-1])
	}
}

func TestService_PostVoucher_Pay_NoNegativeBalance(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	rec := validVoucher()
	rec.FundID = "fund-1"
	recID := createApproved(t, svc, rec)
	if _, err := svc.PostVoucher(ctx, "thuquy", recID); err != nil {
		t.Fatalf("post receive: %v", err)
	}

	pay := validVoucher()
	pay.FundID = "fund-1"
	pay.Type = domaincash.VoucherPay
	pay.Description = "Chi tiền mua hàng"
	pay.AmountMinor = 6_000_000
	pay.CounterpartyName = "Công ty XYZ"
	pay.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "152", AmountMinor: 6_000_000},
		{Seq: 2, CreditAcc: "1111", AmountMinor: 6_000_000},
	}
	id := createApproved(t, svc, pay)

	if _, err := svc.PostVoucher(ctx, "thuquy", id); err != domaincash.ErrNegativeBalance {
		t.Fatalf("expected ErrNegativeBalance, got %v", err)
	}

	pay2 := validVoucher()
	pay2.FundID = "fund-1"
	pay2.Type = domaincash.VoucherPay
	pay2.Description = "Chi ít"
	pay2.AmountMinor = 2_000_000
	pay2.CounterpartyName = "Công ty XYZ"
	pay2.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "152", AmountMinor: 2_000_000},
		{Seq: 2, CreditAcc: "1111", AmountMinor: 2_000_000},
	}
	id2 := createApproved(t, svc, pay2)
	if _, err := svc.PostVoucher(ctx, "thuquy", id2); err != nil {
		t.Fatalf("post smaller pay: %v", err)
	}

	entries, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if got := entries[len(entries)-1].Balance; got != 3_000_000 {
		t.Fatalf("expected closing balance 3000000, got %d", got)
	}
}

func TestService_PostVoucher_NotApproved(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != domaincash.ErrWrongState {
		t.Fatalf("expected ErrWrongState (draft), got %v", err)
	}
}

func TestService_PostVoucher_DoublePost_BR10(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	id := createApproved(t, svc, v)

	if _, err := svc.PostVoucher(ctx, "thuquy", id); err != nil {
		t.Fatalf("first post: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", id); err != domaincash.ErrWrongState {
		t.Fatalf("expected ErrWrongState on double post, got %v", err)
	}
	entries, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 cash book entry (no double-entry), got %d", len(entries))
	}
}

func TestService_PostVoucher_ClosedDay_BR8(t *testing.T) {
	d := openSvcDB(t)
	svc, _ := newSvc(t, d)
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	repo := perscash.NewSqliteRepository(d)
	f, err := repo.GetFund(ctx, "fund-1")
	if err != nil {
		t.Fatalf("get fund: %v", err)
	}
	f.ClosedDays = append(f.ClosedDays, "2026-08-05")
	if err := repo.CreateFund(ctx, f); err != nil {
		t.Fatalf("close day: %v", err)
	}

	v := validVoucher()
	v.FundID = "fund-1"
	id := createApproved(t, svc, v)

	if _, err := svc.PostVoucher(ctx, "thuquy", id); err != domaincash.ErrPeriodClosed {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestService_GetCashBook_Filter(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	for _, day := range []string{"2026-08-03", "2026-08-07"} {
		v := validVoucher()
		v.FundID = "fund-1"
		v.RefDate = day
		id := createApproved(t, svc, v)
		if _, err := svc.PostVoucher(ctx, "thuquy", id); err != nil {
			t.Fatalf("post %s: %v", day, err)
		}
	}

	all, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
	subset, _ := svc.GetCashBook(ctx, "fund-1", "2026-08-05", "")
	if len(subset) != 1 || subset[0].EntryDate != "2026-08-07" {
		t.Fatalf("from-filter mismatch: %+v", subset)
	}
}
