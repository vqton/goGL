package ledger_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
)

// seedBookFixture posts a two-month golden fixture across three periods:
//
//	2026-07  E1  Dr 1111 10.000.000 / Cr 5111 10.000.000   (opening activity)
//	2026-08  E2  Dr 1111  5.000.000 / Cr 5111  5.000.000
//	2026-08  E3  Dr 1311  3.000.000 / Cr 5111  3.000.000
//	2026-08  E4  Dr 3311  2.000.000 / Cr 1111  2.000.000
//	2026-09  E5  Dr 1111  7.000.000 / Cr 5111  7.000.000   (outside range)
//
// E5 exists to prove range filters exclude later periods.
func seedBookFixture(t *testing.T, svc appledger.Service) {
	t.Helper()
	ctx := context.Background()

	mustCreateAccount(t, svc, &domainledger.Account{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "1311", Name: "Phải thu khách hàng", Type: domainledger.AccountAsset, Level: 2, AllowPost: true})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "3311", Name: "Phải trả người bán", Type: domainledger.AccountLiability, Level: 2, AllowPost: true})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "5111", Name: "Doanh thu bán hàng", Type: domainledger.AccountRevenue, Level: 2, AllowPost: true})

	post := func(date string, lines []domainledger.JournalLine) {
		t.Helper()
		e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{VoucherDate: date, Description: "BK " + date, Lines: lines})
		if err != nil {
			t.Fatalf("create %s: %v", date, err)
		}
		if _, err := svc.PostEntry(ctx, "ketoan", e.ID); err != nil {
			t.Fatalf("post %s: %v", date, err)
		}
	}

	post("2026-07-05", []domainledger.JournalLine{
		{LineNo: 1, AccountCode: "1111", Debit: 10_000_000},
		{LineNo: 2, AccountCode: "5111", Credit: 10_000_000},
	})
	post("2026-08-05", []domainledger.JournalLine{
		{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
		{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
	})
	post("2026-08-10", []domainledger.JournalLine{
		{LineNo: 1, AccountCode: "1311", Debit: 3_000_000},
		{LineNo: 2, AccountCode: "5111", Credit: 3_000_000},
	})
	post("2026-08-15", []domainledger.JournalLine{
		{LineNo: 1, AccountCode: "3311", Debit: 2_000_000},
		{LineNo: 2, AccountCode: "1111", Credit: 2_000_000},
	})
	post("2026-09-02", []domainledger.JournalLine{
		{LineNo: 1, AccountCode: "1111", Debit: 7_000_000},
		{LineNo: 2, AccountCode: "5111", Credit: 7_000_000},
	})
}

func TestBooks_GeneralJournal_Golden(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)

	got, err := svc.GetGeneralJournal(context.Background(), "2026-07", "2026-08", nil)
	if err != nil {
		t.Fatalf("general journal: %v", err)
	}
	// 4 entries × 2 lines = 8 rows; September (E5) must be excluded.
	if len(got.Rows) != 8 {
		t.Fatalf("rows = %d, want 8 (E5 outside range excluded)", len(got.Rows))
	}
	// Chronological by (VoucherDate, VoucherNo).
	wantSeq := []string{"2026-07-05", "2026-08-05", "2026-08-10", "2026-08-15"}
	for i, date := range wantSeq {
		if got.Rows[i*2].VoucherDate != date || got.Rows[i*2+1].VoucherDate != date {
			t.Fatalf("row %d: got dates %s/%s, want %s", i, got.Rows[i*2].VoucherDate, got.Rows[i*2+1].VoucherDate, date)
		}
	}
	if got.Rows[0].Debit != 10_000_000 || got.Rows[0].Contra != "5111" {
		t.Fatalf("E1 debit row = %+v, want Dr 1111 10.000.000 contra 5111", got.Rows[0])
	}
	if got.Rows[1].Credit != 10_000_000 || got.Rows[1].Contra != "1111" {
		t.Fatalf("E1 credit row = %+v, want Cr 5111 10.000.000 contra 1111", got.Rows[1])
	}
	if got.TotalDebit != 20_000_000 || got.TotalCredit != 20_000_000 {
		t.Fatalf("totals = %d/%d, want 20.000.000/20.000.000 (ΣNợ = ΣCó)", got.TotalDebit, got.TotalCredit)
	}
}

