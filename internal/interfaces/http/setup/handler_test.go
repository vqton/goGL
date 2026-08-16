package setup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	httpsetup "goGL/internal/interfaces/http/setup"

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
	return out, nil
}

type fakePostings struct {
	entries []*ledger.JournalEntry
	err     error
}

func (f *fakePostings) ListEntries(_ context.Context, _ ledger.EntryFilter) ([]*ledger.JournalEntry, error) {
	return f.entries, f.err
}

type fakeAudit struct{ err error }

func (f *fakeAudit) Record(_ context.Context, _ *audit.AuditLog) error { return f.err }
func (f *fakeAudit) ListRecent(context.Context, string, int) ([]*audit.AuditLog, error) {
	return nil, f.err
}

// --- helpers ----------------------------------------------------------------

func newRepo(t *testing.T) setup.Repository {
	t.Helper()
	dsn := fmt.Sprintf("file:handler_%p?mode=memory&cache=shared", t)
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

type harness struct {
	router  *gin.Engine
	regime  *fakeRegime
	objects *fakeObjects
	audit   *fakeAudit
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fr := &fakeRegime{cur: "TT99-2025"}
	fo := &fakeObjects{recs: map[string]*masterdata.Record{
		"customer:KH-0001": {Code: "KH-0001", Kind: masterdata.KindCustomer, State: masterdata.StateActive},
	}}
	fa := &fakeAudit{}
	svc := appsetup.NewService(newRepo(t), appsetup.Dependencies{
		Regime:  fr,
		Seeder:  &fakeSeeder{n: 24},
		Objects: fo,
		Periods: &fakePeriods{},
		Accounts: &fakeAccounts{accts: map[string]*ledger.Account{
			"1111": {Code: "1111", Status: ledger.AccountActive, AllowPost: true, Level: 2},
			"131":  {Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2},
			"5111": {Code: "5111", Status: ledger.AccountActive, AllowPost: true, Level: 2},
		}},
		Postings: &fakePostings{},
		Audit:    fa,
	})
	r := gin.New()
	httpsetup.NewHandler(svc, "X-User-Id").Register(r.Group("/api/v1"))
	return &harness{router: r, regime: fr, objects: fo, audit: fa}
}

func (h *harness) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "ketoan")
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func validProfile() *setup.CompanyProfile {
	return &setup.CompanyProfile{
		ID:                  setup.ProfileID,
		Name:                "Cty TNHH SX Thép ABC",
		TaxCode:             "0101234567",
		Address:             "Số 1, đường ABC, Hà Nội",
		LegalRepresentative: "Nguyễn Văn A",
		AccountingCurrency:  "VND",
		FiscalYearStart:     time.Now().Format("2006-01") + "-01",
		AccountingRegime:    "TT99-2025",
	}
}

func initializeBody() string {
	p, _ := json.Marshal(validProfile())
	return fmt.Sprintf(`{"profile":%s,"regime":"TT99-2025","fiscal_year_start":%q,"seed_accounts":true,"open_periods":true}`,
		p, time.Now().Format("2006-01")+"-01")
}

func balanceRow(account, side string, amt int64) string {
	return fmt.Sprintf(`{"account_code":%q,"%s":{"amount_minor":%d,"currency":"VND"}}`, account, side, amt)
}

// --- tests ------------------------------------------------------------------

