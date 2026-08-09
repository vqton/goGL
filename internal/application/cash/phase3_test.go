package cash_test

import (
	"context"
	"testing"

	domaincash "goGL/internal/domain/cash"
)

func TestService_ReconcileMonth_Resolved(t *testing.T) {
	svc, a := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)
	postReceive(t, svc, "fund-1", "2026-08-20", 3_000_000)

	rec, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 8_000_000, []string{"thuquy", "ketoan", "giamdoc"})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if rec.State != "resolved" || rec.Difference != 0 || rec.CashierBalance != 8_000_000 || rec.AccountantBalance != 8_000_000 {
		t.Fatalf("reconciliation mismatch: %+v", rec)
	}
	if len(rec.SignedBy) != 3 || rec.SignedBy[0] != "thuquy" || rec.SignedBy[1] != "ketoan" || rec.SignedBy[2] != "giamdoc" {
		t.Fatalf("signature mismatch: %+v", rec.SignedBy)
	}

	// All posted vouchers in the period are now reconciled.
	vouchers, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	for _, v := range vouchers {
		if v.State != domaincash.VoucherReconciled {
			t.Fatalf("expected reconciled, got %s: %+v", v.State, v)
		}
	}

	if a.logs[len(a.logs)-1].Action != "cash.reconcile" {
		t.Fatalf("expected cash.reconcile audit, got %+v", a.logs[len(a.logs)-1])
	}
}

func TestService_ReconcileMonth_ClosesPeriod(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	if _, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 5_000_000, []string{"thuquy", "ketoan", "giamdoc"}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// BR9: posting into the closed period is blocked.
	v := validVoucher()
	v.FundID = "fund-1"
	v.RefDate = "2026-08-25"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != domaincash.ErrPeriodClosed {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestService_ReconcileMonth_Difference(t *testing.T) {
	svc, a := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	rec, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 4_000_000, []string{"thuquy", "ketoan", "giamdoc"})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if rec.State != "diff" || rec.Difference != 1_000_000 {
		t.Fatalf("reconciliation mismatch: %+v", rec)
	}
	if len(rec.SignedBy) != 3 {
		t.Fatalf("expected 3 signers, got %+v", rec.SignedBy)
	}
	if a.logs[len(a.logs)-1].Action != "cash.reconcile.diff" {
		t.Fatalf("expected cash.reconcile.diff audit, got %+v", a.logs[len(a.logs)-1])
	}
}

func TestService_ReconcileMonth_OpenCountBlocks(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	// Open count via a mismatched daily close.
	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := svc.ReconcileMonth(ctx, "ketoan", "fund-1", "2026-08", 5_000_000, []string{"thuquy", "ketoan", "giamdoc"}); err != domaincash.ErrOpenCountPending {
		t.Fatalf("expected ErrOpenCountPending, got %v", err)
	}
}

func TestService_VoidVoucher_DraftAndApproved(t *testing.T) {
	svc, a := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Void draft: no balance impact, no reversal.
	got, err := svc.VoidVoucher(ctx, "ketoan", v.ID, "nhập sai ngày")
	if err != nil {
		t.Fatalf("void draft: %v", err)
	}
	if got.State != domaincash.VoucherVoided || got.VoidReason != "nhập sai ngày" {
		t.Fatalf("voided draft mismatch: %+v", got)
	}
	entries, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if len(entries) != 0 {
		t.Fatalf("void draft must not touch the cash book, got %d entries", len(entries))
	}
	if a.logs[len(a.logs)-1].Action != "voucher.void" {
		t.Fatalf("expected voucher.void audit, got %+v", a.logs[len(a.logs)-1])
	}

	// Void approved also direct.
	v2 := validVoucher()
	v2.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v2); err != nil {
		t.Fatalf("create: %v", err)
	}
	approveVoucher(t, svc, v2.ID, "giamdoc")
	got, err = svc.VoidVoucher(ctx, "ketoan", v2.ID, "hủy trước khi ghi sổ")
	if err != nil {
		t.Fatalf("void approved: %v", err)
	}
	if got.State != domaincash.VoucherVoided {
		t.Fatalf("voided approved mismatch: %+v", got)
	}
}

