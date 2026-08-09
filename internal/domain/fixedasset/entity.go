package fixedasset

import (
	"context"

	"goGL/internal/domain/core"
)

type FixedAsset struct {
	ID                 string      `json:"id" bson:"_id"`
	Code               string      `json:"code" bson:"code"`
	Name               string      `json:"name" bson:"name"`
	Cost               core.Money  `json:"cost" bson:"cost"`
	DepreciationMethod string      `json:"depreciation_method" bson:"depreciation_method"`
	Status             core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, a *FixedAsset) error
	FindByID(ctx context.Context, id string) (*FixedAsset, error)
	Update(ctx context.Context, a *FixedAsset) error
}
