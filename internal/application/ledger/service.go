package ledger

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	domainaudit "goGL/internal/domain/audit"
	"goGL/internal/domain/ledger"
)

type Service interface {
	CreateEntry(ctx context.Context, actor string, e *ledger.JournalEntry) (*ledger.JournalEntry, error)
	GetEntry(ctx context.Context, id string) (*ledger.JournalEntry, error)
	ListEntries(ctx context.Context, f ledger.EntryFilter) ([]*ledger.JournalEntry, error)
	PostEntry(ctx context.Context, actor, id string) (*ledger.JournalEntry, error)
	DeleteEntry(ctx context.Context, actor, id string) error

	CreateAccount(ctx context.Context, actor string, a *ledger.Account) error
	UpdateAccount(ctx context.Context, actor string, a *ledger.Account) error
	GetAccount(ctx context.Context, id string) (*ledger.Account, error)
	ListAccounts(ctx context.Context, f ledger.AccountFilter) ([]*ledger.Account, error)

	ListPeriods(ctx context.Context) ([]*ledger.AccountingPeriod, error)
	OpenPeriod(ctx context.Context, actor, id string) (*ledger.AccountingPeriod, error)
	ClosePeriod(ctx context.Context, actor, id, reason string) (*ledger.AccountingPeriod, error)
	ReopenPeriod(ctx context.Context, actor, id, reason string) (*ledger.AccountingPeriod, error)

	GetGeneralJournal(ctx context.Context, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.GeneralJournal, error)
	GetLedgerBook(ctx context.Context, accountCode, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.LedgerBook, error)
	GetDetailBook(ctx context.Context, accountCode, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.LedgerBook, error)
	GetTrialBalance(ctx context.Context, period string, p *ledger.Page) (*ledger.TrialBalance, error)
}

type AuditRecorder interface {
	Record(ctx context.Context, l *domainaudit.AuditLog) error
}

type noopAuditor struct{}

func (noopAuditor) Record(context.Context, *domainaudit.AuditLog) error { return nil }

type service struct {
	repo  ledger.Repository
	audit AuditRecorder
	now   func() time.Time
}

type Option func(*service)

// WithAuditor wires an audit recorder (period close/reopen and account
// mutations are audited). A no-op keeps callers without one working.
func WithAuditor(a AuditRecorder) Option {
	return func(s *service) { s.audit = a }
}

