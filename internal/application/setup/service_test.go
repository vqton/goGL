package setup_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"

	appsetup "goGL/internal/application/setup"
	"goGL/internal/domain/audit"
	"goGL/internal/domain/core"
	"goGL/internal/domain/ledger"
	"goGL/internal/domain/masterdata"
	"goGL/internal/domain/setup"
	"goGL/internal/infrastructure/db"
	persistence "goGL/internal/infrastructure/persistence/setup"
	_ "modernc.org/sqlite"
) // --- fake seams -------------------------------------------------------------

type fakeRegime struct {
	cur       string
	calls     int
	last      string
	lastActor string
	err       error
}

func (f *fakeRegime) SetRegime(_ context.Context, regime, actor string) error {
	f.calls++
	f.last, f.lastActor = regime, actor
	if f.err != nil {
		return f.err
	}
	f.cur = regime
	return nil
}
func (f *fakeRegime) GetRegime(_ context.Context) (string, error) { return f.cur, nil }

type fakeSeeder struct {
	calls int
	n     int
	err   error
}

func (f *fakeSeeder) SeedAccounts(_ context.Context, _ string) (int, error) {
	f.calls++
	return f.n, f.err
}

type fakeObjects struct {
	recs map[string]*masterdata.Record // key "kind:code"
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

type fakePeriods struct {
	calls int
	ids   []string
	err   error
}

func (f *fakePeriods) OpenPeriod(_ context.Context, _ string, id string) (*ledger.AccountingPeriod, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls++
	f.ids = append(f.ids, id)
	return &ledger.AccountingPeriod{ID: id, Year: 2026, Month: 1, Status: ledger.PeriodOpen}, nil
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
	logs []*audit.AuditLog
}

func (f *fakeAudit) Record(_ context.Context, l *audit.AuditLog) error {
	f.logs = append(f.logs, l)
	return nil
}

func (f *fakeAudit) ListRecent(_ context.Context, module string, limit int) ([]*audit.AuditLog, error) {
	var out []*audit.AuditLog
	// newest-first (records are appended chronologically)
	for i := len(f.logs) - 1; i >= 0 && len(out) < limit; i-- {
		if module != "" && f.logs[i].Module != module {
			continue
		}
		out = append(out, f.logs[i])
	}
	return out, nil
}

// --- helpers ----------------------------------------------------------------

func validProfile() *setup.CompanyProfile {
	return &setup.CompanyProfile{
		ID:                  setup.ProfileID,
		Name:                "Cty TNHH SX Thép ABC",
		TaxCode:             "0101234567",
		Address:             "Số 1, đường ABC, Hà Nội",
		LegalRepresentative: "Nguyễn Văn A",
		AccountingCurrency:  "VND",
		FiscalYearStart:     "2026-01-01",
		AccountingRegime:    "TT99-2025",
	}
}

func leafAccount(code string) *ledger.Account {
	return &ledger.Account{Code: code, Status: ledger.AccountActive, AllowPost: true, Level: 2}
}

func newRepo(t *testing.T) setup.Repository {
	t.Helper()
	dsn := fmt.Sprintf("file:setup_svc_%p?mode=memory&cache=shared", t)
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

func newService(t *testing.T, opts ...func(*deps)) (appsetup.Service, *fakeRegime, *fakeSeeder, *fakeObjects, *fakePeriods, *fakeAccounts, *fakePostings, *fakeAudit, setup.Repository, func() setup.SetupStatus) {
	repo := newRepo(t)
	fr := &fakeRegime{cur: "TT99-2025"}
	fs := &fakeSeeder{n: 5}
	fo := &fakeObjects{recs: map[string]*masterdata.Record{}}
	fp := &fakePeriods{}
	fac := &fakeAccounts{accts: map[string]*ledger.Account{}}
	fpost := &fakePostings{}
	fa := &fakeAudit{}
	d := deps{fr: fr, fs: fs, fo: fo, fp: fp, fac: fac, fpost: fpost, fa: fa}
	for _, o := range opts {
		o(&d)
	}
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime:   fr,
		Seeder:   fs,
		Objects:  fo,
		Periods:  fp,
		Accounts: fac,
		Postings: fpost,
		Audit:    fa,
	})
	getStatus := func() setup.SetupStatus {
		st, _ := repo.GetStatus(context.Background())
		return st
	}
	return svc, fr, fs, fo, fp, fac, fpost, fa, repo, getStatus
}

