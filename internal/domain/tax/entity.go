package tax

import (
	"context"

	"goGL/internal/domain/core"
)

type TaxDeclaration struct {
	ID      string      `json:"id" bson:"_id"`
	TaxType string      `json:"tax_type" bson:"tax_type"`
	Period  core.Period `json:"period" bson:"period"`
	RefNo   string      `json:"ref_no" bson:"ref_no"`
	Status  core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, d *TaxDeclaration) error
	FindByID(ctx context.Context, id string) (*TaxDeclaration, error)
	Update(ctx context.Context, d *TaxDeclaration) error
}
