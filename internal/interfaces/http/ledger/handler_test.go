package ledger_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
	httpledger "goGL/internal/interfaces/http/ledger"
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

func newSvc(t *testing.T, d *sql.DB) appledger.Service {
	t.Helper()
	return appledger.NewService(persledger.NewSqliteRepository(d))
}

func setupRouter(t *testing.T, svc appledger.Service) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	httpledger.NewHandler(svc, "X-User-Id").Register(r.Group("/api/v1"))
	return r
}

func seedPostableAccounts(t *testing.T, svc appledger.Service) {
	t.Helper()
	ctx := t.Context()
	for _, a := range []*domainledger.Account{
		{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true},
		{Code: "5111", Name: "Doanh thu bán hàng", Type: domainledger.AccountRevenue, Level: 3, AllowPost: true},
	} {
		if err := svc.CreateAccount(ctx, "ketoan", a); err != nil {
			t.Fatalf("seed account %s: %v", a.Code, err)
		}
	}
}

func TestHandler_CreateEntry_ReturnsCreated(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	body := `{
		"voucher_date": "2026-08-05",
		"source": "manual",
		"description": "Bút toán kiểm tra",
		"lines": [
			{"line_no": 1, "account_code": "1111", "debit": 5000000},
			{"line_no": 2, "account_code": "5111", "credit": 5000000}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", bytes.NewBufferString(body))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.JournalEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" || got.Status != domainledger.EntryDraft || got.CreatedBy != "ketoan" {
		t.Fatalf("unexpected entry: %+v", got)
	}
	if got.Period != "2026-08" {
		t.Fatalf("period = %q, want 2026-08", got.Period)
	}
}

func TestHandler_CreateEntry_Unbalanced422(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	body := `{
		"voucher_date": "2026-08-05",
		"source": "manual",
		"lines": [
			{"line_no": 1, "account_code": "1111", "debit": 5000000},
			{"line_no": 2, "account_code": "5111", "credit": 4000000}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", bytes.NewBufferString(body))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateEntry_BadRequest(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", strings.NewReader(`{not json`))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetEntryRoundTrip(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	ctx := t.Context()

	created, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Description: "Bút toán kiểm tra",
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/entries/"+created.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.JournalEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != created.ID || len(got.Lines) != 2 {
		t.Fatalf("unexpected entry: %+v", got)
	}
}

func TestHandler_GetEntryNotFound404(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/entries/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListAccountsAndPeriodsEmpty(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	for _, tc := range []struct{ path string }{
		{"/api/v1/ledger/accounts"},
		{"/api/v1/ledger/periods"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200; body: %s", tc.path, w.Code, w.Body.String())
		}
		if strings.TrimSpace(w.Body.String()) != "[]" {
			t.Fatalf("%s: want empty array, got %s", tc.path, w.Body.String())
		}
	}
}

func TestHandler_ListEntriesByPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)
	ctx := t.Context()

	if _, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
		},
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/entries?period=2026-08", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got []domainledger.JournalEntry
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
}

func TestHandler_CreateAccount_ReturnsCreated(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	body := `{
		"code": "1111",
		"name": "Tiền mặt VND",
		"type": "asset",
		"level": 2,
		"allow_post": true
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/accounts", bytes.NewBufferString(body))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.Account
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" || got.Code != "1111" || got.Status != domainledger.AccountActive {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestHandler_GetAccountRoundTrip(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/accounts/"+domainledger.RowID("account", "1111"), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.Account
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "1111" {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestHandler_GetAccountNotFound404(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/accounts/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_UpdateAccount_ReturnsUpdated(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	body := `{"name": "Tiền mặt VND (đóng)", "status": "inactive"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/ledger/accounts/"+domainledger.RowID("account", "1111"), bytes.NewBufferString(body))
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got domainledger.Account
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != domainledger.AccountInactive || got.Code != "1111" {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestHandler_ListAccounts_FilterByType(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/accounts?type=revenue", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got []domainledger.Account
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 || got[0].Code != "5111" {
		t.Fatalf("unexpected accounts: %+v", got)
	}
}

func TestHandler_OpenCloseReopenPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	open := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/open", nil)
	open.Header.Set("X-User-Id", "kttruong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, open)
	if w.Code != http.StatusOK {
		t.Fatalf("open: status = %d, body: %s", w.Code, w.Body.String())
	}
	var p domainledger.AccountingPeriod
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode open: %v", err)
	}
	if p.Status != domainledger.PeriodOpen || p.OpenedBy != "kttruong" {
		t.Fatalf("unexpected period: %+v", p)
	}

	close := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/close",
		bytes.NewBufferString(`{"reason": "khoá sổ cuối tháng"}`))
	close.Header.Set("X-User-Id", "kttruong")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, close)
	if w.Code != http.StatusOK {
		t.Fatalf("close: status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode close: %v", err)
	}
	if p.Status != domainledger.PeriodClosed || p.CloseReason != "khoá sổ cuối tháng" {
		t.Fatalf("unexpected closed period: %+v", p)
	}

	reopen := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/reopen",
		bytes.NewBufferString(`{"reason": "thiếu bút toán"}`))
	reopen.Header.Set("X-User-Id", "kttruong")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, reopen)
	if w.Code != http.StatusOK {
		t.Fatalf("reopen: status = %d, body: %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode reopen: %v", err)
	}
	if p.Status != domainledger.PeriodOpen {
		t.Fatalf("unexpected reopened period: %+v", p)
	}
}

func TestHandler_ClosePeriod_MissingReason422(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	router := setupRouter(t, svc)

	close := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/close",
		bytes.NewBufferString(`{}`))
	close.Header.Set("X-User-Id", "kttruong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, close)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateEntry_ClosedPeriod409(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)
	router := setupRouter(t, svc)

	open := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/open", nil)
	open.Header.Set("X-User-Id", "kttruong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, open)
	if w.Code != http.StatusOK {
		t.Fatalf("open: status = %d", w.Code)
	}

	close := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/periods/2026-08/close",
		bytes.NewBufferString(`{"reason": "khoá sổ"}`))
	close.Header.Set("X-User-Id", "kttruong")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, close)
	if w.Code != http.StatusOK {
		t.Fatalf("close: status = %d", w.Code)
	}

	body := `{
		"voucher_date": "2026-08-20",
		"source": "manual",
		"lines": [
			{"line_no": 1, "account_code": "1111", "debit": 5000000},
			{"line_no": 2, "account_code": "5111", "credit": 5000000}
		]
	}`
	entry := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", bytes.NewBufferString(body))
	entry.Header.Set("X-User-Id", "ketoan")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, entry)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (R4); body: %s", w.Code, w.Body.String())
	}
}
