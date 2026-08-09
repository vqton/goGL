package webcash_test

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	appcash "goGL/internal/application/cash"
	domainaudit "goGL/internal/domain/audit"
	domaincash "goGL/internal/domain/cash"
	"goGL/internal/infrastructure/db"
	perscash "goGL/internal/infrastructure/persistence/cash"
	httpcashweb "goGL/internal/interfaces/http/webcash"
)

type noopAuditor struct{}

func (noopAuditor) Record(_ context.Context, _ *domainaudit.AuditLog) error { return nil }

func openDB(t *testing.T) *sql.DB {
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

// repoRoot resolves the repo root from this test file's location so template
// globs work regardless of the test working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}

func newRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	sqlDB := openDB(t)
	svc := appcash.NewService(perscash.NewSqliteRepository(sqlDB), noopAuditor{})
	return buildRouter(t, svc)
}

func buildRouter(t *testing.T, svc appcash.Service) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	root := repoRoot(t)
	tmpl := template.Must(template.New("").Funcs(httpcashweb.Funcs()).ParseGlob(filepath.Join(root, "web/templates/*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join(root, "web/templates/cash/*.html")))
	r.SetHTMLTemplate(tmpl)
	httpcashweb.NewHandler(svc, "X-User-Id").Register(r)
	return r
}

func doForm(t *testing.T, r http.Handler, method, path string, vals url.Values, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// redirectID extracts the voucher id from a /cash/vouchers/:id redirect.
func redirectID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location %q: %v", w.Header().Get("Location"), err)
	}
	id := strings.TrimPrefix(loc.Path, "/cash/vouchers/")
	if id == loc.Path {
		t.Fatalf("unexpected redirect path %q", loc.Path)
	}
	return id
}

func getWithUser(t *testing.T, r http.Handler, path, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func seedFund(t *testing.T, svc appcash.Service) *domaincash.Fund {
	t.Helper()
	fund := &domaincash.Fund{Name: "Quỹ tiền mặt VND", Currency: "VND", Account: "1111", IsActive: true}
	if err := svc.CreateFund(context.Background(), fund); err != nil {
		t.Fatalf("create fund: %v", err)
	}
	return fund
}

func TestDashboard(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/cash", nil)
	req.Header.Set("X-User-Id", "cashier01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	for _, want := range []string{"Tổng quan quỹ tiền mặt", "Phiếu thu/chi", "Khóa sổ", "Đối chiếu"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestPagesRender(t *testing.T) {
	r := newRouter(t)
	for _, path := range []string{"/cash/vouchers", "/cash/vouchers/new", "/cash/book", "/cash/close-day", "/cash/reconcile"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, w.Code)
		}
	}
}

func TestFundThenFlow(t *testing.T) {
	sqlDB := openDB(t)
	svc := appcash.NewService(perscash.NewSqliteRepository(sqlDB), noopAuditor{})
	r := buildRouter(t, svc)
	fund := seedFund(t, svc)

	vals := url.Values{
		"type":              {"receive"},
		"fund_id":           {fund.ID},
		"ref_date":          {"2026-08-25"},
		"counterparty_name": {"Nguyễn Văn A"},
		"description":       {"Thu tiền bán hàng"},
		"amount_minor":      {"123045067"},
		"other_account":     {"5111"},
	}
	w := doForm(t, r, http.MethodPost, "/cash/vouchers", vals, "cashier01")
	id := redirectID(t, w)

	// Detail page shows the auto-assigned RefNo + VN words.
	w = getWithUser(t, r, "/cash/vouchers/"+id, "cashier01")
	body := w.Body.String()
	for _, want := range []string{"PT/2026-08/000001", "một trăm hai mươi ba triệu"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail missing %q", want)
		}
	}

	// Approve (must differ from creator) → post.
	w = doForm(t, r, http.MethodPost, "/cash/vouchers/"+id+"/approve", url.Values{}, "accountant01")
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("approve failed: %s", w.Header().Get("Location"))
	}
	w = doForm(t, r, http.MethodPost, "/cash/vouchers/"+id+"/post", url.Values{}, "cashier01")
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("post failed: %s", w.Header().Get("Location"))
	}

	// Cash book shows the posted entry.
	w = doForm(t, r, http.MethodGet, "/cash/book?fund_id="+fund.ID, url.Values{}, "cashier01")
	if w.Code != http.StatusOK {
		t.Fatalf("book = %d, want 200", w.Code)
	}
	body = w.Body.String()
	for _, want := range []string{"PT/2026-08/000001", "123.045.067"} {
		if !strings.Contains(body, want) {
			t.Errorf("book missing %q", want)
		}
	}

	// Print endpoints.
	w = doForm(t, r, http.MethodGet, "/cash/print/voucher/"+id, url.Values{}, "cashier01")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "PHIẾU THU") {
		t.Errorf("print voucher: code=%d", w.Code)
	}
	w = doForm(t, r, http.MethodGet, "/cash/print/s07?fund_id="+fund.ID, url.Values{}, "cashier01")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "SỔ QUỸ TIỀN MẶT") {
		t.Errorf("print s07: code=%d", w.Code)
	}
}

func TestCloseDayAndReconcileFromWeb(t *testing.T) {
	sqlDB := openDB(t)
	svc := appcash.NewService(perscash.NewSqliteRepository(sqlDB), noopAuditor{})
	r := buildRouter(t, svc)
	fund := seedFund(t, svc)

	vals := url.Values{
		"type":              {"receive"},
		"fund_id":           {fund.ID},
		"ref_date":          {"2026-08-25"},
		"counterparty_name": {"Nguyễn Văn A"},
		"description":       {"Thu tiền"},
		"amount_minor":      {"1000000"},
		"other_account":     {"5111"},
	}
	w := doForm(t, r, http.MethodPost, "/cash/vouchers", vals, "cashier01")
	id := redirectID(t, w)
	doForm(t, r, http.MethodPost, "/cash/vouchers/"+id+"/approve", url.Values{}, "accountant01")
	doForm(t, r, http.MethodPost, "/cash/vouchers/"+id+"/post", url.Values{}, "cashier01")

	// Close day with matching count → book closes.
	w = doForm(t, r, http.MethodPost, "/cash/close-day", url.Values{
		"fund_id":        {fund.ID},
		"count_date":     {"2026-08-25"},
		"counted_amount": {"1000000"},
		"participants":   {"thuquy01, ktt01"},
	}, "cashier01")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("close-day = %d, want 303 (body %s)", w.Code, w.Body.String())
	}

	// Reconcile the month.
	w = doForm(t, r, http.MethodPost, "/cash/reconcile", url.Values{
		"fund_id":            {fund.ID},
		"period":             {"2026-08"},
		"accountant_balance": {"1000000"},
	}, "cashier01")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("reconcile = %d, want 303 (body %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Location"), "ok=") {
		t.Errorf("reconcile should redirect with success flash, got %q", w.Header().Get("Location"))
	}
}

func TestVoidFromWeb(t *testing.T) {
	sqlDB := openDB(t)
	svc := appcash.NewService(perscash.NewSqliteRepository(sqlDB), noopAuditor{})
	r := buildRouter(t, svc)
	fund := seedFund(t, svc)

	v := &domaincash.Voucher{
		Type: domaincash.VoucherReceive, FundID: fund.ID, RefDate: "2026-08-25",
		Currency: "VND", AmountMinor: 1000000,
		CounterpartyName: "Nguyễn Văn A", Description: "Thu tiền",
		Lines: []domaincash.VoucherLine{
			{Seq: 1, DebitAcc: "1111", AmountMinor: 1000000},
			{Seq: 2, CreditAcc: "5111", AmountMinor: 1000000},
		},
	}
	if err := svc.CreateVoucher(context.Background(), "cashier01", v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}

	vals := url.Values{"reason": {"Nhập sai"}}
	w := doForm(t, r, http.MethodPost, "/cash/vouchers/"+v.ID+"/void", vals, "cashier01")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("void = %d, want 303", w.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/cash/vouchers/"+v.ID, nil)
	req.Header.Set("X-User-Id", "cashier01")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "Đã hủy") {
		t.Errorf("voided detail should show Đã hủy badge")
	}
}
