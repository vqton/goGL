package cash

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	domainaudit "goGL/internal/domain/audit"
	"goGL/internal/domain/cash"
)

// AuditRecorder is the seam the cash service uses to write audit rows. The
// audit module's Service satisfies it; swap freely without coupling.
type AuditRecorder interface {
	Record(ctx context.Context, l *domainaudit.AuditLog) error
}

// LedgerEntry is the posting the cash service hands to the general-ledger
// seam (T2.5). For a receive voucher cash is debited; for a pay voucher it is
// credited, always against the fund's own account (e.g. TK111).
type LedgerEntry struct {
	Date      string
	Account   string
	Debit     int64
	Credit    int64
	RefNo     string
	FundID    string
	VoucherID string
}

// LedgerWriter is the seam for writing entries into the general ledger. The
// ledger module's writer will satisfy it later; a no-op keeps the skeleton
// runnable in the meantime.
type LedgerWriter interface {
	Post(ctx context.Context, e LedgerEntry) error
}

type noopLedger struct{}

func (noopLedger) Post(context.Context, LedgerEntry) error { return nil }

// Notifier is the seam for out-of-band notifications. CloseDay uses it to
// alert the chief accountant when a count mismatch is found (R7). A no-op
// keeps the skeleton runnable until real channels exist.
type Notifier interface {
	Notify(ctx context.Context, recipientRole, subject, body string) error
}

type noopNotifier struct{}

func (noopNotifier) Notify(context.Context, string, string, string) error { return nil }

// VoidApprover is the seam that gates voiding a posted/reconciled voucher.
// Per Điều 30 the void of an already-posted voucher requires the chief
// accountant's approval; main.go wires a Casbin-backed implementation. The
// default approves everything so the skeleton stays runnable until real
// authorization wiring exists.
type VoidApprover interface {
	CanApproveVoid(ctx context.Context, actor string) (bool, error)
}

type alwaysVoidApprover struct{}

func (alwaysVoidApprover) CanApproveVoid(context.Context, string) (bool, error) { return true, nil }

type Service interface {
	CreateFund(ctx context.Context, f *cash.Fund) error
	ListFunds(ctx context.Context) ([]*cash.Fund, error)

	CreateVoucher(ctx context.Context, actor string, v *cash.Voucher) error
	UpdateVoucher(ctx context.Context, actor string, v *cash.Voucher) error
	ApproveVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error)
	PostVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error)
	GetVoucher(ctx context.Context, id string) (*cash.Voucher, error)
	ListVouchers(ctx context.Context, f cash.VoucherFilter) ([]*cash.Voucher, error)

	GetCashBook(ctx context.Context, fundID, from, to string) ([]*cash.CashBookEntry, error)

	CloseDay(ctx context.Context, actor, fundID, date string, countedAmount int64, participants []string) (*cash.CashCount, error)
	CreateCashCount(ctx context.Context, actor, fundID, date string, countedAmount int64, participants []string) (*cash.CashCount, error)
	ResolveCashCount(ctx context.Context, actor, id, resolution string) (*cash.CashCount, error)
	GetCashCount(ctx context.Context, id string) (*cash.CashCount, error)
	ListCashCounts(ctx context.Context, fundID string) ([]*cash.CashCount, error)

	VoidVoucher(ctx context.Context, actor, id, reason string) (*cash.Voucher, error)
	ReconcileMonth(ctx context.Context, actor, fundID, period string, accountantBalance int64, signers []string) (*cash.Reconciliation, error)
	GetReconciliation(ctx context.Context, id string) (*cash.Reconciliation, error)
	ListReconciliations(ctx context.Context, fundID string) ([]*cash.Reconciliation, error)
}

type service struct {
	repo         cash.Repository
	audit        AuditRecorder
	ledger       LedgerWriter
	notifier     Notifier
	voidApprover VoidApprover
	now          func() time.Time

	// mu serializes the read-compute-write mutators (post, close-day,
	// reconcile, void). Cash book balances and closed-day/period lists are
	// derived from a shared aggregate, so concurrent mutators must not
	// interleave between their read and write steps (T5.1).
	mu sync.Mutex
}

