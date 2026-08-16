package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"goGL/internal/domain/core"
)

// ProfileID is the fixed row id of the single company profile (R1: one
// company per install).
const ProfileID = "company"

// SetupStatus is the initialize state machine (docs/setup/02-spec §2.2).
type SetupStatus string

const (
	StatusEmpty          SetupStatus = "empty"           // nothing yet
	StatusProfiled       SetupStatus = "profiled"        // profile saved
	StatusRegimeSet      SetupStatus = "regime_set"      // masterdata regime set
	StatusAccountsSeeded SetupStatus = "accounts_seeded" // COA seeded
	StatusPeriodsOpen    SetupStatus = "periods_open"    // ledger FY periods opened
	StatusBalancesDraft  SetupStatus = "balances_draft"  // opening balances entered
	StatusBalancesLocked SetupStatus = "balances_locked" // approved/locked
	StatusActive         SetupStatus = "active"          // go live (period 1 postable)
)

// statusOrder is the monotonic progression. R6 forbids any backward move
// except the explicit reopen (BALANCES_LOCKED -> BALANCES_DRAFT).
var statusOrder = []SetupStatus{
	StatusEmpty,
	StatusProfiled,
	StatusRegimeSet,
	StatusAccountsSeeded,
	StatusPeriodsOpen,
	StatusBalancesDraft,
	StatusBalancesLocked,
	StatusActive,
}

// Index returns the position of s in the monotonic order, or -1 if unknown.
func (s SetupStatus) Index() int {
	for i, st := range statusOrder {
		if st == s {
			return i
		}
	}
	return -1
}

// Label returns the Vietnamese display label for the status.
func (s SetupStatus) Label() string {
	switch s {
	case StatusEmpty:
		return "Chưa khởi tạo"
	case StatusProfiled:
		return "Đã lưu thông tin doanh nghiệp"
	case StatusRegimeSet:
		return "Đã chọn chế độ kế toán"
	case StatusAccountsSeeded:
		return "Đã tạo sơ đồ tài khoản"
	case StatusPeriodsOpen:
		return "Đã mở kỳ kế toán"
	case StatusBalancesDraft:
		return "Đang nhập số dư đầu kỳ"
	case StatusBalancesLocked:
		return "Đã khóa số dư đầu kỳ"
	case StatusActive:
		return "Đã kích hoạt"
	default:
		return string(s)
	}
}

// CompanyProfile is the statutory identity of the single company (R1–R5).
// Fields mirror docs/setup/02-spec §2.1 (TT 99/2025 Điều 5/11/31, Luật Kế
// toán Điều 12, NĐ 254/2026).
type CompanyProfile struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	NameEN              string      `json:"name_en,omitempty"`
	TaxCode             string      `json:"tax_code"`
	BudgetUnitCode      string      `json:"budget_unit_code,omitempty"`
	Address             string      `json:"address"`
	LegalRepresentative string      `json:"legal_representative"`
	CompanyType         string      `json:"company_type,omitempty"`
	Industry            string      `json:"industry,omitempty"`
	AccountingCurrency  string      `json:"accounting_currency"`
	FiscalYearStart     string      `json:"fiscal_year_start"`
	AccountingRegime    string      `json:"accounting_regime"`
	BooksStartDate      string      `json:"books_start_date,omitempty"`
	Status              SetupStatus `json:"status"`
	CreatedBy           string      `json:"created_by,omitempty"`
	CreatedAt           string      `json:"created_at,omitempty"`
	UpdatedBy           string      `json:"updated_by,omitempty"`
	UpdatedAt           string      `json:"updated_at,omitempty"`
}

// Clone returns a deep copy safe for callers to mutate.
func (p *CompanyProfile) Clone() *CompanyProfile {
	cp := *p
	return &cp
}

// BalanceStatus is the per-balance lifecycle (draft until lock, R8).
type BalanceStatus string