func NewService(repo ledger.Repository, opts ...Option) Service {
	s := &service{repo: repo, audit: noopAuditor{}, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *service) auditAction(ctx context.Context, actor, action, targetID string) error {
	return s.audit.Record(ctx, &domainaudit.AuditLog{
		UserCode:  actor,
		Module:    "ledger",
		Action:    action,
		TargetID:  targetID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

// CreateEntry persists a manual journal entry as a DRAFT after enforcing R1
// (Σ debits = Σ credits), R2 (exactly one side per line) and R3 (every line
// account exists, is ACTIVE and is a postable leaf). R4 (open period) is
// checked in P1.3; R10 (VoucherNo) is assigned at post time (P2). This is the
// manual-entry API, so the source is always MANUAL — operating modules post
// through PostFromSource instead.
func (s *service) CreateEntry(ctx context.Context, actor string, e *ledger.JournalEntry) (*ledger.JournalEntry, error) {
	if e.VoucherDate == "" {
		return nil, ledger.ErrInvalidDate
	}
	if err := validateLines(e.Lines); err != nil {
		return nil, err
	}
	if err := s.validatePostableAccounts(ctx, e.Lines); err != nil {
		return nil, err
	}
	if e.Period == "" {
		e.Period = ledger.PeriodFromDate(e.VoucherDate)
	}
	if err := s.ensurePeriodOpen(ctx, e.Period); err != nil {
		return nil, err
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	e.ID = id
	e.Source = ledger.SourceManual
	e.Status = ledger.EntryDraft
	e.VoucherNo = "" // drafts stay unnumbered; R10 assigns at post time
	e.CreatedBy = actor

	if err := s.repo.CreateEntry(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// validatePostableAccounts enforces R3: every line account must exist, be
// ACTIVE and allow posting (leaves only).
func (s *service) validatePostableAccounts(ctx context.Context, lines []ledger.JournalLine) error {
	for _, l := range lines {
		acc, err := s.repo.GetAccountByCode(ctx, l.AccountCode)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ledger.ErrAccountNotFound
			}
			return err
		}
		if acc.Status != ledger.AccountActive || !acc.AllowPost {
			return ledger.ErrAccountInactive
		}
	}
	return nil
}

func (s *service) GetEntry(ctx context.Context, id string) (*ledger.JournalEntry, error) {
	return s.repo.GetEntry(ctx, id)
}

func (s *service) ListEntries(ctx context.Context, f ledger.EntryFilter) ([]*ledger.JournalEntry, error) {
	return s.repo.ListEntries(ctx, f)
}

// --- P3.1 — statutory books read-model (Sổ Nhật ký chung, Sổ Cái, Sổ chi
// tiết, BCĐPS). Books aggregate over POSTED entries only, in exact int64
// arithmetic; Số dư đầu kỳ carries forward from all activity in periods
// strictly before the range start. ---

// postedEntries returns every POSTED entry — the books' only source of truth.
func (s *service) postedEntries(ctx context.Context) ([]*ledger.JournalEntry, error) {
	return s.repo.ListEntries(ctx, ledger.EntryFilter{Status: ledger.EntryPosted})
}

// isValidPeriodID validates the "YYYY-MM" shape (year >= 1900, month 1..12).
func isValidPeriodID(id string) bool {
	if len(id) != 7 || id[4] != '-' {
		return false
	}
	year, err := strconv.Atoi(id[:4])
	if err != nil || year < 1900 {
		return false
	}
	month, err := strconv.Atoi(id[5:])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	return true
}

func validatePeriodRange(from, to string) error {
	if !isValidPeriodID(from) || !isValidPeriodID(to) {
		return ledger.ErrInvalidPeriod
	}
	if from > to {
		return ledger.ErrInvalidRange
	}
	return nil
}

// sortByDateVoucher orders entries chronologically as books render them
// (UC-L5): by (VoucherDate, VoucherNo).
func sortByDateVoucher(entries []*ledger.JournalEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].VoucherDate != entries[j].VoucherDate {
			return entries[i].VoucherDate < entries[j].VoucherDate
		}
		return entries[i].VoucherNo < entries[j].VoucherNo
	})
}

// contraCodes lists the distinct đối ứng account codes of an entry, excluding
// the row's own account, in line order.
func contraCodes(lines []ledger.JournalLine, exclude string) string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lines {
		if l.AccountCode == exclude || seen[l.AccountCode] {
			continue
		}
		seen[l.AccountCode] = true
		out = append(out, l.AccountCode)
	}
	return strings.Join(out, ", ")
}

// entryNet returns the net debit-credit contribution of one account within an
// entry's lines (Nợ positive).
func entryNet(e *ledger.JournalEntry, code string) int64 {
	var net int64
	for _, l := range e.Lines {
		if l.AccountCode == code {
			net += l.Debit - l.Credit
		}
	}
	return net
}

// pageRows applies an optional offset/limit window to book rows, returning the
// paged slice plus the total row count before paging. A nil Page returns all
// rows untouched. Totals are always computed over the full book, never the
// window.
func pageRows[T any](rows []T, p *ledger.Page) ([]T, int) {
	if p == nil {
		return rows, len(rows)
	}
	start := p.Offset
	if start < 0 {
		start = 0
	}
	if start > len(rows) {
		start = len(rows)
	}
	end := start + p.Limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[start:end], len(rows)
}

// GetGeneralJournal renders Sổ Nhật ký chung for a period range: one row per
// entry line, ordered by (VoucherDate, VoucherNo), with column totals.
func (s *service) GetGeneralJournal(ctx context.Context, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.GeneralJournal, error) {
	if err := validatePeriodRange(fromPeriod, toPeriod); err != nil {
		return nil, err
	}
	entries, err := s.postedEntries(ctx)
	if err != nil {
		return nil, err
	}
	sortByDateVoucher(entries)

	book := &ledger.GeneralJournal{FromPeriod: fromPeriod, ToPeriod: toPeriod}
	for _, e := range entries {
		if e.Period < fromPeriod || e.Period > toPeriod {
			continue
		}
		for _, l := range e.Lines {
			book.Rows = append(book.Rows, ledger.BookRow{
				VoucherDate: e.VoucherDate,
				VoucherNo:   e.VoucherNo,
				Description: e.Description,
				Contra:      contraCodes(e.Lines, l.AccountCode),
				Debit:       l.Debit,
				Credit:      l.Credit,
			})
			book.TotalDebit += l.Debit
			book.TotalCredit += l.Credit
		}
	}
	book.Rows, book.Total = pageRows(book.Rows, p)
	return book, nil
}

// accountBook builds a per-account book (Sổ Cái when withBalance, Sổ chi tiết
// otherwise). The opening balance carries forward from periods before
// FromPeriod; rows are the account's lines in range with running balance.
func (s *service) accountBook(ctx context.Context, accountCode, fromPeriod, toPeriod string, withBalance bool, p *ledger.Page) (*ledger.LedgerBook, error) {
	if err := validatePeriodRange(fromPeriod, toPeriod); err != nil {
		return nil, err
	}
	acc, err := s.repo.GetAccountByCode(ctx, accountCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrAccountNotFound
		}
		return nil, err
	}
	entries, err := s.postedEntries(ctx)
	if err != nil {
		return nil, err
	}

	book := &ledger.LedgerBook{
		AccountCode: acc.Code,
		AccountName: acc.Name,
		FromPeriod:  fromPeriod,
		ToPeriod:    toPeriod,
	}

	var openNet int64
	for _, e := range entries {
		if e.Period >= fromPeriod {
			continue
		}
		openNet += entryNet(e, accountCode)
	}
	open := ledger.BalanceOf(openNet)
	book.OpenDebit, book.OpenCredit = open.Debit, open.Credit

	var balance int64 = openNet
	sortByDateVoucher(entries)
	for _, e := range entries {
		if e.Period < fromPeriod || e.Period > toPeriod {
			continue
		}
		for _, l := range e.Lines {
			if l.AccountCode != accountCode {
				continue
			}
			row := ledger.BookRow{
				VoucherDate: e.VoucherDate,
				VoucherNo:   e.VoucherNo,
				Description: e.Description,
				Contra:      contraCodes(e.Lines, accountCode),
				Debit:       l.Debit,
				Credit:      l.Credit,
			}
			if withBalance {
				balance += l.Debit - l.Credit
				row.Balance = balance
			}
			book.Rows = append(book.Rows, row)
			book.TotalDebit += l.Debit
			book.TotalCredit += l.Credit
		}
	}

	var closeNet int64 = openNet
	for _, r := range book.Rows {
		closeNet += r.Debit - r.Credit
	}
	close := ledger.BalanceOf(closeNet)
	book.CloseDebit, book.CloseCredit = close.Debit, close.Credit
	book.Rows, book.Total = pageRows(book.Rows, p)
	return book, nil
}