type Option func(*service)

func WithLedger(l LedgerWriter) Option {
	return func(s *service) { s.ledger = l }
}

func WithNotifier(n Notifier) Option {
	return func(s *service) { s.notifier = n }
}

func WithVoidApprover(a VoidApprover) Option {
	return func(s *service) { s.voidApprover = a }
}

func NewService(repo cash.Repository, audit AuditRecorder, opts ...Option) Service {
	s := &service{
		repo: repo, audit: audit,
		ledger: noopLedger{}, notifier: noopNotifier{}, voidApprover: alwaysVoidApprover{},
		now: time.Now,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *service) auditAction(ctx context.Context, actor, action, targetID string) error {
	return s.audit.Record(ctx, &domainaudit.AuditLog{
		UserCode:  actor,
		Module:    "cash",
		Action:    action,
		TargetID:  targetID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) CreateFund(ctx context.Context, f *cash.Fund) error {
	if f.Name == "" || f.Currency == "" || f.Account == "" {
		return errors.New("cash: fund name, currency and account are required")
	}
	if f.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		f.ID = id
	}
	if err := s.repo.CreateFund(ctx, f); err != nil {
		return err
	}
	return s.auditAction(ctx, "", "fund.create", f.ID)
}

func (s *service) ListFunds(ctx context.Context) ([]*cash.Fund, error) {
	return s.repo.ListFunds(ctx)
}

func (s *service) CreateVoucher(ctx context.Context, actor string, v *cash.Voucher) error {
	fund, err := s.loadActiveFund(ctx, v.FundID)
	if err != nil {
		return err
	}
	if v.Currency == "" {
		v.Currency = fund.Currency
	}
	if v.Currency != fund.Currency {
		return errors.New("cash: voucher currency must match the fund currency")
	}
	if err := validateVoucher(v, fund.Account); err != nil {
		return err
	}

	id, err := newID()
	if err != nil {
		return err
	}
	v.ID = id
	v.State = cash.VoucherDraft
	v.CreatedBy = actor
	v.ApprovedBy = ""
	v.PostedBy = ""
	v.ApprovedAt = ""
	v.PostedAt = ""
	v.AmountWords = cash.AmountInWords(v.AmountMinor)

	refNo, err := s.repo.NextRefNo(ctx, v.FundID, v.RefDate[:7], v.Type)
	if err != nil {
		return err
	}
	v.RefNo = refNo

	if err := s.repo.CreateVoucher(ctx, v); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "voucher.create", v.ID)
}

func (s *service) UpdateVoucher(ctx context.Context, actor string, v *cash.Voucher) error {
	existing, err := s.repo.GetVoucher(ctx, v.ID)
	if err != nil {
		return err
	}
	if existing.State != cash.VoucherDraft {
		return cash.ErrWrongState
	}
	if v.FundID != existing.FundID || v.Type != existing.Type {
		return errors.New("cash: fund and type are immutable after creation")
	}

	fund, err := s.loadActiveFund(ctx, v.FundID)
	if err != nil {
		return err
	}
	if v.Currency == "" {
		v.Currency = fund.Currency
	}
	if v.Currency != fund.Currency {
		return errors.New("cash: voucher currency must match the fund currency")
	}
	if err := validateVoucher(v, fund.Account); err != nil {
		return err
	}

	v.ID = existing.ID
	v.RefNo = existing.RefNo
	v.CreatedBy = existing.CreatedBy
	v.State = cash.VoucherDraft
	v.ApprovedBy = ""
	v.PostedBy = ""
	v.ApprovedAt = ""
	v.PostedAt = ""
	v.AmountWords = cash.AmountInWords(v.AmountMinor)

	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "voucher.update", v.ID)
}

