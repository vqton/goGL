package bank

import (
	"context"

	"goGL/internal/domain/core"
)

type BankTransaction struct {
	ID        string      `json:"id" bson:"_id"`
	AccountNo string      `json:"account_no" bson:"account_no"`
	BankCode  string      `json:"bank_code" bson:"bank_code"`
	RefDate   string      `json:"ref_date" bson:"ref_date"`
	Amount    core.Money  `json:"amount" bson:"amount"`
	Status    core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, t *BankTransaction) error
	FindByID(ctx context.Context, id string) (*BankTransaction, error)
	Update(ctx context.Context, t *BankTransaction) error
}
