package cash_test

import (
	"net/http"
	"testing"

	domaincash "goGL/internal/domain/cash"
)

func createApprovedViaHTTP(t *testing.T, r http.Handler, fundID, user string) string {
	t.Helper()
	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), user)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create voucher: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var v domaincash.Voucher
	decode(t, rec, &v)
	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/approve", "", "giamdoc")
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d", rec.Code)
	}
	return v.ID
}

func TestHandler_PostVoucherAndBook(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy")
	if rec.Code != http.StatusOK {
		t.Fatalf("post: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var v domaincash.Voucher
	decode(t, rec, &v)
	if v.State != domaincash.VoucherPosted || v.PostedBy != "thuquy" {
		t.Fatalf("posted voucher mismatch: %+v", v)
	}

	rec = do(t, r, http.MethodGet, "/api/v1/cash/book?fund_id="+fundID, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("book: expected 200, got %d", rec.Code)
	}
	var book []domaincash.CashBookEntry
	decode(t, rec, &book)
	if len(book) != 1 || book[0].Receive != 5_000_000 || book[0].Balance != 5_000_000 {
		t.Fatalf("book mismatch: %+v", book)
	}
}

func TestHandler_PostOverdraftUnprocessable(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	body := `{"ref_date":"2026-08-05","type":"pay","fund_id":"` + fundID + `","amount_minor":1000000,"counterparty_name":"X","lines":[{"seq":1,"debit_acc":"152","amount_minor":1000000},{"seq":2,"credit_acc":"1111","amount_minor":1000000}]}`
	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", body, "ketoan")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pay voucher: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var v domaincash.Voucher
	decode(t, rec, &v)
	do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/approve", "", "giamdoc")

	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/post", "", "thuquy")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("overdraft post: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_CloseDayResolved(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}

	body := `{"fund_id":"` + fundID + `","date":"2026-08-05","counted_amount":5000000,"participants":["thuquy","ketoan"]}`
	rec := do(t, r, http.MethodPost, "/api/v1/cash/close-day", body, "thuquy")
	if rec.Code != http.StatusOK {
		t.Fatalf("close-day: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var count domaincash.CashCount
	decode(t, rec, &count)
	if count.State != "resolved" || count.Difference != 0 {
		t.Fatalf("count mismatch: %+v", count)
	}

	// Posting after close is rejected.
	body = `{"ref_date":"2026-08-05","type":"pay","fund_id":"` + fundID + `","amount_minor":1000,"counterparty_name":"X","lines":[{"seq":1,"debit_acc":"152","amount_minor":1000},{"seq":2,"credit_acc":"1111","amount_minor":1000}]}`
	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers", body, "ketoan")
	var v domaincash.Voucher
	decode(t, rec, &v)
	do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/approve", "", "giamdoc")
	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/post", "", "thuquy")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("post after close: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_CloseDayDifference(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}

	body := `{"fund_id":"` + fundID + `","date":"2026-08-05","counted_amount":4700000,"participants":["thuquy"]}`
	rec := do(t, r, http.MethodPost, "/api/v1/cash/close-day", body, "thuquy")
	if rec.Code != http.StatusOK {
		t.Fatalf("close-day: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var count domaincash.CashCount
	decode(t, rec, &count)
	if count.State != "open" || count.Difference != 300_000 {
		t.Fatalf("count mismatch: %+v", count)
	}

	rec = do(t, r, http.MethodGet, "/api/v1/cash/counts?fund_id="+fundID, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("counts: expected 200, got %d", rec.Code)
	}
	var counts []domaincash.CashCount
	decode(t, rec, &counts)
	if len(counts) != 1 || counts[0].State != "open" {
		t.Fatalf("counts mismatch: %+v", counts)
	}
}