func (s *service) GetLedgerBook(ctx context.Context, accountCode, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.LedgerBook, error) {
	return s.accountBook(ctx, accountCode, fromPeriod, toPeriod, true, p)
}

func (s *service) GetDetailBook(ctx context.Context, accountCode, fromPeriod, toPeriod string, p *ledger.Page) (*ledger.LedgerBook, error) {
	return s.accountBook(ctx, accountCode, fromPeriod, toPeriod, false, p)
}

// GetTrialBalance renders BCĐPS (S06) for one period: per account the opening,
// period activity and closing balance as Nợ/Có pairs, plus column totals. The
// balanced flag is asserted on every column (E6).
func (s *service) GetTrialBalance(ctx context.Context, period string, p *ledger.Page) (*ledger.TrialBalance, error) {
	if !isValidPeriodID(period) {
		return nil, ledger.ErrInvalidPeriod
	}
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := s.postedEntries(ctx)
	if err != nil {
		return nil, err
	}

	name := map[string]string{}
	for _, a := range accounts {
		name[a.Code] = a.Name
	}

	openNet := map[string]int64{}
	actDebit := map[string]int64{}
	actCredit := map[string]int64{}
	for _, e := range entries {
		if e.Period > period {
			continue
		}
		for _, l := range e.Lines {
			if e.Period < period {
				openNet[l.AccountCode] += l.Debit - l.Credit
			} else {
				actDebit[l.AccountCode] += l.Debit
				actCredit[l.AccountCode] += l.Credit
			}
		}
	}

	tb := &ledger.TrialBalance{Period: period}
	codes := map[string]bool{}
	for c := range openNet {
		codes[c] = true
	}
	for c := range actDebit {
		codes[c] = true
	}
	for c := range actCredit {
		codes[c] = true
	}
	sorted := make([]string, 0, len(codes))
	for c := range codes {
		sorted = append(sorted, c)
	}
	sort.Strings(sorted)

	for _, code := range sorted {
		row := ledger.TrialBalanceRow{
			AccountCode: code,
			AccountName: name[code],
			Open:        ledger.BalanceOf(openNet[code]),
			Activity:    ledger.Balance{Debit: actDebit[code], Credit: actCredit[code]},
			Close:       ledger.BalanceOf(openNet[code] + actDebit[code] - actCredit[code]),
		}
		tb.Rows = append(tb.Rows, row)
		tb.Totals.Open.Debit += row.Open.Debit
		tb.Totals.Open.Credit += row.Open.Credit
		tb.Totals.Activity.Debit += row.Activity.Debit
		tb.Totals.Activity.Credit += row.Activity.Credit
		tb.Totals.Close.Debit += row.Close.Debit
		tb.Totals.Close.Credit += row.Close.Credit
	}
	tb.Balanced = tb.Totals.Open.Debit == tb.Totals.Open.Credit &&
		tb.Totals.Activity.Debit == tb.Totals.Activity.Credit &&
		tb.Totals.Close.Debit == tb.Totals.Close.Credit
	tb.Rows, tb.Total = pageRows(tb.Rows, p)
	return tb, nil
}

