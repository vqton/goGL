package sales

import (
	"context"

	"goGL/internal/domain/core"
)

type SalesInvoice struct {
	ID           string      `json:"id" bson:"_id"`
	RefNo        string      `json:"ref_no" bson:"ref_no"`
	CustomerCode string      `json:"customer_code" bson:"customer_code"`
	RefDate      string      `json:"ref_date" bson:"ref_date"`
	Amount       core.Money  `json:"amount" bson:"amount"`
	VATAmount    core.Money  `json:"vat_amount" bson:"vat_amount"`
	Status       core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, s *SalesInvoice) error
	FindByID(ctx context.Context, id string) (*SalesInvoice, error)
	Update(ctx context.Context, s *SalesInvoice) error
}