type deps struct {
	fr    *fakeRegime
	fs    *fakeSeeder
	fo    *fakeObjects
	fp    *fakePeriods
	fac   *fakeAccounts
	fpost *fakePostings
	fa    *fakeAudit
}

// --- P0: status & profile ----------------------------------------------------

func TestStatusStartsEmpty(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, getStatus := newService(t)
	st, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.Status != setup.StatusEmpty {
		t.Fatalf("want empty, got %s", st.Status)
	}
	if getStatus() != setup.StatusEmpty {
		t.Fatalf("repo status not empty")
	}
}

func TestGetProfileBeforeInitialize(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _ := newService(t)
	if _, err := svc.GetProfile(context.Background()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("want sql.ErrNoRows, got %v", err)
	}
}

// --- P1: initialize -----------------------------------------------------------

func TestInitializeInvalidProfile(t *testing.T) {
	bad := []func(p *setup.CompanyProfile){
		func(p *setup.CompanyProfile) { p.Name = "" },
		func(p *setup.CompanyProfile) { p.TaxCode = "123" },
		func(p *setup.CompanyProfile) { p.TaxCode = "0101234567 0" },
		func(p *setup.CompanyProfile) { p.AccountingCurrency = "USD" },
		func(p *setup.CompanyProfile) { p.FiscalYearStart = "2026-02-15" },
		func(p *setup.CompanyProfile) { p.FiscalYearStart = "2026-13-01" },
		func(p *setup.CompanyProfile) { p.AccountingRegime = "TT111" },
	}
	for _, mut := range bad {
		p := validProfile()
		mut(p)
		svc, _, _, _, _, _, _, _, _, getStatus := newService(t)
		_, err := svc.Initialize(context.Background(), &appsetup.InitializeRequest{
			Profile: p, Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
			SeedAccounts: true, OpenPeriods: true,
		}, "user-1")
		if err == nil {
			t.Fatalf("expected validation error for %+v", p)
		}
		if st := getStatus(); st != setup.StatusEmpty {
			t.Fatalf("status advanced on invalid profile: %s", st)
		}
	}
}

func TestInitializeFullFlow(t *testing.T) {
	svc, fr, fs, _, fp, _, _, fa, _, getStatus := newService(t)
	p := validProfile()
	view, err := svc.Initialize(context.Background(), &appsetup.InitializeRequest{
		Profile: p, Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1")
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if view.Status != setup.StatusBalancesDraft {
		t.Fatalf("want balances_draft, got %s", view.Status)
	}
	if st := getStatus(); st != setup.StatusBalancesDraft {
		t.Fatalf("repo status %s", st)
	}
	if fr.calls != 1 || fr.last != "TT99-2025" || fr.lastActor != "user-1" {
		t.Fatalf("regime seam not called once: calls=%d last=%q actor=%q", fr.calls, fr.last, fr.lastActor)
	}
	if fs.calls != 1 {
		t.Fatalf("seed seam calls=%d", fs.calls)
	}
	if fp.calls != 12 {
		t.Fatalf("open periods calls=%d, want 12", fp.calls)
	}
	if fp.ids[0] != "2026-01" || fp.ids[11] != "2026-12" {
		t.Fatalf("unexpected periods: %v", fp.ids)
	}
	if len(fa.logs) == 0 {
		t.Fatalf("no audit trail (R13)")
	}
	got, err := svc.GetProfile(context.Background())
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got.Name != p.Name || got.TaxCode != "0101234567" {
		t.Fatalf("profile roundtrip mismatch: %+v", got)
	}
}

func TestInitializeResumesFromProfiled(t *testing.T) {
	repo := newRepo(t)
	fr := &fakeRegime{}
	fs := &fakeSeeder{n: 5}
	fp := &fakePeriods{}
	fac := &fakeAccounts{accts: map[string]*ledger.Account{}}
	fo := &fakeObjects{recs: map[string]*masterdata.Record{}}
	fa := &fakeAudit{}
	ctx := context.Background()
	_ = repo.SetStatus(ctx, setup.StatusProfiled)
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime: fr, Seeder: fs, Objects: fo, Periods: fp, Accounts: fac, Postings: &fakePostings{}, Audit: fa,
	})
	view, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1")
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if view.Status != setup.StatusBalancesDraft {
		t.Fatalf("want balances_draft, got %s", view.Status)
	}
	if fr.calls != 1 || fs.calls != 1 || fp.calls != 12 {
		t.Fatalf("resume should only run pending steps: regime=%d seed=%d periods=%d", fr.calls, fs.calls, fp.calls)
	}
}

