package cash_test

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/cash"
	"goGL/internal/infrastructure/db"
	perscash "goGL/internal/infrastructure/persistence/cash"
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

func newRepo(t *testing.T, d *sql.DB) cash.Repository {
	t.Helper()
	return perscash.NewSqliteRepository(d)
}

func TestRepository_FundRoundTrip(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	in := &cash.Fund{ID: "fund-1", Name: "Quỹ VND", Currency: "VND", Account: "1111", IsActive: true}
	if err := repo.CreateFund(ctx, in); err != nil {
		t.Fatalf("create fund: %v", err)
	}

	got, err := repo.GetFund(ctx, "fund-1")
	if err != nil {
		t.Fatalf("get fund: %v", err)
	}
	if got.Name != in.Name || got.Currency != in.Currency || !got.IsActive {
		t.Fatalf("fund round trip mismatch: %+v", got)
	}

	list, err := repo.ListFunds(ctx)
	if err != nil {
		t.Fatalf("list funds: %v", err)
	}
	if len(list) != 1 || list[0].ID != "fund-1" {
		t.Fatalf("expected 1 fund, got %+v", list)
	}
}

func TestRepository_VoucherRoundTrip(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	v := &cash.Voucher{
		ID:               "v-1",
		RefNo:            "PT/2026-08/000001",
		RefDate:          "2026-08-05",
		Type:             cash.VoucherReceive,
		FundID:           "fund-1",
		Currency:         "VND",
		AmountMinor:      5_000_000,
		AmountWords:      "năm triệu đồng",
		CounterpartyType: "customer",
		CounterpartyID:   "kh-1",
		CounterpartyName: "Công ty ABC",
		Description:      "Thu tiền bán hàng",
		Lines: []cash.VoucherLine{
			{Seq: 1, DebitAcc: "1111", CreditAcc: "131", AmountMinor: 5_000_000, ObjectID: "kh-1"},
		},
		State:     cash.VoucherDraft,
		CreatedBy: "ketoan",
	}
	if err := repo.CreateVoucher(ctx, v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}

	got, err := repo.GetVoucher(ctx, "v-1")
	if err != nil {
		t.Fatalf("get voucher: %v", err)
	}
	if got.RefNo != v.RefNo || got.AmountMinor != v.AmountMinor || got.State != cash.VoucherDraft {
		t.Fatalf("voucher round trip mismatch: %+v", got)
	}
	if len(got.Lines) != 1 || got.Lines[0].DebitAcc != "1111" {
		t.Fatalf("lines mismatch: %+v", got.Lines)
	}

	if _, err := repo.GetVoucher(ctx, "missing"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for missing voucher, got %v", err)
	}
}

