package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type AccountType string

const (
	AccountAsset     AccountType = "asset"
	AccountLiability AccountType = "liability"
	AccountEquity    AccountType = "equity"
	AccountRevenue   AccountType = "revenue"
	AccountExpense   AccountType = "expense"
)

type AccountStatus string

const (
	AccountActive   AccountStatus = "active"
	AccountInactive AccountStatus = "inactive"
	AccountLocked   AccountStatus = "locked"
)

// Account = một tài khoản trong Sơ đồ tài khoản (Hệ thống TK kế toán).
// ID is a deterministic SHA-256 of Code (JSON-doc table pattern). AllowPost is
// false for parent/summary accounts, which never take direct postings (R3).
type Account struct {
	ID         string        `json:"id"`
	Code       string        `json:"code"`
	Name       string        `json:"name"`
	ParentCode string        `json:"parent_code,omitempty"`
	Type       AccountType   `json:"type"`
	Level      int           `json:"level"`
	Status     AccountStatus `json:"status"`
	AllowPost  bool          `json:"allow_post"`
}

type EntrySource string

const (
	SourceManual     EntrySource = "manual"
	SourceCash       EntrySource = "cash"
	SourceBank       EntrySource = "bank"
	SourceInvoice    EntrySource = "invoice"
	SourcePayroll    EntrySource = "payroll"
	SourceFixedAsset EntrySource = "fixedasset"
	SourceOpening    EntrySource = "opening"
	SourceClosing    EntrySource = "closing"
)

type EntryStatus string

const (
	EntryDraft    EntryStatus = "draft"
	EntryPosted   EntryStatus = "posted"
	EntryReversed EntryStatus = "reversed"
)

// JournalLine = một dòng bút toán. Debit and Credit are VND minor units
// (int64); exactly one side must be positive on every line (R2).
type JournalLine struct {
	LineNo      int    `json:"line_no"`
	AccountCode string `json:"account_code"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
	Note        string `json:"note,omitempty"`
}

// JournalEntry = một bút toán trong Sổ Nhật ký chung. Period is "YYYY-MM"
// derived from VoucherDate. VoucherNo "{form}-{5-digit}/{YY}" is assigned when
// the entry is posted (R10); drafts stay unnumbered.
type JournalEntry struct {
	ID          string        `json:"id"`
	VoucherNo   string        `json:"voucher_no,omitempty"`
	VoucherDate string        `json:"voucher_date"`
	Period      string        `json:"period"`
	Source      EntrySource   `json:"source"`
	SourceRef   string        `json:"source_ref,omitempty"`
	Description string        `json:"description"`
	Lines       []JournalLine `json:"lines"`
	Status      EntryStatus   `json:"status"`
	CreatedBy   string        `json:"created_by"`
	PostedBy    string        `json:"posted_by,omitempty"`
	PostedAt    string        `json:"posted_at,omitempty"`
	ReversedBy  string        `json:"reversed_by,omitempty"`
	ReversedOf  string        `json:"reversed_of,omitempty"`
}

// AccountFilter filters accounts on the read path (type, parent, q).
type AccountFilter struct {
	Type       AccountType
	ParentCode string
	Q          string
}

// EntryFilter filters journal entries on the read path (P2/P3 books).
type EntryFilter struct {
	Period   string
	Source   EntrySource
	Status   EntryStatus
	FromDate string
	ToDate   string
}

type PeriodStatus string

const (
	PeriodOpen   PeriodStatus = "open"
	PeriodClosed PeriodStatus = "closed"
)

// AccountingPeriod = một kỳ kế toán ("2026-08"). Postings are rejected while
// the period is CLOSED (R4).
type AccountingPeriod struct {
	ID          string       `json:"id"`
	Year        int          `json:"year"`
	Month       int          `json:"month"`
	Status      PeriodStatus `json:"status"`
	OpenedBy    string       `json:"opened_by,omitempty"`
	ClosedBy    string       `json:"closed_by,omitempty"`
	ClosedAt    string       `json:"closed_at,omitempty"`
	CloseReason string       `json:"close_reason,omitempty"`
}

// Balance is a Nợ/Có pair with exactly one side above zero (a signed net amount
// split at its sign). Used by the statutory books for Số dư đầu/đi kỳ.
type Balance struct {
	Debit  int64 `json:"debit"`
	Credit int64 `json:"credit"`
}

// BalanceOf splits a signed net amount (Nợ positive, Có negative) into a
// Nợ/Có pair. A zero net yields a zero pair.
func BalanceOf(net int64) Balance {
	switch {
	case net > 0:
		return Balance{Debit: net}
	case net < 0:
		return Balance{Credit: -net}
	default:
		return Balance{}
	}
}

// BookRow is one rendered line of a statutory book. Contra (TKHN — đối ứng)
// lists the entry's other account codes, comma-joined. Balance is the running
// balance for Sổ Cái (Nợ positive, Có negative); Sổ chi tiết leaves it zero.
type BookRow struct {
	VoucherDate string `json:"voucher_date"`
	VoucherNo   string `json:"voucher_no"`
	Description string `json:"description"`
	Contra      string `json:"contra"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
	Balance     int64  `json:"balance,omitempty"`
}