func TestService_VoidVoucher_Posted_ReversalPair(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	vouchers, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	orig := vouchers[0]

	got, err := svc.VoidVoucher(ctx, "ketoan", orig.ID, "sai tài khoản đối ứng")
	if err != nil {
		t.Fatalf("void posted: %v", err)
	}
	if got.State != domaincash.VoucherVoided || len(got.RefVouchers) != 1 {
		t.Fatalf("voided voucher mismatch: %+v", got)
	}

	reversalID := got.RefVouchers[0]
	rev, err := svc.GetVoucher(ctx, reversalID)
	if err != nil {
		t.Fatalf("get reversal: %v", err)
	}
	if rev.Type != domaincash.VoucherPay || rev.AmountMinor != 5_000_000 {
		t.Fatalf("reversal mismatch: %+v", rev)
	}
	if len(rev.RefVouchers) != 1 || rev.RefVouchers[0] != orig.ID {
		t.Fatalf("reversal link mismatch: %+v", rev.RefVouchers)
	}
	if rev.State != domaincash.VoucherPosted && rev.State != domaincash.VoucherVoided {
		t.Fatalf("unexpected reversal state: %s", rev.State)
	}

	// Balance restored to zero; two book entries, the reversal offsets.
	entries, _ := svc.GetCashBook(ctx, "fund-1", "", "")
	if len(entries) != 2 {
		t.Fatalf("expected 2 book entries, got %d", len(entries))
	}
	if last := entries[len(entries)-1]; last.Balance != 0 || last.Pay != 5_000_000 {
		t.Fatalf("balance not restored: %+v", last)
	}
}

func TestService_VoidVoucher_Posted_ReversalMismatch(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	vouchers, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	orig := vouchers[0]

	// Pre-create a wrong-amount reversal draft linked to the original (E2).
	rev := validVoucher()
	rev.FundID = "fund-1"
	rev.Type = domaincash.VoucherPay
	rev.AmountMinor = 2_000_000
	rev.RefVouchers = []string{orig.ID}
	rev.Description = "reversal sai số tiền"
	rev.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "152", AmountMinor: 2_000_000},
		{Seq: 2, CreditAcc: "1111", AmountMinor: 2_000_000},
	}
	if err := svc.CreateVoucher(ctx, "ketoan", rev); err != nil {
		t.Fatalf("create reversal: %v", err)
	}

	if _, err := svc.VoidVoucher(ctx, "ketoan", orig.ID, "hủy"); err != domaincash.ErrReversalMismatch {
		t.Fatalf("expected ErrReversalMismatch, got %v", err)
	}
}

func TestService_VoidVoucher_NotFound(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	if _, err := svc.VoidVoucher(ctx, "ketoan", "nope", "x"); err != domaincash.ErrVoucherNotFound {
		t.Fatalf("expected ErrVoucherNotFound, got %v", err)
	}
}

func TestService_VoidVoucher_Posted_GuardAgainstSpentFunds(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	// Spend the balance.
	pay := validVoucher()
	pay.FundID = "fund-1"
	pay.Type = domaincash.VoucherPay
	pay.AmountMinor = 5_000_000
	pay.Description = "chi hết"
	pay.CounterpartyName = "X"
	pay.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "152", AmountMinor: 5_000_000},
		{Seq: 2, CreditAcc: "1111", AmountMinor: 5_000_000},
	}
	if err := svc.CreateVoucher(ctx, "ketoan", pay); err != nil {
		t.Fatalf("create pay: %v", err)
	}
	if _, err := svc.ApproveVoucher(ctx, "giamdoc", pay.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := svc.PostVoucher(ctx, "thuquy", pay.ID); err != nil {
		t.Fatalf("post: %v", err)
	}

	vouchers, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1", State: domaincash.VoucherPosted})
	if len(vouchers) != 2 {
		t.Fatalf("expected 2 posted, got %d", len(vouchers))
	}
	// Voiding the receive after the funds were spent must be rejected: the
	// offsetting pay-reversal would push the balance negative (BR2).
	var receiveID string
	for _, v := range vouchers {
		if v.Type == domaincash.VoucherReceive {
			receiveID = v.ID
		}
	}
	if _, err := svc.VoidVoucher(ctx, "ketoan", receiveID, "hủy"); err != domaincash.ErrNegativeBalance {
		t.Fatalf("expected ErrNegativeBalance, got %v", err)
	}
}