// voucherForm returns the voucher-number prefix for entries the ledger numbers
// itself (manual, closing, opening). Entries originating from an operating
// module already carry the source voucher number as VoucherNo, so they skip
// the ledger sequence entirely.
func voucherForm(src ledger.EntrySource) string {
	switch src {
	case ledger.SourceManual:
		return "PK"
	case ledger.SourceClosing:
		return "KC"
	case ledger.SourceOpening:
		return "DC"
	default:
		return ""
	}
}

// PostEntry posts a DRAFT entry (spec 5.2). R1–R3 are re-validated and R4 is
// re-checked at post time, so a draft created before the period closed cannot
// be posted into it. R10 assigns the VoucherNo inside the atomic repo
// transition. R6: posting is one-way — a POSTED entry cannot be re-posted.
// R5: re-posting a (Source, SourceRef) key that already has a POSTED entry
// returns that entry instead of duplicating it.
func (s *service) PostEntry(ctx context.Context, actor, id string) (*ledger.JournalEntry, error) {
	e, err := s.repo.GetEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.Status == ledger.EntryPosted {
		return e, nil
	}
	if e.Status != ledger.EntryDraft {
		return nil, ledger.ErrWrongState
	}
	if e.SourceRef != "" {
		existing, err := s.repo.GetEntryBySource(ctx, e.Source, e.SourceRef)
		switch {
		case err == nil && existing.ID != e.ID && existing.Status == ledger.EntryPosted:
			return existing, nil
		case err == nil && existing.ID != e.ID:
			return nil, ledger.ErrDuplicateSource
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return nil, err
		}
	}
	if err := validateLines(e.Lines); err != nil {
		return nil, err
	}
	if err := s.validatePostableAccounts(ctx, e.Lines); err != nil {
		return nil, err
	}
	if err := s.ensurePeriodOpen(ctx, e.Period); err != nil {
		return nil, err
	}
	e.Status = ledger.EntryPosted
	e.PostedBy = actor
	e.PostedAt = s.now().UTC().Format(time.RFC3339)

	// Self-numbered sources (manual/closing/opening) always take the next
	// number from the ledger sequence at post time (R10); a client-supplied
	// VoucherNo is discarded so it cannot collide with the sequence or reuse a
	// number. Operating-module sources carry their own source voucher number
	// and skip the sequence — one must be present.
	form := voucherForm(e.Source)
	if form != "" {
		e.VoucherNo = ""
	} else if e.VoucherNo == "" {
		return nil, ledger.ErrWrongState
	}
	posted, err := s.repo.PostEntry(ctx, e, form)
	if err != nil {
		return nil, err
	}
	if posted.ID != e.ID {
		// A concurrent post already claimed this (Source, SourceRef): the repo
		// returned that entry and wrote nothing new (R5) — no audit needed.
		return posted, nil
	}
	if err := s.auditAction(ctx, actor, "entry.post", e.ID); err != nil {
		return nil, err
	}
	return e, nil
}

