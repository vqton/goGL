package cash_test

import (
	"context"
	"errors"
	"testing"

	domaincash "goGL/internal/domain/cash"
)

func TestService_ListVouchersFilter(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	for i := 0; i < 3; i++ {
		v := validVoucher()
		v.FundID = "fund-1"
		if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	// one posted
	first, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	approveVoucher(t, svc, first[0].ID, "giamdoc")
	if _, err := svc.PostVoucher(ctx, "thuquy", first[0].ID); err != nil {
		t.Fatalf("post: %v", err)
	}

	all, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1"})
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
	posted, _ := svc.ListVouchers(ctx, domaincash.VoucherFilter{FundID: "fund-1", State: domaincash.VoucherPosted})
	if len(posted) != 1 {
		t.Fatalf("expected 1 posted, got %d", len(posted))
	}
}

func TestService_CreateFundValidation(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()

	if err := svc.CreateFund(ctx, &domaincash.Fund{Currency: "VND", Account: "1111"}); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := svc.CreateFund(ctx, &domaincash.Fund{Name: "x", Currency: "VND"}); err == nil {
		t.Fatal("expected error for empty account")
	}
}

func TestService_CreateVoucher_CurrencyMismatch(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	v.Currency = "USD"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err == nil {
		t.Fatal("expected currency mismatch error")
	}
}

func TestService_UpdateVoucher_Immutable(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	mustCreateFund(t, svc, "fund-2")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}

	v.FundID = "fund-2"
	if err := svc.UpdateVoucher(ctx, "ketoan", v); err == nil {
		t.Fatal("expected error when fund changes")
	}
}

func TestService_ValidateVoucher_EdgeCases(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	cases := []struct {
		name string
		mut  func(*domaincash.Voucher)
	}{
		{"invalid type", func(v *domaincash.Voucher) { v.Type = "transfer" }},
		{"missing counterparty", func(v *domaincash.Voucher) { v.CounterpartyName = "" }},
		{"zero amount", func(v *domaincash.Voucher) { v.AmountMinor = 0 }},
		{"missing ref date", func(v *domaincash.Voucher) { v.RefDate = "" }},
		{"single line", func(v *domaincash.Voucher) { v.Lines = v.Lines[:1] }},
		{"zero line amount", func(v *domaincash.Voucher) {
			v.Lines[1].AmountMinor = 0
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := validVoucher()
			v.FundID = "fund-1"
			tc.mut(v)
			if err := svc.CreateVoucher(ctx, "ketoan", v); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestService_CloseDay_AlreadyClosed(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-04", 0, nil); err != nil {
		t.Fatalf("close empty day: %v", err)
	}
	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-04", 0, nil); err != domaincash.ErrPeriodClosed {
		t.Fatalf("expected ErrPeriodClosed, got %v", err)
	}
}

func TestService_CloseDay_NoopNotifierDifference(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	// counted > book exercises the negative Difference / formatMoney branch.
	count, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 6_000_000, nil)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if count.State != "open" || count.Difference != -1_000_000 {
		t.Fatalf("count mismatch: %+v", count)
	}
}

func TestService_CloseDay_InactiveFund(t *testing.T) {
	d := openSvcDB(t)
	svc, _ := newSvc(t, d)
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	if err := svc.CreateFund(ctx, &domaincash.Fund{ID: "fund-1", Name: "x", Currency: "VND", Account: "1111", IsActive: false}); err != nil {
		t.Fatalf("deactivate fund: %v", err)
	}
	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 0, nil); err != domaincash.ErrFundInactive {
		t.Fatalf("expected ErrFundInactive, got %v", err)
	}
}

type failNotifier struct{}

func (failNotifier) Notify(context.Context, string, string, string) error {
	return errors.New("notify failed")
}

func TestService_CloseDay_NotifierError(t *testing.T) {
	svc, _ := newSvcWithNotifier(t, openSvcDB(t), failNotifier{})
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	postReceive(t, svc, "fund-1", "2026-08-05", 5_000_000)

	if _, err := svc.CloseDay(ctx, "thuquy", "fund-1", "2026-08-05", 4_000_000, nil); err == nil {
		t.Fatal("expected notifier error to propagate")
	}
}