func (s *service) ApproveVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error) {
	v, err := s.repo.GetVoucher(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.State != cash.VoucherDraft {
		return nil, cash.ErrWrongState
	}
	if actor == v.CreatedBy {
		return nil, cash.ErrSelfApproval
	}
	v.State = cash.VoucherApproved
	v.ApprovedBy = actor
	v.ApprovedAt = s.now().UTC().Format(time.RFC3339)

	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "voucher.approve", id); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *service) PostVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, err := s.repo.GetVoucher(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.State != cash.VoucherApproved {
		return nil, cash.ErrWrongState
	}

	// UC-3 E2: the poster must be a third party — neither the preparer nor
	// the approver (R6).
	if actor == v.CreatedBy || actor == v.ApprovedBy {
		return nil, cash.ErrUnauthorizedActor
	}

	fund, err := s.loadActiveFund(ctx, v.FundID)
	if err != nil {
		return nil, err
	}
	for _, d := range fund.ClosedDays {
		if d == v.RefDate {
			return nil, cash.ErrPeriodClosed
		}
	}
	for _, p := range fund.ClosedPeriods {
		if p == v.RefDate[:7] {
			return nil, cash.ErrPeriodClosed
		}
	}

	entries, err := s.repo.ListCashBook(ctx, v.FundID, "", v.RefDate)
	if err != nil {
		return nil, err
	}
	balance := runningBalance(entries)
	switch v.Type {
	case cash.VoucherReceive:
		balance += v.AmountMinor
	case cash.VoucherPay:
		if balance < v.AmountMinor {
			return nil, cash.ErrNegativeBalance
		}
		balance -= v.AmountMinor
	}

	entry := &cash.CashBookEntry{
		ID:          cash.RowID("cash_book", v.FundID, v.ID),
		FundID:      v.FundID,
		EntryDate:   v.RefDate,
		VoucherDate: v.RefDate,
		RefNo:       v.RefNo,
		Type:        v.Type,
		Description: v.Description,
		Balance:     balance,
		Reconciled:  false,
	}
	if v.Type == cash.VoucherReceive {
		entry.Receive = v.AmountMinor
	} else {
		entry.Pay = v.AmountMinor
	}
	if err := s.repo.AppendCashBookEntry(ctx, entry); err != nil {
		return nil, err
	}
	if err := s.rebuildBalances(ctx, v.FundID); err != nil {
		return nil, err
	}

	// Compensate on any failure after the book row is written: delete the
	// entry and revert the voucher to approved so the two never diverge.
	rollback := func() {
		_ = s.repo.DeleteCashBookEntry(ctx, entry.ID)
		_ = s.rebuildBalances(ctx, v.FundID)
		v.State = cash.VoucherApproved
		v.PostedBy = ""
		v.PostedAt = ""
		_ = s.repo.UpdateVoucher(ctx, v)
	}

	v.State = cash.VoucherPosted
	v.PostedBy = actor
	v.PostedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return nil, err
	}

	le := LedgerEntry{
		Date:      v.RefDate,
		Account:   fund.Account,
		RefNo:     v.RefNo,
		FundID:    v.FundID,
		VoucherID: v.ID,
	}
	if v.Type == cash.VoucherReceive {
		le.Debit = v.AmountMinor
	} else {
		le.Credit = v.AmountMinor
	}
	if err := s.ledger.Post(ctx, le); err != nil {
		rollback()
		return nil, err
	}

	if err := s.auditAction(ctx, actor, "voucher.post", v.ID); err != nil {
		rollback()
		return nil, err
	}
	return v, nil
}

// runningBalance sums Receive-Pay over book entries in chronological order.
// The single helper replaces the repeated inline loops in the mutators.
func runningBalance(entries []*cash.CashBookEntry) int64 {
	var b int64
	for _, e := range entries {
		b += e.Receive - e.Pay
	}
	return b
}

