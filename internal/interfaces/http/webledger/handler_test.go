package webledger_test

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

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
	httpwebledger "goGL/internal/interfaces/http/webledger"
)

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

func buildRouter(t *testing.T) (*gin.Engine, appledger.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	sqlDB := openDB(t)
	svc := appledger.NewService(persledger.NewSqliteRepository(sqlDB))
	root := repoRoot(t)
	tmpl := template.Must(template.New("").Funcs(httpwebledger.Funcs()).ParseGlob(filepath.Join(root, "web/templates/*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join(root, "web/templates/ledger/*.html")))
	r := gin.New()
	r.SetHTMLTemplate(tmpl)
	httpwebledger.NewHandler(svc, "X-User-Id").Register(r)
	return r, svc
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

func mustCreateAccount(t *testing.T, svc appledger.Service, a *domainledger.Account) {
	t.Helper()
	if err := svc.CreateAccount(context.Background(), "ketoan", a); err != nil {
		t.Fatalf("create account %s: %v", a.Code, err)
	}
}

func seedAccounts(t *testing.T, svc appledger.Service) {
	t.Helper()
	mustCreateAccount(t, svc, &domainledger.Account{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "5111", Name: "Doanh thu bán hàng", Type: domainledger.AccountRevenue, Level: 3, AllowPost: true})
}

func manualEntryVals() url.Values {
	return url.Values{
		"voucher_date": {"2026-08-25"},
		"description":  {"Thu tiền bán hàng HĐ 123"},
		"account_code": {"1111", "5111"},
		"debit":        {"123045067", "0"},
		"credit":       {"0", "123045067"},
		"note":         {"Nợ tiền mặt", "Có doanh thu"},
	}
}

// redirectID extracts the entry id from a /ledger/entries/:id redirect.
func redirectID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body %s)", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location %q: %v", w.Header().Get("Location"), err)
	}
	id := strings.TrimPrefix(loc.Path, "/ledger/entries/")
	if id == loc.Path {
		t.Fatalf("unexpected redirect path %q", loc.Path)
	}
	return id
}

func TestLedgerPagesRender(t *testing.T) {
	r, _ := buildRouter(t)
	for _, path := range []string{"/ledger", "/ledger/entries", "/ledger/entries/new"} {
		w := getWithUser(t, r, path, "ketoan01")
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 (body %s)", path, w.Code, w.Body.String())
		}
	}
}

func TestCreateDraftFromWeb(t *testing.T) {
	r, svc := buildRouter(t)
	seedAccounts(t, svc)

	w := doForm(t, r, http.MethodPost, "/ledger/entries", manualEntryVals(), "ketoan01")
	id := redirectID(t, w)

	got, err := svc.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got.Status != domainledger.EntryDraft {
		t.Fatalf("status = %s, want draft", got.Status)
	}
	if got.VoucherNo != "" {
		t.Fatalf("draft must not carry a voucher number, got %q", got.VoucherNo)
	}
	if got.Source != domainledger.SourceManual {
		t.Fatalf("source = %s, want manual", got.Source)
	}

	body := getWithUser(t, r, "/ledger/entries/"+id, "ketoan01").Body.String()
	for _, want := range []string{"Thu tiền bán hàng HĐ 123", "Nháp", "123.045.067"} {
		if !strings.Contains(body, want) {
			t.Errorf("draft detail missing %q", want)
		}
	}
}

func TestCreateAndPostFromWeb(t *testing.T) {
	r, svc := buildRouter(t)
	seedAccounts(t, svc)

	vals := manualEntryVals()
	vals.Set("action", "post")
	w := doForm(t, r, http.MethodPost, "/ledger/entries", vals, "ketoan01")
	id := redirectID(t, w)

	got, err := svc.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got.Status != domainledger.EntryPosted {
		t.Fatalf("status = %s, want posted", got.Status)
	}
	if got.VoucherNo != "PK-00001/26" {
		t.Fatalf("voucher no = %q, want PK-00001/26", got.VoucherNo)
	}
	if got.PostedBy != "ketoan01" || got.PostedAt == "" {
		t.Fatalf("posted by/at not recorded: %s %s", got.PostedBy, got.PostedAt)
	}

	// Entries list page shows the voucher number and "Đã ghi sổ".
	body := getWithUser(t, r, "/ledger/entries", "ketoan01").Body.String()
	for _, want := range []string{"PK-00001/26", "Đã ghi sổ", "123.045.067"} {
		if !strings.Contains(body, want) {
			t.Errorf("entries list missing %q", want)
		}
	}
}

func TestPostRejectsUnbalancedFromWeb(t *testing.T) {
	r, svc := buildRouter(t)
	seedAccounts(t, svc)

	vals := manualEntryVals()
	vals.Set("debit", "100000")
	vals.Set("action", "post")
	w := doForm(t, r, http.MethodPost, "/ledger/entries", vals, "ketoan01")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("unbalanced post = %d, want redirect", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("unbalanced post must redirect with err, got %q", w.Header().Get("Location"))
	}
}

func TestPostAndDeleteFromWeb(t *testing.T) {
	r, svc := buildRouter(t)
	seedAccounts(t, svc)

	// Draft → post via detail page form.
	w := doForm(t, r, http.MethodPost, "/ledger/entries", manualEntryVals(), "ketoan01")
	id := redirectID(t, w)
	w = doForm(t, r, http.MethodPost, "/ledger/entries/"+id+"/post", url.Values{}, "ketoan01")
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("post failed: %s", w.Header().Get("Location"))
	}

	// Posting again is idempotent (R5), returns same voucher no.
	w = doForm(t, r, http.MethodPost, "/ledger/entries/"+id+"/post", url.Values{}, "ketoan01")
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("repost failed: %s", w.Header().Get("Location"))
	}

	// Deleting a posted entry is rejected (R7).
	w = doForm(t, r, http.MethodPost, "/ledger/entries/"+id+"/delete", url.Values{}, "ketoan01")
	if !strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("delete posted must redirect with err, got %q", w.Header().Get("Location"))
	}

	// A fresh draft can be deleted.
	w = doForm(t, r, http.MethodPost, "/ledger/entries", manualEntryVals(), "ketoan01")
	id2 := redirectID(t, w)
	w = doForm(t, r, http.MethodPost, "/ledger/entries/"+id2+"/delete", url.Values{}, "ketoan01")
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("delete draft failed: %s", w.Header().Get("Location"))
	}
	if _, err := svc.GetEntry(context.Background(), id2); err == nil {
		t.Fatalf("deleted entry %s still exists", id2)
	}
}

func TestMutatingRoutes_RejectAnonymous(t *testing.T) {
	r, svc := buildRouter(t)
	seedAccounts(t, svc)

	// Seed a draft to exercise post/delete on.
	w := doForm(t, r, http.MethodPost, "/ledger/entries", manualEntryVals(), "ketoan01")
	id := redirectID(t, w)

	cases := []struct {
		name string
		path string
		vals url.Values
	}{
		{"create", "/ledger/entries", manualEntryVals()},
		{"post", "/ledger/entries/" + id + "/post", url.Values{}},
		{"delete", "/ledger/entries/" + id + "/delete", url.Values{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doForm(t, r, http.MethodPost, tc.path, tc.vals, "")
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s = %d, want 401", tc.name, w.Code)
			}
		})
	}

	got, err := svc.GetEntry(context.Background(), id)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got.Status != domainledger.EntryDraft {
		t.Fatalf("status = %s, want draft (anonymous requests must not mutate)", got.Status)
	}
}