func TestBooks_LedgerCard_OpeningCarriedForward(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)
	ctx := context.Background()

	// Sổ Cái 1111, August only: opening must carry the July activity.
	got, err := svc.GetLedgerBook(ctx, "1111", "2026-08", "2026-08", nil)
	if err != nil {
		t.Fatalf("ledger card: %v", err)
	}
	if got.AccountName != "Tiền mặt VND" {
		t.Fatalf("account name = %q", got.AccountName)
	}
	if got.OpenDebit != 10_000_000 || got.OpenCredit != 0 {
		t.Fatalf("opening = %d/%d, want Dr 10.000.000 (carried from July)", got.OpenDebit, got.OpenCredit)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (E2, E4)", len(got.Rows))
	}
	// E2 Dr 1111 5.000.000 — running balance 15.000.000, contra 5111.
	if got.Rows[0].Debit != 5_000_000 || got.Rows[0].Balance != 15_000_000 || got.Rows[0].Contra != "5111" {
		t.Fatalf("E2 row = %+v, want Dr 5.000.000 balance 15.000.000 contra 5111", got.Rows[0])
	}
	// E4 Cr 1111 2.000.000 — running balance back to 13.000.000, contra 3311.
	if got.Rows[1].Credit != 2_000_000 || got.Rows[1].Balance != 13_000_000 || got.Rows[1].Contra != "3311" {
		t.Fatalf("E4 row = %+v, want Cr 2.000.000 balance 13.000.000 contra 3311", got.Rows[1])
	}
	if got.TotalDebit != 5_000_000 || got.TotalCredit != 2_000_000 {
		t.Fatalf("activity = %d/%d, want 5.000.000/2.000.000", got.TotalDebit, got.TotalCredit)
	}
	if got.CloseDebit != 13_000_000 || got.CloseCredit != 0 {
		t.Fatalf("closing = %d/%d, want Dr 13.000.000", got.CloseDebit, got.CloseCredit)
	}

	// Credit-natured account: opening and closing land on the Có side.
	rev, err := svc.GetLedgerBook(ctx, "5111", "2026-07", "2026-08", nil)
	if err != nil {
		t.Fatalf("revenue card: %v", err)
	}
	if rev.OpenDebit != 0 || rev.OpenCredit != 0 {
		t.Fatalf("revenue opening = %d/%d, want 0 (nothing before July)", rev.OpenDebit, rev.OpenCredit)
	}
	if rev.TotalCredit != 18_000_000 || rev.TotalDebit != 0 {
		t.Fatalf("revenue activity = %d/%d, want 0/18.000.000", rev.TotalDebit, rev.TotalCredit)
	}
	if rev.CloseDebit != 0 || rev.CloseCredit != 18_000_000 {
		t.Fatalf("revenue closing = %d/%d, want Có 18.000.000", rev.CloseDebit, rev.CloseCredit)
	}
}

func TestBooks_Detail_NoRunningBalance(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)
	ctx := context.Background()

	got, err := svc.GetDetailBook(ctx, "1111", "2026-08", "2026-08", nil)
	if err != nil {
		t.Fatalf("detail book: %v", err)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(got.Rows))
	}
	for _, r := range got.Rows {
		if r.Balance != 0 {
			t.Fatalf("detail row carries a running balance %d; Sổ chi tiết has no balance column", r.Balance)
		}
	}
	if got.OpenDebit != 10_000_000 || got.CloseDebit != 13_000_000 {
		t.Fatalf("detail opening/closing = %d/%d, want 10.000.000/13.000.000", got.OpenDebit, got.CloseDebit)
	}
}

func TestBooks_TrialBalance_Golden(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)

	tb, err := svc.GetTrialBalance(context.Background(), "2026-08", nil)
	if err != nil {
		t.Fatalf("trial balance: %v", err)
	}
	if len(tb.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(tb.Rows))
	}
	wantRows := map[string]domainledger.TrialBalanceRow{
		"1111": {
			Open:     domainledger.Balance{Debit: 10_000_000},
			Activity: domainledger.Balance{Debit: 5_000_000, Credit: 2_000_000},
			Close:    domainledger.Balance{Debit: 13_000_000},
		},
		"1311": {
			Activity: domainledger.Balance{Debit: 3_000_000},
			Close:    domainledger.Balance{Debit: 3_000_000},
		},
		"3311": {
			Activity: domainledger.Balance{Debit: 2_000_000},
			Close:    domainledger.Balance{Debit: 2_000_000},
		},
		"5111": {
			Open:     domainledger.Balance{Credit: 10_000_000},
			Activity: domainledger.Balance{Credit: 8_000_000},
			Close:    domainledger.Balance{Credit: 18_000_000},
		},
	}
	seen := map[string]bool{}
	for _, r := range tb.Rows {
		seen[r.AccountCode] = true
		want := wantRows[r.AccountCode]
		if r.Open != want.Open || r.Activity != want.Activity || r.Close != want.Close {
			t.Fatalf("row %s = %+v, want %+v", r.AccountCode, r, want)
		}
	}
	for code := range wantRows {
		if !seen[code] {
			t.Fatalf("missing trial balance row for %s", code)
		}
	}

	tot := tb.Totals
	if tot.Open != (domainledger.Balance{Debit: 10_000_000, Credit: 10_000_000}) {
		t.Fatalf("open totals = %+v, want 10.000.000/10.000.000", tot.Open)
	}
	if tot.Activity != (domainledger.Balance{Debit: 10_000_000, Credit: 10_000_000}) {
		t.Fatalf("activity totals = %+v, want 10.000.000/10.000.000", tot.Activity)
	}
	if tot.Close != (domainledger.Balance{Debit: 18_000_000, Credit: 18_000_000}) {
		t.Fatalf("close totals = %+v, want 18.000.000/18.000.000", tot.Close)
	}
	if !tb.Balanced {
		t.Fatal("trial balance must be balanced (ΣNợ = ΣCó on every column)")
	}
}