// rebuildBalances re-derives every cash-book running balance for a fund after
// an entry is appended or removed, so a back-dated post (or a compensated
// failure) never leaves later rows with stale balances.
func (s *service) rebuildBalances(ctx context.Context, fundID string) error {
	entries, err := s.repo.ListCashBook(ctx, fundID, "", "")
	if err != nil {
		return err
	}
	var balance int64
	for _, e := range entries {
		balance += e.Receive - e.Pay
		if e.Balance != balance {
			e.Balance = balance
			if err := s.repo.UpdateCashBookEntry(ctx, e); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) GetCashBook(ctx context.Context, fundID, from, to string) ([]*cash.CashBookEntry, error) {
	return s.repo.ListCashBook(ctx, fundID, from, to)
}

func (s *service) CloseDay(ctx context.Context, actor, fundID, date string, countedAmount int64, participants []string) (*cash.CashCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fund, err := s.loadActiveFund(ctx, fundID)
	if err != nil {
		return nil, err
	}

	// Every voucher on or before the close date must already be posted (or
	// voided/reconciled); a draft or approved voucher would miss the count.
	vouchers, err := s.repo.ListVouchers(ctx, cash.VoucherFilter{FundID: fundID, To: date})
	if err != nil {
		return nil, err
	}
	for _, v := range vouchers {
		if v.State == cash.VoucherDraft || v.State == cash.VoucherApproved {
			return nil, cash.ErrUnpostedVouchers
		}
	}

	count, err := s.createCashCount(ctx, actor, fundID, date, countedAmount, participants)
	if err != nil {
		return nil, err
	}
	if count.State != cash.CashCountResolved {
		return count, nil
	}
	fund.ClosedDays = append(fund.ClosedDays, date)
	if err := s.repo.CreateFund(ctx, fund); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "cash.close_day", count.ID); err != nil {
		return nil, err
	}
	return count, nil
}

// CreateCashCount records a standalone biên bản kiểm kê quỹ. Unlike CloseDay
// it never closes the day: a matching count is a snapshot, an open one waits
// for ResolveCashCount to lock the day.
func (s *service) CreateCashCount(ctx context.Context, actor, fundID, date string, countedAmount int64, participants []string) (*cash.CashCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createCashCount(ctx, actor, fundID, date, countedAmount, participants)
}

// ResolveCashCount resolves an open count (UC-4) and then closes the day: the
// mismatch is recorded, signed off via the resolution text, and the fund's
// closed-day set is extended so later posts are blocked.
func (s *service) ResolveCashCount(ctx context.Context, actor, id, resolution string) (*cash.CashCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, err := s.repo.GetCashCount(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, cash.ErrNotFound
		}
		return nil, err
	}
	if c.State != cash.CashCountOpen {
		return nil, cash.ErrWrongState
	}
	fund, err := s.loadActiveFund(ctx, c.FundID)
	if err != nil {
		return nil, err
	}

	c.State = cash.CashCountResolved
	c.Resolution = resolution
	if err := s.repo.CreateCashCount(ctx, c); err != nil {
		return nil, err
	}
	fund.ClosedDays = append(fund.ClosedDays, c.CountDate)
	if err := s.repo.CreateFund(ctx, fund); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "cash.count.resolve", c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

// createCashCount is the shared count-recording core behind CloseDay and
// CreateCashCount: validates, computes the book balance, persists the count,
// and notifies the chief accountant on a mismatch (R7). It never touches the
// fund's closed-day set — callers decide that.
func (s *service) createCashCount(ctx context.Context, actor, fundID, date string, countedAmount int64, participants []string) (*cash.CashCount, error) {
	fund, err := s.loadActiveFund(ctx, fundID)
	if err != nil {
		return nil, err
	}
	if countedAmount < 0 {
		return nil, cash.ErrInvalidCount
	}
	for _, d := range fund.ClosedDays {
		if d == date {
			return nil, cash.ErrPeriodClosed
		}
	}

	counts, err := s.repo.ListCashCounts(ctx, fundID)
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		if c.CountDate == date && c.State == cash.CashCountOpen {
			return nil, cash.ErrOpenCountPending
		}
	}

	entries, err := s.repo.ListCashBook(ctx, fundID, "", date)
	if err != nil {
		return nil, err
	}
	bookBalance := runningBalance(entries)

	count := &cash.CashCount{
		ID:            cash.RowID("cash_count", fundID, date),
		FundID:        fundID,
		CountDate:     date,
		BookBalance:   bookBalance,
		CountedAmount: countedAmount,
		Difference:    bookBalance - countedAmount,
		Participants:  participants,
	}
	if countedAmount == bookBalance {
		count.State = cash.CashCountResolved
	} else {
		count.State = cash.CashCountOpen
	}

	if err := s.repo.CreateCashCount(ctx, count); err != nil {
		return nil, err
	}
	if count.State == cash.CashCountOpen {
		if err := s.auditAction(ctx, actor, "cash.count.open", count.ID); err != nil {
			return nil, err
		}
		if err := s.notifier.Notify(ctx, "chief_accountant",
			"Chênh lệch quỹ "+date,
			"Sổ quỹ "+cash.FormatVNDMinor(bookBalance)+", kiểm kê "+cash.FormatVNDMinor(countedAmount)); err != nil {
			return nil, err
		}
		return count, nil
	}
	if err := s.auditAction(ctx, actor, "cash.count", count.ID); err != nil {
		return nil, err
	}
	return count, nil
}