func TestRepository_UpdateVoucher(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	v := &cash.Voucher{ID: "v-1", FundID: "fund-1", AmountMinor: 100, State: cash.VoucherDraft}
	if err := repo.CreateVoucher(ctx, v); err != nil {
		t.Fatalf("create: %v", err)
	}
	v.AmountMinor = 200
	v.State = cash.VoucherApproved
	if err := repo.UpdateVoucher(ctx, v); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetVoucher(ctx, "v-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AmountMinor != 200 || got.State != cash.VoucherApproved {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestRepository_ListVouchersFilter(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	for _, v := range []*cash.Voucher{
		{ID: "v-1", FundID: "f1", RefDate: "2026-08-01", Type: cash.VoucherReceive, State: cash.VoucherDraft},
		{ID: "v-2", FundID: "f1", RefDate: "2026-08-02", Type: cash.VoucherPay, State: cash.VoucherApproved},
		{ID: "v-3", FundID: "f2", RefDate: "2026-08-03", Type: cash.VoucherReceive, State: cash.VoucherDraft},
	} {
		if err := repo.CreateVoucher(ctx, v); err != nil {
			t.Fatalf("create %s: %v", v.ID, err)
		}
	}

	got, err := repo.ListVouchers(ctx, cash.VoucherFilter{FundID: "f1"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 vouchers for f1, got %d", len(got))
	}

	got, err = repo.ListVouchers(ctx, cash.VoucherFilter{FundID: "f1", State: cash.VoucherApproved})
	if err != nil {
		t.Fatalf("list by state: %v", err)
	}
	if len(got) != 1 || got[0].ID != "v-2" {
		t.Fatalf("expected only v-2, got %+v", got)
	}

	got, err = repo.ListVouchers(ctx, cash.VoucherFilter{FundID: "f1", From: "2026-08-01", To: "2026-08-01"})
	if err != nil {
		t.Fatalf("list by date: %v", err)
	}
	if len(got) != 1 || got[0].ID != "v-1" {
		t.Fatalf("expected only v-1 by date, got %+v", got)
	}
}

func TestRepository_NextRefNoSequential(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	refs := make([]string, 0, 5)
	for i := 0; i < 3; i++ {
		ref, err := repo.NextRefNo(ctx, "f1", "2026-08", cash.VoucherReceive)
		if err != nil {
			t.Fatalf("next ref: %v", err)
		}
		refs = append(refs, ref)
	}
	want := []string{"PT/2026-08/000001", "PT/2026-08/000002", "PT/2026-08/000003"}
	for i, ref := range refs {
		if ref != want[i] {
			t.Fatalf("ref[%d] = %q, want %q", i, ref, want[i])
		}
	}

	// Different fund and type each start their own sequence.
	if ref, err := repo.NextRefNo(ctx, "f2", "2026-08", cash.VoucherReceive); err != nil || ref != "PT/2026-08/000001" {
		t.Fatalf("f2 first ref = %q, err=%v", ref, err)
	}
	if ref, err := repo.NextRefNo(ctx, "f1", "2026-08", cash.VoucherPay); err != nil || ref != "PC/2026-08/000001" {
		t.Fatalf("f1 PC first ref = %q, err=%v", ref, err)
	}
}

func TestRepository_NextRefNoParallelUnique(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	const n = 20
	refs := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ref, err := repo.NextRefNo(ctx, "f1", "2026-08", cash.VoucherReceive)
			if err != nil {
				t.Errorf("next ref: %v", err)
				return
			}
			refs[i] = ref
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for _, ref := range refs {
		if ref == "" {
			t.Fatal("empty ref returned")
		}
		if seen[ref] {
			t.Fatalf("duplicate ref: %s", ref)
		}
		seen[ref] = true
	}
	for i := 1; i <= n; i++ {
		want := fmt.Sprintf("PT/2026-08/%06d", i)
		if !seen[want] {
			t.Fatalf("missing continuous ref %s; got %v", want, refs)
		}
	}
}

func TestRepository_CashBookAppendList(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	entries := []*cash.CashBookEntry{
		{ID: "b-1", FundID: "f1", EntryDate: "2026-08-05", RefNo: "PT/2026-08/000001", Type: cash.VoucherReceive, Receive: 5_000_000, Balance: 5_000_000},
		{ID: "b-2", FundID: "f1", EntryDate: "2026-08-06", RefNo: "PC/2026-08/000001", Type: cash.VoucherPay, Pay: 2_000_000, Balance: 3_000_000},
		{ID: "b-3", FundID: "f2", EntryDate: "2026-08-06", RefNo: "PT/2026-08/000001", Type: cash.VoucherReceive, Receive: 100, Balance: 100},
	}
	for _, e := range entries {
		if err := repo.AppendCashBookEntry(ctx, e); err != nil {
			t.Fatalf("append %s: %v", e.ID, err)
		}
	}

	rows, err := repo.ListCashBook(ctx, "f1", "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows for f1, got %d", len(rows))
	}
	if rows[0].Balance != 5_000_000 || rows[1].Balance != 3_000_000 {
		t.Fatalf("balance order mismatch: %+v", rows)
	}
}

func TestRepository_CashCountRoundTrip(t *testing.T) {
	repo := newRepo(t, openTestDB(t))
	ctx := context.Background()

	c := &cash.CashCount{
		ID: "count-1", FundID: "f1", CountDate: "2026-08-31",
		BookBalance: 3_000_000, CountedAmount: 3_000_000, State: "open",
		Participants: []string{"truongquy", "kttruong"},
	}
	if err := repo.CreateCashCount(ctx, c); err != nil {
		t.Fatalf("create count: %v", err)
	}

	list, err := repo.ListCashCounts(ctx, "f1")
	if err != nil {
		t.Fatalf("list counts: %v", err)
	}
	if len(list) != 1 || list[0].CountedAmount != 3_000_000 {
		t.Fatalf("count round trip mismatch: %+v", list)
	}
	if len(list[0].Participants) != 2 || list[0].Participants[0] != "truongquy" {
		t.Fatalf("participants mismatch: %+v", list[0].Participants)
	}
}
