package ledger_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainledger "goGL/internal/domain/ledger"
)

const manualEntryBody = `{
	"voucher_date": "2026-08-05",
	"source": "manual",
	"description": "Bút toán kiểm tra",
	"lines": [
		{"line_no": 1, "account_code": "1111", "debit": 5000000},
		{"line_no": 2, "account_code": "5111", "credit": 5000000}
	]
}`

func createEntryViaAPI(t *testing.T, router http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", bytes.NewBufferString(manualEntryBody))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create entry: status = %d, body: %s", w.Code, w.Body.String())
	}
	var got domainledger.JournalEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode entry: %v", err)
	}
	return got.ID
}

func TestHandler_PostEntry_ReturnsPosted(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	id := createEntryViaAPI(t, router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries/"+id+"/post", nil)
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.JournalEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != domainledger.EntryPosted {
		t.Fatalf("status = %q, want posted", got.Status)
	}
	if got.VoucherNo != "PK-00001/26" {
		t.Fatalf("VoucherNo = %q, want PK-00001/26", got.VoucherNo)
	}
	if got.PostedBy != "ketoan" || got.PostedAt == "" {
		t.Fatalf("poster fields missing: %+v", got)
	}
}

func TestHandler_PostEntry_RepostIsIdempotent(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	id := createEntryViaAPI(t, router)

	post := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries/"+id+"/post", nil)
		req.Header.Set("X-User-Id", "ketoan")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("post: status = %d, body: %s", w.Code, w.Body.String())
		}
		var got domainledger.JournalEntry
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return got.VoucherNo
	}

	first := post()
	second := post()
	if first == "" || first != second {
		t.Fatalf("repost not idempotent: %q vs %q", first, second)
	}
}

func TestHandler_PostEntry_NotFound404(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries/missing/post", nil)
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_DeleteEntry_DraftAndPosted(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	id := createEntryViaAPI(t, router)

	del := httptest.NewRequest(http.MethodDelete, "/api/v1/ledger/entries/"+id, nil)
	del.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, del)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete draft: status = %d, want 204; body: %s", w.Code, w.Body.String())
	}

	// Posted entries are append-only (R6/R7).
	postedID := createEntryViaAPI(t, router)
	post := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries/"+postedID+"/post", nil)
	post.Header.Set("X-User-Id", "ketoan")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, post)
	if w.Code != http.StatusOK {
		t.Fatalf("post: status = %d, body: %s", w.Code, w.Body.String())
	}

	del = httptest.NewRequest(http.MethodDelete, "/api/v1/ledger/entries/"+postedID, nil)
	del.Header.Set("X-User-Id", "ketoan")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, del)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete posted: status = %d, want 409 (R7); body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_PostEntry_ClosedPeriod409(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	id := createEntryViaAPI(t, router)

	open := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/open", nil)
	open.Header.Set("X-User-Id", "kttruong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, open)
	if w.Code != http.StatusOK {
		t.Fatalf("open: status = %d, body: %s", w.Code, w.Body.String())
	}

	close := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/close",
		bytes.NewBufferString(`{"reason": "khoá sổ"}`))
	close.Header.Set("X-User-Id", "kttruong")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, close)
	if w.Code != http.StatusOK {
		t.Fatalf("close: status = %d, body: %s", w.Code, w.Body.String())
	}

	post := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries/"+id+"/post", nil)
	post.Header.Set("X-User-Id", "ketoan")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, post)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (R4 re-checked at post); body: %s", w.Code, w.Body.String())
	}
}