// DeleteEntry removes a DRAFT entry only (R7). POSTED and REVERSED entries are
// append-only (R6) — they can be reversed, never deleted.
func (s *service) DeleteEntry(ctx context.Context, actor, id string) error {
	e, err := s.repo.GetEntry(ctx, id)
	if err != nil {
		return err
	}
	if e.Status != ledger.EntryDraft {
		return ledger.ErrWrongState
	}
	if err := s.repo.DeleteEntry(ctx, id); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "entry.delete", id)
}

// CreateAccount validates the chart-of-accounts hierarchy (P1.2): the parent
// must exist, the level is one more than the parent's, the type matches the
// parent's, and an account with children never allows direct posting. Level-1
// root accounts are summary groups and are never postable.
func (s *service) CreateAccount(ctx context.Context, actor string, a *ledger.Account) error {
	if a.Code == "" || a.Name == "" {
		return ledger.ErrInvalidAccount
	}
	parent, err := s.resolveParent(ctx, a.ParentCode)
	if err != nil {
		return err
	}
	if parent != nil {
		a.Level = parent.Level + 1
		if a.Type == "" {
			a.Type = parent.Type
		}
		if a.Type != parent.Type {
			return ledger.ErrTypeMismatch
		}
	}
	if err := validateTypeLevel(a); err != nil {
		return err
	}
	a.ID = ledger.RowID("account", a.Code)
	if a.Status == "" {
		a.Status = ledger.AccountActive
	}
	s.applyPostableRule(ctx, a)
	if err := s.repo.CreateAccount(ctx, a); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "account.create", a.ID)
}

// UpdateAccount mutates only Name, Status and AllowPost. Code, type, parent
// and level are immutable after creation (they define the account's slot in
// the chart). The leaf-only invariant is re-applied on every save.
func (s *service) UpdateAccount(ctx context.Context, actor string, a *ledger.Account) error {
	existing, err := s.repo.GetAccount(ctx, a.ID)
	if err != nil {
		return err
	}
	a.Code = existing.Code
	a.Type = existing.Type
	a.ParentCode = existing.ParentCode
	a.Level = existing.Level
	if a.Name == "" {
		return ledger.ErrInvalidAccount
	}
	s.applyPostableRule(ctx, a)
	if err := s.repo.UpdateAccount(ctx, a); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "account.update", a.ID)
}

func (s *service) GetAccount(ctx context.Context, id string) (*ledger.Account, error) {
	return s.repo.GetAccount(ctx, id)
}

