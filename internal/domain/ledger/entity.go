package ledger

import (
	"context"

	"goGL/internal/domain/core"
)

type JournalLine struct {
	AccountCode string     `json:"account_code" bson:"account_code"`
	Debit       core.Money `json:"debit" bson:"debit"`
	Credit      core.Money `json:"credit" bson:"credit"`
}

type JournalEntry struct {
	ID          string        `json:"id" bson:"_id"`
	VoucherNo   string        `json:"voucher_no" bson:"voucher_no"`
	VoucherDate string        `json:"voucher_date" bson:"voucher_date"`
	Lines       []JournalLine `json:"lines" bson:"lines"`
	Status      core.Status   `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, e *JournalEntry) error
	FindByID(ctx context.Context, id string) (*JournalEntry, error)
	Update(ctx context.Context, e *JournalEntry) error
}
