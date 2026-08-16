package ledger_test

import (
	"context"
	"database/sql"
	"regexp"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
	"goGL/internal/infrastructure/db"
	persledger "goGL/internal/infrastructure/persistence/ledger"
)

// openP2DB opens a shared-cache in-memory SQLite DB with a busy timeout, so
// concurrent write tests exercise SQLite's write-lock serialization instead of
// erroring with SQLITE_BUSY. It deliberately does NOT cap the connection pool.
func openP2DB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	d, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

func newP2Svc(t *testing.T, d *sql.DB) appledger.Service {
	t.Helper()
	return appledger.NewService(persledger.NewSqliteRepository(d))
}

func createDraft(t *testing.T, svc appledger.Service, date, desc string) *domainledger.JournalEntry {
	t.Helper()
	e, err := svc.CreateEntry(context.Background(), "ketoan", &domainledger.JournalEntry{
		VoucherDate: date,
		Description: desc,
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
		},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	return e
}

var voucherNoRe = regexp.MustCompile(`^PK-\d{5}/\d{2}$`)

func TestFormatVoucherNo(t *testing.T) {
	cases := []struct {
		form, period string
		n            int64
		want         string
	}{
		{"PK", "2026-08", 1, "PK-00001/26"},
		{"KC", "2025-12", 123, "KC-00123/25"},
		{"PK", "", 7, "PK-00007/"},
	}
	for _, c := range cases {
		if got := domainledger.FormatVoucherNo(c.form, c.n, c.period); got != c.want {
			t.Errorf("FormatVoucherNo(%q, %d, %q) = %q, want %q", c.form, c.n, c.period, got, c.want)
		}
	}
}

func TestService_PostEntry_AssignsVoucherNoAndPoster(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	draft := createDraft(t, svc, "2026-08-05", "Thu tiền bán hàng")
	if draft.VoucherNo != "" {
		t.Fatalf("draft VoucherNo = %q, want empty (assigned only at post, R10)", draft.VoucherNo)
	}

	posted, err := svc.PostEntry(ctx, "ketoan", draft.ID)
	if err != nil {
		t.Fatalf("post entry: %v", err)
	}
	if posted.Status != domainledger.EntryPosted {
		t.Fatalf("status = %q, want posted", posted.Status)
	}
	if posted.PostedBy != "ketoan" || posted.PostedAt == "" {
		t.Fatalf("poster fields missing: %+v", posted)
	}
	if !voucherNoRe.MatchString(posted.VoucherNo) {
		t.Fatalf("VoucherNo = %q, want PK-<5-digit>/YY", posted.VoucherNo)
	}
	if posted.VoucherNo != "PK-00001/26" {
		t.Fatalf("VoucherNo = %q, want PK-00001/26 (first of period)", posted.VoucherNo)
	}

	// The next post in the same period advances the sequence.
	second := createDraft(t, svc, "2026-08-06", "Thu tiền lần hai")
	p2, err := svc.PostEntry(ctx, "ketoan", second.ID)
	if err != nil {
		t.Fatalf("post second: %v", err)
	}
	if p2.VoucherNo != "PK-00002/26" {
		t.Fatalf("VoucherNo = %q, want PK-00002/26 (per-period sequence)", p2.VoucherNo)
	}

	// A different period restarts the sequence.
	sep := createDraft(t, svc, "2026-09-01", "Bút toán tháng 9")
	p3, err := svc.PostEntry(ctx, "ketoan", sep.ID)
	if err != nil {
		t.Fatalf("post september: %v", err)
	}
	if p3.VoucherNo != "PK-00001/26" {
		t.Fatalf("VoucherNo = %q, want PK-00001/26 (sequence resets per period)", p3.VoucherNo)
	}
}

func TestService_PostEntry_IgnoresClientVoucherNo(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	// A manual entry is numbered by the ledger sequence (R10). A client-supplied
	// VoucherNo must not bypass it, or a later post could collide with it.
	e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
		VoucherDate: "2026-08-05",
		VoucherNo:   "PK-99999/26",
		Description: "Bút toán tự đánh số",
		Lines: []domainledger.JournalLine{
			{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
			{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
		},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if e.VoucherNo != "" {
		t.Fatalf("draft VoucherNo = %q, want empty (drafts stay unnumbered)", e.VoucherNo)
	}

	posted, err := svc.PostEntry(ctx, "ketoan", e.ID)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if posted.VoucherNo != "PK-00001/26" {
		t.Fatalf("VoucherNo = %q, want PK-00001/26 (client value must not bypass R10)", posted.VoucherNo)
	}

	// The sequence was consumed normally: the next post has no collision.
	next := createDraft(t, svc, "2026-08-06", "Bút toán sau")
	p2, err := svc.PostEntry(ctx, "ketoan", next.ID)
	if err != nil {
		t.Fatalf("post next: %v", err)
	}
	if p2.VoucherNo != "PK-00002/26" {
		t.Fatalf("VoucherNo = %q, want PK-00002/26", p2.VoucherNo)
	}
}

func TestService_PostEntry_RepostIsIdempotent(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	draft := createDraft(t, svc, "2026-08-05", "Bút toán")
	posted, err := svc.PostEntry(ctx, "ketoan", draft.ID)
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	again, err := svc.PostEntry(ctx, "ketoan", draft.ID)
	if err != nil {
		t.Fatalf("repost: %v", err)
	}
	if again.ID != posted.ID || again.VoucherNo != posted.VoucherNo {
		t.Fatalf("repost returned %+v, want the original posted entry", again)
	}

	// The idempotent repost must not have consumed a sequence number.
	fresh := createDraft(t, svc, "2026-08-07", "Bút toán sau repost")
	p, err := svc.PostEntry(ctx, "ketoan", fresh.ID)
	if err != nil {
		t.Fatalf("post after repost: %v", err)
	}
	if p.VoucherNo != "PK-00002/26" {
		t.Fatalf("VoucherNo = %q, want PK-00002/26 (repost burned no number)", p.VoucherNo)
	}
}

func TestService_PostEntry_RejectsClosedPeriodAtPostTime(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	if _, err := svc.OpenPeriod(ctx, "kttruong", "2026-08"); err != nil {
		t.Fatalf("open period: %v", err)
	}
	draft := createDraft(t, svc, "2026-08-05", "Draft trước khi khoá sổ")
	if _, err := svc.ClosePeriod(ctx, "kttruong", "2026-08", "khoá sổ cuối tháng"); err != nil {
		t.Fatalf("close period: %v", err)
	}

	if _, err := svc.PostEntry(ctx, "ketoan", draft.ID); err != domainledger.ErrPeriodClosed {
		t.Fatalf("post into closed period: got %v, want ErrPeriodClosed (R4 re-checked at post)", err)
	}
}

func TestService_PostEntry_RevalidatesAccountsAtPostTime(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	draft := createDraft(t, svc, "2026-08-05", "Bút toán")

	acc, err := svc.GetAccount(ctx, domainledger.RowID("account", "5111"))
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	acc.Status = domainledger.AccountInactive
	if err := svc.UpdateAccount(ctx, "ketoan", acc); err != nil {
		t.Fatalf("deactivate account: %v", err)
	}

	if _, err := svc.PostEntry(ctx, "ketoan", draft.ID); err != domainledger.ErrAccountInactive {
		t.Fatalf("post with deactivated line account: got %v, want ErrAccountInactive (R3 re-checked)", err)
	}
}

func TestService_DeleteEntry_OnlyDraft(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	// Draft can be deleted (R7).
	draft := createDraft(t, svc, "2026-08-05", "Bút toán xoá")
	if err := svc.DeleteEntry(ctx, "ketoan", draft.ID); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
	if _, err := svc.GetEntry(ctx, draft.ID); err != sql.ErrNoRows {
		t.Fatalf("after delete: got %v, want sql.ErrNoRows", err)
	}

	// Posted entry is append-only (R6): delete is rejected.
	posted := createDraft(t, svc, "2026-08-06", "Bút toán đã ghi sổ")
	if _, err := svc.PostEntry(ctx, "ketoan", posted.ID); err != nil {
		t.Fatalf("post: %v", err)
	}
	if err := svc.DeleteEntry(ctx, "ketoan", posted.ID); err != domainledger.ErrWrongState {
		t.Fatalf("delete posted: got %v, want ErrWrongState (R7)", err)
	}

	// Missing entry is a plain not-found.
	if err := svc.DeleteEntry(ctx, "ketoan", "nope"); err != sql.ErrNoRows {
		t.Fatalf("delete missing: got %v, want sql.ErrNoRows", err)
	}
}

// TestRepository_DeleteEntry_RefusesPosted exercises the CAS guard directly at
// the persistence seam: a POSTED entry is append-only (R6/R7), so even a direct
// repo delete must be refused and the row retained. This is the guard that a
// concurrent post-vs-delete race depends on.
func TestRepository_DeleteEntry_RefusesPosted(t *testing.T) {
	d := openP2DB(t)
	repo := persledger.NewSqliteRepository(d)
	svc := newP2Svc(t, d)
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	draft := createDraft(t, svc, "2026-08-05", "Bút toán đã ghi sổ")
	if _, err := svc.PostEntry(ctx, "ketoan", draft.ID); err != nil {
		t.Fatalf("post: %v", err)
	}

	if err := repo.DeleteEntry(ctx, draft.ID); err != domainledger.ErrWrongState {
		t.Fatalf("delete posted via repo: got %v, want ErrWrongState (R6/R7 CAS guard)", err)
	}
	if _, err := svc.GetEntry(ctx, draft.ID); err != nil {
		t.Fatalf("posted entry must survive a refused delete: %v", err)
	}

	// A DRAFT row is still deletable through the repo.
	draft2 := createDraft(t, svc, "2026-08-06", "Bút toán nháp")
	if err := repo.DeleteEntry(ctx, draft2.ID); err != nil {
		t.Fatalf("delete draft via repo: %v", err)
	}
}

func TestService_PostEntry_ReturnsExistingForSameSourceRef(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	mk := func() *domainledger.JournalEntry {
		e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
			VoucherDate: "2026-08-05",
			SourceRef:   "CASH-V-100",
			Description: "Từ phiếu thu V-100",
			Lines: []domainledger.JournalLine{
				{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
				{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
			},
		})
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		return e
	}

	first := mk()
	second := mk()

	posted, err := svc.PostEntry(ctx, "ketoan", first.ID)
	if err != nil {
		t.Fatalf("post first: %v", err)
	}

	// Posting the second draft with the same (Source, SourceRef) returns the
	// existing POSTED entry — no duplicate is created (R5).
	ret, err := svc.PostEntry(ctx, "ketoan", second.ID)
	if err != nil {
		t.Fatalf("post duplicate source ref: %v", err)
	}
	if ret.ID != posted.ID || ret.VoucherNo != posted.VoucherNo {
		t.Fatalf("R5: got %+v, want the existing entry %+v", ret, posted)
	}

	got, err := svc.ListEntries(ctx, domainledger.EntryFilter{Status: domainledger.EntryPosted})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].SourceRef != "CASH-V-100" {
		t.Fatalf("expected exactly one posted entry for the source ref, got %d", len(got))
	}
}

func TestService_PostEntry_DuplicateSourceRefWhileDraft(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	mk := func(desc string) *domainledger.JournalEntry {
		e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
			VoucherDate: "2026-08-05",
			SourceRef:   "V-1",
			Description: desc,
			Lines: []domainledger.JournalLine{
				{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
				{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
			},
		})
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		return e
	}

	mk("Bút toán gốc")
	dup := mk("Bút toán trùng")

	// While the original is still DRAFT, a second entry on the same key cannot
	// be posted (the key must resolve to exactly one entry).
	if _, err := svc.PostEntry(ctx, "ketoan", dup.ID); err != domainledger.ErrDuplicateSource {
		t.Fatalf("got %v, want ErrDuplicateSource (R5 while original is draft)", err)
	}
}

// TestRepository_PostEntry_DuplicateSourceRefInTx exercises the serialized R5
// guard at the persistence seam, bypassing the service's fast-path lookup: a
// direct repo post of a second entry sharing (Source, SourceRef) must return
// the already-POSTED entry instead of creating a duplicate, and must not leave
// a second posted row behind (R5, "never double-post").
func TestRepository_PostEntry_DuplicateSourceRefInTx(t *testing.T) {
	d := openP2DB(t)
	repo := persledger.NewSqliteRepository(d)
	svc := newP2Svc(t, d)
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	mk := func(desc string) *domainledger.JournalEntry {
		e, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
			VoucherDate: "2026-08-05",
			SourceRef:   "SR-1",
			Description: desc,
			Lines: []domainledger.JournalLine{
				{LineNo: 1, AccountCode: "1111", Debit: 5_000_000},
				{LineNo: 2, AccountCode: "5111", Credit: 5_000_000},
			},
		})
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		return e
	}

	first := mk("Bút toán gốc")

	postedFirst, err := svc.PostEntry(ctx, "ketoan", first.ID)
	if err != nil {
		t.Fatalf("post first: %v", err)
	}

	second := mk("Bút toán trùng")

	got, err := repo.PostEntry(ctx, second, "PK")
	if err != nil {
		t.Fatalf("repo post duplicate key: %v", err)
	}
	if got.ID != postedFirst.ID || got.VoucherNo != postedFirst.VoucherNo {
		t.Fatalf("repo PostEntry returned %s/%s, want existing %s/%s (R5 inside tx)",
			got.ID, got.VoucherNo, postedFirst.ID, postedFirst.VoucherNo)
	}

	posted, err := svc.ListEntries(ctx, domainledger.EntryFilter{Status: domainledger.EntryPosted})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(posted) != 1 || posted[0].ID != first.ID {
		t.Fatalf("want exactly one posted entry, got %d", len(posted))
	}
}