func TestInitializeIdempotentAtBalancesDraft(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	_ = repo.SetStatus(ctx, setup.StatusBalancesDraft)
	fr, fs, fp := &fakeRegime{}, &fakeSeeder{}, &fakePeriods{}
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime: fr, Seeder: fs, Objects: &fakeObjects{}, Periods: fp,
		Accounts: &fakeAccounts{}, Postings: &fakePostings{}, Audit: &fakeAudit{},
	})
	view, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1")
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if view.Status != setup.StatusBalancesDraft {
		t.Fatalf("want balances_draft, got %s", view.Status)
	}
	if fr.calls+fs.calls+fp.calls != 0 {
		t.Fatalf("idempotent re-run must not re-apply steps: regime=%d seed=%d periods=%d", fr.calls, fs.calls, fp.calls)
	}
}

func TestInitializeBlockedAfterLock(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	_ = repo.SetStatus(ctx, setup.StatusBalancesLocked)
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime: &fakeRegime{}, Seeder: &fakeSeeder{}, Objects: &fakeObjects{},
		Periods: &fakePeriods{}, Accounts: &fakeAccounts{}, Postings: &fakePostings{}, Audit: &fakeAudit{},
	})
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); !errors.Is(err, setup.ErrAlreadyInitialized) {
		t.Fatalf("want ErrAlreadyInitialized, got %v", err)
	}
}

func TestInitializeConcurrent(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	var fr fakeRegime
	var fs fakeSeeder
	var fp fakePeriods
	fr.cur = "TT99-2025"
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime: &fr, Seeder: &fs, Objects: &fakeObjects{}, Periods: &fp,
		Accounts: &fakeAccounts{}, Postings: &fakePostings{}, Audit: &fakeAudit{},
	})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.Initialize(ctx, &appsetup.InitializeRequest{
				Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
				SeedAccounts: true, OpenPeriods: true,
			}, "user-1")
		}(i)
	}
	wg.Wait()
	st, _ := repo.GetStatus(ctx)
	if st != setup.StatusBalancesDraft {
		t.Fatalf("final status %s, want balances_draft", st)
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("init %d: %v", i, err)
		}
	}
	if fp.calls != 12 {
		t.Fatalf("periods opened %d times, want exactly 12 (once)", fp.calls)
	}
}

// --- P2: opening balances ----------------------------------------------------

func TestSaveBalanceRequiresDraftState(t *testing.T) {
	for _, st := range []setup.SetupStatus{setup.StatusEmpty, setup.StatusProfiled, setup.StatusBalancesLocked, setup.StatusActive} {
		repo := newRepo(t)
		ctx := context.Background()
		_ = repo.SetStatus(ctx, st)
		svc := appsetup.NewService(repo, appsetup.Dependencies{
			Regime: &fakeRegime{}, Seeder: &fakeSeeder{}, Objects: &fakeObjects{},
			Periods: &fakePeriods{}, Accounts: &fakeAccounts{accts: map[string]*ledger.Account{"1111": leafAccount("1111")}},
			Postings: &fakePostings{}, Audit: &fakeAudit{},
		})
		_, err := svc.SaveBalance(ctx, &setup.OpeningBalance{
			AccountCode: "1111",
			Debit:       core.Money{AmountMinor: 100, Currency: "VND"},
		}, "user-1")
		if err == nil {
			t.Fatalf("status %s: expected error", st)
		}
	}
}

