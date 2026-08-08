package print

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goGL/internal/domain/cash"
)

var update = flag.Bool("update", false, "rewrite golden fixtures")

// golden compares normalized output against a frozen fixture in testdata/.
// Run with -update to (re)generate fixtures, then review the diff.
func golden(t *testing.T, name, got string) {
	t.Helper()
	got = normalize(got)
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update)", path, err)
	}
	if got != normalize(string(want)) {
		t.Errorf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, normalize(string(want)))
	}
}

func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func testFund() *cash.Fund {
	return &cash.Fund{
		ID:          "fund_vnd",
		Name:        "Quỹ tiền mặt VND",
		Currency:    "VND",
		Account:     "1111",
		Description: "Quỹ chính",
		IsActive:    true,
	}
}

func TestVoucherForm_Receive_01TT(t *testing.T) {
	v := &cash.Voucher{
		ID:               "v1",
		RefNo:            "PT/2026-08/000001",
		RefDate:          "2026-08-25",
		Type:             cash.VoucherReceive,
		FundID:           "fund_vnd",
		Currency:         "VND",
		AmountMinor:      123045067,
		AmountWords:      cash.AmountInWords(123045067),
		CounterpartyName: "Nguyễn Văn A",
		Description:      "Thu tiền bán hàng",
		Lines: []cash.VoucherLine{
			{Seq: 1, DebitAcc: "1111", CreditAcc: "5111", AmountMinor: 123045067},
		},
		CreatedBy:  "cashier01",
		ApprovedBy: "acc01",
		PostedBy:   "cashier01",
	}
	got, err := VoucherForm(v, testFund(), "01")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "voucher_receive_01tt", got)
}

func TestVoucherForm_Pay_02TT(t *testing.T) {
	v := &cash.Voucher{
		ID:               "v2",
		RefNo:            "PC/2026-08/000002",
		RefDate:          "2026-08-05",
		Type:             cash.VoucherPay,
		FundID:           "fund_vnd",
		Currency:         "VND",
		AmountMinor:      2000000,
		AmountWords:      cash.AmountInWords(2000000),
		FXRate:           0,
		CounterpartyName: "Công ty TNHH XYZ",
		Description:      "Chi thanh toán tiền mua văn phòng phẩm",
		Lines: []cash.VoucherLine{
			{Seq: 1, DebitAcc: "642", CreditAcc: "1111", AmountMinor: 2000000},
		},
		CreatedBy:  "cashier01",
		ApprovedBy: "acc01",
		PostedBy:   "cashier01",
	}
	got, err := VoucherForm(v, testFund(), "01")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "voucher_pay_02tt", got)
}

func s07Data() S07Data {
	return S07Data{
		Fund:    testFund(),
		Opening: 5000000,
		Year:    "2026",
		Entries: []*cash.CashBookEntry{
			{
				ID: "e1", EntryDate: "2026-08-01", VoucherDate: "2026-08-01",
				RefNo: "PT/2026-08/000001", Type: cash.VoucherReceive,
				Description: "Thu tiền bán hàng", Receive: 5000000, Pay: 0, Balance: 10000000,
			},
			{
				ID: "e2", EntryDate: "2026-08-05", VoucherDate: "2026-08-05",
				RefNo: "PC/2026-08/000002", Type: cash.VoucherPay,
				Description: "Chi mua văn phòng phẩm", Receive: 0, Pay: 2000000, Balance: 8000000,
			},
		},
	}
}

func TestS07DN(t *testing.T) {
	got, err := S07DN(s07Data())
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "s07dn", got)
}

func TestS07aDN(t *testing.T) {
	got, err := S07aDN(s07Data())
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "s07adn", got)
}

func TestKiemKeQuy(t *testing.T) {
	c := &cash.CashCount{
		ID:            "count1",
		FundID:        "fund_vnd",
		CountDate:     "2026-08-31",
		BookBalance:   8000000,
		CountedAmount: 7950000,
		Difference:    -50000,
		Resolution:    "Chờ xác minh chênh lệch.",
		Participants:  []string{"Nguyễn Văn A", "Trần Thị B"},
		State:         "open",
	}
	got, err := KiemKeQuy(testFund(), c)
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "kiem_ke", got)
}

func TestBienBanDoiChieu(t *testing.T) {
	rec := &cash.Reconciliation{
		ID:                "rec1",
		FundID:            "fund_vnd",
		Period:            "2026-08",
		CashierBalance:    8000000,
		AccountantBalance: 8000000,
		Difference:        0,
		State:             "resolved",
		SignedBy:          []string{"thuquy01", "ktt01", "giamdoc01"},
		CreatedAt:         "2026-08-31T17:00:00Z",
	}
	got, err := BienBanDoiChieu(testFund(), rec)
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "doi_chieu", got)
}

func TestFormatVN(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1234567, "1.234.567"},
		{123045067, "123.045.067"},
		{-50000, "-50.000"},
		{1000000000, "1.000.000.000"},
	}
	for _, c := range cases {
		if got := FormatVN(c.in); got != c.want {
			t.Errorf("FormatVN(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	if got, want := FormatDate("2026-08-25"), "25/08/2026"; got != want {
		t.Errorf("FormatDate = %q, want %q", got, want)
	}
	if got := FormatDate("2026-08"); got != "2026-08" {
		t.Errorf("FormatDate(period) = %q, want passthrough", got)
	}
}
