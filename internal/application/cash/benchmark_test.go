package cash_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	appcash "goGL/internal/application/cash"
	domaincash "goGL/internal/domain/cash"
	"goGL/internal/infrastructure/db"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

// openBenchDB mirrors openSvcDB but without the testing t for benchmarks.
func openBenchDB() *sql.DB {
	d, err := sql.Open("sqlite", "file:bench?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}
	d.SetMaxOpenConns(1)
	if err := db.Migrate(context.Background(), d); err != nil {
		panic(err)
	}
	return d
}

func benchVoucher(fundID string, i int) *domaincash.Voucher {
	amount := int64(1_000_000 + i)
	return &domaincash.Voucher{
		FundID: fundID, RefDate: "2026-08-05", Type: domaincash.VoucherReceive,
		Currency: "VND", AmountMinor: amount,
		CounterpartyName: fmt.Sprintf("KH-%d", i), Description: "Thu tiền",
		Lines: []domaincash.VoucherLine{
			{Seq: 1, DebitAcc: "1111", AmountMinor: amount},
			{Seq: 2, CreditAcc: "5111", AmountMinor: amount},
		},
	}
}

// BenchmarkPostVoucher measures the full create→approve→post path. The state
// machine only allows one post per voucher, so each iteration builds its own.
func BenchmarkPostVoucher(b *testing.B) {
	ctx := context.Background()
	svc := appcash.NewService(perscash.NewSqliteRepository(openBenchDB()), noopAuditor{})
	if err := svc.CreateFund(ctx, &domaincash.Fund{ID: "bench", Name: "Q", Currency: "VND", Account: "1111", IsActive: true}); err != nil {
		b.Fatalf("fund: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := benchVoucher("bench", i)
		if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
			b.Fatalf("create: %v", err)
		}
		if _, err := svc.ApproveVoucher(ctx, "kttruong", v.ID); err != nil {
			b.Fatalf("approve: %v", err)
		}
		if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != nil {
			b.Fatalf("post: %v", err)
		}
	}
}

// BenchmarkGetCashBook12Month measures a 12-month book query. Entries are
// seeded once per run with 30 business days x 5 vouchers per day (~1800 rows).
func BenchmarkGetCashBook12Month(b *testing.B) {
	ctx := context.Background()
	svc := appcash.NewService(perscash.NewSqliteRepository(openBenchDB()), noopAuditor{})
	if err := svc.CreateFund(ctx, &domaincash.Fund{ID: "bench", Name: "Q", Currency: "VND", Account: "1111", IsActive: true}); err != nil {
		b.Fatalf("fund: %v", err)
	}

	const days = 30 * 12 // one entry per voucher across 12 months
	for i := 0; i < days*5; i++ {
		v := benchVoucher("bench", i)
		v.RefDate = fmt.Sprintf("2026-%02d-%02d", i/150%12+1, 1+i%28)
		if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
			b.Fatalf("create: %v", err)
		}
		if _, err := svc.ApproveVoucher(ctx, "kttruong", v.ID); err != nil {
			b.Fatalf("approve: %v", err)
		}
		if _, err := svc.PostVoucher(ctx, "thuquy", v.ID); err != nil {
			b.Fatalf("post: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetCashBook(ctx, "bench", "2026-01-01", "2026-12-31"); err != nil {
			b.Fatalf("book: %v", err)
		}
	}
}