// GeneralJournal = Sổ Nhật ký chung (UC-L5). Rows are per entry-line, ordered
// by (VoucherDate, VoucherNo). Column totals always balance for POSTED entries.
// Total is the row count before paging — equal to len(Rows) for an unpaged book.
type GeneralJournal struct {
	FromPeriod  string    `json:"from_period"`
	ToPeriod    string    `json:"to_period"`
	Rows        []BookRow `json:"rows"`
	TotalDebit  int64     `json:"total_debit"`
	TotalCredit int64     `json:"total_credit"`
	Total       int       `json:"total"`
}

// LedgerBook = Sổ Cái / Sổ chi tiết tài khoản (UC-L3). Open balance carries
// forward from posted activity in periods strictly before FromPeriod; Close =
// Open + period activity. Số dư render as Nợ/Có pairs, one side above zero.
type LedgerBook struct {
	AccountCode string    `json:"account_code"`
	AccountName string    `json:"account_name"`
	FromPeriod  string    `json:"from_period"`
	ToPeriod    string    `json:"to_period"`
	OpenDebit   int64     `json:"open_debit"`
	OpenCredit  int64     `json:"open_credit"`
	Rows        []BookRow `json:"rows"`
	TotalDebit  int64     `json:"total_debit"`
	TotalCredit int64     `json:"total_credit"`
	CloseDebit  int64     `json:"close_debit"`
	CloseCredit int64     `json:"close_credit"`
	Total       int       `json:"total"`
}

// TrialBalanceRow is one BCĐPS (S06) row: per-account opening, period activity
// and closing balances, each a Nợ/Có pair.
type TrialBalanceRow struct {
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	Open        Balance `json:"open"`
	Activity    Balance `json:"activity"`
	Close       Balance `json:"close"`
}

// TrialBalance = Bảng cân đối số phát sinh (UC-L4). Balanced is true when the
// totals row balances on every column (ΣNợ = ΣCó), which holds for a store of
// balanced POSTED entries (R1) and is asserted on render (E6).
type TrialBalance struct {
	Period   string            `json:"period"`
	Rows     []TrialBalanceRow `json:"rows"`
	Totals   TrialBalanceRow   `json:"totals"`
	Balanced bool              `json:"balanced"`
	Total    int               `json:"total"`
}

// Page is an offset/limit paging request for book reads. A nil Page returns
// the full book; a non-nil Page must carry Limit >= 1. Book services set the
// Total field to the full row count regardless of the window.
type Page struct {
	Offset int
	Limit  int
}

// VoucherSeq tracks the per-form-per-period voucher number (R10). One row per
// (form, period) lives in the ledger_sequences table. The counter only ever
// increments, so deleted or reversed entries never cause a number to be
// reused.
type VoucherSeq struct {
	Form   string `json:"form"`
	Period string `json:"period"`
	N      int64  `json:"n"`
}

// FormatVoucherNo renders "{form}-{5-digit}/{YY}" (e.g. "PK-00045/26") from a
// per-form-per-period counter. The two-digit year is derived from the
// "YYYY-MM" accounting period.
func FormatVoucherNo(form string, n int64, period string) string {
	yy := ""
	if len(period) == 7 {
		yy = period[2:4]
	}
	return fmt.Sprintf("%s-%05d/%s", form, n, yy)
}

// Repository is the persistence contract for the ledger module. Rows are
// JSON-document tables in the shape (id TEXT PRIMARY KEY, data TEXT NOT NULL).
type Repository interface {
	CreateEntry(ctx context.Context, e *JournalEntry) error
	UpdateEntry(ctx context.Context, e *JournalEntry) error
	GetEntry(ctx context.Context, id string) (*JournalEntry, error)
	GetEntryBySource(ctx context.Context, source EntrySource, ref string) (*JournalEntry, error)
	ListEntries(ctx context.Context, f EntryFilter) ([]*JournalEntry, error)
	// PostEntry atomically transitions a DRAFT entry to POSTED inside one
	// transaction: it acquires the next VoucherNo for form+period (R10) unless
	// the entry already carries a VoucherNo from its source system, then
	// CAS-updates the row. The R5 duplicate-key guard runs in the same locked
	// statement: when another entry already owns (Source, SourceRef) it returns
	// that entry (idempotent retry), or ErrDuplicateSource while that entry is
	// still DRAFT; ErrWrongState when this entry is no longer DRAFT.
	PostEntry(ctx context.Context, e *JournalEntry, form string) (*JournalEntry, error)
	DeleteEntry(ctx context.Context, id string) error

	CreateAccount(ctx context.Context, a *Account) error
	UpdateAccount(ctx context.Context, a *Account) error
	GetAccount(ctx context.Context, id string) (*Account, error)
	GetAccountByCode(ctx context.Context, code string) (*Account, error)
	ListAccounts(ctx context.Context) ([]*Account, error)

	CreatePeriod(ctx context.Context, p *AccountingPeriod) error
	GetPeriod(ctx context.Context, id string) (*AccountingPeriod, error)
	ListPeriods(ctx context.Context) ([]*AccountingPeriod, error)
}

// RowID derives a deterministic SHA-256 row id for a document key so that
// re-saving the same logical document is an upsert (the (id, data) table
// shape documented in AGENTS.md). All ledger-module row ids derive from it.
func RowID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// PeriodFromDate derives the "YYYY-MM" accounting period from a "YYYY-MM-DD"
// date. An empty or malformed input yields "" so the caller can reject it.
func PeriodFromDate(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return ""
}
