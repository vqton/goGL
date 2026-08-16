package websetup_test

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appsetup "goGL/internal/application/setup"
	"goGL/internal/domain/audit"
	"goGL/internal/domain/ledger"
	"goGL/internal/domain/masterdata"
	"goGL/internal/domain/setup"
	"goGL/internal/infrastructure/db"
	persistence "goGL/internal/infrastructure/persistence/setup"
	httpwebsetup "goGL/internal/interfaces/http/websetup"

	_ "modernc.org/sqlite"
)

// --- fake seams (mirror internal/application/setup/service_test.go) ---------

type fakeRegime struct {
	cur string
	err error
}

func (f *fakeRegime) SetRegime(_ context.Context, regime, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.cur = regime
	return nil
}
func (f *fakeRegime) GetRegime(_ context.Context) (string, error) { return f.cur, nil }

type fakeSeeder struct {
	n   int
	err error
}

func (f *fakeSeeder) SeedAccounts(_ context.Context, _ string) (int, error) { return f.n, f.err }

type fakeObjects struct {
	recs map[string]*masterdata.Record
	err  error
}

func (f *fakeObjects) Get(_ context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
	if f.err != nil {
		return nil, f.err
	}
	r, ok := f.recs[string(kind)+":"+code]
	if !ok {
		return nil, masterdata.ErrNotFound
	}
	return r, nil
}

type fakePeriods struct{ err error }

func (f *fakePeriods) OpenPeriod(_ context.Context, _ string, id string) (*ledger.AccountingPeriod, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &ledger.AccountingPeriod{ID: id, Status: ledger.PeriodOpen}, nil
}
func (f *fakePeriods) ListPeriods(context.Context) ([]*ledger.AccountingPeriod, error) {
	return nil, nil
}

type fakeAccounts struct {
	accts map[string]*ledger.Account
	err   error
}

func (f *fakeAccounts) GetAccountByCode(_ context.Context, code string) (*ledger.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	a, ok := f.accts[code]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return a, nil
}