func TestBooks_EmptyRange_OpeningBalanceOnly(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)
	ctx := context.Background()

	// 1311 has no September activity: the book renders its carried balance only.
	got, err := svc.GetLedgerBook(ctx, "1311", "2026-09", "2026-09", nil)
	if err != nil {
		t.Fatalf("ledger card: %v", err)
	}
	if len(got.Rows) != 0 {
		t.Fatalf("rows = %d, want 0 (no activity in range)", len(got.Rows))
	}
	if got.OpenDebit != 3_000_000 || got.CloseDebit != 3_000_000 {
		t.Fatalf("opening/closing = %d/%d, want 3.000.000 carried", got.OpenDebit, got.CloseDebit)
	}
}

func TestBooks_Validation(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	seedBookFixture(t, svc)
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
		want error
	}{
		{"journal bad from", func() error {
			_, err := svc.GetGeneralJournal(ctx, "2026-8", "2026-08", nil)
			return err
		}, domainledger.ErrInvalidPeriod},
		{"journal from after to", func() error {
			_, err := svc.GetGeneralJournal(ctx, "2026-09", "2026-08", nil)
			return err
		}, domainledger.ErrInvalidRange},
		{"ledger bad period", func() error {
			_, err := svc.GetLedgerBook(ctx, "1111", "2026-08", "abc", nil)
			return err
		}, domainledger.ErrInvalidPeriod},
		{"ledger unknown account", func() error {
			_, err := svc.GetLedgerBook(ctx, "9999", "2026-08", "2026-08", nil)
			return err
		}, domainledger.ErrAccountNotFound},
		{"trial balance bad period", func() error {
			_, err := svc.GetTrialBalance(ctx, "2026-13", nil)
			return err
		}, domainledger.ErrInvalidPeriod},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// TestBooks_TotalsAlwaysBalance is a small property test over randomized
// balanced entries: for every period and book, the totals must balance.
func TestBooks_TotalsAlwaysBalance(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()

	for _, a := range []*domainledger.Account{
		{Code: "1111", Name: "Tiền mặt", Type: domainledger.AccountAsset, Level: 2, AllowPost: true},
		{Code: "1311", Name: "Phải thu", Type: domainledger.AccountAsset, Level: 2, AllowPost: true},
		{Code: "1561", Name: "Hàng hóa", Type: domainledger.AccountAsset, Level: 2, AllowPost: true},
		{Code: "3311", Name: "Phải trả", Type: domainledger.AccountLiability, Level: 2, AllowPost: true},
		{Code: "5111", Name: "Doanh thu", Type: domainledger.AccountRevenue, Level: 2, AllowPost: true},
		{Code: "6321", Name: "Giá vốn", Type: domainledger.AccountExpense, Level: 2, AllowPost: true},
	} {
		mustCreateAccount(t, svc, a)
	}

	rng := rand.New(rand.NewSource(42))
	debitAccts := []string{"1111", "1311", "1561", "6321"}
	creditAccts := []string{"3311", "5111"}
	for i := 0; i < 40; i++ {
		period := "2026-0" + fmt.Sprint(7+(i%3))
		day := 1 + rng.Intn(28)
		amount := int64(100_000 + rng.Intn(9_900_000))
		lines := []domainledger.JournalLine{
			{LineNo: 1, AccountCode: debitAccts[rng.Intn(len(debitAccts))], Debit: amount},
			{LineNo: 2, AccountCode: creditAccts[rng.Intn(len(creditAccts))], Credit: amount},
		}
		e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
			VoucherDate: fmt.Sprintf("%s-%02d", period, day),
			Description: "random",
			Lines:       lines,
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		if _, err := svc.PostEntry(ctx, "ketoan", e.ID); err != nil {
			t.Fatalf("post %d: %v", i, err)
		}
	}

	for _, period := range []string{"2026-07", "2026-08", "2026-09"} {
		tb, err := svc.GetTrialBalance(ctx, period, nil)
		if err != nil {
			t.Fatalf("trial balance %s: %v", period, err)
		}
		if !tb.Balanced {
			t.Fatalf("%s: totals out of balance: %+v", period, tb.Totals)
		}
		if tb.Totals.Open.Debit != tb.Totals.Open.Credit ||
			tb.Totals.Activity.Debit != tb.Totals.Activity.Credit ||
			tb.Totals.Close.Debit != tb.Totals.Close.Credit {
			t.Fatalf("%s: column totals differ: %+v", period, tb.Totals)
		}
	}

	gj, err := svc.GetGeneralJournal(ctx, "2026-07", "2026-09", nil)
	if err != nil {
		t.Fatalf("general journal: %v", err)
	}
	if gj.TotalDebit != gj.TotalCredit {
		t.Fatalf("journal totals differ: %d/%d", gj.TotalDebit, gj.TotalCredit)
	}
	if len(gj.Rows) != 80 {
		t.Fatalf("journal rows = %d, want 80 (40 entries × 2 lines)", len(gj.Rows))
	}
}

// BenchmarkBooks_12Months50k measures the read-model at the spec's scale
// ceiling (§6): 50,000 posted entries across 12 periods. P3.2 acceptance is a
// book render under 2s. Seeding is bulk SQL (not the repo write path), since
// the benchmark targets the render, not the write path. Each book is measured
// on its own. Run with:
//
//	go test ./internal/application/ledger -run '^$' -bench Books -benchtime=1x
func benchStore(b *testing.B, n int) appledger.Service {
	b.Helper()
	ctx := b.Context()
	d, err := sql.Open("sqlite", "file:bench_books?mode=memory&cache=shared")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	b.Cleanup(func() { _ = d.Close() })
	if err := db.Migrate(ctx, d); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	if err := seedBenchStore(ctx, b, d, n); err != nil {
		b.Fatalf("seed: %v", err)
	}
	return appledger.NewService(persledger.NewSqliteRepository(d))
}

func BenchmarkBooks_GeneralJournal_12Months50k(b *testing.B) {
	svc := benchStore(b, 50_000)
	ctx := b.Context()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetGeneralJournal(ctx, "2026-01", "2026-12", nil); err != nil {
			b.Fatalf("general journal: %v", err)
		}
	}
}

