package cash_test

import (
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

	appcash "goGL/internal/application/cash"
	domainaudit "goGL/internal/domain/audit"
	domaincash "goGL/internal/domain/cash"
	"goGL/internal/infrastructure/db"
	perscash "goGL/internal/infrastructure/persistence/cash"
	httpcash "goGL/internal/interfaces/http/cash"
)

type noopAuditor struct{}

func (noopAuditor) Record(_ context.Context, _ *domainaudit.AuditLog) error { return nil }

func openHTTPDB(t *testing.T) *sql.DB {
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

func newRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := appcash.NewService(perscash.NewSqliteRepository(openHTTPDB(t)), noopAuditor{})
	httpcash.NewHandler(svc, "X-User-Id").Register(r.Group("/api/v1"))
	return r
}

func do(t *testing.T, r http.Handler, method, path, body, user string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

func createFundViaHTTP(t *testing.T, r http.Handler) string {
	t.Helper()
	body := `{"name":"Quỹ VND","currency":"VND","account":"1111","is_active":true}`
	rec := do(t, r, http.MethodPost, "/api/v1/cash/funds", body, "admin")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create fund: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var fund domaincash.Fund
	decode(t, rec, &fund)
	return fund.ID
}

func validVoucherJSON(fundID string) string {
	return `{"ref_date":"2026-08-05","type":"receive","fund_id":"` + fundID + `","currency":"VND","amount_minor":5000000,"counterparty_type":"customer","counterparty_id":"kh-1","counterparty_name":"Công ty ABC","description":"Thu tiền bán hàng","lines":[{"seq":1,"debit_acc":"1111","amount_minor":5000000},{"seq":2,"credit_acc":"131","amount_minor":3000000,"object_id":"kh-1"},{"seq":3,"credit_acc":"5111","amount_minor":2000000}]}`
}

func TestHandler_CreateFundAndVoucher(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create voucher: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var v domaincash.Voucher
	decode(t, rec, &v)
	if v.ID == "" || v.RefNo != "PT/2026-08/000001" || v.State != domaincash.VoucherDraft || v.CreatedBy != "ketoan" {
		t.Fatalf("voucher mismatch: %+v", v)
	}

	rec = do(t, r, http.MethodGet, "/api/v1/cash/vouchers/"+v.ID, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get voucher: expected 200, got %d", rec.Code)
	}
}

func TestHandler_ApproveFlow(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
	var v domaincash.Voucher
	decode(t, rec, &v)

	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/approve", "", "giamdoc")
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got domaincash.Voucher
	decode(t, rec, &got)
	if got.State != domaincash.VoucherApproved || got.ApprovedBy != "giamdoc" {
		t.Fatalf("approved voucher mismatch: %+v", got)
	}
}

func TestHandler_SelfApprovalForbidden(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
	var v domaincash.Voucher
	decode(t, rec, &v)

	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers/"+v.ID+"/approve", "", "ketoan")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("self-approval: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_UpdateDraft(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
	var v domaincash.Voucher
	decode(t, rec, &v)

	updated := validVoucherJSON(fundID)
	updated = strings.Replace(updated, `"amount_minor":5000000`, `"amount_minor":6000000`, 2)
	updated = strings.Replace(updated, `"amount_minor":3000000`, `"amount_minor":4000000`, 1)
	rec = do(t, r, http.MethodPatch, "/api/v1/cash/vouchers/"+v.ID, updated, "ketoan")
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got domaincash.Voucher
	decode(t, rec, &got)
	if got.AmountMinor != 6_000_000 || got.RefNo != "PT/2026-08/000001" {
		t.Fatalf("updated voucher mismatch: %+v", got)
	}
}

func TestHandler_ListVouchers(t *testing.T) {
	r := newRouter(t)
	fundID := createFundViaHTTP(t, r)

	for i := 0; i < 2; i++ {
		rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON(fundID), "ketoan")
		if rec.Code != http.StatusCreated {
			t.Fatalf("create voucher %d: %d", i, rec.Code)
		}
	}

	rec := do(t, r, http.MethodGet, "/api/v1/cash/vouchers?fund_id="+fundID+"&state=draft", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var list []domaincash.Voucher
	decode(t, rec, &list)
	if len(list) != 2 {
		t.Fatalf("expected 2 vouchers, got %d", len(list))
	}
}

func TestHandler_Errors(t *testing.T) {
	r := newRouter(t)

	rec := do(t, r, http.MethodPost, "/api/v1/cash/vouchers", validVoucherJSON("fund-nope"), "ketoan")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown fund: expected 404, got %d (%s)", rec.Code, rec.Body.String())
	}

	fundID := createFundViaHTTP(t, r)
	bad := `{"ref_date":"2026-08-05","type":"receive","fund_id":"` + fundID + `","amount_minor":5000000,"counterparty_name":"X","lines":[{"seq":1,"debit_acc":"1111","amount_minor":1000000},{"seq":2,"credit_acc":"5111","amount_minor":2000000}]}`
	rec = do(t, r, http.MethodPost, "/api/v1/cash/vouchers", bad, "ketoan")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unbalanced lines: expected 422, got %d (%s)", rec.Code, rec.Body.String())
	}
}
