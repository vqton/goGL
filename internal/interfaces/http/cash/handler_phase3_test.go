package cash_test

import (
	"net/http"
	"testing"

	domaincash "goGL/internal/domain/cash"
)

func TestHandler_VoidDraft(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
	var v domaincash.Voucher
	decode(t, rec, &v)

	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/void", `{"reason":"nhập sai"}`, "ketoan")
	if rec.Code != http.StatusOK {
		t.Fatalf("void draft: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	decode(t, rec, &v)
	if v.State != domaincash.VoucherVoided || v.VoidReason != "nhập sai" {
		t.Fatalf("voided voucher mismatch: %+v", v)
	}
}

func TestHandler_VoidPosted(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/void", `{"reason":"sai đối ứng"}`, "ketoan")
	if rec.Code != http.StatusOK {
		t.Fatalf("void posted: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var v domaincash.Voucher
	decode(t, rec, &v)
	if v.State != domaincash.VoucherVoided || len(v.RefVouchers) != 1 {
		t.Fatalf("voided voucher mismatch: %+v", v)
	}

	rec = do(t, r, http.MethodGet, "/api/v1/cash/book?fund_id="+fundID, "", "")
	var book []domaincash.CashBookEntry
	decode(t, rec, &book)
	if len(book) != 2 || book[1].Balance != 0 {
		t.Fatalf("book balance not restored: %+v", book)
	}
}

func TestHandler_ReconcileResolved(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}

	body := `{"fund_id":"` + fundID + `","period":"2026-08","accountant_balance":5000000,"signers":["thuquy","ketoan","giamdoc"]}`
	rec := do(t, r, http.MethodPost, "/api/v1/cash/reconcile", body, "ketoan")
	if rec.Code != http.StatusOK {
		t.Fatalf("reconcile: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var recd domaincash.Reconciliation
	decode(t, rec, &recd)
	if recd.State != "resolved" || recd.Difference != 0 {
		t.Fatalf("reconciliation mismatch: %+v", recd)
	}

	rec = do(t, r, http.MethodGet, "/api/v1/cash/reconciliations?fund_id="+fundID, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list reconciliations: expected 200, got %d", rec.Code)
	}
	var list []domaincash.Reconciliation
	decode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 reconciliation, got %d", len(list))
	}
}

func TestHandler_ReconcileOpenCountBlocks(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}
	rec := do(t, r, http.MethodPost, "/api/v1/cash/close-day",
		`{"fund_id":"`+fundID+`","date":"2026-08-05","counted_amount":4000000}`, "thuquy")
	if rec.Code != http.StatusOK {
		t.Fatalf("close-day: %d", rec.Code)
	}

	rec = do(t, r, http.MethodPost, "/api/v1/cash/reconcile",
		`{"fund_id":"`+fundID+`","period":"2026-08","accountant_balance":5000000}`, "ketoan")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reconcile with open count: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_ResolveCount_ClosesDay(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}

	// Mismatched close-day leaves an open count.
	rec := do(t, r, http.MethodPost, "/api/v1/cash/close-day",
		`{"fund_id":"`+fundID+`","date":"2026-08-05","counted_amount":4000000}`, "thuquy")
	if rec.Code != http.StatusOK {
		t.Fatalf("close-day: %d", rec.Code)
	}
	var count domaincash.CashCount
	decode(t, rec, &count)
	if count.State != domaincash.CashCountOpen {
		t.Fatalf("count must stay open: %+v", count)
	}

	// A standalone count on the same date is rejected while one is pending.
	rec = do(t, r, http.MethodPost, "/api/v1/cash/counts",
		`{"fund_id":"`+fundID+`","date":"2026-08-05","counted_amount":4000000}`, "thuquy")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate open count: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Resolve → day closes.
	rec = do(t, r, http.MethodPost, "/api/v1/cash/counts/"+count.ID+"/resolve",
		`{"resolution":"đã xử lý chênh lệch"}`, "ketoan")
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resolved domaincash.CashCount
	decode(t, rec, &resolved)
	if resolved.State != domaincash.CashCountResolved {
		t.Fatalf("resolve mismatch: %+v", resolved)
	}

	// Day is now closed: a fresh count for the same date is rejected.
	rec = do(t, r, http.MethodPost, "/api/v1/cash/counts",
		`{"fund_id":"`+fundID+`","date":"2026-08-05","counted_amount":4000000}`, "thuquy")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("count on closed day: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostVoucher_UnauthorizedActor(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")

	// Poster == preparer (ketoan created the voucher) must be rejected.
	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "ketoan")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("post by preparer: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_Reconcile_InvalidSigners(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)
	id := createApprovedViaHTTP(t, r, fundID, "ketoan")
	if rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+id+"/post", "", "thuquy"); rec.Code != http.StatusOK {
		t.Fatalf("post: %d", rec.Code)
	}
	rec := do(t, r, http.MethodPost, "/api/v1/cash/reconcile",
		`{"fund_id":"`+fundID+`","period":"2026-08","accountant_balance":5000000,"signers":["a","b"]}`, "ketoan")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reconcile with 2 signers: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}