func BenchmarkBooks_LedgerBook_12Months50k(b *testing.B) {
	svc := benchStore(b, 50_000)
	ctx := b.Context()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetLedgerBook(ctx, "1111", "2026-01", "2026-12", nil); err != nil {
			b.Fatalf("ledger book: %v", err)
		}
	}
}

func BenchmarkBooks_TrialBalance_12Months50k(b *testing.B) {
	svc := benchStore(b, 50_000)
	ctx := b.Context()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetTrialBalance(ctx, "2026-12", nil); err != nil {
			b.Fatalf("trial balance: %v", err)
		}
	}
}

// seedBenchStore bulk-loads two postable accounts and n posted 1111/5111
// entries spread across 12 months, each batch a single multi-row INSERT.
func seedBenchStore(ctx context.Context, b *testing.B, d *sql.DB, n int) error {
	b.Helper()
	for _, a := range []*domainledger.Account{
		{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true},
		{Code: "5111", Name: "Doanh thu bán hàng", Type: domainledger.AccountRevenue, Level: 2, AllowPost: true},
	} {
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if _, err := d.ExecContext(ctx,
			`INSERT INTO ledger_accounts (id, data) VALUES (?, ?)`,
			domainledger.RowID("account", a.Code), string(data)); err != nil {
			return err
		}
	}

	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06",
		"2026-07", "2026-08", "2026-09", "2026-10", "2026-11", "2026-12"}
	const batch = 100
	var placeholders []string
	var vals []any
	perPeriod := map[string]int{}
	flush := func() error {
		if len(vals) == 0 {
			return nil
		}
		q := `INSERT INTO ledger_journals (id, data) VALUES ` + strings.Join(placeholders, ", ")
		if _, err := d.ExecContext(ctx, q, vals...); err != nil {
			return err
		}
		placeholders = placeholders[:0]
		vals = vals[:0]
		return nil
	}
	for i := 0; i < n; i++ {
		period := months[i%len(months)]
		perPeriod[period]++
		e := &domainledger.JournalEntry{
			ID:          fmt.Sprintf("bench-%07d", i),
			VoucherDate: fmt.Sprintf("%s-%02d", period, 1+i%28),
			Period:      period,
			Source:      domainledger.SourceManual,
			Status:      domainledger.EntryPosted,
			VoucherNo:   domainledger.FormatVoucherNo("PK", int64(perPeriod[period]), period),
			Description: "Benchmark entry",
			Lines: []domainledger.JournalLine{
				{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
				{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
			},
		}
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		placeholders = append(placeholders, "(?, ?)")
		vals = append(vals, e.ID, string(data))
		if len(vals) >= batch*2 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}
