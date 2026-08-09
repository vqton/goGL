package masterdata

import (
	"context"

	"goGL/internal/domain/core"
)

type CatalogItem struct {
	ID       string      `json:"id" bson:"_id"`
	Category string      `json:"category" bson:"category"`
	Code     string      `json:"code" bson:"code"`
	Name     string      `json:"name" bson:"name"`
	Status   core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, i *CatalogItem) error
	FindByID(ctx context.Context, id string) (*CatalogItem, error)
	Update(ctx context.Context, i *CatalogItem) error
}