func TestStatus_Empty(t *testing.T) {
	h := newHarness(t)
	w := h.do(http.MethodGet, "/api/v1/setup/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var v appsetup.StatusView
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v.Status != setup.StatusEmpty {
		t.Fatalf("status = %q, want empty", v.Status)
	}
}

func TestInitialize_FullRun(t *testing.T) {
	h := newHarness(t)
	w := h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var v appsetup.StatusView
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v.Status != setup.StatusBalancesDraft {
		t.Fatalf("status = %q, want balances_draft", v.Status)
	}
	if v.Profile == nil || v.Profile.Name != "Cty TNHH SX Thép ABC" {
		t.Fatalf("profile missing: %+v", v.Profile)
	}
	if h.regime.cur != "TT99-2025" {
		t.Fatalf("regime = %q", h.regime.cur)
	}
}

func TestInitialize_ResumeIdempotent(t *testing.T) {
	h := newHarness(t)
	if w := h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody()); w.Code != http.StatusOK {
		t.Fatalf("first init: %d %s", w.Code, w.Body.String())
	}
	w := h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	if w.Code != http.StatusOK {
		t.Fatalf("re-init status = %d, want 200 (idempotent); body: %s", w.Code, w.Body.String())
	}
	var v appsetup.StatusView
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if v.Status != setup.StatusBalancesDraft {
		t.Fatalf("status = %q, want balances_draft", v.Status)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	h := newHarness(t)
	w := h.do(http.MethodGet, "/api/v1/setup/profile", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestGetProfile_AfterInit(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodGet, "/api/v1/setup/profile", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var p setup.CompanyProfile
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.Name != "Cty TNHH SX Thép ABC" {
		t.Fatalf("name = %q", p.Name)
	}
}

func TestUpdateProfile(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	p := validProfile()
	p.Name = "Cty TNHH SX Thép ABC (đổi tên)"
	body, _ := json.Marshal(p)
	w := h.do(http.MethodPut, "/api/v1/setup/profile", string(body))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var got setup.CompanyProfile
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Name != p.Name {
		t.Fatalf("name = %q, want %q", got.Name, p.Name)
	}
}

func TestUpdateProfile_InvalidBody(t *testing.T) {
	h := newHarness(t)
	w := h.do(http.MethodPut, "/api/v1/setup/profile", `{not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProfile_Validation422(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	p := validProfile()
	p.TaxCode = "bogus"
	body, _ := json.Marshal(p)
	w := h.do(http.MethodPut, "/api/v1/setup/profile", string(body))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestSaveBalance_ThenCheckAndList(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())

	// two sides must balance; post two rows so check passes
	rows := []string{
		balanceRow("1111", "debit", 500000000),
		balanceRow("5111", "credit", 500000000),
	}
	for _, row := range rows {
		w := h.do(http.MethodPost, "/api/v1/setup/opening-balances", row)
		if w.Code != http.StatusOK {
			t.Fatalf("save balance: %d %s", w.Code, w.Body.String())
		}
	}
	w := h.do(http.MethodGet, "/api/v1/setup/opening-balances", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	var bl appsetup.BalanceList
	if err := json.Unmarshal(w.Body.Bytes(), &bl); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(bl.Balances) != 2 {
		t.Fatalf("len = %d, want 2", len(bl.Balances))
	}
	if bl.Check == nil || !bl.Check.Balanced {
		t.Fatalf("check = %+v, want balanced", bl.Check)
	}

	// check endpoint agrees
	w = h.do(http.MethodPost, "/api/v1/setup/opening-balances/check", "")
	if w.Code != http.StatusOK {
		t.Fatalf("check: %d %s", w.Code, w.Body.String())
	}
	var ck setup.BalanceCheck
	if err := json.Unmarshal(w.Body.Bytes(), &ck); err != nil {
		t.Fatalf("decode check: %v", err)
	}
	if !ck.Balanced || ck.Diff != 0 {
		t.Fatalf("check = %+v, want balanced diff 0", ck)
	}
}

func TestSaveBalance_ObjectRequired422(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances", balanceRow("131", "debit", 1000000))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestSaveBalance_ObjectOK(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances",
		`{"account_code":"131","object_type":"customer","object_code":"KH-0001","debit":{"amount_minor":1000000,"currency":"VND"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestSaveBalance_BadJSON(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances", `{nope`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestLock_Unbalanced409(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/lock", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", w.Code, w.Body.String())
	}
}

func TestLock_Reopen_Activate_Lifecycle(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	for _, row := range []string{
		balanceRow("1111", "debit", 500000000),
		balanceRow("5111", "credit", 500000000),
	} {
		if w := h.do(http.MethodPost, "/api/v1/setup/opening-balances", row); w.Code != http.StatusOK {
			t.Fatalf("save: %d %s", w.Code, w.Body.String())
		}
	}

	if w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/lock", ""); w.Code != http.StatusOK {
		t.Fatalf("lock: %d %s", w.Code, w.Body.String())
	}
	// locked → save rejected
	if w := h.do(http.MethodPost, "/api/v1/setup/opening-balances",
		balanceRow("1111", "credit", 1000000)); w.Code != http.StatusConflict {
		t.Fatalf("save after lock: %d, want 409; body: %s", w.Code, w.Body.String())
	}

	if w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/reopen", `{"reason":"cần chỉnh sửa"}`); w.Code != http.StatusOK {
		t.Fatalf("reopen: %d %s", w.Code, w.Body.String())
	}

	if w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/lock", ""); w.Code != http.StatusOK {
		t.Fatalf("lock again: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodPost, "/api/v1/setup/activate", ""); w.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", w.Code, w.Body.String())
	}
	// already active → initialize is 409
	if w := h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody()); w.Code != http.StatusConflict {
		t.Fatalf("init after activate: %d, want 409; body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBalance(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances", balanceRow("1111", "debit", 500000000))
	var saved setup.OpeningBalance
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	if saved.ID == "" {
		t.Fatalf("no id returned")
	}
	if w := h.do(http.MethodDelete, "/api/v1/setup/opening-balances/"+saved.ID, ""); w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodDelete, "/api/v1/setup/opening-balances/missing-id", ""); w.Code != http.StatusNotFound {
		t.Fatalf("delete missing: %d, want 404", w.Code)
	}
}

func TestImportBalances_DryRun(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	rows := `{"dry_run":true,"rows":[
		["account","object_type","object_code","debit","credit"],
		["1111","","","500000000",""],
		["5111","","","","500000000"]
	]}`
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/import", rows)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var res setup.ImportResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Created != 0 || res.Updated != 0 || len(res.Errors) != 0 {
		t.Fatalf("dry-run must not persist: %+v", res)
	}
	if res.Total != 2 {
		t.Fatalf("total = %d, want 2; %+v", res.Total, res)
	}
}

func TestImportBalances_Commit(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	rows := `{"dry_run":false,"rows":[
		["account","object_type","object_code","debit","credit"],
		["1111","","","500000000",""],
		["5111","","","","500000000"]
	]}`
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/import", rows)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var res setup.ImportResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Created != 2 {
		t.Fatalf("created = %d, want 2; %+v", res.Created, res)
	}
}

