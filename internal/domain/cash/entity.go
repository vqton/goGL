package cash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

type VoucherType string

const (
	VoucherReceive VoucherType = "receive"
	VoucherPay     VoucherType = "pay"
)

type VoucherState string

const (
	VoucherDraft      VoucherState = "draft"
	VoucherApproved   VoucherState = "approved"
	VoucherPosted     VoucherState = "posted"
	VoucherReconciled VoucherState = "reconciled"
	VoucherVoided     VoucherState = "voided"
)

// Voucher = phiếu thu/chi (Mẫu 01-TT / 02-TT). AmountMinor is in the fund
// currency's minor units; AmountWords is the statutory VN number-in-words.
type Voucher struct {
	ID               string        `json:"id"`
	RefNo            string        `json:"ref_no"`
	RefDate          string        `json:"ref_date"`
	Type             VoucherType   `json:"type"`
	FundID           string        `json:"fund_id"`
	Currency         string        `json:"currency"`
	AmountMinor      int64         `json:"amount_minor"`
	AmountWords      string        `json:"amount_words"`
	FXRate           float64       `json:"fx_rate,omitempty"`
	CounterpartyType string        `json:"counterparty_type"`
	CounterpartyID   string        `json:"counterparty_id"`
	CounterpartyName string        `json:"counterparty_name"`
	Description      string        `json:"description"`
	RefVouchers      []string      `json:"ref_vouchers,omitempty"`
	Lines            []VoucherLine `json:"lines"`
	State            VoucherState  `json:"state"`
	CreatedBy        string        `json:"created_by"`
	ApprovedBy       string        `json:"approved_by,omitempty"`
	PostedBy         string        `json:"posted_by,omitempty"`
	ReceiverName     string        `json:"receiver_name"`
	ApprovedAt       string        `json:"approved_at,omitempty"`
	PostedAt         string        `json:"posted_at,omitempty"`
	VoidedBy         string        `json:"voided_by,omitempty"`
	VoidReason       string        `json:"void_reason,omitempty"`
	VoidedAt         string        `json:"voided_at,omitempty"`
}

type VoucherLine struct {
	Seq         int    `json:"seq"`
	DebitAcc    string `json:"debit_acc"`
	CreditAcc   string `json:"credit_acc"`
	AmountMinor int64  `json:"amount_minor"`
	ObjectID    string `json:"object_id,omitempty"`
}

// Fund = a cash balance per currency (quỹ). BR6: one fund per currency.
// Code is the stable business code (e.g. "QTM-VND"); OpeningBalanceMinor is
// the opening balance in minor units carried into the fund; FXRate applies to
// foreign-currency funds; Cashiers lists the cashier user codes responsible
// for the fund. ClosedDays holds yyyy-mm-dd dates on which posting is blocked
// (BR8); ClosedPeriods holds closed yyyy-mm accounting periods (BR9).
type Fund struct {
	ID                  string   `json:"id"`
	Code                string   `json:"code,omitempty"`
	Name                string   `json:"name"`
	Currency            string   `json:"currency"`
	Account             string   `json:"account"`
	OpeningBalanceMinor int64    `json:"opening_balance_minor,omitempty"`
	FXRate              float64  `json:"fx_rate,omitempty"`
	Cashiers            []string `json:"cashiers,omitempty"`
	Description         string   `json:"description,omitempty"`
	IsActive            bool     `json:"is_active"`
	ClosedDays          []string `json:"closed_days,omitempty"`
	ClosedPeriods       []string `json:"closed_periods,omitempty"`
}

// CashBookEntry = row of Sổ quỹ tiền mặt (S07-DN).
type CashBookEntry struct {
	ID          string      `json:"id"`
	FundID      string      `json:"fund_id"`
	EntryDate   string      `json:"entry_date"`
	VoucherDate string      `json:"voucher_date"`
	RefNo       string      `json:"ref_no"`
	Type        VoucherType `json:"type"`
	Description string      `json:"description"`
	Receive     int64       `json:"receive"`
	Pay         int64       `json:"pay"`
	Balance     int64       `json:"balance"`
	Reconciled  bool        `json:"reconciled,omitempty"`
}

// CashCountState = trạng thái biên bản kiểm kê quỹ.
type CashCountState string

const (
	CashCountOpen     CashCountState = "open"
	CashCountResolved CashCountState = "resolved"
)

// ReconciliationState = trạng thái biên bản đối chiếu quỹ cuối tháng (UC-5).
type ReconciliationState string

const (
	ReconciliationDiff     ReconciliationState = "diff"
	ReconciliationResolved ReconciliationState = "resolved"
)

// CashCount = biên bản kiểm kê quỹ.
type CashCount struct {
	ID            string         `json:"id"`
	FundID        string         `json:"fund_id"`
	CountDate     string         `json:"count_date"`
	BookBalance   int64          `json:"book_balance"`
	CountedAmount int64          `json:"counted_amount"`
	Difference    int64          `json:"difference"`
	Resolution    string         `json:"resolution,omitempty"`
	Participants  []string       `json:"participants"`
	State         CashCountState `json:"state"`
}

// Reconciliation = biên bản đối chiếu quỹ cuối tháng (UC-5). SignedBy holds
// the three electronic signatories (thủ quỹ, kế toán, kế toán trưởng).
type Reconciliation struct {
	ID                string              `json:"id"`
	FundID            string              `json:"fund_id"`
	Period            string              `json:"period"`
	CashierBalance    int64               `json:"cashier_balance"`
	AccountantBalance int64               `json:"accountant_balance"`
	Difference        int64               `json:"difference"`
	State             ReconciliationState `json:"state"`
	SignedBy          []string            `json:"signed_by,omitempty"`
	CreatedAt         string              `json:"created_at"`
}

type VoucherFilter struct {
	FundID string
	State  VoucherState
	From   string
	To     string
	Type   VoucherType
}

type Repository interface {
	CreateFund(ctx context.Context, f *Fund) error
	ListFunds(ctx context.Context) ([]*Fund, error)
	GetFund(ctx context.Context, id string) (*Fund, error)

	CreateVoucher(ctx context.Context, v *Voucher) error
	UpdateVoucher(ctx context.Context, v *Voucher) error
	GetVoucher(ctx context.Context, id string) (*Voucher, error)
	ListVouchers(ctx context.Context, f VoucherFilter) ([]*Voucher, error)
	NextRefNo(ctx context.Context, fundID, period string, typ VoucherType) (string, error)

	ListCashBook(ctx context.Context, fundID, from, to string) ([]*CashBookEntry, error)
	AppendCashBookEntry(ctx context.Context, e *CashBookEntry) error
	DeleteCashBookEntry(ctx context.Context, id string) error
	UpdateCashBookEntry(ctx context.Context, e *CashBookEntry) error

	CreateCashCount(ctx context.Context, c *CashCount) error
	GetCashCount(ctx context.Context, id string) (*CashCount, error)
	ListCashCounts(ctx context.Context, fundID string) ([]*CashCount, error)

	CreateReconciliation(ctx context.Context, r *Reconciliation) error
	GetReconciliation(ctx context.Context, id string) (*Reconciliation, error)
	ListReconciliations(ctx context.Context, fundID string) ([]*Reconciliation, error)
}

// RowID derives a deterministic SHA-256 row id for a document key so that
// re-saving the same logical document is an upsert (the (id, data) table
// shape documented in AGENTS.md). All cash-module row ids derive from it.
func RowID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
