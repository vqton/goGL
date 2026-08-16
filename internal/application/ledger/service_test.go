package ledger_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
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

func mustCreateAccount(t *testing.T, svc appledger.Service, a *domainledger.Account) {
	t.Helper()
	if err := svc.CreateAccount(context.Background(), "ketoan", a); err != nil {
		t.Fatalf("create account %s: %v", a.Code, err)
	}
}

// seedPostableAccounts creates the leaf accounts the entry tests reference.
func seedPostableAccounts(t *testing.T, svc appledger.Service) {
	t.Helper()
	mustCreateAccount(t, svc, &domainledger.Account{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "5111", Name: "Doanh thu bán hàng", Type: domainledger.AccountRevenue, Level: 3, AllowPost: true})
}

func TestService_CreateEntry_AssignsIDStatusAndPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Description: "Bút toán kiểm tra",
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	}

	created, err := svc.CreateEntry(ctx, "ketoan", e)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected service to assign entry ID")
	}
	if created.Status != domainledger.EntryDraft {
		t.Fatalf("status = %q, want draft", created.Status)
	}
	if created.CreatedBy != "ketoan" {
		t.Fatalf("created_by = %q, want ketoan", created.CreatedBy)
	}
	if created.Period != "2026-08" {
		t.Fatalf("period = %q, want 2026-08 (derived from voucher date)", created.Period)
	}

	got, err := svc.GetEntry(ctx, created.ID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got.ID != created.ID || len(got.Lines) != 2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestService_CreateEntry_RejectsUnbalancedEntry(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 4_000_000},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err != domainledger.ErrUnbalanced {
		t.Fatalf("got %v, want ErrUnbalanced", err)
	}
}

func TestService_CreateEntry_RejectsBothSidesOnLine(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000, Credit: 5_000_000},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err != domainledger.ErrInvalidLine {
		t.Fatalf("got %v, want ErrInvalidLine", err)
	}
}

func TestService_CreateEntry_RejectsZeroSideLine(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111"},
			{LineNo: 2, AccountCode: "5111", Credit: 0},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err != domainledger.ErrInvalidLine {
		t.Fatalf("got %v, want ErrInvalidLine", err)
	}
}

func TestService_CreateEntry_RejectsNegativeAmount(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: -1_000},
			{LineNo: 2, AccountCode: "5111", Credit: -1_000},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err != domainledger.ErrInvalidLine {
		t.Fatalf("got %v, want ErrInvalidLine", err)
	}
}

func TestService_CreateEntry_RejectsMissingDate(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	e := &domainledger.JournalEntry{
		Source: domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err == nil {
		t.Fatal("expected missing voucher date to be rejected")
	}
}

func TestService_GetEntryNotFound(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	_, err := svc.GetEntry(context.Background(), "nope")
	if err != sql.ErrNoRows {
		t.Fatalf("got %v, want sql.ErrNoRows", err)
	}
}

func TestService_ListEntriesByPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	for _, d := range []string{"2026-08-05", "2026-09-01"} {
		_, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
			VoucherDate: d,
			Source:      domainledger.SourceManual,
			Lines: []domainledger.JournalLine{
				{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
				{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
			},
		})
		if err != nil {
			t.Fatalf("create entry %s: %v", d, err)
		}
	}

	got, err := svc.ListEntries(ctx, domainledger.EntryFilter{Period: "2026-08"})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry in 2026-08, got %d", len(got))
	}
}

// --- P1.1 / P1.2 — chart of accounts + hierarchy (R3) ---