func TestService_PostEntry_ConcurrentPosts_UniqueVoucherNos(t *testing.T) {
	svc := newP2Svc(t, openP2DB(t))
	ctx := context.Background()
	seedPostableAccounts(t, svc)

	const n = 20
	start := make(chan struct{})
	errs := make(chan error, n)
	nos := make(chan string, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			draft, err := svc.CreateEntry(ctx, "ketoan", &domainledger.JournalEntry{
				VoucherDate: "2026-08-05",
				Description: "Bút toán đồng thời",
				Lines: []domainledger.JournalLine{
					{LineNo: 1, AccountCode: "1111", Debit: 1_000_000},
					{LineNo: 2, AccountCode: "5111", Credit: 1_000_000},
				},
			})
			if err != nil {
				errs <- err
				return
			}
			posted, err := svc.PostEntry(ctx, "ketoan", draft.ID)
			if err != nil {
				errs <- err
				return
			}
			nos <- posted.VoucherNo
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	close(nos)

	for err := range errs {
		t.Fatalf("concurrent post: %v", err)
	}

	seen := make(map[string]bool, n)
	for no := range nos {
		if seen[no] {
			t.Fatalf("duplicate VoucherNo %q (R10 violated)", no)
		}
		seen[no] = true
	}
	if len(seen) != n {
		t.Fatalf("got %d unique VoucherNos, want %d", len(seen), n)
	}
}