func TestSaveBalanceRules(t *testing.T) {
	svc, _, _, _, _, fac, _, _, _, _ := newService(t)
	ctx := context.Background()
	// reach BALANCES_DRAFT
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["11"] = &ledger.Account{Code: "11", Status: ledger.AccountActive, AllowPost: false, Level: 1}
	fac.accts["131"] = leafAccount("131")
	fac.accts["331"] = leafAccount("331")
	fac.accts["152"] = leafAccount("152")
	fac.accts["211"] = leafAccount("211")

	cases := []struct {
		name string
		b    *setup.OpeningBalance
		want error
	}{
		{"both sides zero", &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{Currency: "VND"}, Credit: core.Money{Currency: "VND"}}, setup.ErrInvalidBalance},
		{"both sides non-zero", &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{AmountMinor: 100, Currency: "VND"}, Credit: core.Money{AmountMinor: 100, Currency: "VND"}}, setup.ErrInvalidBalance},
		{"negative amount", &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{AmountMinor: -1, Currency: "VND"}}, setup.ErrInvalidBalance},
		{"unknown account", &setup.OpeningBalance{AccountCode: "9999", Debit: core.Money{AmountMinor: 100, Currency: "VND"}}, setup.ErrAccountNotFound},
		{"summary account not postable", &setup.OpeningBalance{AccountCode: "11", Debit: core.Money{AmountMinor: 100, Currency: "VND"}, ObjectType: "customer", ObjectCode: "KH-0001"}, setup.ErrAccountNotFound},
		{"131 needs object", &setup.OpeningBalance{AccountCode: "131", Debit: core.Money{AmountMinor: 100, Currency: "VND"}}, setup.ErrObjectRequired},
		{"131 unknown object", &setup.OpeningBalance{AccountCode: "131", Debit: core.Money{AmountMinor: 100, Currency: "VND"}, ObjectType: "customer", ObjectCode: "KH-0002"}, setup.ErrObjectNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SaveBalance(ctx, tc.b, "user-1")
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
	list, err := svc.ListBalances(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Balances) != 0 {
		t.Fatalf("no balance should be saved after failed validations, got %d", len(list.Balances))
	}
}

func TestSaveBalanceHappyPathAndUpsert(t *testing.T) {
	svc, _, _, fo, _, fac, _, _, repo, _ := newService(t)
	ctx := context.Background()
	fo.recs["customer:KH-0001"] = &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-0001", Name: "Cty B", State: masterdata.StateActive}
	fac.accts["131"] = &ledger.Account{Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2}

	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	b := &setup.OpeningBalance{AccountCode: "131", ObjectType: "customer", ObjectCode: "KH-0001",
		Debit: core.Money{AmountMinor: 500_000_000, Currency: "VND"}}
	saved, err := svc.SaveBalance(ctx, b, "user-1")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Period.From != "2026-01-01" || saved.Period.To != "2026-01-01" {
		t.Fatalf("period not set to FY start: %+v", saved.Period)
	}
	if saved.Credit.AmountMinor != 0 || saved.Debit.AmountMinor != 500_000_000 {
		t.Fatalf("side not preserved: %+v", saved)
	}
	// upsert idempotency
	saved.Debit.AmountMinor = 600_000_000
	if _, err := svc.SaveBalance(ctx, saved, "user-1"); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	list, _ := svc.ListBalances(ctx, "")
	if len(list.Balances) != 1 || list.Balances[0].Debit.AmountMinor != 600_000_000 {
		t.Fatalf("want 1 upserted balance: %+v", list.Balances)
	}
	_ = repo
}

func TestCheckBalancesUnbalanced(t *testing.T) {
	svc, _, _, fo, _, fac, _, _, _, _ := newService(t)
	ctx := context.Background()
	fo.recs["customer:KH-0001"] = &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-0001", State: masterdata.StateActive}
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["131"] = &ledger.Account{Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2}
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, _ = svc.SaveBalance(ctx, &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{AmountMinor: 100, Currency: "VND"}}, "u")
	check, err := svc.CheckBalances(ctx)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if check.Balanced {
		t.Fatal("should be unbalanced")
	}
	if check.Debit != 100 || check.Credit != 0 || check.Diff != 100 {
		t.Fatalf("totals wrong: %+v", check)
	}
	if len(check.Offending) != 1 || check.Offending[0] != "1111" {
		t.Fatalf("offending accounts: %v", check.Offending)
	}
}