func (s *service) ListAccounts(ctx context.Context, f ledger.AccountFilter) ([]*ledger.Account, error) {
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := []*ledger.Account{}
	q := strings.ToLower(f.Q)
	for _, a := range accounts {
		if f.Type != "" && a.Type != f.Type {
			continue
		}
		if f.ParentCode != "" && a.ParentCode != f.ParentCode {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(a.Code+a.Name), q) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *service) ListPeriods(ctx context.Context) ([]*ledger.AccountingPeriod, error) {
	return s.repo.ListPeriods(ctx)
}

// parsePeriodID validates the "YYYY-MM" shape and fills Year/Month. A period id
// like "2026-08" is a single "YYYY" then "-" then "MM" (1..12).
func parsePeriodID(p *ledger.AccountingPeriod) error {
	if !isValidPeriodID(p.ID) {
		return ledger.ErrInvalidPeriod
	}
	p.Year, _ = strconv.Atoi(p.ID[:4])
	p.Month, _ = strconv.Atoi(p.ID[5:])
	return nil
}

// OpenPeriod ensures the "YYYY-MM" accounting period exists and is open. It is
// idempotent: an already-open period is returned unchanged.
func (s *service) OpenPeriod(ctx context.Context, actor, id string) (*ledger.AccountingPeriod, error) {
	p := &ledger.AccountingPeriod{ID: id, Year: -1, Month: -1, Status: ledger.PeriodOpen, OpenedBy: actor}
	if err := parsePeriodID(p); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetPeriod(ctx, id)
	if err == nil {
		if existing.Status == ledger.PeriodClosed {
			return nil, ledger.ErrWrongState
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err := s.repo.CreatePeriod(ctx, p); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "period.open", id); err != nil {
		return nil, err
	}
	return p, nil
}

// ClosePeriod locks the period against further postings (R4). A close reason is
// mandatory and recorded with the closer, mirroring the paper hand-over. Closing
// an already-closed period is idempotent.
func (s *service) ClosePeriod(ctx context.Context, actor, id, reason string) (*ledger.AccountingPeriod, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, ledger.ErrCloseReasonRequired
	}
	p := &ledger.AccountingPeriod{ID: id}
	if err := parsePeriodID(p); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Closing a period that was never opened is a plain 404.
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if existing.Status == ledger.PeriodClosed {
		return existing, nil
	}
	existing.Status = ledger.PeriodClosed
	existing.ClosedBy = actor
	existing.ClosedAt = s.now().UTC().Format(time.RFC3339)
	existing.CloseReason = reason
	if err := s.repo.CreatePeriod(ctx, existing); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "period.close", id); err != nil {
		return nil, err
	}
	return existing, nil
}

// ReopenPeriod un-locks a closed period. Like closing, it requires a reason and
// records the actor.
func (s *service) ReopenPeriod(ctx context.Context, actor, id, reason string) (*ledger.AccountingPeriod, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, ledger.ErrCloseReasonRequired
	}
	p := &ledger.AccountingPeriod{ID: id}
	if err := parsePeriodID(p); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.Status == ledger.PeriodOpen {
		return existing, nil
	}
	existing.Status = ledger.PeriodOpen
	existing.ClosedBy = ""
	existing.ClosedAt = ""
	existing.CloseReason = ""
	if err := s.repo.CreatePeriod(ctx, existing); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "period.reopen", id); err != nil {
		return nil, err
	}
	return existing, nil
}

// ensurePeriodOpen enforces R4 in CreateEntry: postings into a CLOSED period are
// rejected. Periods that were never explicitly opened are treated as open.
func (s *service) ensurePeriodOpen(ctx context.Context, period string) error {
	p, err := s.repo.GetPeriod(ctx, period)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if p.Status == ledger.PeriodClosed {
		return ledger.ErrPeriodClosed
	}
	return nil
}

func validateTypeLevel(a *ledger.Account) error {
	switch a.Type {
	case ledger.AccountAsset, ledger.AccountLiability, ledger.AccountEquity,
		ledger.AccountRevenue, ledger.AccountExpense:
	default:
		return ledger.ErrInvalidType
	}
	if a.Level < 1 || a.Level > 6 {
		return ledger.ErrInvalidLevel
	}
	return nil
}

// applyPostableRule enforces R3's leaf-only invariant: level-1 root accounts
// and any account that has children are summary accounts and never postable.
func (s *service) applyPostableRule(ctx context.Context, a *ledger.Account) {
	if a.Level == 1 || s.hasChildren(ctx, a.Code) {
		a.AllowPost = false
	}
}

func (s *service) resolveParent(ctx context.Context, code string) (*ledger.Account, error) {
	if code == "" {
		return nil, nil
	}
	parent, err := s.repo.GetAccountByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrParentNotFound
		}
		return nil, err
	}
	return parent, nil
}

func (s *service) hasChildren(ctx context.Context, code string) bool {
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return false
	}
	for _, a := range accounts {
		if a.ParentCode == code {
			return true
		}
	}
	return false
}

// validateLines enforces R1 (balanced) and R2 (exactly one side per line).
func validateLines(lines []ledger.JournalLine) error {
	var totalDebit, totalCredit int64
	for _, l := range lines {
		if l.Debit < 0 || l.Credit < 0 {
			return ledger.ErrInvalidLine
		}
		oneSide := (l.Debit > 0) != (l.Credit > 0)
		if !oneSide {
			return ledger.ErrInvalidLine
		}
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if totalDebit != totalCredit {
		return ledger.ErrUnbalanced
	}
	return nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