func TestImportBalances_InvalidRow422(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())
	rows := `{"dry_run":true,"rows":[
		["account","object_type","object_code","debit","credit"],
		["131","","","1000000",""]
	]}`
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/import", rows)
	if w.Code != http.StatusOK {
		t.Fatalf("import status = %d, want 200 (row errors reported inline); body: %s", w.Code, w.Body.String())
	}
	var res setup.ImportResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Errors) != 1 || res.Errors[0].Message == "" {
		t.Fatalf("expected 1 row error: %+v", res.Errors)
	}
}

func TestImportReport_PersistedJob(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())

	rows := `{"dry_run":true,"rows":[
		["account","object_type","object_code","debit","credit"],
		["1111","","","1000000",""],
		["9999","","","1",""]
	]}`
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/import", rows)
	if w.Code != http.StatusOK {
		t.Fatalf("import status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var res setup.ImportResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.JobID == "" {
		t.Fatal("import must return a persisted job id")
	}

	// report endpoint serves the same per-row errors later
	w = h.do(http.MethodGet, "/api/v1/setup/opening-balances/import/"+res.JobID+"/report", "")
	if w.Code != http.StatusOK {
		t.Fatalf("report status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var job setup.ImportJob
	_ = json.Unmarshal(w.Body.Bytes(), &job)
	if job.ID != res.JobID || job.Status != setup.JobErrored || job.Total != 2 || len(job.Errors) != 1 {
		t.Fatalf("report mismatch: %+v", job)
	}
	if !job.DryRun || job.CreatedBy != "ketoan" {
		t.Fatalf("report metadata: dry_run=%v created_by=%s", job.DryRun, job.CreatedBy)
	}

	// unknown job -> 404
	w = h.do(http.MethodGet, "/api/v1/setup/opening-balances/import/nope/report", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown job status = %d, want 404", w.Code)
	}
}

func TestImportReport_ErrorsCSV(t *testing.T) {
	h := newHarness(t)
	h.do(http.MethodPost, "/api/v1/setup/initialize", initializeBody())

	rows := `{"dry_run":true,"rows":[
		["account","object_type","object_code","debit","credit"],
		["1111","","","1000000",""],
		["9999","","","1",""]
	]}`
	w := h.do(http.MethodPost, "/api/v1/setup/opening-balances/import", rows)
	var res setup.ImportResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	w = h.do(http.MethodGet, "/api/v1/setup/opening-balances/import/"+res.JobID+"/errors.csv", "")
	if w.Code != http.StatusOK {
		t.Fatalf("errors.csv status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment;") {
		t.Fatalf("content-disposition = %q", cd)
	}
	body := w.Body.String()
	if !strings.Contains(body, "row,column,message") || !strings.Contains(body, "account does not exist") {
		t.Fatalf("errors.csv must contain header and failed-row reason: %q", body)
	}
}