func TestLockReopenActivate(t *testing.T) {
	svc, _, _, fo, _, fac, fpost, fa, repo, getStatus := newService(t)
	ctx := context.Background()
	fo.recs["customer:KH-0001"] = &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-0001", State: masterdata.StateActive}
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["131"] = &ledger.Account{Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2}
	fac.accts["331"] = &ledger.Account{Code: "331", Status: ledger.AccountActive, AllowPost: true, Level: 2}

	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	// unbalanced -> lock blocked
	if err := svc.Lock(ctx, "ke-toan-truong"); !errors.Is(err, setup.ErrUnbalanced) {
		t.Fatalf("lock on unbalanced must fail, got %v", err)
	}
	// balance it: 1111 Nợ 100, 331 Có 100
	_, _ = svc.SaveBalance(ctx, &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{AmountMinor: 100, Currency: "VND"}}, "u")
	_, _ = svc.SaveBalance(ctx, &setup.OpeningBalance{AccountCode: "331", Credit: core.Money{AmountMinor: 100, Currency: "VND"}}, "u")
	if err := svc.Lock(ctx, "ke-toan-truong"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if st := getStatus(); st != setup.StatusBalancesLocked {
		t.Fatalf("after lock: %s", st)
	}
	// edit blocked while locked (R8)
	if _, err := svc.SaveBalance(ctx, &setup.OpeningBalance{AccountCode: "1111", Debit: core.Money{AmountMinor: 1, Currency: "VND"}}, "u"); !errors.Is(err, setup.ErrBalanceLocked) {
		t.Fatalf("edit while locked must fail, got %v", err)
	}
	// reopen without reason -> error (R12/R13)
	if err := svc.Reopen(ctx, "ke-toan-truong", ""); err == nil {
		t.Fatal("reopen without reason must fail")
	}
	// reopen blocked while posted refs exist
	fpost.entries = []*ledger.JournalEntry{{
		ID: "e1", Lines: []ledger.JournalLine{{AccountCode: "1111", Debit: 50}}, Status: ledger.EntryPosted,
	}}
	if err := svc.Reopen(ctx, "ke-toan-truong", "fix typo"); !errors.Is(err, setup.ErrReopenBlocked) {
		t.Fatalf("reopen with posted refs must be blocked, got %v", err)
	}
	fpost.entries = nil
	if err := svc.Reopen(ctx, "ke-toan-truong", "fix typo"); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if st := getStatus(); st != setup.StatusBalancesDraft {
		t.Fatalf("after reopen: %s", st)
	}
	// lock again then activate
	if err := svc.Lock(ctx, "ke-toan-truong"); err != nil {
		t.Fatalf("re-lock: %v", err)
	}
	if err := svc.Activate(ctx, "ke-toan-truong"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if st := getStatus(); st != setup.StatusActive {
		t.Fatalf("after activate: %s", st)
	}
	if len(fa.logs) < 4 {
		t.Fatalf("want lock/reopen/lock/activate audit entries, got %d", len(fa.logs))
	}
	_ = repo
}

func TestActivateRequiresLocked(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _ := newService(t)
	if err := svc.Activate(context.Background(), "u"); !errors.Is(err, setup.ErrWrongState) {
		t.Fatalf("activate from empty must fail, got %v", err)
	}
}

// --- P3: import ---------------------------------------------------------------

func TestImportBalances(t *testing.T) {
	svc, _, _, fo, _, fac, _, fa, repo, _ := newService(t)
	ctx := context.Background()
	fo.recs["customer:KH-0001"] = &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-0001", State: masterdata.StateActive}
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["331"] = leafAccount("331")
	fac.accts["131"] = &ledger.Account{Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2}
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}

	rows := [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "100", ""},
		{"331", "", "", "", "100"},
		{"131", "customer", "KH-0001", "50", ""}, // unbalanced overall
		{"131", "customer", "KH-0002", "50", ""}, // missing object -> error
		{"9999", "", "", "1", ""},                // unknown account
		{"1111", "", "", "abc", ""},              // bad amount
	}
	dry, err := svc.ImportBalances(ctx, rows, "user-1", true)
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}
	if !dry.DryRun {
		t.Fatal("dry run flag lost")
	}
	if dry.Total != 6 || dry.Created != 0 {
		t.Fatalf("dry run counts: total=%d created=%d", dry.Total, dry.Created)
	}
	if len(dry.Errors) != 3 {
		t.Fatalf("want 3 row errors (missing object, unknown account, bad amount), got %+v", dry.Errors)
	}
	// nothing persisted by dry-run
	list, _ := svc.ListBalances(ctx, "")
	if len(list.Balances) != 0 {
		t.Fatalf("dry-run must not persist, got %d rows", len(list.Balances))
	}

	// balanced subset without the errors
	clean := [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "150", ""},
		{"331", "", "", "", "100"},
		{"131", "customer", "KH-0001", "50", ""},
	}
	// wrong-side for 131 must produce object-required gap but balance passes
	_ = clean
	commit, err := svc.ImportBalances(ctx, [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "100", ""},
		{"331", "", "", "", "100"},
	}, "user-1", false)
	if err != nil {
		t.Fatalf("commit import: %v", err)
	}
	if commit.Created != 2 || commit.Total != 2 || len(commit.Errors) != 0 {
		t.Fatalf("commit counts: %+v", commit)
	}
	list, _ = svc.ListBalances(ctx, "")
	if len(list.Balances) != 2 {
		t.Fatalf("want 2 rows, got %d", len(list.Balances))
	}
	// re-import same -> updated, not created
	again, err := svc.ImportBalances(ctx, [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "100", ""},
		{"331", "", "", "", "100"},
	}, "user-1", false)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if again.Updated != 2 || again.Created != 0 {
		t.Fatalf("re-import should update 2: %+v", again)
	}
	if len(fa.logs) == 0 {
		t.Fatal("import must audit (R13)")
	}
	_ = repo
}

