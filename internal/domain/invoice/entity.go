package invoice

import (
	"context"

	"goGL/internal/domain/core"
)

type Invoice struct {
	ID        string      `json:"id" bson:"_id"`
	SerialNo  string      `json:"serial_no" bson:"serial_no"`
	InvoiceNo string      `json:"invoice_no" bson:"invoice_no"`
	TaxCode   string      `json:"tax_code" bson:"tax_code"`
	IssueDate string      `json:"issue_date" bson:"issue_date"`
	Amount    core.Money  `json:"amount" bson:"amount"`
	Status    core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, i *Invoice) error
	FindByID(ctx context.Context, id string) (*Invoice, error)
	Update(ctx context.Context, i *Invoice) error
}