func (f *fakeAccounts) ListAccounts(_ context.Context, _ ledger.AccountFilter) ([]*ledger.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]*ledger.Account, 0, len(f.accts))
	for _, a := range f.accts {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

type fakePostings struct {
	entries []*ledger.JournalEntry
	err     error
}

func (f *fakePostings) ListEntries(_ context.Context, _ ledger.EntryFilter) ([]*ledger.JournalEntry, error) {
	return f.entries, f.err
}

type fakeAudit struct {
	err  error
	logs []*audit.AuditLog
}

func (f *fakeAudit) Record(_ context.Context, l *audit.AuditLog) error {
	if f.err != nil {
		return f.err
	}
	f.logs = append(f.logs, l)
	return nil
}

func (f *fakeAudit) ListRecent(_ context.Context, module string, limit int) ([]*audit.AuditLog, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*audit.AuditLog
	for i := len(f.logs) - 1; i >= 0 && len(out) < limit; i-- {
		if module != "" && f.logs[i].Module != module {
			continue
		}
		out = append(out, f.logs[i])
	}
	return out, nil
}

// --- helpers ----------------------------------------------------------------

func newRepo(t *testing.T) setup.Repository {
	t.Helper()
	dsn := fmt.Sprintf("file:websetup_%p?mode=memory&cache=shared", t)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return persistence.NewSqliteRepository(d)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}

type harness struct {
	router *gin.Engine
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fr := &fakeRegime{cur: "TT99-2025"}
	fo := &fakeObjects{recs: map[string]*masterdata.Record{
		"customer:KH-0001": {Code: "KH-0001", Kind: masterdata.KindCustomer, State: masterdata.StateActive},
		"vendor:NCC-0003":  {Code: "NCC-0003", Kind: masterdata.KindSupplier, State: masterdata.StateActive},
	}}
	svc := appsetup.NewService(newRepo(t), appsetup.Dependencies{
		Regime:  fr,
		Seeder:  &fakeSeeder{n: 24},
		Objects: fo,
		Periods: &fakePeriods{},
		Accounts: &fakeAccounts{accts: map[string]*ledger.Account{
			"1111": {Code: "1111", Status: ledger.AccountActive, AllowPost: true, Level: 2},
			"131":  {Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2},
			"331":  {Code: "331", Status: ledger.AccountActive, AllowPost: true, Level: 2},
			"5111": {Code: "5111", Status: ledger.AccountActive, AllowPost: true, Level: 2},
		}},
		Postings: &fakePostings{},
		Audit:    &fakeAudit{},
	})
	root := repoRoot(t)
	tmpl := template.Must(template.New("").Funcs(httpwebsetup.Funcs()).ParseGlob(filepath.Join(root, "web/templates/*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join(root, "web/templates/setup/*.html")))
	r := gin.New()
	r.SetHTMLTemplate(tmpl)
	httpwebsetup.NewHandler(svc, "X-User-Id").Register(r)
	return &harness{router: r}
}

func (h *harness) get(path, user string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func (h *harness) post(path string, vals url.Values, user string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func mustInit(t *testing.T, h *harness) {
	t.Helper()
	fy := time.Now().Format("2006-01") + "-01"
	w := h.post("/setup/start", url.Values{
		"name":                 {"Cty TNHH SX Thép ABC"},
		"tax_code":             {"0101234567"},
		"address":              {"Số 1, đường ABC, Hà Nội"},
		"legal_representative": {"Nguyễn Văn A"},
		"regime":               {"TT99-2025"},
		"fiscal_year_start":    {fy},
		"seed_accounts":        {"on"},
		"open_periods":         {"on"},
	}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("init status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
}

// --- tests ------------------------------------------------------------------

func TestWizard_EmptyShowsStart(t *testing.T) {
	h := newHarness(t)
	w := h.get("/setup", "ketoan")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Chào mừng đến goGL", "Bắt đầu khởi tạo", "Thông tin doanh nghiệp"} {
		if !strings.Contains(body, want) {
			t.Errorf("wizard body missing %q", want)
		}
	}
	if strings.Contains(body, "Trạng thái hệ thống") {
		t.Error("wizard shows status dashboard when EMPTY")
	}
}

func TestWizard_AfterInitShowsSteps(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	w := h.get("/setup", "ketoan")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Trạng thái hệ thống", "Đang nhập số dư đầu kỳ", "Số dư đầu kỳ", "Nhập số dư đầu kỳ", "Nhật ký hoạt động"} {
		if !strings.Contains(body, want) {
			t.Errorf("wizard body missing %q", want)
		}
	}
	// R13: the audit trail section shows at least one setup action
	if !strings.Contains(body, "initialize.profile") {
		t.Errorf("wizard trail must list setup actions, got body head:\n%s", body[:500])
	}
}

func TestAccountsPage_ListsSeededChart(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	w := h.get("/setup/accounts", "ketoan")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Bước 3 · Sơ đồ tài khoản", "TK đã tạo (4)", "1111", "5111", "postable", "Đến bước 4"} {
		if !strings.Contains(body, want) {
			t.Errorf("accounts body missing %q", want)
		}
	}
	// wizard links to the preview from the step list
	w = h.get("/setup", "ketoan")
	if !strings.Contains(w.Body.String(), "/setup/accounts") {
		t.Error("wizard must link to the accounts preview")
	}
}

func TestStartForm_RequiresActor(t *testing.T) {
	h := newHarness(t)
	w := h.post("/setup/start", url.Values{"name": {"x"}}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestStartForm_RendersForm(t *testing.T) {
	h := newHarness(t)
	w := h.get("/setup/start", "ketoan")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Thông tin doanh nghiệp", "Chế độ kế toán", "fiscal_year_start", "TT99-2025"} {
		if !strings.Contains(body, want) {
			t.Errorf("start form missing %q", want)
		}
	}
}

func TestStart_Initializes(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	w := h.get("/setup", "ketoan")
	if !strings.Contains(w.Body.String(), "Đang nhập số dư đầu kỳ") {
		t.Errorf("after init expected BALANCES_DRAFT, got:\n%s", w.Body.String())
	}
}

func TestStart_InvalidRegimeRedirectsWithErr(t *testing.T) {
	h := newHarness(t)
	fy := time.Now().Format("2006-01") + "-01"
	w := h.post("/setup/start", url.Values{
		"name": {"Cty"}, "tax_code": {"0101234567"}, "address": {"a"},
		"legal_representative": {"A"}, "regime": {"TT-OLD"}, "fiscal_year_start": {fy},
	}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/setup/start") || !strings.Contains(loc, "err=") {
		t.Errorf("redirect = %q, want /setup/start?err=...", loc)
	}
}

func TestBalances_EmptyState(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	w := h.get("/setup/balances", "ketoan")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Chưa có dữ liệu") {
		t.Errorf("empty balances page should show empty state")
	}
}

func TestBalances_AddRowAndCheck(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)

	w := h.post("/setup/balances", url.Values{
		"account_code": {"1111"}, "debit": {"500000000"}, "credit": {"0"},
	}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
	w = h.post("/setup/balances", url.Values{
		"account_code": {"131"}, "object_type": {"customer"}, "object_code": {"KH-0001"},
		"debit": {"45500000"}, "credit": {"0"},
	}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add obj status = %d, want 303", w.Code)
	}
	w = h.post("/setup/balances", url.Values{
		"account_code": {"331"}, "object_type": {"vendor"}, "object_code": {"NCC-0003"},
		"debit": {"0"}, "credit": {"50000000"},
	}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add credit status = %d, want 303", w.Code)
	}

	page := h.get("/setup/balances", "ketoan")
	body := page.Body.String()
	for _, want := range []string{"1111", "131", "KH-0001", "500.000.000", "45.500.000"} {
		if !strings.Contains(body, want) {
			t.Errorf("balances page missing %q", want)
		}
	}
	if !strings.Contains(body, "Chưa cân đối") && !strings.Contains(body, "Lệch") {
		t.Errorf("expected unbalanced banner, body:\n%s", body)
	}
}

func TestBalances_LockAfterBalanced(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")
	h.post("/setup/balances", url.Values{"account_code": {"5111"}, "credit": {"1000000"}}, "ketoan")

	w := h.post("/setup/balances/lock", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("lock status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if strings.Contains(loc, "err=") {
		t.Fatalf("lock failed: %s", loc)
	}

	page := h.get("/setup/balances", "ketoan")
	body := page.Body.String()
	for _, want := range []string{"Số dư đã khóa", "Lý do mở lại"} {
		if !strings.Contains(body, want) {
			t.Errorf("locked page missing %q", want)
		}
	}
	if strings.Contains(body, "+ Thêm số dư") {
		t.Error("locked page should not show add form")
	}
}

func TestBalances_ReopenRequiresReason(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")
	h.post("/setup/balances", url.Values{"account_code": {"5111"}, "credit": {"1000000"}}, "ketoan")
	h.post("/setup/balances/lock", nil, "ketoan")

	w := h.post("/setup/balances/reopen", url.Values{"reason": {"sai số dư"}}, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("reopen status = %d, want 303", w.Code)
	}
	if strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("reopen with reason failed: %s", w.Header().Get("Location"))
	}
	page := h.get("/setup/balances", "ketoan")
	if !strings.Contains(page.Body.String(), "+ Thêm số dư") {
		t.Error("after reopen the add form should be back")
	}
}

func TestBalances_CheckPost(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")

	w := h.post("/setup/balances/check", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("check status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("check failed: %s", w.Header().Get("Location"))
	}
}

func TestReopen_NoReasonRejected(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")
	h.post("/setup/balances", url.Values{"account_code": {"5111"}, "credit": {"1000000"}}, "ketoan")
	h.post("/setup/balances/lock", nil, "ketoan")

	w := h.post("/setup/balances/reopen", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "err=") {
		t.Errorf("reopen without reason should redirect with err")
	}
}

func TestActivate_AfterLock(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")
	h.post("/setup/balances", url.Values{"account_code": {"5111"}, "credit": {"1000000"}}, "ketoan")
	h.post("/setup/balances/lock", nil, "ketoan")

	w := h.post("/setup/activate", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("activate status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("activate failed: %s", w.Header().Get("Location"))
	}
	page := h.get("/setup", "ketoan")
	if !strings.Contains(page.Body.String(), "Hệ thống đã hoạt động") {
		t.Errorf("wizard should show ACTIVE state, body:\n%s", page.Body.String())
	}
}

func TestActivate_BeforeLockFails(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	w := h.post("/setup/activate", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "err=") {
		t.Errorf("activate before lock should fail with err, got %q", w.Header().Get("Location"))
	}
}

func TestDeleteBalance(t *testing.T) {
	h := newHarness(t)
	mustInit(t, h)
	h.post("/setup/balances", url.Values{"account_code": {"1111"}, "debit": {"1000000"}}, "ketoan")

	page := h.get("/setup/balances", "ketoan")
	// find the delete form action: /setup/balances/<id>/delete
	body := page.Body.String()
	idx := strings.Index(body, "/setup/balances/")
	if idx < 0 {
		t.Fatal("no balance row rendered with delete action")
	}
	start := idx + len("/setup/balances/")
	end := strings.Index(body[start:], "/delete")
	if end < 0 {
		t.Fatal("no /delete action found")
	}
	id := body[start : start+end]

	w := h.post("/setup/balances/"+id+"/delete", nil, "ketoan")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", w.Code)
	}
	if strings.Contains(w.Header().Get("Location"), "err=") {
		t.Fatalf("delete failed: %s", w.Header().Get("Location"))
	}
	page = h.get("/setup/balances", "ketoan")
	if strings.Contains(page.Body.String(), "Chưa có dữ liệu") == false {
		t.Error("deleted balance still shown")
	}
}