func TestImportBalances_TemplateVersionRejected(t *testing.T) {
	svc, _, _, _, _, fac, _, _, _, _ := newService(t)
	ctx := context.Background()
	fac.accts["1111"] = leafAccount("1111")
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}

	bad := [][]string{
		{"account", "object", "object_code", "debit", "credit"}, // v2 renamed column
		{"1111", "", "", "100", ""},
	}
	if _, err := svc.ImportBalances(ctx, bad, "user-1", true); !errors.Is(err, setup.ErrInvalidImport) {
		t.Fatalf("wrong header: err = %v, want ErrInvalidImport", err)
	}
	badOrder := [][]string{
		{"debit", "credit", "account", "object_type", "object_code"}, // shuffled
		{"100", "", "1111", "", ""},
	}
	if _, err := svc.ImportBalances(ctx, badOrder, "user-1", true); !errors.Is(err, setup.ErrInvalidImport) {
		t.Fatalf("shuffled header: err = %v, want ErrInvalidImport", err)
	}
	// header tolerates case/space differences but not missing columns
	if _, err := svc.ImportBalances(ctx, [][]string{{"account", "object_type", "object_code", "debit"}}, "user-1", true); !errors.Is(err, setup.ErrInvalidImport) {
		t.Fatalf("short header: err = %v, want ErrInvalidImport", err)
	}
}

