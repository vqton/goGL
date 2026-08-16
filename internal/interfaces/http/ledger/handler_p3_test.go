package ledger_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
)

// seedPostedEntry creates and posts one manual 1111/5111 entry via the
// service, the same way the UI would after posting a voucher.
func seedPostedEntry(t *testing.T, svc appledger.Service, date string, amount int64) {
	t.Helper()
	e, err := svc.CreateEntry(t.Context(), "ketoan", &domainledger.JournalEntry{
		VoucherDate: date,
		Source:      domainledger.SourceManual,
		Description: "BK " + date,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: amount},
			{LineNo: 2, AccountCode: "5111", Credit: amount},
		},
	})
	if err != nil {
		t.Fatalf("create %s: %v", date, err)
	}
	if _, err := svc.PostEntry(t.Context(), "ketoan", e.ID); err != nil {
		t.Fatalf("post %s: %v", date, err)
	}
}

func getBook(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestHandler_Books_GeneralJournal(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	seedPostedEntry(t, svc, "2026-08-05", 5_000_000)
	seedPostedEntry(t, svc, "2026-08-12", 3_000_000)

	w := getBook(t, router, "/api/v1/ledger/books/general-journal?from=2026-07&to=2026-08")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.GeneralJournal
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Rows) != 4 {
		t.Fatalf("rows = %d, want 4 (2 entries × 2 lines)", len(got.Rows))
	}
	if got.TotalDebit != 8_000_000 || got.TotalCredit != 8_000_000 {
		t.Fatalf("totals = %d/%d, want 8.000.000/8.000.000", got.TotalDebit, got.TotalCredit)
	}
	if got.Total != 4 {
		t.Fatalf("total = %d, want 4 (unpaged book reports full row count)", got.Total)
	}
}

func TestHandler_Books_LedgerRunningBalance(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	seedPostedEntry(t, svc, "2026-08-05", 5_000_000)
	seedPostedEntry(t, svc, "2026-08-12", 2_000_000)

	w := getBook(t, router, "/api/v1/ledger/books/ledger?account=1111&from=2026-08&to=2026-08")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.LedgerBook
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(got.Rows))
	}
	if got.Rows[0].Debit != 5_000_000 || got.Rows[0].Balance != 5_000_000 {
		t.Fatalf("row 0 = %+v, want Dr 5.000.000, balance 5.000.000", got.Rows[0])
	}
	if got.Rows[1].Debit != 2_000_000 || got.Rows[1].Balance != 7_000_000 {
		t.Fatalf("row 1 = %+v, want Dr 2.000.000, running balance 7.000.000", got.Rows[1])
	}
	if got.CloseDebit != 7_000_000 || got.OpenDebit != 0 {
		t.Fatalf("open/close = %d/%d, want 0/7.000.000", got.OpenDebit, got.CloseDebit)
	}
}

func TestHandler_Books_DetailNoRunningBalance(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	seedPostedEntry(t, svc, "2026-08-05", 5_000_000)

	w := getBook(t, router, "/api/v1/ledger/books/detail?account=1111&from=2026-08&to=2026-08")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.LedgerBook
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Rows) != 1 || got.Rows[0].Balance != 0 {
		t.Fatalf("detail rows = %+v, want 1 row with zero running balance", got.Rows)
	}
}

func TestHandler_Books_TrialBalance(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	seedPostedEntry(t, svc, "2026-08-05", 5_000_000)

	w := getBook(t, router, "/api/v1/ledger/books/trial-balance?period=2026-08")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.TrialBalance
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Balanced {
		t.Fatalf("balanced = false, want true: %+v", got)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (1111, 5111)", len(got.Rows))
	}
	if got.Totals.Close.Debit != got.Totals.Close.Credit {
		t.Fatalf("close totals = %+v, want ΣNợ = ΣCó", got.Totals.Close)
	}
}

func TestHandler_Books_Paging(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	for _, d := range []string{"2026-08-01", "2026-08-02", "2026-08-03"} {
		seedPostedEntry(t, svc, d, 1_000_000)
	}

	for _, tc := range []struct {
		page, pageSize string
		wantRows       int
	}{
		{"1", "2", 2},
		{"2", "2", 2},
		{"3", "2", 2},
		{"4", "2", 0},
	} {
		path := "/api/v1/ledger/books/general-journal?from=2026-08&to=2026-08&page=" + tc.page + "&page_size=" + tc.pageSize
		w := getBook(t, router, path)
		if w.Code != http.StatusOK {
			t.Fatalf("page %s: status = %d; body: %s", tc.page, w.Code, w.Body.String())
		}
		var got domainledger.GeneralJournal
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("page %s: decode: %v", tc.page, err)
		}
		if len(got.Rows) != tc.wantRows {
			t.Fatalf("page %s: rows = %d, want %d", tc.page, len(got.Rows), tc.wantRows)
		}
		if got.Total != 6 {
			t.Fatalf("page %s: total = %d, want 6 (full row count, not page size)", tc.page, got.Total)
		}
	}
}

func TestHandler_Books_BadQuery(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	for _, path := range []string{
		"/api/v1/ledger/books/general-journal?from=2026-08&to=2026-08&page=abc",
		"/api/v1/ledger/books/general-journal?from=2026-08&to=2026-08&page_size=0",
		"/api/v1/ledger/books/general-journal?from=2026-09&to=2026-08",
		"/api/v1/ledger/books/general-journal?from=2026-8&to=2026-08",
	} {
		w := getBook(t, router, path)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: status = %d, want 400/422; body: %s", path, w.Code, w.Body.String())
		}
	}
}

func TestHandler_Books_LedgerUnknownAccount(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	// ErrAccountNotFound maps to 422 across the module (established in the
	// write path), so an unknown account filter reports the same code.
	w := getBook(t, router, "/api/v1/ledger/books/ledger?account=9999&from=2026-08&to=2026-08")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", w.Code, w.Body.String())
	}
}