func (s *service) ListCashCounts(ctx context.Context, fundID string) ([]*cash.CashCount, error) {
	return s.repo.ListCashCounts(ctx, fundID)
}

func (s *service) GetCashCount(ctx context.Context, id string) (*cash.CashCount, error) {
	return s.repo.GetCashCount(ctx, id)
}

// ReconcileMonth runs the month-end reconciliation (UC-5). signers carries the
// three electronic signatories — thủ quỹ, kế toán, kế toán trưởng — who must
// be distinct and non-empty; all three sign the biên bản regardless of whether
// the balances match.
func (s *service) ReconcileMonth(ctx context.Context, actor, fundID, period string, accountantBalance int64, signers []string) (*cash.Reconciliation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(signers) != 3 || signers[0] == "" || signers[1] == "" || signers[2] == "" ||
		signers[0] == signers[1] || signers[1] == signers[2] || signers[0] == signers[2] {
		return nil, cash.ErrInvalidSigners
	}

	fund, err := s.loadActiveFund(ctx, fundID)
	if err != nil {
		return nil, err
	}

	counts, err := s.repo.ListCashCounts(ctx, fundID)
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		if c.State == cash.CashCountOpen {
			return nil, cash.ErrOpenCountPending
		}
	}

	entries, err := s.repo.ListCashBook(ctx, fundID, period+"-01", period+"-31")
	if err != nil {
		return nil, err
	}
	cashierBalance := runningBalance(entries)

	rec := &cash.Reconciliation{
		ID:                cash.RowID("cash_recon", fundID, period),
		FundID:            fundID,
		Period:            period,
		CashierBalance:    cashierBalance,
		AccountantBalance: accountantBalance,
		Difference:        cashierBalance - accountantBalance,
		SignedBy:          signers,
		CreatedAt:         s.now().UTC().Format(time.RFC3339),
	}

	if rec.Difference != 0 {
		rec.State = cash.ReconciliationDiff
		if err := s.repo.CreateReconciliation(ctx, rec); err != nil {
			return nil, err
		}
		if err := s.auditAction(ctx, actor, "cash.reconcile.diff", rec.ID); err != nil {
			return nil, err
		}
		return rec, nil
	}

	rec.State = cash.ReconciliationResolved

	fund.ClosedPeriods = append(fund.ClosedPeriods, period)
	if err := s.repo.CreateFund(ctx, fund); err != nil {
		return nil, err
	}

	vouchers, err := s.repo.ListVouchers(ctx, cash.VoucherFilter{FundID: fundID, From: period + "-01", To: period + "-31"})
	if err != nil {
		return nil, err
	}
	for _, v := range vouchers {
		if v.State != cash.VoucherPosted {
			continue
		}
		v.State = cash.VoucherReconciled
		if err := s.repo.UpdateVoucher(ctx, v); err != nil {
			return nil, err
		}
	}

	if err := s.repo.CreateReconciliation(ctx, rec); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "cash.reconcile", rec.ID); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *service) ListReconciliations(ctx context.Context, fundID string) ([]*cash.Reconciliation, error) {
	return s.repo.ListReconciliations(ctx, fundID)
}

