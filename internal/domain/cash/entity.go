package cash

import (
	"context"
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
}

type VoucherLine struct {
	Seq         int    `json:"seq"`
	DebitAcc    string `json:"debit_acc"`
	CreditAcc   string `json:"credit_acc"`
	AmountMinor int64  `json:"amount_minor"`
	ObjectID    string `json:"object_id,omitempty"`
}

// Fund = a cash balance per currency (quỹ). BR6: one fund per currency.
type Fund struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Currency    string `json:"currency"`
	Account     string `json:"account"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
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

// CashCount = biên bản kiểm kê quỹ.
type CashCount struct {
	ID            string   `json:"id"`
	FundID        string   `json:"fund_id"`
	CountDate     string   `json:"count_date"`
	BookBalance   int64    `json:"book_balance"`
	CountedAmount int64    `json:"counted_amount"`
	Difference    int64    `json:"difference"`
	Resolution    string   `json:"resolution,omitempty"`
	Participants  []string `json:"participants"`
	State         string   `json:"state"`
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

	CreateCashCount(ctx context.Context, c *CashCount) error
	ListCashCounts(ctx context.Context, fundID string) ([]*CashCount, error)
}
