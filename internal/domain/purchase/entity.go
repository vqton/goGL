package purchase

import (
	"context"

	"goGL/internal/domain/core"
)

type PurchaseInvoice struct {
	ID           string      `json:"id" bson:"_id"`
	RefNo        string      `json:"ref_no" bson:"ref_no"`
	SupplierCode string      `json:"supplier_code" bson:"supplier_code"`
	RefDate      string      `json:"ref_date" bson:"ref_date"`
	Amount       core.Money  `json:"amount" bson:"amount"`
	VATAmount    core.Money  `json:"vat_amount" bson:"vat_amount"`
	Status       core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, p *PurchaseInvoice) error
	FindByID(ctx context.Context, id string) (*PurchaseInvoice, error)
	Update(ctx context.Context, p *PurchaseInvoice) error
}