func (s *service) GetReconciliation(ctx context.Context, id string) (*cash.Reconciliation, error) {
	return s.repo.GetReconciliation(ctx, id)
}

func oppositeType(t cash.VoucherType) cash.VoucherType {
	if t == cash.VoucherReceive {
		return cash.VoucherPay
	}
	return cash.VoucherReceive
}

// buildReversal constructs the Điều 30 offsetting voucher: opposite type,
// identical amount, inverted lines, linked back to the original.
func buildReversal(v *cash.Voucher, reason string) *cash.Voucher {
	rev := &cash.Voucher{
		RefDate:          v.RefDate,
		Type:             oppositeType(v.Type),
		FundID:           v.FundID,
		Currency:         v.Currency,
		AmountMinor:      v.AmountMinor,
		FXRate:           v.FXRate,
		CounterpartyType: v.CounterpartyType,
		CounterpartyID:   v.CounterpartyID,
		CounterpartyName: v.CounterpartyName,
		Description:      "Điều chỉnh hủy " + v.RefNo,
		RefVouchers:      []string{v.ID},
	}
	if reason != "" {
		rev.Description += ": " + reason
	}
	for _, l := range v.Lines {
		rev.Lines = append(rev.Lines, cash.VoucherLine{
			Seq:         l.Seq,
			DebitAcc:    l.CreditAcc,
			CreditAcc:   l.DebitAcc,
			AmountMinor: l.AmountMinor,
			ObjectID:    l.ObjectID,
		})
	}
	return rev
}

// postReversal appends the reversal's offsetting cash-book entry, enforcing
// the no-negative-balance rule against the same running balance.
func (s *service) postReversal(ctx context.Context, rev *cash.Voucher) error {
	entries, err := s.repo.ListCashBook(ctx, rev.FundID, "", rev.RefDate)
	if err != nil {
		return err
	}
	balance := runningBalance(entries)

	entry := &cash.CashBookEntry{
		ID:          cash.RowID("cash_book", rev.FundID, rev.ID),
		FundID:      rev.FundID,
		EntryDate:   rev.RefDate,
		VoucherDate: rev.RefDate,
		RefNo:       rev.RefNo,
		Type:        rev.Type,
		Description: rev.Description,
		Reconciled:  false,
	}
	switch rev.Type {
	case cash.VoucherReceive:
		entry.Receive = rev.AmountMinor
		balance += rev.AmountMinor
	case cash.VoucherPay:
		if balance < rev.AmountMinor {
			return cash.ErrNegativeBalance
		}
		entry.Pay = rev.AmountMinor
		balance -= rev.AmountMinor
	}
	entry.Balance = balance
	if err := s.repo.AppendCashBookEntry(ctx, entry); err != nil {
		return err
	}
	return s.rebuildBalances(ctx, rev.FundID)
}

func (s *service) VoidVoucher(ctx context.Context, actor, id, reason string) (*cash.Voucher, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, err := s.repo.GetVoucher(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, cash.ErrVoucherNotFound
		}
		return nil, err
	}
	switch v.State {
	case cash.VoucherDraft, cash.VoucherApproved:
		return s.markVoided(ctx, actor, v, reason)
	case cash.VoucherPosted, cash.VoucherReconciled:
		return s.voidPosted(ctx, actor, v, reason)
	default:
		return nil, cash.ErrWrongState
	}
}

func (s *service) markVoided(ctx context.Context, actor string, v *cash.Voucher, reason string) (*cash.Voucher, error) {
	v.State = cash.VoucherVoided
	v.VoidedBy = actor
	v.VoidReason = reason
	v.VoidedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "voucher.void", v.ID); err != nil {
		return nil, err
	}
	return v, nil
}