func TestImportBalances_PersistsJobReport(t *testing.T) {
	svc, _, _, _, _, fac, _, _, _, _ := newService(t)
	ctx := context.Background()
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["331"] = leafAccount("331")
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}

	rows := [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "1000000", ""},
		{"9999", "", "", "1", ""},
	}
	res, err := svc.ImportBalances(ctx, rows, "user-1", true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if res.JobID == "" {
		t.Fatal("dry-run must persist a job")
	}
	job, err := svc.GetImportReport(ctx, res.JobID)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if job.ID != res.JobID || job.Status != setup.JobErrored || job.Total != 2 || len(job.Errors) != 1 {
		t.Fatalf("report mismatch: %+v", job)
	}
	if !job.DryRun || job.CreatedBy != "user-1" {
		t.Fatalf("report metadata: dry_run=%v created_by=%s", job.DryRun, job.CreatedBy)
	}
	// commit of the same file gets its own job id (dry-run/commit never collide)
	commit, err := svc.ImportBalances(ctx, [][]string{
		{"account", "object_type", "object_code", "debit", "credit"},
		{"1111", "", "", "1000000", ""},
	}, "user-1", false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if commit.JobID == res.JobID {
		t.Fatal("commit and dry-run job ids must differ")
	}
	cjob, err := svc.GetImportReport(ctx, commit.JobID)
	if err != nil {
		t.Fatalf("commit report: %v", err)
	}
	if cjob.DryRun || cjob.Created != 1 || cjob.Status != setup.JobOK {
		t.Fatalf("commit report mismatch: %+v", cjob)
	}
	if _, err := svc.GetImportReport(ctx, "nope"); !errors.Is(err, setup.ErrImportNotFound) {
		t.Fatalf("unknown job: err = %v, want ErrImportNotFound", err)
	}
}

func TestPreviewAccounts(t *testing.T) {
	svc, _, _, _, _, fac, _, _, _, _ := newService(t)
	ctx := context.Background()
	fac.accts["1111"] = leafAccount("1111")
	fac.accts["331"] = &ledger.Account{Code: "331", Name: "Phải trả người bán", Status: ledger.AccountActive, AllowPost: true, Level: 2, Type: ledger.AccountLiability}

	preview, err := svc.PreviewAccounts(ctx)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Total != 2 || len(preview.Accounts) != 2 {
		t.Fatalf("preview counts: %+v", preview)
	}
	// sorted by code
	if preview.Accounts[0].Code != "1111" || preview.Accounts[1].Code != "331" {
		t.Fatalf("preview order: %+v", preview.Accounts)
	}
	if preview.Accounts[1].Name != "Phải trả người bán" || preview.Accounts[1].Type != "liability" || !preview.Accounts[1].Postable {
		t.Fatalf("preview fields: %+v", preview.Accounts[1])
	}

	// seam failure surfaces
	fac.err = errors.New("boom")
	if _, err := svc.PreviewAccounts(ctx); err == nil {
		t.Fatal("preview must surface seam error")
	}
}

func TestAuditTrail(t *testing.T) {
	svc, _, _, _, _, _, _, fa, _, _ := newService(t)
	ctx := context.Background()
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	trail, err := svc.AuditTrail(ctx, "setup", 20)
	if err != nil {
		t.Fatalf("trail: %v", err)
	}
	if len(trail) == 0 {
		t.Fatal("init must leave a setup audit trail (R13)")
	}
	// newest-first, module-filtered, all entries setup.*
	for _, l := range trail {
		if l.Module != "setup" {
			t.Fatalf("trail leaked module %q", l.Module)
		}
		if l.Action == "" || l.UserCode == "" {
			t.Fatalf("trail entry incomplete: %+v", l)
		}
	}
	if len(fa.logs) > 0 && trail[0].Action != fa.logs[len(fa.logs)-1].Action {
		t.Fatalf("trail must be newest-first: got %q, want %q", trail[0].Action, fa.logs[len(fa.logs)-1].Action)
	}
}

func TestUpdateProfileGuarded(t *testing.T) {
	svc, _, _, _, _, _, _, _, repo, _ := newService(t)
	ctx := context.Background()
	_ = repo.SetStatus(ctx, setup.StatusBalancesLocked)
	p := validProfile()
	if _, err := svc.UpdateProfile(ctx, p, "u"); !errors.Is(err, setup.ErrWrongState) {
		t.Fatalf("update while locked must fail, got %v", err)
	}
}

func TestUpdateProfileHappyPath(t *testing.T) {
	svc, _, _, _, _, _, _, fa, repo, _ := newService(t)
	ctx := context.Background()
	if _, err := svc.UpdateProfile(ctx, nil, "u"); !errors.Is(err, setup.ErrInvalidProfile) {
		t.Fatalf("nil profile = %v, want ErrInvalidProfile", err)
	}

	// profile.update is allowed in any state before balances_locked — drive to
	// balances_draft via Initialize, then edit.
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}

	p := validProfile()
	p.Name = "Cty TNHH SX Thép ABC (sửa)"
	p.LegalRepresentative = "Nguyễn Văn B"
	got, err := svc.UpdateProfile(ctx, p, "ke-toan-truong")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "Cty TNHH SX Thép ABC (sửa)" || got.LegalRepresentative != "Nguyễn Văn B" {
		t.Fatalf("update not persisted: %+v", got)
	}
	if got.ID != setup.ProfileID {
		t.Fatalf("profile ID must stay %s, got %q", setup.ProfileID, got.ID)
	}
	// UpdateProfile preserves CreatedBy/CreatedAt from the existing row.
	before, err := svc.GetProfile(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CreatedAt != before.CreatedAt || got.UpdatedBy != "ke-toan-truong" {
		t.Fatalf("timestamps/author wrong: %+v vs %+v", got, before)
	}
	foundAudit := false
	for _, l := range fa.logs {
		if l.Action == "profile.update" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Fatal("profile.update audit missing")
	}
	// After re-initialize the profile must be retrievable.
	stored, err := repo.GetProfile(ctx, setup.ProfileID)
	if err != nil {
		t.Fatalf("repo get: %v", err)
	}
	if stored.Name != got.Name {
		t.Fatalf("repo copy out of sync: %+v", stored)
	}
}

func TestDeleteBalanceGuards(t *testing.T) {
	for _, st := range []setup.SetupStatus{setup.StatusEmpty, setup.StatusProfiled, setup.StatusBalancesLocked, setup.StatusActive} {
		repo := newRepo(t)
		ctx := context.Background()
		_ = repo.SetStatus(ctx, st)
		svc := appsetup.NewService(repo, appsetup.Dependencies{
			Regime: &fakeRegime{}, Seeder: &fakeSeeder{}, Objects: &fakeObjects{},
			Periods: &fakePeriods{}, Accounts: &fakeAccounts{}, Postings: &fakePostings{}, Audit: &fakeAudit{},
		})
		err := svc.DeleteBalance(ctx, setup.BalanceID("1111", ""), "user-1")
		switch st {
		case setup.StatusBalancesLocked, setup.StatusActive:
			if !errors.Is(err, setup.ErrBalanceLocked) {
				t.Fatalf("status %s: delete = %v, want ErrBalanceLocked", st, err)
			}
		default:
			if !errors.Is(err, setup.ErrWrongState) {
				t.Fatalf("status %s: delete = %v, want ErrWrongState", st, err)
			}
		}
	}
}

func TestDeleteBalanceHappyPath(t *testing.T) {
	svc, _, _, fo, _, fac, _, fa, repo, _ := newService(t)
	ctx := context.Background()
	fo.recs["customer:KH-0001"] = &masterdata.Record{Kind: masterdata.KindCustomer, Code: "KH-0001", Name: "Cty B", State: masterdata.StateActive}
	fac.accts["131"] = &ledger.Account{Code: "131", Status: ledger.AccountActive, AllowPost: true, Level: 2}

	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init: %v", err)
	}
	saved, err := svc.SaveBalance(ctx, &setup.OpeningBalance{
		AccountCode: "131", ObjectType: "customer", ObjectCode: "KH-0001",
		Debit: core.Money{AmountMinor: 500_000_000, Currency: "VND"},
	}, "user-1")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := svc.DeleteBalance(ctx, saved.ID, "user-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ := svc.ListBalances(ctx, "")
	if len(list.Balances) != 0 {
		t.Fatalf("want empty after delete, got %d", len(list.Balances))
	}
	if err := svc.DeleteBalance(ctx, saved.ID, "user-1"); !errors.Is(err, setup.ErrBalanceNotFound) {
		t.Fatalf("second delete = %v, want ErrBalanceNotFound", err)
	}
	foundAudit := false
	for _, l := range fa.logs {
		if l.Action == "balance.delete" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Fatal("balance.delete audit missing")
	}
	_ = repo
}

func TestAuditNilSeamDoesNotPanic(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	svc := appsetup.NewService(repo, appsetup.Dependencies{
		Regime: &fakeRegime{}, Seeder: &fakeSeeder{}, Objects: &fakeObjects{},
		Periods: &fakePeriods{}, Accounts: &fakeAccounts{accts: map[string]*ledger.Account{"1111": leafAccount("1111")}},
		Postings: &fakePostings{}, Audit: nil,
	})
	if _, err := svc.Initialize(ctx, &appsetup.InitializeRequest{
		Profile: validProfile(), Regime: "TT99-2025", FiscalYearStart: "2026-01-01",
		SeedAccounts: true, OpenPeriods: true,
	}, "user-1"); err != nil {
		t.Fatalf("init without audit seam: %v", err)
	}
	if _, err := svc.SaveBalance(ctx, &setup.OpeningBalance{
		AccountCode: "1111", Debit: core.Money{AmountMinor: 100, Currency: "VND"},
	}, "user-1"); err != nil {
		t.Fatalf("save without audit seam: %v", err)
	}
}

// --- StatusView steps ----------------------------------------------------------

func TestStatusViewSteps(t *testing.T) {
	svc, _, _, _, _, _, _, _, repo, _ := newService(t)
	ctx := context.Background()
	view, _ := svc.Status(ctx)
	if len(view.Steps) != 7 {
		t.Fatalf("want 7 steps, got %d", len(view.Steps))
	}
	for _, s := range view.Steps {
		if s.Done {
			t.Fatalf("step %s done at empty status", s.Key)
		}
	}
	_ = repo.SetStatus(ctx, setup.StatusBalancesLocked)
	view, _ = svc.Status(ctx)
	dones := map[string]bool{}
	for _, s := range view.Steps {
		dones[s.Key] = s.Done
	}
	if !dones["profile"] || !dones["regime"] || !dones["accounts"] || !dones["periods"] || !dones["balances"] || !dones["lock"] || dones["active"] {
		t.Fatalf("step done map wrong at locked: %+v", dones)
	}
}