func TestService_CreateAccount_AssignsIDAndDefaultStatus(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()

	a := &domainledger.Account{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true}
	if err := svc.CreateAccount(ctx, "ketoan", a); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if a.ID == "" {
		t.Fatal("expected service to assign account ID")
	}
	if a.Status != domainledger.AccountActive {
		t.Fatalf("status = %q, want active", a.Status)
	}

	got, err := svc.GetAccount(ctx, a.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if got.Code != "1111" || got.Type != domainledger.AccountAsset {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestService_CreateAccount_RejectsMissingFields(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	cases := []struct {
		name string
		a    *domainledger.Account
		want error
	}{
		{"missing code", &domainledger.Account{Name: "X", Type: domainledger.AccountAsset, Level: 1}, domainledger.ErrInvalidAccount},
		{"missing name", &domainledger.Account{Code: "1111", Type: domainledger.AccountAsset, Level: 1}, domainledger.ErrInvalidAccount},
		{"bad type", &domainledger.Account{Code: "1111", Name: "X", Type: "bogus", Level: 1}, domainledger.ErrInvalidType},
		{"bad level", &domainledger.Account{Code: "1111", Name: "X", Type: domainledger.AccountAsset, Level: 7}, domainledger.ErrInvalidLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.CreateAccount(context.Background(), "ketoan", tc.a); err != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestService_CreateAccount_InheritsLevelAndTypeFromParent(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()

	mustCreateAccount(t, svc, &domainledger.Account{Code: "11", Name: "Tiền và tương đương tiền", Type: domainledger.AccountAsset, Level: 1})

	child := &domainledger.Account{Code: "111", Name: "Tiền mặt", Type: domainledger.AccountAsset, ParentCode: "11", AllowPost: true}
	if err := svc.CreateAccount(ctx, "ketoan", child); err != nil {
		t.Fatalf("create child: %v", err)
	}
	if child.Level != 2 {
		t.Fatalf("child level = %d, want 2 (parent level + 1)", child.Level)
	}
}

func TestService_CreateAccount_ParentNotFound(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	a := &domainledger.Account{Code: "1111", Name: "Tiền mặt", Type: domainledger.AccountAsset, ParentCode: "99", Level: 2}
	if err := svc.CreateAccount(context.Background(), "ketoan", a); err != domainledger.ErrParentNotFound {
		t.Fatalf("got %v, want ErrParentNotFound", err)
	}
}

func TestService_CreateAccount_TypeMustMatchParent(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	mustCreateAccount(t, svc, &domainledger.Account{Code: "11", Name: "Tiền", Type: domainledger.AccountAsset, Level: 1})

	a := &domainledger.Account{Code: "511", Name: "Doanh thu", Type: domainledger.AccountRevenue, ParentCode: "11"}
	if err := svc.CreateAccount(context.Background(), "ketoan", a); err != domainledger.ErrTypeMismatch {
		t.Fatalf("got %v, want ErrTypeMismatch", err)
	}
}

func TestService_CreateAccount_ParentWithChildrenIsNotPostable(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()

	parent := &domainledger.Account{Code: "11", Name: "Tiền", Type: domainledger.AccountAsset, Level: 1, AllowPost: true}
	mustCreateAccount(t, svc, parent)
	if parent.AllowPost {
		t.Fatal("root with no children may be postable")
	}

	// Adding a child must flip the parent off AllowPost.
	mustCreateAccount(t, svc, &domainledger.Account{Code: "111", Name: "Tiền mặt", Type: domainledger.AccountAsset, ParentCode: "11", AllowPost: true})
	got, err := svc.GetAccount(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	if got.AllowPost {
		t.Fatal("parent account with children must not allow posting (R3)")
	}
}

func TestService_UpdateAccount_Inactivates(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	mustCreateAccount(t, svc, &domainledger.Account{Code: "1111", Name: "Tiền mặt VND", Type: domainledger.AccountAsset, Level: 2, AllowPost: true})

	a := &domainledger.Account{ID: "does-not-matter", Name: "Tiền mặt VND (đóng)", Status: domainledger.AccountInactive, AllowPost: true}
	// The service resolves by ID from the store, so fetch the real one.
	existing, err := svc.GetAccount(ctx, domainledger.RowID("account", "1111"))
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	a.ID = existing.ID
	if err := svc.UpdateAccount(ctx, "ketoan", a); err != nil {
		t.Fatalf("update account: %v", err)
	}
	got, err := svc.GetAccount(ctx, existing.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if got.Status != domainledger.AccountInactive {
		t.Fatalf("status = %q, want inactive", got.Status)
	}
	if got.Code != "1111" || got.Type != domainledger.AccountAsset {
		t.Fatalf("structural fields must be immutable: %+v", got)
	}
}

func TestService_ListAccounts_Filters(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)
	mustCreateAccount(t, svc, &domainledger.Account{Code: "3311", Name: "Phải trả người bán", Type: domainledger.AccountLiability, Level: 2, AllowPost: true})

	liability, err := svc.ListAccounts(ctx, domainledger.AccountFilter{Type: domainledger.AccountLiability})
	if err != nil {
		t.Fatalf("list by type: %v", err)
	}
	if len(liability) != 1 || liability[0].Code != "3311" {
		t.Fatalf("type filter: got %+v", liability)
	}

	byQ, err := svc.ListAccounts(ctx, domainledger.AccountFilter{Q: "mặt"})
	if err != nil {
		t.Fatalf("list by q: %v", err)
	}
	if len(byQ) != 1 || byQ[0].Code != "1111" {
		t.Fatalf("q filter: got %+v", byQ)
	}

	all, err := svc.ListAccounts(ctx, domainledger.AccountFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(all))
	}
}

func TestService_CreateEntry_RejectsUnknownAccount(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	seedPostableAccounts(t, svc)

	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "9999", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	}
	_, err := svc.CreateEntry(context.Background(), "ketoan", e)
	if err != domainledger.ErrAccountNotFound {
		t.Fatalf("got %v, want ErrAccountNotFound", err)
	}
}

func TestService_CreateEntry_RejectsNonPostableAccount(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()

	// 11 is a parent (summary) account — not postable.
	mustCreateAccount(t, svc, &domainledger.Account{Code: "11", Name: "Tiền", Type: domainledger.AccountAsset, Level: 1})
	mustCreateAccount(t, svc, &domainledger.Account{Code: "5111", Name: "Doanh thu", Type: domainledger.AccountRevenue, Level: 3, AllowPost: true})

	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "11", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	}
	_, err := svc.CreateEntry(ctx, "ketoan", e)
	if err != domainledger.ErrAccountInactive {
		t.Fatalf("got %v, want ErrAccountInactive (parent account is not postable)", err)
	}
}

// --- P1.3 — accounting periods (R4) ---

func TestService_OpenPeriod_CreatesAndParses(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()

	p, err := svc.OpenPeriod(ctx, "kttruong", "2026-08")
	if err != nil {
		t.Fatalf("open period: %v", err)
	}
	if p.Year != 2026 || p.Month != 8 || p.Status != domainledger.PeriodOpen || p.OpenedBy != "kttruong" {
		t.Fatalf("unexpected period: %+v", p)
	}
}

func TestService_OpenPeriod_Idempotent(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("first open: %v", err)
	}
	p, err := svc.OpenPeriod(ctx, "kttruong", "2026-08")
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	if p.Status != domainledger.PeriodOpen {
		t.Fatalf("status = %q, want open", p.Status)
	}
}

func TestService_OpenPeriod_InvalidID(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	for _, id := range []string{"2026-8", "2026-13", "abc", "2026/08"} {
		if _, err := svc.OpenPeriod(context.Background(), "kttruong", id); err != domainledger.ErrInvalidPeriod {
			t.Fatalf("open %q: got %v, want ErrInvalidPeriod", id, err)
		}
	}
}

func TestService_OpenPeriod_ClosedPeriodReturnsWrongState(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "khoá sổ cuối tháng"); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != domainledger.ErrWrongState {
		t.Fatalf("got %v, want ErrWrongState (reopen via ReopenPeriod)", err)
	}
}

func TestService_ClosePeriod_RequiresReason(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "  "); err != domainledger.ErrCloseReasonRequired {
		t.Fatalf("got %v, want ErrCloseReasonRequired", err)
	}
	if _, err := svc.ReopenPeriod(ctx, "kttruong", "2026-08", ""); err != domainledger.ErrCloseReasonRequired {
		t.Fatalf("reopen without reason: got %v, want ErrCloseReasonRequired", err)
	}
}

func TestService_CloseAndReopenPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("open: %v", err)
	}

	closed, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "khoá sổ cuối tháng")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closed.Status != domainledger.PeriodClosed || closed.ClosedBy != "kttruong" ||
		closed.ClosedAt == "" || closed.CloseReason != "khoá sổ cuối tháng" {
		t.Fatalf("unexpected closed period: %+v", closed)
	}

	// Closing again is idempotent.
	again, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "lần hai")
	if err != nil {
		t.Fatalf("close again: %v", err)
	}
	if again.ClosedAt != closed.ClosedAt {
		t.Fatalf("idempotent close must not rewrite the record: %+v", again)
	}

	reopened, err := svc.ReopenPeriod(ctx, "kttruong", "2026-08", "phát hiện thiếu bút toán")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Status != domainledger.PeriodOpen || reopened.ClosedAt != "" || reopened.CloseReason != "" {
		t.Fatalf("unexpected reopened period: %+v", reopened)
	}
}

func TestService_CreateEntry_RejectsClosedPeriod(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "khoá sổ"); err != nil {
		t.Fatalf("close: %v", err)
	}

	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-20",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
		},
	}
	_, err := svc.CreateEntry(ctx, "ketoan", e)
	if err != domainledger.ErrPeriodClosed {
		t.Fatalf("got %v, want ErrPeriodClosed (R4)", err)
	}
}

func TestService_CreateEntry_ImplicitOpenPeriodAllowed(t *testing.T) {
	svc := newSvc(t, openTestDB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	e := &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		Source:      domainledger.SourceManual,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
		},
	}
	if _, err := svc.CreateEntry(ctx, "ketoan", e); err != nil {
		t.Fatalf("create entry in never-opened period: %v", err)
	}
}