const (
	BalanceDraft  BalanceStatus = "draft"
	BalanceLocked BalanceStatus = "locked"
)

// OpeningBalance is one opening-balance row, per TK + per đối tượng
// (docs/setup/02-spec §2.3, R7/R10). Exactly one of Debit/Credit is above
// zero; amounts are int64 VND minor units via core.Money.
type OpeningBalance struct {
	ID          string        `json:"id"`
	AccountCode string        `json:"account_code"`
	ObjectType  string        `json:"object_type,omitempty"`
	ObjectCode  string        `json:"object_code,omitempty"`
	Period      core.Period   `json:"period"`
	Debit       core.Money    `json:"debit"`
	Credit      core.Money    `json:"credit"`
	Status      BalanceStatus `json:"status"`
	EnteredBy   string        `json:"entered_by,omitempty"`
	EnteredAt   string        `json:"entered_at,omitempty"`
	UpdatedBy   string        `json:"updated_by,omitempty"`
	UpdatedAt   string        `json:"updated_at,omitempty"`
}

// Clone returns a deep copy safe for callers to mutate.
func (b *OpeningBalance) Clone() *OpeningBalance {
	cp := *b
	return &cp
}

// BalanceID derives the deterministic row id: sha256("OB\x00"+account+
// "\x00"+object) hex. Deterministic ⇒ re-saving the same logical balance is an
// upsert (idempotent, R7). Hashed (not literal "OB:{acct}:{obj}") so TKs and
// codes containing colons never collide or leak into the row id.
func BalanceID(account, object string) string {
	h := sha256.New()
	h.Write([]byte("OB"))
	h.Write([]byte{0})
	h.Write([]byte(account))
	h.Write([]byte{0})
	h.Write([]byte(object))
	return hex.EncodeToString(h.Sum(nil))
}

// BalanceCheck is the R9 summary: totals, diff, the accounts that break
// the ΣNợ == ΣCó invariant (Offending), and the object-required gaps (R10).
type BalanceCheck struct {
	Debit     int64    `json:"debit"`
	Credit    int64    `json:"credit"`
	Diff      int64    `json:"diff"`
	Balanced  bool     `json:"balanced"`
	Offending []string `json:"offending,omitempty"`
	Gaps      []string `json:"gaps,omitempty"`
}

// SupportedRegimes are the accounting regimes setup may select (R5).
// Mirror of docs/setup/02-spec §3 R5 = {"TT99-2025","TT133-2016"}. The seam
// (masterdata.SetRegime) validates again against its own whitelist.
var SupportedRegimes = map[string]bool{
	"TT99-2025":  true,
	"TT133-2016": true,
}

// NormalizeTaxCode strips all whitespace (R3, mirrors masterdata).
func NormalizeTaxCode(v string) string {
	return strings.ReplaceAll(strings.TrimSpace(v), " ", "")
}

// RowError reports a single CSV row import problem.
type RowError struct {
	Row     int    `json:"row"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

// ImportResult summarizes a CSV opening-balance import (create/update counts
// + per-row errors). Mirror of masterdata.ImportResult.
type ImportResult struct {
	Total   int        `json:"total"`
	Created int        `json:"created"`
	Updated int        `json:"updated"`
	Errors  []RowError `json:"errors,omitempty"`
	DryRun  bool       `json:"dry_run"`
}

// Repository persists the setup module's own data: the single company
// profile, opening balances and the status machine row. All rows use the
// shared (id, data) JSON-document shape.
type Repository interface {
	SaveProfile(ctx context.Context, p *CompanyProfile) error
	GetProfile(ctx context.Context, id string) (*CompanyProfile, error)
	SaveBalance(ctx context.Context, b *OpeningBalance) error
	ListBalances(ctx context.Context, accountCode string) ([]*OpeningBalance, error)
	DeleteBalance(ctx context.Context, id string) error
	GetStatus(ctx context.Context) (SetupStatus, error)
	SetStatus(ctx context.Context, s SetupStatus) error
}