// voidPosted voids a posted/reconciled voucher via an offsetting reversal
// (Điều 30): a reversal draft linked to the voucher is reused if present
// (amount must match, E2), otherwise one is created and posted internally.
// The chief accountant's approval is required (Điều 30) via the VoidApprover
// seam; main.go wires a Casbin-backed implementation.
func (s *service) voidPosted(ctx context.Context, actor string, v *cash.Voucher, reason string) (*cash.Voucher, error) {
	ok, err := s.voidApprover.CanApproveVoid(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, cash.ErrUnauthorizedActor
	}

	vouchers, err := s.repo.ListVouchers(ctx, cash.VoucherFilter{FundID: v.FundID})
	if err != nil {
		return nil, err
	}
	var reversal *cash.Voucher
	for _, cand := range vouchers {
		if cand.State != cash.VoucherDraft || cand.Type == v.Type || cand.ID == v.ID {
			continue
		}
		for _, ref := range cand.RefVouchers {
			if ref == v.ID {
				reversal = cand
				break
			}
		}
		if reversal != nil {
			break
		}
	}

	reuse := reversal != nil
	if reuse {
		if reversal.AmountMinor != v.AmountMinor {
			return nil, cash.ErrReversalMismatch
		}
	} else {
		reversal = buildReversal(v, reason)
		id, err := newID()
		if err != nil {
			return nil, err
		}
		reversal.ID = id
		refNo, err := s.repo.NextRefNo(ctx, reversal.FundID, reversal.RefDate[:7], reversal.Type)
		if err != nil {
			return nil, err
		}
		reversal.RefNo = refNo
	}

	if err := s.postReversal(ctx, reversal); err != nil {
		return nil, err
	}

	reversal.State = cash.VoucherVoided
	reversal.VoidedBy = actor
	reversal.VoidReason = "đối ứng " + v.RefNo
	reversal.VoidedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.repo.UpdateVoucher(ctx, reversal); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "voucher.reversal.create", reversal.ID); err != nil {
		return nil, err
	}

	v.RefVouchers = append(v.RefVouchers, reversal.ID)
	return s.markVoided(ctx, actor, v, reason)
}

func (s *service) GetVoucher(ctx context.Context, id string) (*cash.Voucher, error) {
	return s.repo.GetVoucher(ctx, id)
}

func (s *service) ListVouchers(ctx context.Context, f cash.VoucherFilter) ([]*cash.Voucher, error) {
	return s.repo.ListVouchers(ctx, f)
}

func (s *service) loadActiveFund(ctx context.Context, id string) (*cash.Fund, error) {
	fund, err := s.repo.GetFund(ctx, id)
	if err != nil {
		return nil, cash.ErrFundNotFound
	}
	if !fund.IsActive {
		return nil, cash.ErrFundInactive
	}
	return fund, nil
}

func validateVoucher(v *cash.Voucher, cashAccount string) error {
	if v.Type != cash.VoucherReceive && v.Type != cash.VoucherPay {
		return errors.New("cash: invalid voucher type")
	}
	if v.FundID == "" {
		return errors.New("cash: fund_id is required")
	}
	if v.RefDate == "" || len(v.RefDate) < 7 {
		return errors.New("cash: ref_date is required as yyyy-mm-dd")
	}
	if _, err := time.Parse("2006-01-02", v.RefDate); err != nil {
		return errors.New("cash: ref_date must be yyyy-mm-dd (T5.2)")
	}
	if v.AmountMinor <= 0 {
		return errors.New("cash: amount must be positive")
	}
	if v.CounterpartyName == "" {
		return errors.New("cash: counterparty name is required (R1)")
	}
	if len(v.Lines) < 2 {
		return cash.ErrInvalidLines
	}

	var sumDebit, sumCredit int64
	cashLines := 0
	for _, l := range v.Lines {
		if l.AmountMinor <= 0 {
			return cash.ErrInvalidLines
		}
		if l.DebitAcc != "" {
			sumDebit += l.AmountMinor
		}
		if l.CreditAcc != "" {
			sumCredit += l.AmountMinor
		}
		if strings.HasPrefix(l.DebitAcc, cashAccount) || strings.HasPrefix(l.CreditAcc, cashAccount) {
			cashLines++
		}
	}
	if sumDebit != v.AmountMinor || sumCredit != v.AmountMinor {
		return cash.ErrInvalidLines
	}
	if cashLines != 1 {
		return cash.ErrInvalidLines
	}
	return nil
}
